// SPDX-License-Identifier: MIT

// Package swift_operation implements the filemanager_swift_operation resource.
package swift_operation

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

	"github.com/ebogdum/filemanager/internal/backends/swift"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SwiftOperationResource{}
	_ resource.ResourceWithImportState = &SwiftOperationResource{}
)

// Supported Swift operations.
const (
	OpHead        = "head"
	OpCopy        = "copy"
	OpSetMetadata = "set_metadata"
)

// NewSwiftOperationResource creates a new Swift operation resource.
func NewSwiftOperationResource() resource.Resource {
	return &SwiftOperationResource{}
}

// SwiftOperationResource defines the resource implementation.
type SwiftOperationResource struct {
	config *common.ProviderConfig
}

// SwiftOperationResourceModel describes the resource data model.
type SwiftOperationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Service         types.String `tfsdk:"service"`
	ObjectPath      types.String `tfsdk:"object_path"`
	Operation       types.String `tfsdk:"operation"`
	DestinationPath types.String `tfsdk:"destination_path"`
	Metadata        types.Map    `tfsdk:"metadata"`

	// Computed outputs
	Size            types.Int64  `tfsdk:"size"`
	LastModified    types.String `tfsdk:"last_modified"`
	ContentType     types.String `tfsdk:"content_type"`
	ETag            types.String `tfsdk:"etag"`
	IsDir           types.Bool   `tfsdk:"is_dir"`
	Name            types.String `tfsdk:"name"`
	CurrentMetadata types.Map    `tfsdk:"current_metadata"`
}

// Metadata returns the resource type name.
func (r *SwiftOperationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swift_operation"
}

