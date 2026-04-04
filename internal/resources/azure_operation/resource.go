// SPDX-License-Identifier: MIT

// Package azure_operation implements the filemanager_azure_operation resource.
package azure_operation

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	azurebackend "github.com/ebogdum/filemanager/internal/backends/azure"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &AzureOperationResource{}
	_ resource.ResourceWithImportState = &AzureOperationResource{}
)

// NewAzureOperationResource creates a new Azure operation resource.
func NewAzureOperationResource() resource.Resource {
	return &AzureOperationResource{}
}

// AzureOperationResource defines the resource implementation.
type AzureOperationResource struct {
	config *common.ProviderConfig
}

// AzureOperationResourceModel describes the resource data model.
type AzureOperationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Service         types.String `tfsdk:"service"`
	BlobPath        types.String `tfsdk:"blob_path"`
	Operation       types.String `tfsdk:"operation"`
	DestinationPath types.String `tfsdk:"destination_path"`
	Metadata        types.Map    `tfsdk:"metadata"`
	Tags            types.Map    `tfsdk:"tags"`
	AccessTier      types.String `tfsdk:"access_tier"`
	LeaseDuration   types.Int64  `tfsdk:"lease_duration"`
	Triggers        types.Map    `tfsdk:"triggers"`

	// Computed outputs
	ETag              types.String `tfsdk:"etag"`
	ContentType       types.String `tfsdk:"content_type"`
	ContentMD5        types.String `tfsdk:"content_md5"`
	BlobType          types.String `tfsdk:"blob_type"`
	CurrentAccessTier types.String `tfsdk:"current_access_tier"`
	LeaseStatus       types.String `tfsdk:"lease_status"`
	LeaseID           types.String `tfsdk:"lease_id"`
	CreationTime      types.String `tfsdk:"creation_time"`
	LastModified      types.String `tfsdk:"last_modified"`
	VersionID         types.String `tfsdk:"version_id"`
	IsCurrentVersion  types.Bool   `tfsdk:"is_current_version"`
	ServerEncrypted   types.Bool   `tfsdk:"server_encrypted"`
	ArchiveStatus     types.String `tfsdk:"archive_status"`
	CurrentMetadata   types.Map    `tfsdk:"current_metadata"`
	CurrentTags       types.Map    `tfsdk:"current_tags"`
}

// Metadata returns the resource type name.
func (r *AzureOperationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azure_operation"
}

