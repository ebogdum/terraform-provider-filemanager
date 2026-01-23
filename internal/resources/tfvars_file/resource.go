// SPDX-License-Identifier: MIT

// Package tfvars_file implements the filemanager_tfvars_file resource.
package tfvars_file

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TfvarsFileResource{}
	_ resource.ResourceWithImportState = &TfvarsFileResource{}
)

// NewTfvarsFileResource creates a new tfvars file resource.
func NewTfvarsFileResource() resource.Resource {
	return &TfvarsFileResource{}
}

// TfvarsFileResource defines the resource implementation.
type TfvarsFileResource struct {
	config *common.ProviderConfig
}

// TfvarsFileResourceModel describes the resource data model.
type TfvarsFileResourceModel struct {
	ID                  types.String  `tfsdk:"id"`
	Path                types.String  `tfsdk:"path"`
	Service             types.String  `tfsdk:"service"`
	Vars                types.Dynamic `tfsdk:"vars"`
	TemplateVars        types.Map     `tfsdk:"template_vars"`
	DeleteVars          types.List    `tfsdk:"delete_vars"`
	MergeWithExisting   types.Bool    `tfsdk:"merge_with_existing"`
	MergeStrategy       types.String  `tfsdk:"merge_strategy"`
	SortKeys            types.Bool    `tfsdk:"sort_keys"`
	JSONFormat          types.Bool    `tfsdk:"json_format"`
	LeftDelim           types.String  `tfsdk:"left_delim"`
	RightDelim          types.String  `tfsdk:"right_delim"`
	FilePermission      types.String  `tfsdk:"file_permission"`
	DirectoryPermission types.String  `tfsdk:"directory_permission"`
	CreateParentDirs    types.Bool    `tfsdk:"create_parent_dirs"`
	AtomicWrite         types.Bool    `tfsdk:"atomic_write"`

	// Computed
	Size         types.Int64  `tfsdk:"size"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	Rendered     types.String `tfsdk:"rendered"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
	VarCount     types.Int64  `tfsdk:"var_count"`
}

// Metadata returns the resource type name.
func (r *TfvarsFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tfvars_file"
}

