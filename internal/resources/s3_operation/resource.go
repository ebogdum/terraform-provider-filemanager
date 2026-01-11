// SPDX-License-Identifier: MIT

// Package s3_operation implements the filemanager_s3_operation resource.
package s3_operation

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	s3backend "github.com/ebogdum/filemanager/internal/backends/s3"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3OperationResource{}
	_ resource.ResourceWithImportState = &S3OperationResource{}
)

// Supported S3 operations.
const (
	OpHead            = "head"
	OpCopy            = "copy"
	OpSetMetadata     = "set_metadata"
	OpSetTags         = "set_tags"
	OpDeleteTags      = "delete_tags"
	OpSetStorageClass = "set_storage_class"
	OpRestore         = "restore"
)

// NewS3OperationResource creates a new S3 operation resource.
func NewS3OperationResource() resource.Resource {
	return &S3OperationResource{}
}

// S3OperationResource defines the resource implementation.
type S3OperationResource struct {
	config *common.ProviderConfig
}

// S3OperationResourceModel describes the resource data model.
type S3OperationResourceModel struct {
	ID             types.String `tfsdk:"id"`
	Service        types.String `tfsdk:"service"`
	Key            types.String `tfsdk:"key"`
	Operation      types.String `tfsdk:"operation"`
	DestinationKey types.String `tfsdk:"destination_key"`
	Metadata       types.Map    `tfsdk:"metadata"`
	Tags           types.Map    `tfsdk:"tags"`
	StorageClass   types.String `tfsdk:"storage_class"`
	RestoreDays    types.Int64  `tfsdk:"restore_days"`

	// Computed outputs
	ETag                  types.String `tfsdk:"etag"`
	Size                  types.Int64  `tfsdk:"size"`
	ContentType           types.String `tfsdk:"content_type"`
	CurrentStorageClass   types.String `tfsdk:"current_storage_class"`
	VersionID             types.String `tfsdk:"version_id"`
	LastModified          types.String `tfsdk:"last_modified"`
	CurrentMetadata       types.Map    `tfsdk:"current_metadata"`
	CurrentTags           types.Map    `tfsdk:"current_tags"`
	ChecksumCRC32         types.String `tfsdk:"checksum_crc32"`
	ChecksumSHA1          types.String `tfsdk:"checksum_sha1"`
	ChecksumSHA256        types.String `tfsdk:"checksum_sha256"`
	ReplicationStatus     types.String `tfsdk:"replication_status"`
	ObjectLockMode        types.String `tfsdk:"object_lock_mode"`
	ObjectLockRetainUntil types.String `tfsdk:"object_lock_retain_until"`
	LegalHold             types.Bool   `tfsdk:"legal_hold"`
}

// Metadata returns the resource type name.
func (r *S3OperationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_operation"
}