// Schema defines the schema for the resource.
func (r *AzureOperationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Performs Azure Blob Storage operations on existing blobs.",
		MarkdownDescription: "Performs Azure Blob Storage operations on existing blobs.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Azure service alias configured in the provider.",
				Required:    true,
			},
			"blob_path": schema.StringAttribute{
				Description: "Path to the blob within the container.",
				Required:    true,
			},
			"operation": schema.StringAttribute{
				Description: "Operation to perform: head, copy, set_metadata, set_tags, set_access_tier, acquire_lease, release_lease.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf(
						"head",
						"copy",
						"set_metadata",
						"set_tags",
						"set_access_tier",
						"acquire_lease",
						"release_lease",
					),
				},
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path for copy operation.",
				Optional:    true,
			},
			"metadata": schema.MapAttribute{
				Description: "Metadata to set on the blob (for set_metadata operation).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"tags": schema.MapAttribute{
				Description: "Tags to set on the blob (for set_tags operation).",
				Optional:    true,
				ElementType: types.StringType,
			},
			"access_tier": schema.StringAttribute{
				Description: "Access tier to set: Hot, Cool, or Archive (for set_access_tier operation).",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("Hot", "Cool", "Archive"),
				},
			},
			"lease_duration": schema.Int64Attribute{
				Description: "Lease duration in seconds: 15-60 or -1 for infinite (for acquire_lease operation).",
				Optional:    true,
			},
			"triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger re-execution of the operation.",
				Optional:    true,
				ElementType: types.StringType,
			},

			// Computed outputs
			"etag": schema.StringAttribute{
				Description: "ETag of the blob.",
				Computed:    true,
			},
			"content_type": schema.StringAttribute{
				Description: "Content type of the blob.",
				Computed:    true,
			},
			"content_md5": schema.StringAttribute{
				Description: "MD5 hash of the blob content.",
				Computed:    true,
			},
			"blob_type": schema.StringAttribute{
				Description: "Type of the blob (BlockBlob, PageBlob, AppendBlob).",
				Computed:    true,
			},
			"current_access_tier": schema.StringAttribute{
				Description: "Current access tier of the blob.",
				Computed:    true,
			},
			"lease_status": schema.StringAttribute{
				Description: "Lease status of the blob (locked, unlocked).",
				Computed:    true,
			},
			"lease_id": schema.StringAttribute{
				Description: "Lease ID if a lease was acquired.",
				Computed:    true,
				Sensitive:   true,
			},
			"creation_time": schema.StringAttribute{
				Description: "Creation time of the blob in RFC3339 format.",
				Computed:    true,
			},
			"last_modified": schema.StringAttribute{
				Description: "Last modification time of the blob in RFC3339 format.",
				Computed:    true,
			},
			"version_id": schema.StringAttribute{
				Description: "Version ID of the blob (if versioning is enabled).",
				Computed:    true,
			},
			"is_current_version": schema.BoolAttribute{
				Description: "Whether this is the current version of the blob.",
				Computed:    true,
			},
			"server_encrypted": schema.BoolAttribute{
				Description: "Whether the blob is encrypted at rest.",
				Computed:    true,
			},
			"archive_status": schema.StringAttribute{
				Description: "Archive status if blob is being rehydrated.",
				Computed:    true,
			},
			"current_metadata": schema.MapAttribute{
				Description: "Current metadata on the blob.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"current_tags": schema.MapAttribute{
				Description: "Current tags on the blob.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *AzureOperationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *AzureOperationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AzureOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Azure operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"blob_path": data.BlobPath.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform Azure operation", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		data.Service.ValueString(),
		data.BlobPath.ValueString(),
		data.Operation.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *AzureOperationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AzureOperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Refresh blob properties
	if err := r.refreshBlobProperties(ctx, &data); err != nil {
		// If blob no longer exists, remove from state
		if err == plugin.ErrPathNotFound {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to read blob properties", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *AzureOperationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AzureOperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating Azure operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"blob_path": data.BlobPath.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	if err := r.performOperation(ctx, &data); err != nil {
		resp.Diagnostics.AddError("Failed to perform Azure operation", err.Error())
		return
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *AzureOperationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AzureOperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For operations, delete is mostly a no-op.
	// However, if we acquired a lease, we should release it.
	if data.Operation.ValueString() == "acquire_lease" && !data.LeaseID.IsNull() && data.LeaseID.ValueString() != "" {
		tflog.Debug(ctx, "Releasing lease on delete", map[string]any{
			"blob_path": data.BlobPath.ValueString(),
			"lease_id":  data.LeaseID.ValueString(),
		})

		backend, err := r.getBackend(ctx, data.Service.ValueString())
		if err != nil {
			resp.Diagnostics.AddError(
				"Failed to get backend for lease release",
				fmt.Sprintf("Could not obtain backend to release lease: %s", err),
			)
			return
		}

		azBackend, ok := backend.(azurebackend.AzureBackend)
		if !ok {
			resp.Diagnostics.AddError(
				"Lease Release Not Supported",
				"The configured backend does not support Azure lease operations. The lease may remain active until it expires. Please release the lease manually.",
			)
			return
		}

		if err := azBackend.ReleaseLease(ctx, data.BlobPath.ValueString(), data.LeaseID.ValueString()); err != nil {
			resp.Diagnostics.AddError(
				"Failed to Release Lease",
				fmt.Sprintf("Could not release lease %s on blob %s: %s. The lease may remain active until it expires.",
					data.LeaseID.ValueString(), data.BlobPath.ValueString(), err),
			)
			return
		}

		tflog.Info(ctx, "Lease released successfully on delete", map[string]any{
			"blob_path": data.BlobPath.ValueString(),
		})
	}

	// Resource is removed from state automatically
}

// ImportState imports an existing resource.
func (r *AzureOperationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// performOperation performs the specified Azure blob operation.
func (r *AzureOperationResource) performOperation(ctx context.Context, data *AzureOperationResourceModel) error {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	operation := data.Operation.ValueString()
	blobPath := data.BlobPath.ValueString()

	tflog.Debug(ctx, "Performing Azure operation", map[string]any{
		"operation": operation,
		"blob_path": blobPath,
	})

	switch operation {
	case "head":
		return r.operationHead(ctx, backend, data)
	case "copy":
		return r.operationCopy(ctx, backend, data)
	case "set_metadata":
		return r.operationSetMetadata(ctx, backend, data)
	case "set_tags":
		return r.operationSetTags(ctx, backend, data)
	case "set_access_tier":
		return r.operationSetAccessTier(ctx, backend, data)
	case "acquire_lease":
		return r.operationAcquireLease(ctx, backend, data)
	case "release_lease":
		return r.operationReleaseLease(ctx, backend, data)
	default:
		return fmt.Errorf("unknown operation: %s", operation)
	}
}

// operationHead gets blob properties (head operation).
func (r *AzureOperationResource) operationHead(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	return r.refreshBlobProperties(ctx, data)
}

// operationCopy copies a blob to a new location.
func (r *AzureOperationResource) operationCopy(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	if data.DestinationPath.IsNull() || data.DestinationPath.ValueString() == "" {
		return fmt.Errorf("destination_path is required for copy operation")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Copying Azure blob", map[string]any{
		"source":      data.BlobPath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	if err := azBackend.CopyBlob(ctx, data.BlobPath.ValueString(), data.DestinationPath.ValueString()); err != nil {
		return fmt.Errorf("failed to copy blob: %w", err)
	}

	return r.refreshBlobProperties(ctx, data)
}

// operationSetMetadata sets metadata on a blob.
func (r *AzureOperationResource) operationSetMetadata(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	if data.Metadata.IsNull() {
		return fmt.Errorf("metadata is required for set_metadata operation")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Setting Azure blob metadata", map[string]any{
		"blob_path": data.BlobPath.ValueString(),
	})

	// Convert types.Map to map[string]*string (Azure SDK format)
	metadata := make(map[string]*string)
	for k, v := range data.Metadata.Elements() {
		if strVal, ok := v.(types.String); ok {
			val := strVal.ValueString()
			metadata[k] = &val
		}
	}

	if err := azBackend.SetMetadata(ctx, data.BlobPath.ValueString(), metadata); err != nil {
		return fmt.Errorf("failed to set metadata: %w", err)
	}

	// Update current_metadata to reflect what was set
	data.CurrentMetadata = data.Metadata

	return r.refreshBlobProperties(ctx, data)
}

// operationSetTags sets tags on a blob.
func (r *AzureOperationResource) operationSetTags(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	if data.Tags.IsNull() {
		return fmt.Errorf("tags is required for set_tags operation")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Setting Azure blob tags", map[string]any{
		"blob_path": data.BlobPath.ValueString(),
	})

	// Convert types.Map to map[string]string
	tags := make(map[string]string)
	for k, v := range data.Tags.Elements() {
		if strVal, ok := v.(types.String); ok {
			tags[k] = strVal.ValueString()
		}
	}

	if err := azBackend.SetTags(ctx, data.BlobPath.ValueString(), tags); err != nil {
		return fmt.Errorf("failed to set tags: %w", err)
	}

	// Update current_tags to reflect what was set
	data.CurrentTags = data.Tags

	return r.refreshBlobProperties(ctx, data)
}

// operationSetAccessTier sets the access tier on a blob.
func (r *AzureOperationResource) operationSetAccessTier(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	if data.AccessTier.IsNull() || data.AccessTier.ValueString() == "" {
		return fmt.Errorf("access_tier is required for set_access_tier operation")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Setting Azure blob access tier", map[string]any{
		"blob_path":   data.BlobPath.ValueString(),
		"access_tier": data.AccessTier.ValueString(),
	})

	if err := azBackend.SetAccessTier(ctx, data.BlobPath.ValueString(), data.AccessTier.ValueString()); err != nil {
		return fmt.Errorf("failed to set access tier: %w", err)
	}

	// Update current_access_tier to reflect what was set
	data.CurrentAccessTier = data.AccessTier

	return r.refreshBlobProperties(ctx, data)
}

// operationAcquireLease acquires a lease on a blob.
func (r *AzureOperationResource) operationAcquireLease(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	duration := data.LeaseDuration.ValueInt64()
	if duration != -1 && (duration < 15 || duration > 60) {
		return fmt.Errorf("lease_duration must be -1 (infinite) or between 15-60 seconds")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Acquiring Azure blob lease", map[string]any{
		"blob_path":      data.BlobPath.ValueString(),
		"lease_duration": duration,
	})

	leaseID, err := azBackend.AcquireLease(ctx, data.BlobPath.ValueString(), int32(duration))
	if err != nil {
		return fmt.Errorf("failed to acquire lease: %w", err)
	}

	data.LeaseID = types.StringValue(leaseID)
	data.LeaseStatus = types.StringValue("locked")

	return r.refreshBlobProperties(ctx, data)
}