// Schema defines the schema for the resource.
func (r *TfvarsFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a Terraform .tfvars file with native dynamic type support and variable interpolation.",
		MarkdownDescription: `
Manages a Terraform ` + "`.tfvars`" + ` file with native dynamic type support, internal variable interpolation, and merge capabilities.

## Features

- **Native types**: Pass maps, lists, numbers, booleans directly via the ` + "`vars`" + ` attribute
- **Internal interpolation**: Variables can reference each other using ` + "`{{ .var_name }}`" + ` syntax
- **Merge with existing**: Preserve unmanaged variables in existing files
- **Variable-level operations**: Set/delete individual variables
- **Dual format**: Output as HCL (` + "`.tfvars`" + `) or JSON (` + "`.tfvars.json`" + `)
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path where the tfvars file will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"vars": schema.DynamicAttribute{
				Description: "Variables to write to the tfvars file. Accepts any Terraform object with native types (strings, numbers, booleans, lists, maps).",
				Required:    true,
			},
			"template_vars": schema.MapAttribute{
				Description: "Additional template variables for interpolation. These are available as {{ .key }} in string values but are not written to the output file.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"delete_vars": schema.ListAttribute{
				Description: "List of variable names to delete from the output (useful with merge_with_existing).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"merge_with_existing": schema.BoolAttribute{
				Description: "When true, merges vars with any existing file content instead of replacing entirely.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"merge_strategy": schema.StringAttribute{
				Description: "How to handle conflicts when merging: 'replace' (new vars win) or 'keep_existing' (existing values preserved).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("replace"),
				Validators: []validator.String{
					stringvalidator.OneOf("replace", "keep_existing"),
				},
			},
			"sort_keys": schema.BoolAttribute{
				Description: "Sort variable names alphabetically in the output.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"json_format": schema.BoolAttribute{
				Description: "Output in JSON format (.tfvars.json) instead of HCL (.tfvars).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"left_delim": schema.StringAttribute{
				Description: "Left delimiter for template interpolation.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{{"),
			},
			"right_delim": schema.StringAttribute{
				Description: "Right delimiter for template interpolation.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("}}"),
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode in octal format.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode for parent directories.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"create_parent_dirs": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"atomic_write": schema.BoolAttribute{
				Description: "Use atomic write operations.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "MD5 checksum of the file content.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the file content.",
				Computed:    true,
			},
			"rendered": schema.StringAttribute{
				Description: "The rendered tfvars content as a string.",
				Computed:    true,
				Sensitive:   true,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the file.",
				Computed:    true,
			},
			"extension": schema.StringAttribute{
				Description: "The file extension without the leading dot.",
				Computed:    true,
			},
			"absolute_path": schema.StringAttribute{
				Description: "The absolute resolved path.",
				Computed:    true,
			},
			"var_count": schema.Int64Attribute{
				Description: "Number of variables in the output file.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *TfvarsFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*common.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.config = config
}

// Create creates the resource.
func (r *TfvarsFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TfvarsFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating tfvars file", map[string]any{
		"path": data.Path.ValueString(),
	})

	content, varCount, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate tfvars content", err.Error())
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	writeOpts := plugin.WriteOptions{
		Mode:       common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:    common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs: data.CreateParentDirs.ValueBool(),
		Overwrite:  true,
		Atomic:     data.AtomicWrite.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	r.updateComputedValues(&data, content, varCount)
	data.ID = data.Path

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *TfvarsFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TfvarsFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	reader, err := backend.Read(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file", err.Error())
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file content", err.Error())
		return
	}

	// Count vars in the existing file
	varCount := int64(0)
	filePath := data.Path.ValueString()
	format := detectFormat(filePath)
	if "json" == format {
		parsed, parseErr := ParseTfvarsJSON(content)
		if nil == parseErr {
			varCount = int64(len(parsed))
		}
	} else {
		parsed, parseErr := ParseTfvarsHCL(content)
		if nil == parseErr {
			varCount = int64(len(parsed))
		}
	}

	r.updateComputedValues(&data, content, varCount)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *TfvarsFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TfvarsFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating tfvars file", map[string]any{
		"path": data.Path.ValueString(),
	})

	content, varCount, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate tfvars content", err.Error())
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	writeOpts := plugin.WriteOptions{
		Mode:       common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:    common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs: data.CreateParentDirs.ValueBool(),
		Overwrite:  true,
		Atomic:     data.AtomicWrite.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	r.updateComputedValues(&data, content, varCount)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *TfvarsFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TfvarsFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	if err := backend.Delete(ctx, data.Path.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete file", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *TfvarsFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// generateContent generates the final tfvars content.
func (r *TfvarsFileResource) generateContent(ctx context.Context, data *TfvarsFileResourceModel) ([]byte, int64, error) {
	// Convert Vars (types.Dynamic) to map[string]any
	goVars, err := common.TerraformDynamicToGoValue(ctx, data.Vars)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to convert vars: %w", err)
	}

	vars, ok := goVars.(map[string]any)
	if !ok {
		if nil == goVars {
			vars = make(map[string]any)
		} else {
			return nil, 0, fmt.Errorf("vars must be an object/map, got %T", goVars)
		}
	}

	// Extract template_vars
	var templateVars map[string]string
	if !data.TemplateVars.IsNull() && !data.TemplateVars.IsUnknown() {
		elems := data.TemplateVars.Elements()
		templateVars = make(map[string]string, len(elems))
		for k, v := range elems {
			templateVars[k] = v.(types.String).ValueString()
		}
	}

	// Resolve interpolation
	leftDelim := data.LeftDelim.ValueString()
	rightDelim := data.RightDelim.ValueString()

	resolved, err := ResolveInterpolation(vars, templateVars, leftDelim, rightDelim)
	if err != nil {
		return nil, 0, fmt.Errorf("interpolation error: %w", err)
	}

	// Merge with existing if requested
	if data.MergeWithExisting.ValueBool() {
		backend, backendErr := r.getBackend(ctx, data.Service.ValueString())
		if nil == backendErr {
			exists, existsErr := backend.Exists(ctx, data.Path.ValueString())
			if nil == existsErr && exists {
				reader, readErr := backend.Read(ctx, data.Path.ValueString())
				if nil == readErr {
					existingContent, ioErr := io.ReadAll(reader)
					reader.Close()
					if nil == ioErr && len(existingContent) > 0 {
						var existing map[string]any
						format := detectFormat(data.Path.ValueString())
						if "json" == format {
							existing, _ = ParseTfvarsJSON(existingContent)
						} else {
							existing, _ = ParseTfvarsHCL(existingContent)
						}

						if nil != existing {
							strategy := data.MergeStrategy.ValueString()
							if "keep_existing" == strategy {
								// Existing values take precedence
								for k, v := range resolved {
									if _, exists := existing[k]; !exists {
										existing[k] = v
									}
								}
								resolved = existing
							} else {
								// "replace": resolved vars override existing
								for k, v := range existing {
									if _, exists := resolved[k]; !exists {
										resolved[k] = v
									}
								}
							}
						}
					}
				}
			}
		}
	}

	// Apply delete_vars
	if !data.DeleteVars.IsNull() && !data.DeleteVars.IsUnknown() {
		elems := data.DeleteVars.Elements()
		for _, elem := range elems {
			key := elem.(types.String).ValueString()
			delete(resolved, key)
		}
	}

	// Serialize
	var content []byte
	if data.JSONFormat.ValueBool() {
		content, err = SerializeTfvarsJSON(resolved, "  ")
	} else {
		content, err = SerializeTfvars(resolved, data.SortKeys.ValueBool())
	}
	if err != nil {
		return nil, 0, err
	}

	return content, int64(len(resolved)), nil
}

// getBackend returns the appropriate backend.
func (r *TfvarsFileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if "" == backendName || "local" == backendName {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *TfvarsFileResource) updateComputedValues(data *TfvarsFileResourceModel, content []byte, varCount int64) {
	data.Size = types.Int64Value(int64(len(content)))
	data.Rendered = types.StringValue(string(content))
	data.VarCount = types.Int64Value(varCount)

	md5sum := md5.Sum(content)
	data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
