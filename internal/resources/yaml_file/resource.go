// SPDX-License-Identifier: MIT

// Package yaml_file implements the filemanager_yaml_file resource.
package yaml_file

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	yamlFormat "github.com/ebogdum/filemanager/internal/formats/yaml"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &YAMLFileResource{}
	_ resource.ResourceWithImportState = &YAMLFileResource{}
)

// NewYAMLFileResource creates a new YAML file resource.
func NewYAMLFileResource() resource.Resource {
	return &YAMLFileResource{
		format: yamlFormat.New(),
	}
}

// YAMLFileResource defines the resource implementation.
type YAMLFileResource struct {
	config *common.ProviderConfig
	format *yamlFormat.Format
}

// YAMLFileResourceModel describes the resource data model.
type YAMLFileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	Content             types.String `tfsdk:"content"`
	MergeWith           types.String `tfsdk:"merge_with"`
	MergeStrategy       types.String `tfsdk:"merge_strategy"`
	Indent              types.Int64  `tfsdk:"indent"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`
	AtomicWrite         types.Bool   `tfsdk:"atomic_write"`

	// Computed
	Size         types.Int64  `tfsdk:"size"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	Rendered     types.String `tfsdk:"rendered"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *YAMLFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_yaml_file"
}

// Schema defines the schema for the resource.
func (r *YAMLFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a YAML file with deep merge support.",
		MarkdownDescription: `Manages a YAML file with support for deep merge and structured content manipulation.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path where the YAML file will be created.",
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
			"content": schema.StringAttribute{
				Description: "The YAML content as a string. Use yamlencode() to convert HCL to YAML.",
				Required:    true,
			},
			"merge_with": schema.StringAttribute{
				Description: "Additional YAML content to merge with the main content. Use yamlencode() to convert HCL to YAML.",
				Optional:    true,
			},
			"merge_strategy": schema.StringAttribute{
				Description: "Merge strategy: replace, deep, append, concat, union.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("deep"),
			},
			"indent": schema.Int64Attribute{
				Description: "Indentation spaces for YAML output.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
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
				Description: "Deprecated insecure checksum field. Always null.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the file content.",
				Computed:    true,
			},
			"rendered": schema.StringAttribute{
				Description: "The rendered YAML content as a string.",
				Computed:    true,
				Sensitive:   true,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the file (e.g., 'config.json').",
				Computed:    true,
			},
			"extension": schema.StringAttribute{
				Description: "The file extension without the leading dot (e.g., 'json').",
				Computed:    true,
			},
			"absolute_path": schema.StringAttribute{
				Description: "The absolute resolved path.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *YAMLFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *YAMLFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data YAMLFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating YAML file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Generate YAML content
	content, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate YAML content", err.Error())
		return
	}

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Write file
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

	// Update computed values
	r.updateComputedValues(&data, content)
	data.ID = data.Path

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *YAMLFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data YAMLFileResourceModel

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

	r.updateComputedValues(&data, content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *YAMLFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data YAMLFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating YAML file", map[string]any{
		"path": data.Path.ValueString(),
	})

	content, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate YAML content", err.Error())
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

	r.updateComputedValues(&data, content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *YAMLFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data YAMLFileResourceModel

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
func (r *YAMLFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// generateContent generates the final YAML content.
func (r *YAMLFileResource) generateContent(ctx context.Context, data *YAMLFileResourceModel) ([]byte, error) {
	// Parse the YAML content string
	content, err := r.format.Parse([]byte(data.Content.ValueString()))
	if err != nil {
		return nil, fmt.Errorf("invalid YAML in content: %w", err)
	}

	// Merge if merge_with is specified
	if !data.MergeWith.IsNull() && !data.MergeWith.IsUnknown() && data.MergeWith.ValueString() != "" {
		mergeWith, err := r.format.Parse([]byte(data.MergeWith.ValueString()))
		if err != nil {
			return nil, fmt.Errorf("invalid YAML in merge_with: %w", err)
		}

		strategy := plugin.MergeStrategy(data.MergeStrategy.ValueString())
		content, err = r.format.Merge(content, mergeWith, strategy)
		if err != nil {
			return nil, fmt.Errorf("failed to merge content: %w", err)
		}
	}

	// Serialize to YAML
	opts := plugin.SerializeOptions{
		Indent:          int(data.Indent.ValueInt64()),
		TrailingNewline: true,
	}

	return r.format.Serialize(content, opts)
}

// getBackend returns the appropriate backend.
func (r *YAMLFileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *YAMLFileResource) updateComputedValues(data *YAMLFileResourceModel, content []byte) {
	data.Size = types.Int64Value(int64(len(content)))
	data.Rendered = types.StringValue(string(content))

	data.MD5 = types.StringNull()

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
