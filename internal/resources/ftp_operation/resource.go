// SPDX-License-Identifier: MIT

// Package ftp_operation implements the filemanager_ftp_operation resource.
package ftp_operation

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	ftpbackend "github.com/ebogdum/filemanager/internal/backends/ftp"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &FTPOperationResource{}
	_ resource.ResourceWithImportState = &FTPOperationResource{}
)

// Supported FTP operations.
const (
	OpHead   = "head"
	OpCopy   = "copy"
	OpRename = "rename"
)

// NewFTPOperationResource creates a new FTP operation resource.
func NewFTPOperationResource() resource.Resource {
	return &FTPOperationResource{}
}

// FTPOperationResource defines the resource implementation.
type FTPOperationResource struct {
	config *common.ProviderConfig
}

// FTPOperationResourceModel describes the resource data model.
type FTPOperationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Service         types.String `tfsdk:"service"`
	Path            types.String `tfsdk:"path"`
	Operation       types.String `tfsdk:"operation"`
	DestinationPath types.String `tfsdk:"destination_path"`

	// Computed outputs
	Size        types.Int64  `tfsdk:"size"`
	ModTime     types.String `tfsdk:"mod_time"`
	Permissions types.String `tfsdk:"permissions"`
	IsDir       types.Bool   `tfsdk:"is_dir"`
	Name        types.String `tfsdk:"name"`
}

// Metadata returns the resource type name.
func (r *FTPOperationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ftp_operation"
}

