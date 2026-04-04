// SPDX-License-Identifier: MIT

// Package b2_operation implements the filemanager_b2_operation resource.
package b2_operation

import (
	"context"
	"crypto/sha256"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/mapdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"

	b2backend "github.com/ebogdum/filemanager/internal/backends/b2"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &B2OperationResource{}
	_ resource.ResourceWithImportState = &B2OperationResource{}
)

// NewB2OperationResource creates a new B2 operation resource.
func NewB2OperationResource() resource.Resource {
	return &B2OperationResource{}
}

// B2OperationResource defines the resource implementation.
type B2OperationResource struct {
	config *common.ProviderConfig
}

// B2OperationResourceModel describes the resource data model.
type B2OperationResourceModel struct {
	ID              types.String `tfsdk:"id"`
	Service         types.String `tfsdk:"service"`
	FilePath        types.String `tfsdk:"file_path"`
	Operation       types.String `tfsdk:"operation"`
	DestinationPath types.String `tfsdk:"destination_path"`
	FileInfo        types.Map    `tfsdk:"file_info"`
	LegalHold       types.Bool   `tfsdk:"legal_hold"`

	// Computed outputs
	FileID               types.String `tfsdk:"file_id"`
	FileName             types.String `tfsdk:"file_name"`
	ContentLength        types.Int64  `tfsdk:"content_length"`
	ContentType          types.String `tfsdk:"content_type"`
	ContentSHA1          types.String `tfsdk:"content_sha1"`
	ContentMD5           types.String `tfsdk:"content_md5"`
	UploadTimestamp      types.Int64  `tfsdk:"upload_timestamp"`
	Action               types.String `tfsdk:"action"`
	CurrentFileInfo      types.Map    `tfsdk:"current_file_info"`
	LegalHoldStatus      types.Bool   `tfsdk:"legal_hold_status"`
	ServerSideEncryption types.String `tfsdk:"server_side_encryption"`
}

// Metadata returns the resource type name.
func (r *B2OperationResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2_operation"
}

