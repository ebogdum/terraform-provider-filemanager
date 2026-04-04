// SPDX-License-Identifier: MIT

// Package symlink implements the filemanager_symlink resource.
package symlink

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

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
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SymlinkResource{}
	_ resource.ResourceWithImportState = &SymlinkResource{}
)

// NewSymlinkResource creates a new symlink resource.
func NewSymlinkResource() resource.Resource {
	return &SymlinkResource{}
}

// SymlinkResource defines the resource implementation.
type SymlinkResource struct {
	config *common.ProviderConfig
}

// SymlinkResourceModel describes the resource data model.
type SymlinkResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Path             types.String `tfsdk:"path"`
	Target           types.String `tfsdk:"target"`
	TargetType       types.String `tfsdk:"target_type"`
	Service          types.String `tfsdk:"service"`
	CreateParentDirs types.Bool   `tfsdk:"create_parent_dirs"`
	Force            types.Bool   `tfsdk:"force"`

	// Computed
	Exists       types.Bool   `tfsdk:"exists"`
	ResolvedPath types.String `tfsdk:"resolved_path"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *SymlinkResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_symlink"
}

// Schema defines the schema for the resource.
func (r *SymlinkResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a symbolic link.",
		MarkdownDescription: "Manages a symbolic link on the filesystem.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path of the symbolic link to create.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"target": schema.StringAttribute{
				Description: "The target path that the symlink points to.",
				Required:    true,
			},
			"target_type": schema.StringAttribute{
				Description: "How to treat the target path: 'absolute' resolves relative targets to absolute paths (symlink breaks if directory tree moves), 'relative' keeps the target as-is (symlink is portable). Defaults to 'absolute'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("absolute"),
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"create_parent_dirs": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"force": schema.BoolAttribute{
				Description: "Remove existing file/symlink at path before creating.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"exists": schema.BoolAttribute{
				Description: "Whether the symbolic link exists.",
				Computed:    true,
			},
			"resolved_path": schema.StringAttribute{
				Description: "The fully resolved target path.",
				Computed:    true,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the symlink.",
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
		},
	}
}

// Configure configures the resource with provider data.
func (r *SymlinkResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SymlinkResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SymlinkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating symlink", map[string]any{
		"path":   data.Path.ValueString(),
		"target": data.Target.ValueString(),
	})

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Create parent directories if needed
	if data.CreateParentDirs.ValueBool() {
		parentDir := filepath.Dir(data.Path.ValueString())
		if parentDir != "" && parentDir != "." {
			mkdirOpts := plugin.MkdirOptions{
				Mode:      0755,
				Recursive: true,
			}
			if err := backend.Mkdir(ctx, parentDir, mkdirOpts); err != nil && err != plugin.ErrPathExists {
				resp.Diagnostics.AddError("Failed to create parent directories", err.Error())
				return
			}
		}
	}

	// Remove existing if force is enabled
	if data.Force.ValueBool() {
		exists, _ := backend.Exists(ctx, data.Path.ValueString())
		if exists {
			if err := backend.Delete(ctx, data.Path.ValueString()); err != nil {
				resp.Diagnostics.AddError("Failed to remove existing path", err.Error())
				return
			}
		}
	}

	// Resolve target path based on target_type
	targetPath := data.Target.ValueString()
	if data.TargetType.ValueString() == "absolute" && !filepath.IsAbs(targetPath) {
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			resp.Diagnostics.AddError("Failed to resolve target path", err.Error())
			return
		}
		targetPath = absTarget
	}

	// Create symlink
	if err := backend.Symlink(ctx, targetPath, data.Path.ValueString()); err != nil {
		if err == plugin.ErrNotSupported {
			resp.Diagnostics.AddError(
				"Symlinks not supported",
				"The current backend does not support symbolic links",
			)
			return
		}
		resp.Diagnostics.AddError("Failed to create symlink", err.Error())
		return
	}

	// Update computed values
	data.ID = data.Path
	data.Exists = types.BoolValue(true)
	data.ResolvedPath = types.StringValue(targetPath)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *SymlinkResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SymlinkResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Check if symlink exists
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		if err == plugin.ErrPathNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to stat symlink", err.Error())
		return
	}

	// Verify it's a symlink (Mode will have ModeSymlink set)
	if info.Mode&os.ModeSymlink == 0 {
		// Path exists but is not a symlink
		resp.Diagnostics.AddWarning(
			"Path is not a symlink",
			fmt.Sprintf("%s exists but is not a symbolic link", data.Path.ValueString()),
		)
	}

	// Read actual symlink target from disk for drift detection
	linkPath := data.Path.ValueString()
	actualTarget, readlinkErr := os.Readlink(linkPath)
	if readlinkErr != nil {
		if os.IsNotExist(readlinkErr) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read symlink", readlinkErr.Error())
		return
	}
	// Update target if it has changed (drift detection)
	data.Target = types.StringValue(actualTarget)

	// Compute resolved path based on target_type
	targetPath := actualTarget
	if data.TargetType.ValueString() == "absolute" && !filepath.IsAbs(targetPath) {
		if absTarget, err := filepath.Abs(targetPath); err == nil {
			targetPath = absTarget
		}
	}

	data.Exists = types.BoolValue(true)
	data.ResolvedPath = types.StringValue(targetPath)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *SymlinkResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SymlinkResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating symlink", map[string]any{
		"path":   data.Path.ValueString(),
		"target": data.Target.ValueString(),
	})

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Remove existing symlink
	if err := backend.Delete(ctx, data.Path.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to remove existing symlink", err.Error())
			return
		}
	}

	// Resolve target path based on target_type
	targetPath := data.Target.ValueString()
	if data.TargetType.ValueString() == "absolute" && !filepath.IsAbs(targetPath) {
		absTarget, err := filepath.Abs(targetPath)
		if err != nil {
			resp.Diagnostics.AddError("Failed to resolve target path", err.Error())
			return
		}
		targetPath = absTarget
	}

	// Create new symlink
	if err := backend.Symlink(ctx, targetPath, data.Path.ValueString()); err != nil {
		resp.Diagnostics.AddError("Failed to create symlink", err.Error())
		return
	}

	data.Exists = types.BoolValue(true)
	data.ResolvedPath = types.StringValue(targetPath)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *SymlinkResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SymlinkResourceModel

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
			resp.Diagnostics.AddError("Failed to delete symlink", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *SymlinkResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// getBackend returns the appropriate backend.
func (r *SymlinkResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