// Schema defines the schema for the resource.
func (r *FTPOperationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Performs operations on FTP files such as head, copy, or rename.",
		MarkdownDescription: `Performs operations on FTP files such as head, copy, or rename.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "FTP service alias (from filemanager_ftp_service resource).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"path": schema.StringAttribute{
				Description: "Path to the file on the FTP server.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operation": schema.StringAttribute{
				Description: "Operation to perform. Valid values: head, copy, rename.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(OpHead, OpCopy, OpRename),
				},
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path for copy or rename operations.",
				Optional:    true,
			},

			// Computed outputs
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"mod_time": schema.StringAttribute{
				Description: "Last modification time of the file.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"permissions": schema.StringAttribute{
				Description: "File permissions in octal format.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"is_dir": schema.BoolAttribute{
				Description: "Whether the path is a directory.",
				Computed:    true,
			},
			"name": schema.StringAttribute{
				Description: "Base name of the file.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *FTPOperationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *FTPOperationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FTPOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating FTP operation", map[string]any{
		"service":   data.Service.ValueString(),
		"path":      data.Path.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform FTP operation", err.Error())
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		data.Service.ValueString(), data.Path.ValueString(), data.Operation.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *FTPOperationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FTPOperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Determine which path to check (for rename, check destination)
	pathToCheck := data.Path.ValueString()
	if data.Operation.ValueString() == OpRename && !data.DestinationPath.IsNull() {
		pathToCheck = data.DestinationPath.ValueString()
	}

	// Check if file still exists
	exists, err := backend.Exists(ctx, pathToCheck)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Refresh computed values
	if err := r.refreshFileInfo(ctx, backend, &data, pathToCheck); err != nil {
		resp.Diagnostics.AddError("Failed to refresh file info", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *FTPOperationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FTPOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating FTP operation", map[string]any{
		"service":   data.Service.ValueString(),
		"path":      data.Path.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform FTP operation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *FTPOperationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FTPOperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// FTP operation resources are metadata-only; we don't delete the actual file
	tflog.Debug(ctx, "Deleting FTP operation resource (no file deletion)", map[string]any{
		"service":   data.Service.ValueString(),
		"path":      data.Path.ValueString(),
		"operation": data.Operation.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *FTPOperationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// validateOperation validates operation-specific configuration.
func (r *FTPOperationResource) validateOperation(ctx context.Context, data *FTPOperationResourceModel) error {
	op := data.Operation.ValueString()

	switch op {
	case OpCopy, OpRename:
		if data.DestinationPath.IsNull() || data.DestinationPath.ValueString() == "" {
			return fmt.Errorf("destination_path is required for %s operation", op)
		}
	case OpHead:
		// No additional validation required
	}

	return nil
}

// performOperation executes the FTP operation.
func (r *FTPOperationResource) performOperation(ctx context.Context, data *FTPOperationResourceModel) error {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	op := data.Operation.ValueString()

	tflog.Info(ctx, "Performing FTP operation", map[string]any{
		"operation": op,
		"path":      data.Path.ValueString(),
	})

	switch op {
	case OpHead:
		return r.performHead(ctx, backend, data)
	case OpCopy:
		return r.performCopy(ctx, backend, data)
	case OpRename:
		return r.performRename(ctx, backend, data)
	default:
		return fmt.Errorf("unsupported operation: %s", op)
	}
}

// performHead retrieves file metadata.
func (r *FTPOperationResource) performHead(ctx context.Context, backend plugin.Backend, data *FTPOperationResourceModel) error {
	return r.refreshFileInfo(ctx, backend, data, data.Path.ValueString())
}

// performCopy copies a file.
func (r *FTPOperationResource) performCopy(ctx context.Context, backend plugin.Backend, data *FTPOperationResourceModel) error {
	ftpBack, ok := backend.(*ftpbackend.Backend)
	if !ok {
		return fmt.Errorf("backend does not support FTP operations")
	}

	tflog.Debug(ctx, "Copying FTP file", map[string]any{
		"source":      data.Path.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	opts := plugin.WriteOptions{
		CreateDirs: true,
		Overwrite:  true,
	}

	if err := ftpBack.CopyFile(ctx, data.Path.ValueString(), data.DestinationPath.ValueString(), opts); err != nil {
		return fmt.Errorf("failed to copy file: %w", err)
	}

	return r.refreshFileInfo(ctx, backend, data, data.DestinationPath.ValueString())
}

// performRename renames/moves a file.
func (r *FTPOperationResource) performRename(ctx context.Context, backend plugin.Backend, data *FTPOperationResourceModel) error {
	ftpBack, ok := backend.(*ftpbackend.Backend)
	if !ok {
		return fmt.Errorf("backend does not support FTP operations")
	}

	tflog.Debug(ctx, "Renaming FTP file", map[string]any{
		"source":      data.Path.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	if err := ftpBack.Rename(ctx, data.Path.ValueString(), data.DestinationPath.ValueString()); err != nil {
		return fmt.Errorf("failed to rename file: %w", err)
	}

	return r.refreshFileInfo(ctx, backend, data, data.DestinationPath.ValueString())
}

// refreshFileInfo retrieves current file information.
func (r *FTPOperationResource) refreshFileInfo(ctx context.Context, backend plugin.Backend, data *FTPOperationResourceModel, filePath string) error {
	info, err := backend.Stat(ctx, filePath)
	if err != nil {
		if err == plugin.ErrPathNotFound {
			r.setPlaceholderValues(data)
			return nil
		}
		return fmt.Errorf("failed to stat file: %w", err)
	}

	// Map FileInfo to computed values
	data.Size = types.Int64Value(info.Size)
	data.ModTime = types.StringValue(info.ModTime.Format("2006-01-02T15:04:05Z07:00"))
	data.Permissions = types.StringValue(fmt.Sprintf("%04o", info.Mode.Perm()))
	data.IsDir = types.BoolValue(info.IsDir)
	data.Name = types.StringValue(info.Name)

	return nil
}

// setPlaceholderValues sets placeholder values for computed attributes.
func (r *FTPOperationResource) setPlaceholderValues(data *FTPOperationResourceModel) {
	data.Size = types.Int64Value(0)
	data.ModTime = types.StringValue("")
	data.Permissions = types.StringValue("")
	data.IsDir = types.BoolValue(false)
	data.Name = types.StringValue("")
}

// getBackend returns the appropriate backend.
func (r *FTPOperationResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}