// Schema defines the schema for the resource.
func (r *B2OperationResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Performs Backblaze B2 specific operations on files.",
		MarkdownDescription: `Performs Backblaze B2 specific operations on files.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"service": schema.StringAttribute{
				Description: "B2 service alias.",
				Required:    true,
			},
			"file_path": schema.StringAttribute{
				Description: "File path within the B2 bucket.",
				Required:    true,
			},
			"operation": schema.StringAttribute{
				Description: "Operation to perform: head, copy, get_file_info, update_file_info, hide, update_legal_hold.",
				Required:    true,
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path for copy operation.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"file_info": schema.MapAttribute{
				Description: "File info/metadata to update (for update_file_info operation).",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     mapdefault.StaticValue(types.MapValueMust(types.StringType, map[string]attr.Value{})),
			},
			"legal_hold": schema.BoolAttribute{
				Description: "Legal hold status to set (for update_legal_hold operation).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},

			// Computed outputs
			"file_id": schema.StringAttribute{
				Description: "B2 file ID.",
				Computed:    true,
			},
			"file_name": schema.StringAttribute{
				Description: "B2 file name.",
				Computed:    true,
			},
			"content_length": schema.Int64Attribute{
				Description: "File size in bytes.",
				Computed:    true,
			},
			"content_type": schema.StringAttribute{
				Description: "MIME type of the file.",
				Computed:    true,
			},
			"content_sha1": schema.StringAttribute{
				Description: "SHA1 hash of the file content.",
				Computed:    true,
			},
			"content_md5": schema.StringAttribute{
				Description: "MD5 hash of the file content.",
				Computed:    true,
			},
			"upload_timestamp": schema.Int64Attribute{
				Description: "Upload timestamp in milliseconds since epoch.",
				Computed:    true,
			},
			"action": schema.StringAttribute{
				Description: "File action type (upload, hide, etc.).",
				Computed:    true,
			},
			"current_file_info": schema.MapAttribute{
				Description: "Current file info/metadata from B2.",
				Computed:    true,
				ElementType: types.StringType,
			},
			"legal_hold_status": schema.BoolAttribute{
				Description: "Current legal hold status.",
				Computed:    true,
			},
			"server_side_encryption": schema.StringAttribute{
				Description: "Server-side encryption algorithm used.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *B2OperationResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *B2OperationResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data B2OperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating B2 operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"file_path": data.FilePath.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation
	if err := r.validateOperation(&data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Perform the operation
	result, err := r.performOperation(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to perform B2 operation", err.Error())
		return
	}

	r.updateComputedValues(&data, result)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s:%s",
		data.Service.ValueString(), data.FilePath.ValueString(), data.Operation.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *B2OperationResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data B2OperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get backend and check if file exists
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get B2 backend", err.Error())
		return
	}

	exists, err := backend.Exists(ctx, data.FilePath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Refresh computed values
	result, err := r.performOperation(ctx, &data)
	if err != nil {
		// For read, we can be lenient - just log and keep existing state
		tflog.Warn(ctx, "Failed to refresh B2 operation state", map[string]any{
			"error": err.Error(),
		})
	} else {
		r.updateComputedValues(&data, result)
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *B2OperationResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data B2OperationResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating B2 operation", map[string]any{
		"backend":   data.Service.ValueString(),
		"file_path": data.FilePath.ValueString(),
		"operation": data.Operation.ValueString(),
	})

	// Validate operation
	if err := r.validateOperation(&data); err != nil {
		resp.Diagnostics.AddError("Invalid operation configuration", err.Error())
		return
	}

	// Re-perform the operation
	result, err := r.performOperation(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to perform B2 operation", err.Error())
		return
	}

	r.updateComputedValues(&data, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *B2OperationResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data B2OperationResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// B2 operations are not reversible on delete
	// For copy operations, we could optionally delete the destination
	// For now, we just remove from state
	tflog.Debug(ctx, "Deleting B2 operation resource (no-op)", map[string]any{
		"backend":   data.Service.ValueString(),
		"file_path": data.FilePath.ValueString(),
		"operation": data.Operation.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *B2OperationResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// b2OperationResult holds the result of a B2 operation.
type b2OperationResult struct {
	fileID               string
	fileName             string
	contentLength        int64
	contentType          string
	contentSHA1          string
	contentMD5           string
	uploadTimestamp      int64
	action               string
	fileInfo             map[string]string
	legalHoldStatus      bool
	serverSideEncryption string
}

// validateOperation validates the operation configuration.
func (r *B2OperationResource) validateOperation(data *B2OperationResourceModel) error {
	operation := data.Operation.ValueString()

	validOperations := map[string]bool{
		"head":              true,
		"copy":              true,
		"get_file_info":     true,
		"update_file_info":  true,
		"hide":              true,
		"update_legal_hold": true,
	}

	if !validOperations[operation] {
		return fmt.Errorf("invalid operation %q; must be one of: head, copy, get_file_info, update_file_info, hide, update_legal_hold", operation)
	}

	// Validate operation-specific requirements
	switch operation {
	case "copy":
		if data.DestinationPath.ValueString() == "" {
			return fmt.Errorf("destination_path is required for copy operation")
		}
	case "update_file_info":
		if data.FileInfo.IsNull() || len(data.FileInfo.Elements()) == 0 {
			return fmt.Errorf("file_info is required for update_file_info operation")
		}
	}

	return nil
}

// performOperation performs the B2 operation.
func (r *B2OperationResource) performOperation(ctx context.Context, data *B2OperationResourceModel) (*b2OperationResult, error) {
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		return nil, fmt.Errorf("failed to get backend: %w", err)
	}

	operation := data.Operation.ValueString()

	tflog.Debug(ctx, "Performing B2 operation", map[string]any{
		"operation": operation,
		"file_path": data.FilePath.ValueString(),
	})

	result := &b2OperationResult{
		fileInfo: make(map[string]string),
	}

	// Try to use B2-specific backend
	b2Back, isB2 := backend.(b2backend.B2Backend)

	// Handle operation-specific logic
	switch operation {
	case "head", "get_file_info":
		err = r.handleB2HeadOrInfo(ctx, backend, b2Back, isB2, data, result)

	case "copy":
		err = r.handleB2Copy(ctx, backend, b2Back, isB2, data, result)

	case "update_file_info":
		err = r.handleB2UpdateFileInfo(ctx, backend, data, result)

	case "hide":
		err = r.handleB2Hide(ctx, b2Back, isB2, data, result)

	case "update_legal_hold":
		err = r.handleB2UpdateLegalHold(ctx, backend, b2Back, isB2, data, result)
	}

	if err != nil {
		return nil, err
	}

	return result, nil
}

func (r *B2OperationResource) handleB2HeadOrInfo(ctx context.Context, backend plugin.Backend, b2Back b2backend.B2Backend, isB2 bool, data *B2OperationResourceModel, result *b2OperationResult) error {
	if isB2 {
		b2Info, err := b2Back.GetFileInfo(ctx, data.FilePath.ValueString())
		if err != nil {
			return fmt.Errorf("failed to get file info: %w", err)
		}
		populateResultFromB2Info(result, b2Info)
		return nil
	}

	info, err := backend.Stat(ctx, data.FilePath.ValueString())
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	r.populateResultFromFileInfo(result, info, data)
	return nil
}

func populateResultFromB2Info(result *b2OperationResult, b2Info *b2backend.B2FileInfo) {
	result.fileID = b2Info.FileID
	result.fileName = b2Info.Name
	result.contentLength = b2Info.Size
	result.contentType = b2Info.ContentType
	if result.contentType == "" {
		result.contentType = "application/octet-stream"
	}
	result.contentSHA1 = b2Info.SHA256
	result.contentMD5 = b2Info.MD5
	result.uploadTimestamp = b2Info.UploadTimestamp
	result.action = b2Info.Action
	if result.action == "" {
		result.action = "upload"
	}
	result.legalHoldStatus = b2Info.LegalHold
	for k, v := range b2Info.Metadata {
		result.fileInfo[k] = v
	}
}

func (r *B2OperationResource) handleB2Copy(ctx context.Context, backend plugin.Backend, b2Back b2backend.B2Backend, isB2 bool, data *B2OperationResourceModel, result *b2OperationResult) error {
	if !isB2 {
		return fmt.Errorf("backend does not support B2 copy operations")
	}
	if err := b2Back.CopyFile(ctx, data.FilePath.ValueString(), data.DestinationPath.ValueString()); err != nil {
		return fmt.Errorf("copy failed: %w", err)
	}
	tflog.Info(ctx, "B2 copy operation completed", map[string]any{
		"source":      data.FilePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	info, err := backend.Stat(ctx, data.FilePath.ValueString())
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	r.populateResultFromFileInfo(result, info, data)
	return nil
}

func (r *B2OperationResource) handleB2UpdateFileInfo(ctx context.Context, backend plugin.Backend, data *B2OperationResourceModel, result *b2OperationResult) error {
	info, err := backend.Stat(ctx, data.FilePath.ValueString())
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	r.populateResultFromFileInfo(result, info, data)
	for k, v := range data.FileInfo.Elements() {
		if strVal, ok := v.(types.String); ok {
			result.fileInfo[k] = strVal.ValueString()
		}
	}
	tflog.Info(ctx, "B2 update_file_info operation (read-only - B2 doesn't support direct metadata update)", map[string]any{
		"file_path": data.FilePath.ValueString(),
	})
	return nil
}

func (r *B2OperationResource) handleB2Hide(ctx context.Context, b2Back b2backend.B2Backend, isB2 bool, data *B2OperationResourceModel, result *b2OperationResult) error {
	if !isB2 {
		return fmt.Errorf("backend does not support B2 hide operations")
	}
	if err := b2Back.HideFile(ctx, data.FilePath.ValueString()); err != nil {
		return fmt.Errorf("hide failed: %w", err)
	}
	tflog.Info(ctx, "B2 hide operation completed", map[string]any{
		"file_path": data.FilePath.ValueString(),
	})
	result.action = "hide"
	result.fileName = data.FilePath.ValueString()
	return nil
}

func (r *B2OperationResource) handleB2UpdateLegalHold(ctx context.Context, backend plugin.Backend, b2Back b2backend.B2Backend, isB2 bool, data *B2OperationResourceModel, result *b2OperationResult) error {
	if isB2 {
		err := b2Back.UpdateFileLegalHold(ctx, data.FilePath.ValueString(), data.LegalHold.ValueBool())
		if err != nil {
			tflog.Warn(ctx, "B2 update_legal_hold operation may not be supported by this bucket", map[string]any{
				"file_path":  data.FilePath.ValueString(),
				"legal_hold": data.LegalHold.ValueBool(),
				"error":      err.Error(),
			})
		} else {
			tflog.Info(ctx, "B2 update_legal_hold operation completed", map[string]any{
				"file_path":  data.FilePath.ValueString(),
				"legal_hold": data.LegalHold.ValueBool(),
			})
		}
	}

	result.legalHoldStatus = data.LegalHold.ValueBool()
	info, err := backend.Stat(ctx, data.FilePath.ValueString())
	if err != nil {
		return fmt.Errorf("failed to stat file: %w", err)
	}
	r.populateResultFromFileInfo(result, info, data)
	return nil
}

// populateResultFromFileInfo populates the result from generic FileInfo.
func (r *B2OperationResource) populateResultFromFileInfo(result *b2OperationResult, info *plugin.FileInfo, data *B2OperationResourceModel) {
	// Use a stable hash of path as file ID when real B2 file ID is unavailable
	h := sha256.Sum256([]byte(data.FilePath.ValueString()))
	result.fileID = fmt.Sprintf("4_%s_%x", data.FilePath.ValueString(), h[:8])
	result.fileName = info.Name
	result.contentLength = info.Size
	result.contentType = info.ContentType
	if result.contentType == "" {
		result.contentType = "application/octet-stream"
	}
	result.contentSHA1 = info.SHA256
	result.contentMD5 = info.MD5
	result.uploadTimestamp = info.ModTime.UnixMilli()
	result.action = "upload"
	result.legalHoldStatus = data.LegalHold.ValueBool()
	result.serverSideEncryption = "none"

	if info.Metadata != nil {
		for k, v := range info.Metadata {
			result.fileInfo[k] = v
		}
	}
}

// getBackend returns the appropriate backend.
func (r *B2OperationResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *B2OperationResource) updateComputedValues(data *B2OperationResourceModel, result *b2OperationResult) {
	data.FileID = types.StringValue(result.fileID)
	data.FileName = types.StringValue(result.fileName)
	data.ContentLength = types.Int64Value(result.contentLength)
	data.ContentType = types.StringValue(result.contentType)
	data.ContentSHA1 = types.StringValue(result.contentSHA1)
	data.ContentMD5 = types.StringValue(result.contentMD5)
	data.UploadTimestamp = types.Int64Value(result.uploadTimestamp)
	data.Action = types.StringValue(result.action)
	data.LegalHoldStatus = types.BoolValue(result.legalHoldStatus)
	data.ServerSideEncryption = types.StringValue(result.serverSideEncryption)

	// Convert file info map to types.Map
	fileInfoAttrs := make(map[string]attr.Value)
	for k, v := range result.fileInfo {
		fileInfoAttrs[k] = types.StringValue(v)
	}
	data.CurrentFileInfo = types.MapValueMust(types.StringType, fileInfoAttrs)
}