// operationReleaseLease releases a lease on a blob.
func (r *AzureOperationResource) operationReleaseLease(ctx context.Context, backend plugin.Backend, data *AzureOperationResourceModel) error {
	if data.LeaseID.IsNull() || data.LeaseID.ValueString() == "" {
		return fmt.Errorf("lease_id is required for release_lease operation (must acquire lease first)")
	}

	azBackend, ok := backend.(azurebackend.AzureBackend)
	if !ok {
		return fmt.Errorf("backend does not support Azure operations")
	}

	tflog.Debug(ctx, "Releasing Azure blob lease", map[string]any{
		"blob_path": data.BlobPath.ValueString(),
	})

	if err := azBackend.ReleaseLease(ctx, data.BlobPath.ValueString(), data.LeaseID.ValueString()); err != nil {
		return fmt.Errorf("failed to release lease: %w", err)
	}

	data.LeaseID = types.StringNull()
	data.LeaseStatus = types.StringValue("unlocked")

	return r.refreshBlobProperties(ctx, data)
}

// refreshBlobProperties refreshes the blob properties from Azure.
func (r *AzureOperationResource) refreshBlobProperties(ctx context.Context, data *AzureOperationResourceModel) error {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return fmt.Errorf("failed to get backend: %w", err)
	}

	// Try to use Azure-specific HeadBlob for extended info
	if azBackend, ok := backend.(azurebackend.AzureBackend); ok {
		return r.refreshFromAzureBackend(ctx, azBackend, data)
	}

	// Fall back to generic Stat
	info, err := backend.Stat(ctx, data.BlobPath.ValueString())
	if err != nil {
		if err == plugin.ErrPathNotFound {
			return plugin.ErrPathNotFound
		}
		return fmt.Errorf("failed to stat blob: %w", err)
	}

	// Set values from the actual file info
	data.ETag = types.StringValue(info.ETag)
	data.ContentType = types.StringValue(info.ContentType)
	data.ContentMD5 = types.StringValue(info.MD5)
	data.LastModified = types.StringValue(info.ModTime.Format(time.RFC3339))
	data.CreationTime = types.StringValue(info.CreationTime.Format(time.RFC3339))
	data.VersionID = types.StringValue(info.VersionID)

	// Set metadata and tags from file info
	if info.Metadata != nil {
		data.CurrentMetadata = r.stringMapToTypesMap(ctx, info.Metadata)
	} else {
		data.CurrentMetadata = types.MapNull(types.StringType)
	}

	if info.Tags != nil {
		data.CurrentTags = r.stringMapToTypesMap(ctx, info.Tags)
	} else {
		data.CurrentTags = types.MapNull(types.StringType)
	}

	// Set defaults for Azure-specific fields
	data.BlobType = types.StringValue("BlockBlob")
	data.CurrentAccessTier = types.StringValue("Hot")
	data.LeaseStatus = types.StringValue("unlocked")
	data.IsCurrentVersion = types.BoolValue(true)
	data.ServerEncrypted = types.BoolValue(true)
	data.ArchiveStatus = types.StringNull()

	return nil
}

