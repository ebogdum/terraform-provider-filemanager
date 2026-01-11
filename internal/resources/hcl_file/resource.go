// SPDX-License-Identifier: MIT

// Package hcl_file implements the filemanager_hcl_file resource.
package hcl_file

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	hclFormat "github.com/ebogdum/filemanager/internal/formats/hcl"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &HCLFileResource{}
	_ resource.ResourceWithImportState = &HCLFileResource{}
)

// NewHCLFileResource creates a new HCL file resource.
func NewHCLFileResource() resource.Resource {
	return &HCLFileResource{
		format: hclFormat.New(),
	}
}

// HCLFileResource defines the resource implementation.
type HCLFileResource struct {
	config *common.ProviderConfig
	format *hclFormat.Format
}

// HCLFileResourceModel describes the resource data model.
type HCLFileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	Content             types.String `tfsdk:"content"`
	MergeWith           types.String `tfsdk:"merge_with"`
	MergeStrategy       types.String `tfsdk:"merge_strategy"`
	SortKeys            types.Bool   `tfsdk:"sort_keys"`
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
func (r *HCLFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hcl_file"
}

// Schema defines the schema for the resource.
func (r *HCLFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages an HCL file (Terraform/HashiCorp configuration format).",
		MarkdownDescription: `
Manages an HCL file with support for merge and structured content manipulation.

HCL (HashiCorp Configuration Language) is used by Terraform, Consul, Vault, and other HashiCorp tools.

## Content Format

Content should be provided as JSON (use jsonencode()) and will be converted to HCL format.
Blocks are represented as nested objects. Block labels can be specified using a special "__labels__" key.
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
				Description: "The path where the HCL file will be created.",
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
				Description: "The content as a JSON string (use jsonencode()). Will be converted to HCL.",
				Required:    true,
			},
			"merge_with": schema.StringAttribute{
				Description: "Additional content to merge (as JSON string). Use jsonencode() to convert HCL.",
				Optional:    true,
			},
			"merge_strategy": schema.StringAttribute{
				Description: "Merge strategy: replace, deep, append, concat, union.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("deep"),
			},
			"sort_keys": schema.BoolAttribute{
				Description: "Sort HCL attribute names alphabetically.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
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
				Description: "The rendered HCL content as a string.",
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
func (r *HCLFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *HCLFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data HCLFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating HCL file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Generate HCL content
	content, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate HCL content", err.Error())
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
func (r *HCLFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data HCLFileResourceModel

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
func (r *HCLFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data HCLFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating HCL file", map[string]any{
		"path": data.Path.ValueString(),
	})

	content, err := r.generateContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate HCL content", err.Error())
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
func (r *HCLFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data HCLFileResourceModel

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
func (r *HCLFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// generateContent generates the final HCL content.
func (r *HCLFileResource) generateContent(ctx context.Context, data *HCLFileResourceModel) ([]byte, error) {
	// Parse the JSON content string into a map
	var content any
	if err := json.Unmarshal([]byte(data.Content.ValueString()), &content); err != nil {
		return nil, fmt.Errorf("invalid JSON in content: %w", err)
	}

	// Merge if merge_with is specified
	if !data.MergeWith.IsNull() && !data.MergeWith.IsUnknown() && data.MergeWith.ValueString() != "" {
		var mergeWith any
		if err := json.Unmarshal([]byte(data.MergeWith.ValueString()), &mergeWith); err != nil {
			return nil, fmt.Errorf("invalid JSON in merge_with: %w", err)
		}

		strategy := plugin.MergeStrategy(data.MergeStrategy.ValueString())
		var err error
		content, err = r.format.Merge(content, mergeWith, strategy)
		if err != nil {
			return nil, fmt.Errorf("failed to merge content: %w", err)
		}
	}

	// Serialize to HCL
	opts := plugin.SerializeOptions{
		SortKeys:        data.SortKeys.ValueBool(),
		TrailingNewline: true,
	}

	return r.format.Serialize(content, opts)
}

// getBackend returns the appropriate backend.
func (r *HCLFileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *HCLFileResource) updateComputedValues(data *HCLFileResourceModel, content []byte) {
	data.Size = types.Int64Value(int64(len(content)))
	data.Rendered = types.StringValue(string(content))

	md5sum := md5.Sum(content)
	data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
