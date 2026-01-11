// SPDX-License-Identifier: MIT

// Package upload implements the filemanager_upload resource.
package upload

import (
	"context"
	"fmt"
	"path/filepath"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
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
	_ resource.Resource                = &UploadResource{}
	_ resource.ResourceWithImportState = &UploadResource{}
)

// NewUploadResource creates a new upload resource.
func NewUploadResource() resource.Resource {
	return &UploadResource{}
}

// UploadResource defines the resource implementation.
type UploadResource struct {
	config *common.ProviderConfig
}

// UploadResourceModel describes the resource data model.
type UploadResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	SourceBackend      types.String `tfsdk:"source_backend"`
	SourcePath         types.String `tfsdk:"source_path"`
	DestinationBackend types.String `tfsdk:"destination_backend"`
	DestinationPath    types.String `tfsdk:"destination_path"`
	Recursive          types.Bool   `tfsdk:"recursive"`
	Includes           types.List   `tfsdk:"includes"`
	Excludes           types.List   `tfsdk:"excludes"`
	ChecksumVerify     types.Bool   `tfsdk:"checksum_verify"`
	Concurrency        types.Int64  `tfsdk:"concurrency"`
	PartSizeMB         types.Int64  `tfsdk:"part_size_mb"`
	PreserveTimestamps types.Bool   `tfsdk:"preserve_timestamps"`
	Overwrite          types.Bool   `tfsdk:"overwrite"`
	Triggers           types.Map    `tfsdk:"triggers"`

	// Computed
	BytesTransferred types.Int64  `tfsdk:"bytes_transferred"`
	FilesTransferred types.Int64  `tfsdk:"files_transferred"`
	DurationMs       types.Int64  `tfsdk:"duration_ms"`
	Directory        types.String `tfsdk:"directory"`
	Filename         types.String `tfsdk:"filename"`
	Extension        types.String `tfsdk:"extension"`
	AbsolutePath     types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *UploadResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_upload"
}