// Schema defines the schema for the resource.
func (r *S3OperationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Performs operations on S3 objects such as head, copy, set metadata, set tags, change storage class, or restore from Glacier.",
		MarkdownDescription: "Performs operations on S3 objects such as head, copy, set metadata, set tags, change storage class, or restore from Glacier.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "S3 service alias.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"key": schema.StringAttribute{
				Description: "S3 object key.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"operation": schema.StringAttribute{
				Description: "Operation to perform. Valid values: head, copy, set_metadata, set_tags, delete_tags, set_storage_class, restore.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(OpHead, OpCopy, OpSetMetadata, OpSetTags, OpDeleteTags, OpSetStorageClass, OpRestore),
				},
			},
			"destination_key": schema.StringAttribute{
				Description: "Destination key for copy operation.",
				Optional:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Metadata to set on the object (for set_metadata operation).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.MapAttribute{
				Description: "Tags to set on the object (for set_tags operation).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"storage_class": schema.StringAttribute{
				Description: "Storage class for set_storage_class operation. Valid values: STANDARD, REDUCED_REDUNDANCY, STANDARD_IA, ONEZONE_IA, INTELLIGENT_TIERING, GLACIER, DEEP_ARCHIVE, GLACIER_IR.",
				Optional:    true,
			},
			"restore_days": schema.Int64Attribute{
				Description: "Number of days for restore operation (Glacier/Deep Archive).",
				Optional:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},

			// Computed outputs
			"etag": schema.StringAttribute{
				Description: "Entity tag of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"size": schema.Int64Attribute{
				Description: "Size of the object in bytes.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"content_type": schema.StringAttribute{
				Description: "Content type of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"current_storage_class": schema.StringAttribute{
				Description: "Current storage class of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"version_id": schema.StringAttribute{
				Description: "Version ID of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modified timestamp of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"current_metadata": schema.MapAttribute{
				Description: "Current metadata on the object.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"current_tags": schema.MapAttribute{
				Description: "Current tags on the object.",
				Computed:    true,
				ElementType: types.StringType,
				PlanModifiers: []planmodifier.Map{
					mapplanmodifier.UseStateForUnknown(),
				},
			},
			"checksum_crc32": schema.StringAttribute{
				Description: "CRC32 checksum of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checksum_sha1": schema.StringAttribute{
				Description: "SHA1 checksum of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"checksum_sha256": schema.StringAttribute{
				Description: "SHA256 checksum of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"replication_status": schema.StringAttribute{
				Description: "Replication status of the object.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"object_lock_mode": schema.StringAttribute{
				Description: "Object lock mode (GOVERNANCE or COMPLIANCE).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"object_lock_retain_until": schema.StringAttribute{
				Description: "Object lock retain until date.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"legal_hold": schema.BoolAttribute{
				Description: "Whether the object has a legal hold.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *S3OperationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *S3OperationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3OperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating S3 operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"key":       data.Key.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform S3 operation", err.Error())
		return
	}

	// Set the ID
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		data.Service.ValueString(), data.Key.ValueString(), data.Operation.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *S3OperationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3OperationResourceModel

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

	// Check if object still exists
	exists, err := backend.Exists(ctx, data.Key.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check object existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Refresh computed values
	if err := r.refreshObjectInfo(ctx, backend, &data); err != nil {
		resp.Diagnostics.AddError("Failed to refresh object info", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *S3OperationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data S3OperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating S3 operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"key":       data.Key.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation-specific requirements
	if err := r.validateOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform S3 operation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *S3OperationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3OperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// S3 operation resources are metadata-only; we don't delete the actual object
	// unless the operation was a copy, in which case we might clean up the destination
	tflog.Debug(ctx, "Deleting S3 operation resource (no object deletion)", map[string]any{
		"backend":   data.Service.ValueString(),
		"key":       data.Key.ValueString(),
		"operation": data.Operation.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *S3OperationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// validateOperation validates operation-specific configuration.
func (r *S3OperationResource) validateOperation(ctx context.Context, data *S3OperationResourceModel) error {
	op := data.Operation.ValueString()

	switch op {
	case OpCopy:
		if data.DestinationKey.IsNull() || data.DestinationKey.ValueString() == "" {
			return fmt.Errorf("destination_key is required for copy operation")
		}
	case OpSetMetadata:
		if data.Metadata.IsNull() || len(data.Metadata.Elements()) == 0 {
			return fmt.Errorf("metadata is required for set_metadata operation")
		}
	case OpSetTags:
		if data.Tags.IsNull() || len(data.Tags.Elements()) == 0 {
			return fmt.Errorf("tags is required for set_tags operation")
		}
	case OpSetStorageClass:
		if data.StorageClass.IsNull() || data.StorageClass.ValueString() == "" {
			return fmt.Errorf("storage_class is required for set_storage_class operation")
		}
	case OpRestore:
		if data.RestoreDays.IsNull() || data.RestoreDays.ValueInt64() <= 0 {
			return fmt.Errorf("restore_days must be a positive integer for restore operation")
		}
	case OpHead, OpDeleteTags:
		// No additional validation required
	}

	return nil
}

// performOperation executes the S3 operation.
func (r *S3OperationResource) performOperation(ctx context.Context, data *S3OperationResourceModel) error {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	op := data.Operation.ValueString()

	tflog.Info(ctx, "Performing S3 operation", map[string]any{
		"operation": op,
		"key":       data.Key.ValueString(),
	})

	switch op {
	case OpHead:
		return r.performHead(ctx, backend, data)
	case OpCopy:
		return r.performCopy(ctx, backend, data)
	case OpSetMetadata:
		return r.performSetMetadata(ctx, backend, data)
	case OpSetTags:
		return r.performSetTags(ctx, backend, data)
	case OpDeleteTags:
		return r.performDeleteTags(ctx, backend, data)
	case OpSetStorageClass:
		return r.performSetStorageClass(ctx, backend, data)
	case OpRestore:
		return r.performRestore(ctx, backend, data)
	default:
		return fmt.Errorf("unsupported operation: %s", op)
	}
}

// performHead retrieves object metadata.
func (r *S3OperationResource) performHead(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	return r.refreshObjectInfo(ctx, backend, data)
}

// performCopy copies an object to a new key.
func (r *S3OperationResource) performCopy(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Copying S3 object", map[string]any{
		"source":      data.Key.ValueString(),
		"destination": data.DestinationKey.ValueString(),
	})

	if err := s3Backend.CopyObject(ctx, data.Key.ValueString(), data.DestinationKey.ValueString()); err != nil {
		return fmt.Errorf("failed to copy object: %w", err)
	}

	return r.refreshObjectInfo(ctx, backend, data)
}

// performSetMetadata sets object metadata.
func (r *S3OperationResource) performSetMetadata(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Setting S3 object metadata", map[string]any{
		"key": data.Key.ValueString(),
	})

	// Convert types.Map to map[string]string
	metadata := make(map[string]string)
	for k, v := range data.Metadata.Elements() {
		if strVal, ok := v.(types.String); ok {
			metadata[k] = strVal.ValueString()
		}
	}

	if err := s3Backend.SetMetadata(ctx, data.Key.ValueString(), metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Update current_metadata to reflect what was set
	data.CurrentMetadata = data.Metadata

	return r.refreshObjectInfo(ctx, backend, data)
}

// performSetTags sets object tags.
func (r *S3OperationResource) performSetTags(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Setting S3 object tags", map[string]any{
		"key": data.Key.ValueString(),
	})

	// Convert types.Map to map[string]string
	tags := make(map[string]string)
	for k, v := range data.Tags.Elements() {
		if strVal, ok := v.(types.String); ok {
			tags[k] = strVal.ValueString()
		}
	}

	if err := s3Backend.SetTags(ctx, data.Key.ValueString(), tags); err != nil {
		return fmt.Errorf("failed to set tags: %w", err)
	}

	// Update current_tags to reflect what was set
	data.CurrentTags = data.Tags

	return r.refreshObjectInfo(ctx, backend, data)
}

// performDeleteTags deletes object tags.
func (r *S3OperationResource) performDeleteTags(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Deleting S3 object tags", map[string]any{
		"key": data.Key.ValueString(),
	})

	if err := s3Backend.DeleteTags(ctx, data.Key.ValueString()); err != nil {
		return fmt.Errorf("failed to delete tags: %w", err)
	}

	// Clear current_tags to reflect deletion
	data.CurrentTags = types.MapNull(types.StringType)

	return r.refreshObjectInfo(ctx, backend, data)
}