// Schema defines the schema for the resource.
func (r *SwiftOperationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Performs operations on OpenStack Swift objects such as head, copy, or set metadata.",
		MarkdownDescription: `Performs operations on OpenStack Swift objects such as head, copy, or set metadata.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Swift service alias (from filemanager_swift_service resource).",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"object_path": schema.StringAttribute{
				Description: "Path to the object in the Swift container.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operation": schema.StringAttribute{
				Description: "Operation to perform. Valid values: head, copy, set_metadata.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(OpHead, OpCopy, OpSetMetadata),
				},
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path for copy operation.",
				Optional:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Metadata to set on the object (for set_metadata operation).",
				Optional:    true,
				ElementType: types.StringType,
			},

			// Computed outputs
			"size": schema.Int64Attribute{
				Description: "Size of the object in bytes.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modification time of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"content_type": schema.StringAttribute{
				Description: "Content type of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"etag": schema.StringAttribute{
				Description: "Entity tag (hash) of the object.",
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
				Description: "Base name of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"current_metadata": schema.MapAttribute{
				Description: "Current metadata on the object.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *SwiftOperationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SwiftOperationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SwiftOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Swift operation", map[string]any{
		"service":     data.Service.ValueString(),
		"object_path": data.ObjectPath.ValueString(),
		"operation":   data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform Swift operation", err.Error())
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		data.Service.ValueString(), data.ObjectPath.ValueString(), data.Operation.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *SwiftOperationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SwiftOperationResourceModel

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

	// Determine which path to check (for copy, check destination)
	pathToCheck := data.ObjectPath.ValueString()
	if data.Operation.ValueString() == OpCopy && !data.DestinationPath.IsNull() {
		pathToCheck = data.DestinationPath.ValueString()
	}

	// Check if object still exists
	exists, err := backend.Exists(ctx, pathToCheck)
	if err != nil {
		resp.Diagnostics.AddError("Failed to check object existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Refresh computed values
	if err := r.refreshObjectInfo(ctx, backend, &data, pathToCheck); err != nil {
		resp.Diagnostics.AddError("Failed to refresh object info", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *SwiftOperationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SwiftOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Swift operation", map[string]any{
		"service":     data.Service.ValueString(),
		"object_path": data.ObjectPath.ValueString(),
		"operation":   data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform Swift operation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *SwiftOperationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SwiftOperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Swift operation resources are metadata-only; we don't delete the actual object
	tflog.Debug(ctx, "Deleting Swift operation resource (no object deletion)", map[string]any{
		"service":     data.Service.ValueString(),
		"object_path": data.ObjectPath.ValueString(),
		"operation":   data.Operation.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *SwiftOperationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// validateOperation validates operation-specific configuration.
func (r *SwiftOperationResource) validateOperation(ctx context.Context, data *SwiftOperationResourceModel) error {
	op := data.Operation.ValueString()

	switch op {
	case OpCopy:
		if data.DestinationPath.IsNull() || data.DestinationPath.ValueString() == "" {
			return fmt.Errorf("destination_path is required for copy operation")
		}
	case OpSetMetadata:
		if data.Metadata.IsNull() {
			return fmt.Errorf("metadata is required for set_metadata operation")
		}
	case OpHead:
		// No additional validation required
	}

	return nil
}

// performOperation executes the Swift operation.
func (r *SwiftOperationResource) performOperation(ctx context.Context, data *SwiftOperationResourceModel) error {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	op := data.Operation.ValueString()

	tflog.Info(ctx, "Performing Swift operation", map[string]any{
		"operation":   op,
		"object_path": data.ObjectPath.ValueString(),
	})

	switch op {
	case OpHead:
		return r.performHead(ctx, backend, data)
	case OpCopy:
		return r.performCopy(ctx, backend, data)
	case OpSetMetadata:
		return r.performSetMetadata(ctx, backend, data)
	default:
		return fmt.Errorf("unsupported operation: %s", op)
	}
}

// performHead retrieves object metadata.
func (r *SwiftOperationResource) performHead(ctx context.Context, backend plugin.Backend, data *SwiftOperationResourceModel) error {
	return r.refreshObjectInfo(ctx, backend, data, data.ObjectPath.ValueString())
}

// performCopy copies an object.
func (r *SwiftOperationResource) performCopy(ctx context.Context, backend plugin.Backend, data *SwiftOperationResourceModel) error {
	swiftBack, ok := backend.(*swift.Backend)
	if !ok {
		return fmt.Errorf("backend does not support Swift operations")
	}

	tflog.Debug(ctx, "Copying Swift object", map[string]any{
		"source":      data.ObjectPath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	opts := plugin.WriteOptions{
		CreateDirs: true,
		Overwrite:  true,
	}

	if err := swiftBack.CopyFile(ctx, data.ObjectPath.ValueString(), data.DestinationPath.ValueString(), opts); err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return r.refreshObjectInfo(ctx, backend, data, data.DestinationPath.ValueString())
}

// performSetMetadata sets metadata on an object.
func (r *SwiftOperationResource) performSetMetadata(ctx context.Context, backend plugin.Backend, data *SwiftOperationResourceModel) error {
	swiftBack, ok := backend.(*swift.Backend)
	if !ok {
		return fmt.Errorf("backend does not support Swift operations")
	}

	tflog.Debug(ctx, "Setting Swift object metadata", map[string]any{
		"object_path": data.ObjectPath.ValueString(),
	})

	// Convert types.Map to map[string]string
	metadata := make(map[string]string)
	if !data.Metadata.IsNull() {
		elements := data.Metadata.Elements()
		for key, val := range elements {
			if strVal, ok := val.(types.String); ok {
				metadata[key] = strVal.ValueString()
			}
		}
	}

	if err := swiftBack.SetMetadata(ctx, data.ObjectPath.ValueString(), metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	return r.refreshObjectInfo(ctx, backend, data, data.ObjectPath.ValueString())
}

// refreshObjectInfo retrieves current object information.
func (r *SwiftOperationResource) refreshObjectInfo(ctx context.Context, backend plugin.Backend, data *SwiftOperationResourceModel, objectPath string) error {
	info, err := backend.Stat(ctx, objectPath)
	if err != nil {
		if err == plugin.ErrPathNotFound {
			r.setPlaceholderValues(ctx, data)
			return nil
		}
		return fmt.Errorf("failed to stat object: %w", err)
	}

	// Map FileInfo to computed values
	data.Size = types.Int64Value(info.Size)
	data.LastModified = types.StringValue(info.ModTime.Format("2006-01-02T15:04:05Z07:00"))
	data.ContentType = types.StringValue(info.ContentType)
	data.ETag = types.StringValue(info.ETag)
	data.IsDir = types.BoolValue(info.IsDir)
	data.Name = types.StringValue(info.Name)

	// Get current metadata
	swiftBack, ok := backend.(*swift.Backend)
	if ok {
		metadata, err := swiftBack.GetMetadata(ctx, objectPath)
		if err == nil && metadata != nil {
			metadataMap := make(map[string]types.String)
			for key, value := range metadata {
				metadataMap[key] = types.StringValue(value)
			}
			mapVal, diags := types.MapValueFrom(ctx, types.StringType, metadataMap)
			if !diags.HasError() {
				data.CurrentMetadata = mapVal
			} else {
				data.CurrentMetadata = types.MapNull(types.StringType)
			}
		} else {
			data.CurrentMetadata = types.MapNull(types.StringType)
		}
	} else {
		data.CurrentMetadata = types.MapNull(types.StringType)
	}

	return nil
}

// setPlaceholderValues sets placeholder values for computed attributes.
func (r *SwiftOperationResource) setPlaceholderValues(ctx context.Context, data *SwiftOperationResourceModel) {
	data.Size = types.Int64Value(0)
	data.LastModified = types.StringValue("")
	data.ContentType = types.StringValue("")
	data.ETag = types.StringValue("")
	data.IsDir = types.BoolValue(false)
	data.Name = types.StringValue("")
	data.CurrentMetadata = types.MapNull(types.StringType)
}

// getBackend returns the appropriate backend.
func (r *SwiftOperationResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}