// Schema defines the schema for the resource.
func (r *UploadResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Uploads files or directories from a source backend to a destination backend.",
		MarkdownDescription: "Uploads files or directories from a source backend to a destination backend.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_backend": schema.StringAttribute{
				Description: "Source backend alias. Defaults to 'local'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"source_path": schema.StringAttribute{
				Description: "Source path (file or directory) within the source backend.",
				Required:    true,
			},
			"destination_backend": schema.StringAttribute{
				Description: "Destination backend alias (required for upload).",
				Required:    true,
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path within the destination backend.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Upload directories recursively.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"includes": schema.ListAttribute{
				Description: "Glob patterns to include in the upload.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from the upload.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"checksum_verify": schema.BoolAttribute{
				Description: "Verify checksums after upload.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"concurrency": schema.Int64Attribute{
				Description: "Number of parallel upload workers.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},
			"part_size_mb": schema.Int64Attribute{
				Description: "Part size in MB for multipart uploads.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(64),
			},
			"preserve_timestamps": schema.BoolAttribute{
				Description: "Preserve modification timestamps.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"overwrite": schema.BoolAttribute{
				Description: "Overwrite existing files at destination.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger a re-upload.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"bytes_transferred": schema.Int64Attribute{
				Description: "Total bytes transferred.",
				Computed:    true,
			},
			"files_transferred": schema.Int64Attribute{
				Description: "Number of files transferred.",
				Computed:    true,
			},
			"duration_ms": schema.Int64Attribute{
				Description: "Duration of the upload operation in milliseconds.",
				Computed:    true,
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
		},
	}
}

// Configure configures the resource with provider data.
func (r *UploadResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *UploadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data UploadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating upload", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performUpload(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload", err.Error())
		return
	}

	r.updateComputedValues(&data, result)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s->%s:%s",
		data.SourceBackend.ValueString(), data.SourcePath.ValueString(),
		data.DestinationBackend.ValueString(), data.DestinationPath.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *UploadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data UploadResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For uploads, we just verify the destination exists
	backend, err := r.getBackend(ctx, data.DestinationBackend.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get destination backend", err.Error())
		return
	}

	exists, err := backend.Exists(ctx, data.DestinationPath.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check destination existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.DestinationPath.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *UploadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data UploadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating upload", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performUpload(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to upload", err.Error())
		return
	}

	r.updateComputedValues(&data, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *UploadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data UploadResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get destination backend
	backend, err := r.getBackend(ctx, data.DestinationBackend.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get destination backend", err.Error())
		return
	}

	// Delete the uploaded content
	if err := backend.Delete(ctx, data.DestinationPath.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete uploaded content", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *UploadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// uploadResult holds the result of an upload operation.
type uploadResult struct {
	bytesTransferred int64
	filesTransferred int
	duration         time.Duration
}

// performUpload uploads files from source to destination.
func (r *UploadResource) performUpload(ctx context.Context, data *UploadResourceModel) (*uploadResult, error) {
	srcBackend, err := r.getBackend(ctx, data.SourceBackend.ValueString())
	if err != nil {
		return nil, fmt.Errorf("failed to get source backend: %w", err)
	}

	dstBackend, err := r.getBackend(ctx, data.DestinationBackend.ValueString())
	if err != nil {
		return nil, fmt.Errorf("failed to get destination backend: %w", err)
	}

	// Parse include/exclude patterns
	includes := r.getListValues(data.Includes)
	excludes := r.getListValues(data.Excludes)

	start := time.Now()
	result := &uploadResult{}

	// Check if source is a file or directory
	srcInfo, err := srcBackend.Stat(ctx, data.SourcePath.ValueString())
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	opts := plugin.TransferOptions{
		Concurrency:    int(data.Concurrency.ValueInt64()),
		PartSize:       data.PartSizeMB.ValueInt64() * 1024 * 1024,
		ChecksumVerify: data.ChecksumVerify.ValueBool(),
		PreserveMtime:  data.PreserveTimestamps.ValueBool(),
	}

	if srcInfo.IsDir {
		if !data.Recursive.ValueBool() {
			return nil, fmt.Errorf("source is a directory but recursive is false")
		}

		// List all files in source directory
		files, err := srcBackend.List(ctx, data.SourcePath.ValueString(), plugin.ListOptions{Recursive: true})
		if err != nil {
			return nil, fmt.Errorf("failed to list source directory: %w", err)
		}

		for _, file := range files {
			if file.IsDir {
				continue
			}

			// Apply filters
			if !r.matchesFilters(file.Name, includes, excludes) {
				continue
			}

			// Calculate relative path
			relPath := file.Name

			// Upload file
			srcPath := file.Path
			dstPath := filepath.Join(data.DestinationPath.ValueString(), relPath)

			bytes, err := r.uploadFile(ctx, srcBackend, srcPath, dstBackend, dstPath, opts)
			if err != nil {
				return nil, fmt.Errorf("failed to upload %s: %w", srcPath, err)
			}

			result.bytesTransferred += bytes
			result.filesTransferred++
		}
	} else {
		// Single file upload
		bytes, err := r.uploadFile(ctx, srcBackend, data.SourcePath.ValueString(), dstBackend, data.DestinationPath.ValueString(), opts)
		if err != nil {
			return nil, err
		}
		result.bytesTransferred = bytes
		result.filesTransferred = 1
	}

	result.duration = time.Since(start)
	return result, nil
}

// uploadFile uploads a single file.
func (r *UploadResource) uploadFile(ctx context.Context, srcBackend plugin.Backend, srcPath string, dstBackend plugin.Backend, dstPath string, opts plugin.TransferOptions) (int64, error) {
	// Read from source
	reader, err := srcBackend.Read(ctx, srcPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read source: %w", err)
	}
	defer reader.Close()

	// Create counting reader
	cr := common.NewCountingReader(reader)

	// Write to destination
	writeOpts := plugin.WriteOptions{
		CreateDirs:       true,
		Overwrite:        true,
		Atomic:           true,
		VerifyAfterWrite: opts.ChecksumVerify,
	}

	if err := dstBackend.Write(ctx, dstPath, cr, writeOpts); err != nil {
		return 0, fmt.Errorf("failed to write destination: %w", err)
	}

	return cr.Count, nil
}

// getBackend returns the appropriate backend.
func (r *UploadResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// getListValues extracts string values from a types.List.
func (r *UploadResource) getListValues(list types.List) []string {
	if list.IsNull() || list.IsUnknown() {
		return nil
	}
	elements := list.Elements()
	result := make([]string, 0, len(elements))
	for _, e := range elements {
		if strVal, ok := e.(types.String); ok {
			result = append(result, strVal.ValueString())
		}
	}
	return result
}

// matchesFilters checks if a filename matches the include/exclude filters.
func (r *UploadResource) matchesFilters(name string, includes, excludes []string) bool {
	// If includes are specified, file must match at least one
	if len(includes) > 0 {
		matched := false
		for _, pattern := range includes {
			if m, _ := filepath.Match(pattern, name); m {
				matched = true
				break
			}
			// Also try matching against basename
			if m, _ := filepath.Match(pattern, filepath.Base(name)); m {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}

	// Check excludes
	for _, pattern := range excludes {
		if m, _ := filepath.Match(pattern, name); m {
			return false
		}
		if m, _ := filepath.Match(pattern, filepath.Base(name)); m {
			return false
		}
	}

	return true
}

// updateComputedValues updates the computed values in the model.
func (r *UploadResource) updateComputedValues(data *UploadResourceModel, result *uploadResult) {
	data.BytesTransferred = types.Int64Value(result.bytesTransferred)
	data.FilesTransferred = types.Int64Value(int64(result.filesTransferred))
	data.DurationMs = types.Int64Value(result.duration.Milliseconds())

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.DestinationPath.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
