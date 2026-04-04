// SPDX-License-Identifier: MIT

// Package directory implements the filemanager_directory resource.
package directory

import (
	"context"
	"fmt"
	"os/user"
	"strconv"

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
	_ resource.Resource                = &DirectoryResource{}
	_ resource.ResourceWithImportState = &DirectoryResource{}
)

// NewDirectoryResource creates a new directory resource.
func NewDirectoryResource() resource.Resource {
	return &DirectoryResource{}
}

// DirectoryResource defines the resource implementation.
type DirectoryResource struct {
	config *common.ProviderConfig
}

// DirectoryResourceModel describes the resource data model.
type DirectoryResourceModel struct {
	ID            types.String `tfsdk:"id"`
	Path          types.String `tfsdk:"path"`
	Service       types.String `tfsdk:"service"`
	Permission    types.String `tfsdk:"permission"`
	CreateParents types.Bool   `tfsdk:"create_parents"`
	ForceDelete   types.Bool   `tfsdk:"force_delete"`
	UID           types.Int64  `tfsdk:"uid"`
	GID           types.Int64  `tfsdk:"gid"`
	Owner         types.String `tfsdk:"owner"`
	Group         types.String `tfsdk:"group"`

	// Computed
	Exists       types.Bool   `tfsdk:"exists"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *DirectoryResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

// Schema defines the schema for the resource.
func (r *DirectoryResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Manages a directory on the filesystem.",
		MarkdownDescription: "Manages a directory on the filesystem or remote backend.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path of the directory to manage.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service to use for operations. Defaults to local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"permission": schema.StringAttribute{
				Description: "Permission mode in octal format (e.g., \"0755\").",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"create_parents": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"force_delete": schema.BoolAttribute{
				Description: "Delete directory even if not empty (recursive delete).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"uid": schema.Int64Attribute{
				Description: "User ID for directory ownership (Unix only).",
				Optional:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "Group ID for directory ownership (Unix only).",
				Optional:    true,
			},
			"owner": schema.StringAttribute{
				Description: "User name for directory ownership. Alternative to uid.",
				Optional:    true,
			},
			"group": schema.StringAttribute{
				Description: "Group name for directory ownership. Alternative to gid.",
				Optional:    true,
			},
			"exists": schema.BoolAttribute{
				Description: "Whether the directory exists.",
				Computed:    true,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the directory.",
				Computed:    true,
			},
			"extension": schema.StringAttribute{
				Description: "The file extension without the leading dot (empty for directories).",
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
func (r *DirectoryResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *DirectoryResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DirectoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating directory", map[string]any{
		"path": data.Path.ValueString(),
	})

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Create directory
	mkdirOpts := plugin.MkdirOptions{
		Mode:      common.ParseDirMode(data.Permission.ValueString()),
		Recursive: data.CreateParents.ValueBool(),
	}

	if err := backend.Mkdir(ctx, data.Path.ValueString(), mkdirOpts); err != nil {
		if err != plugin.ErrPathExists {
			resp.Diagnostics.AddError("Failed to create directory", err.Error())
			return
		}
	}

	// Set ownership if specified
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	// Set permissions explicitly (mkdir might not set exact mode due to umask)
	mode := common.ParseDirMode(data.Permission.ValueString())
	if err := backend.Chmod(ctx, data.Path.ValueString(), mode); err != nil {
		if err != plugin.ErrNotSupported {
			resp.Diagnostics.AddError(
				"Failed to set directory permissions",
				fmt.Sprintf("Directory was created but permissions could not be set: %s", err),
			)
			return
		}
	}

	data.ID = data.Path
	data.Exists = types.BoolValue(true)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	tflog.Debug(ctx, "Directory created successfully", map[string]any{
		"path": data.Path.ValueString(),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *DirectoryResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DirectoryResourceModel

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
		resp.Diagnostics.AddError("Failed to check directory existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Verify it's a directory
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat directory", err.Error())
		return
	}

	if !info.IsDir {
		resp.Diagnostics.AddError(
			"Path is not a directory",
			fmt.Sprintf("Expected %s to be a directory but it is a file", data.Path.ValueString()),
		)
		return
	}

	data.Exists = types.BoolValue(true)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *DirectoryResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DirectoryResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating directory", map[string]any{
		"path": data.Path.ValueString(),
	})

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Update permissions
	mode := common.ParseDirMode(data.Permission.ValueString())
	if err := backend.Chmod(ctx, data.Path.ValueString(), mode); err != nil {
		if err != plugin.ErrNotSupported {
			resp.Diagnostics.AddError("Failed to set directory permissions", err.Error())
				return
		}
	}

	// Update ownership
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	data.Exists = types.BoolValue(true)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *DirectoryResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DirectoryResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting directory", map[string]any{
		"path":         data.Path.ValueString(),
		"force_delete": data.ForceDelete.ValueBool(),
	})

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	if err := backend.Rmdir(ctx, data.Path.ValueString(), data.ForceDelete.ValueBool()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete directory", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "Directory deleted successfully", map[string]any{
		"path": data.Path.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *DirectoryResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// getBackend returns the appropriate backend.
func (r *DirectoryResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// setOwnership sets directory ownership if specified.
func (r *DirectoryResource) setOwnership(ctx context.Context, backend plugin.Backend, path string, data *DirectoryResourceModel) error {
	uid := -1
	gid := -1

	// Numeric UID takes precedence
	if !data.UID.IsNull() {
		uid = int(data.UID.ValueInt64())
	} else if !data.Owner.IsNull() && data.Owner.ValueString() != "" {
		// Lookup by name
		u, err := user.Lookup(data.Owner.ValueString())
		if err != nil {
			return fmt.Errorf("failed to lookup user %s: %w", data.Owner.ValueString(), err)
		}
		uidInt, _ := strconv.Atoi(u.Uid)
		uid = uidInt
	}

	// Numeric GID takes precedence
	if !data.GID.IsNull() {
		gid = int(data.GID.ValueInt64())
	} else if !data.Group.IsNull() && data.Group.ValueString() != "" {
		// Lookup by name
		g, err := user.LookupGroup(data.Group.ValueString())
		if err != nil {
			return fmt.Errorf("failed to lookup group %s: %w", data.Group.ValueString(), err)
		}
		gidInt, _ := strconv.Atoi(g.Gid)
		gid = gidInt
	}

	if uid >= 0 || gid >= 0 {
		if err := backend.Chown(ctx, path, uid, gid); err != nil {
			if err != plugin.ErrNotSupported {
				return err
			}
		}
	}

	return nil
}