// performSetStorageClass changes the object's storage class.
func (r *S3OperationResource) performSetStorageClass(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Setting S3 object storage class", map[string]any{
		"key":           data.Key.ValueString(),
		"storage_class": data.StorageClass.ValueString(),
	})

	if err := s3Backend.SetStorageClass(ctx, data.Key.ValueString(), data.StorageClass.ValueString()); err != nil {
		return fmt.Errorf("failed to set storage class: %w", err)
	}

	// Update current_storage_class to reflect what was set
	data.CurrentStorageClass = data.StorageClass

	return r.refreshObjectInfo(ctx, backend, data)
}

// performRestore initiates a restore from Glacier/Deep Archive.
func (r *S3OperationResource) performRestore(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	s3Backend, ok := backend.(s3backend.S3Backend)
	if !ok {
		return fmt.Errorf("backend does not support S3 operations")
	}

	tflog.Debug(ctx, "Restoring S3 object from Glacier/Deep Archive", map[string]any{
		"key":          data.Key.ValueString(),
		"restore_days": data.RestoreDays.ValueInt64(),
	})

	// Default tier is "standard" if not specified
	tier := "standard"

	if err := s3Backend.RestoreObject(ctx, data.Key.ValueString(), int32(data.RestoreDays.ValueInt64()), tier); err != nil {
		return fmt.Errorf("failed to restore object: %w", err)
	}

	return r.refreshObjectInfo(ctx, backend, data)
}

// refreshObjectInfo retrieves current object information.
func (r *S3OperationResource) refreshObjectInfo(ctx context.Context, backend plugin.Backend, data *S3OperationResourceModel) error {
	// Try to use S3-specific HeadObject for extended info
	if s3Backend, ok := backend.(s3backend.S3Backend); ok {
		return r.refreshFromS3Backend(ctx, s3Backend, data)
	}

	// Fall back to generic Stat
	info, err := backend.Stat(ctx, data.Key.ValueString())
	if err != nil {
		if err == plugin.ErrPathNotFound {
			r.setPlaceholderValues(data)
			return nil
		}
		return fmt.Errorf("failed to stat object: %w", err)
	}

	// Map FileInfo to computed values
	data.ETag = types.StringValue(info.ETag)
	data.Size = types.Int64Value(info.Size)
	data.ContentType = types.StringValue(info.ContentType)
	data.VersionID = types.StringValue(info.VersionID)
	data.LastModified = types.StringValue(info.ModTime.Format(time.RFC3339))

	// Storage class
	if info.StorageClass != "" {
		data.CurrentStorageClass = types.StringValue(info.StorageClass)
	} else if data.CurrentStorageClass.IsNull() {
		data.CurrentStorageClass = types.StringValue("STANDARD")
	}

	// Checksums
	data.ChecksumCRC32 = types.StringValue(info.CRC32)
	data.ChecksumSHA1 = types.StringValue("")
	data.ChecksumSHA256 = types.StringValue(info.SHA256)

	// Metadata from FileInfo
	if len(info.Metadata) > 0 {
		metadataMap := make(map[string]string)
		for k, v := range info.Metadata {
			metadataMap[k] = v
		}
		mapValue, diags := types.MapValueFrom(ctx, types.StringType, metadataMap)
		if !diags.HasError() {
			data.CurrentMetadata = mapValue
		}
	} else if data.CurrentMetadata.IsNull() {
		data.CurrentMetadata = types.MapNull(types.StringType)
	}

	// Tags from FileInfo
	if len(info.Tags) > 0 {
		tagsMap := make(map[string]string)
		for k, v := range info.Tags {
			tagsMap[k] = v
		}
		mapValue, diags := types.MapValueFrom(ctx, types.StringType, tagsMap)
		if !diags.HasError() {
			data.CurrentTags = mapValue
		}
	} else if data.CurrentTags.IsNull() {
		data.CurrentTags = types.MapNull(types.StringType)
	}

	// Set default values for S3-specific fields
	data.ReplicationStatus = types.StringValue("")
	data.ObjectLockMode = types.StringValue("")
	data.ObjectLockRetainUntil = types.StringValue("")
	data.LegalHold = types.BoolValue(false)

	return nil
}