// refreshFromAzureBackend retrieves extended Azure blob information.
func (r *AzureOperationResource) refreshFromAzureBackend(ctx context.Context, azBackend azurebackend.AzureBackend, data *AzureOperationResourceModel) error {
	info, err := azBackend.HeadBlob(ctx, data.BlobPath.ValueString())
	if err != nil {
		if err == plugin.ErrPathNotFound {
			return plugin.ErrPathNotFound
		}
		return fmt.Errorf("failed to get blob properties: %w", err)
	}

	// Map basic FileInfo fields
	data.ETag = types.StringValue(info.FileInfo.ETag)
	data.ContentType = types.StringValue(info.FileInfo.ContentType)
	data.ContentMD5 = types.StringValue(info.FileInfo.MD5)
	data.LastModified = types.StringValue(info.FileInfo.ModTime.Format(time.RFC3339))
	data.CreationTime = types.StringValue(info.FileInfo.CreationTime.Format(time.RFC3339))
	data.VersionID = types.StringValue(info.FileInfo.VersionID)

	// Metadata
	if info.FileInfo.Metadata != nil {
		data.CurrentMetadata = r.stringMapToTypesMap(ctx, info.FileInfo.Metadata)
	} else {
		data.CurrentMetadata = types.MapNull(types.StringType)
	}

	// Tags
	if info.FileInfo.Tags != nil {
		data.CurrentTags = r.stringMapToTypesMap(ctx, info.FileInfo.Tags)
	} else {
		data.CurrentTags = types.MapNull(types.StringType)
	}

	// Azure-specific fields
	data.BlobType = types.StringValue(info.BlobType)
	data.CurrentAccessTier = types.StringValue(info.AccessTier)
	data.LeaseStatus = types.StringValue(info.LeaseStatus)
	data.ServerEncrypted = types.BoolValue(info.ServerEncrypted)
	data.ArchiveStatus = types.StringValue(info.ArchiveStatus)
	data.IsCurrentVersion = types.BoolValue(true)

	return nil
}

// getBackend returns the appropriate backend.
func (r *AzureOperationResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// stringMapToTypesMap converts a Go string map to a types.Map.
func (r *AzureOperationResource) stringMapToTypesMap(ctx context.Context, m map[string]string) types.Map {
	if m == nil {
		return types.MapNull(types.StringType)
	}

	elements := make(map[string]types.String)
	for k, v := range m {
		elements[k] = types.StringValue(v)
	}

	result, _ := types.MapValueFrom(ctx, types.StringType, m)
	return result
}