// refreshFromS3Backend retrieves extended S3 object information.
func (r *S3OperationResource) refreshFromS3Backend(ctx context.Context, s3Backend s3backend.S3Backend, data *S3OperationResourceModel) error {
	info, err := s3Backend.HeadObject(ctx, data.Key.ValueString())
	if err != nil {
		if err == plugin.ErrPathNotFound {
			r.setPlaceholderValues(data)
			return nil
		}
		return fmt.Errorf("failed to head object: %w", err)
	}

	// Map basic FileInfo fields
	data.ETag = types.StringValue(info.FileInfo.ETag)
	data.Size = types.Int64Value(info.FileInfo.Size)
	data.ContentType = types.StringValue(info.FileInfo.ContentType)
	data.VersionID = types.StringValue(info.FileInfo.VersionID)
	data.LastModified = types.StringValue(info.FileInfo.ModTime.Format(time.RFC3339))

	// Storage class
	if info.FileInfo.StorageClass != "" {
		data.CurrentStorageClass = types.StringValue(info.FileInfo.StorageClass)
	} else {
		data.CurrentStorageClass = types.StringValue("STANDARD")
	}

	// Checksums
	data.ChecksumCRC32 = types.StringValue(info.FileInfo.CRC32)
	data.ChecksumSHA1 = types.StringValue("")
	data.ChecksumSHA256 = types.StringValue(info.FileInfo.SHA256)

	// Metadata
	if len(info.FileInfo.Metadata) > 0 {
		mapValue, diags := types.MapValueFrom(ctx, types.StringType, info.FileInfo.Metadata)
		if !diags.HasError() {
			data.CurrentMetadata = mapValue
		}
	} else {
		data.CurrentMetadata = types.MapNull(types.StringType)
	}

	// Tags
	if len(info.FileInfo.Tags) > 0 {
		mapValue, diags := types.MapValueFrom(ctx, types.StringType, info.FileInfo.Tags)
		if !diags.HasError() {
			data.CurrentTags = mapValue
		}
	} else {
		data.CurrentTags = types.MapNull(types.StringType)
	}

	// S3-specific extended fields
	data.ReplicationStatus = types.StringValue(info.ReplicationStatus)
	data.ObjectLockMode = types.StringValue(info.ObjectLockMode)
	if info.ObjectLockRetainUntil != nil {
		data.ObjectLockRetainUntil = types.StringValue(info.ObjectLockRetainUntil.Format(time.RFC3339))
	} else {
		data.ObjectLockRetainUntil = types.StringValue("")
	}
	data.LegalHold = types.BoolValue(info.LegalHold)

	return nil
}

// setPlaceholderValues sets placeholder/mock values for computed attributes.
func (r *S3OperationResource) setPlaceholderValues(data *S3OperationResourceModel) {
	data.ETag = types.StringValue("")
	data.Size = types.Int64Value(0)
	data.ContentType = types.StringValue("")
	data.CurrentStorageClass = types.StringValue("STANDARD")
	data.VersionID = types.StringValue("")
	data.LastModified = types.StringValue("")
	data.CurrentMetadata = types.MapNull(types.StringType)
	data.CurrentTags = types.MapNull(types.StringType)
	data.ChecksumCRC32 = types.StringValue("")
	data.ChecksumSHA1 = types.StringValue("")
	data.ChecksumSHA256 = types.StringValue("")
	data.ReplicationStatus = types.StringValue("")
	data.ObjectLockMode = types.StringValue("")
	data.ObjectLockRetainUntil = types.StringValue("")
	data.LegalHold = types.BoolValue(false)
}

// getBackend returns the appropriate backend.
func (r *S3OperationResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}
