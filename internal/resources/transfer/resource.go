// SPDX-License-Identifier: MIT

// Package transfer implements the filemanager_transfer resource.
package transfer

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TransferResource{}
	_ resource.ResourceWithImportState = &TransferResource{}
)

// NewTransferResource creates a new transfer resource.
func NewTransferResource() resource.Resource {
	return &TransferResource{}
}

// TransferResource defines the resource implementation.
type TransferResource struct {
	config *common.ProviderConfig
}

// TransferResourceModel describes the resource data model.
type TransferResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	SourceBackend       types.String `tfsdk:"source_backend"`
	SourcePath          types.String `tfsdk:"source_path"`
	DestinationBackend  types.String `tfsdk:"destination_backend"`
	DestinationPath     types.String `tfsdk:"destination_path"`
	Recursive           types.Bool   `tfsdk:"recursive"`
	Includes            types.List   `tfsdk:"includes"`
	Excludes            types.List   `tfsdk:"excludes"`
	ChecksumVerify      types.Bool   `tfsdk:"checksum_verify"`
	Concurrency         types.Int64  `tfsdk:"concurrency"`
	BufferSizeKB        types.Int64  `tfsdk:"buffer_size_kb"`
	PreserveTimestamps  types.Bool   `tfsdk:"preserve_timestamps"`
	PreservePermissions types.Bool   `tfsdk:"preserve_permissions"`
	Overwrite           types.Bool   `tfsdk:"overwrite"`
	Streaming           types.Bool   `tfsdk:"streaming"`
	ZeroCopy            types.Bool   `tfsdk:"zero_copy"`
	Triggers            types.Map    `tfsdk:"triggers"`

	// Computed
	BytesTransferred types.Int64  `tfsdk:"bytes_transferred"`
	FilesTransferred types.Int64  `tfsdk:"files_transferred"`
	FilesFailed      types.Int64  `tfsdk:"files_failed"`
	DurationMs       types.Int64  `tfsdk:"duration_ms"`
	TransferMethod   types.String `tfsdk:"transfer_method"`
	Directory        types.String `tfsdk:"directory"`
	Filename         types.String `tfsdk:"filename"`
	Extension        types.String `tfsdk:"extension"`
	AbsolutePath     types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *TransferResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_transfer"
}

// Schema defines the schema for the resource.
func (r *TransferResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Transfers files or directories between any two backends.",
		MarkdownDescription: "Transfers files or directories between any two backends.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_backend": schema.StringAttribute{
				Description: "Source backend alias.",
				Required:    true,
			},
			"source_path": schema.StringAttribute{
				Description: "Source path (file or directory) within the source backend.",
				Required:    true,
			},
			"destination_backend": schema.StringAttribute{
				Description: "Destination backend alias.",
				Required:    true,
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path within the destination backend.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Transfer directories recursively.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"includes": schema.ListAttribute{
				Description: "Glob patterns to include in the transfer.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from the transfer.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"checksum_verify": schema.BoolAttribute{
				Description: "Verify checksums after transfer.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"concurrency": schema.Int64Attribute{
				Description: "Number of parallel transfer workers.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
			},
			"buffer_size_kb": schema.Int64Attribute{
				Description: "Buffer size in KB for transfer operations.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(32),
			},
			"preserve_timestamps": schema.BoolAttribute{
				Description: "Preserve modification timestamps.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"preserve_permissions": schema.BoolAttribute{
				Description: "Preserve file permissions.",
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
			"streaming": schema.BoolAttribute{
				Description: "Use streaming mode for transfers.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"zero_copy": schema.BoolAttribute{
				Description: "Attempt zero-copy transfers where supported.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger a re-transfer.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"bytes_transferred": schema.Int64Attribute{
				Description: "Total bytes transferred.",
				Computed:    true,
			},
			"files_transferred": schema.Int64Attribute{
				Description: "Number of files transferred successfully.",
				Computed:    true,
			},
			"files_failed": schema.Int64Attribute{
				Description: "Number of files that failed to transfer.",
				Computed:    true,
			},
			"duration_ms": schema.Int64Attribute{
				Description: "Duration of the transfer operation in milliseconds.",
				Computed:    true,
			},
			"transfer_method": schema.StringAttribute{
				Description: "Method used for transfer (direct, parallel_chunked, kernel_zero_copy, streaming_buffer, standard).",
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
func (r *TransferResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *TransferResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TransferResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating transfer", map[string]any{
		"source_backend":      data.SourceBackend.ValueString(),
		"source_path":         data.SourcePath.ValueString(),
		"destination_backend": data.DestinationBackend.ValueString(),
		"destination_path":    data.DestinationPath.ValueString(),
	})

	result, err := r.performTransfer(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to transfer", err.Error())
		return
	}

	r.updateComputedValues(&data, result)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s->%s:%s",
		data.SourceBackend.ValueString(), data.SourcePath.ValueString(),
		data.DestinationBackend.ValueString(), data.DestinationPath.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *TransferResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TransferResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For transfers, we just verify the destination exists
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
func (r *TransferResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TransferResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating transfer", map[string]any{
		"source_backend":      data.SourceBackend.ValueString(),
		"source_path":         data.SourcePath.ValueString(),
		"destination_backend": data.DestinationBackend.ValueString(),
		"destination_path":    data.DestinationPath.ValueString(),
	})

	result, err := r.performTransfer(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to transfer", err.Error())
		return
	}

	r.updateComputedValues(&data, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *TransferResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TransferResourceModel

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

	// Delete the transferred content
	if err := backend.Delete(ctx, data.DestinationPath.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete transferred content", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *TransferResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse composite ID format: "backend:path"
	parts := strings.SplitN(req.ID, ":", 2)
	if len(parts) != 2 {
		resp.Diagnostics.AddError(
			"Invalid Import ID",
			fmt.Sprintf("Expected format 'backend:path', got: %s", req.ID),
		)
		return
	}
	resp.State.SetAttribute(ctx, path.Root("destination_backend"), parts[0])
	resp.State.SetAttribute(ctx, path.Root("destination_path"), parts[1])
}

// transferResult holds the result of a transfer operation.
type transferResult struct {
	bytesTransferred int64
	filesTransferred int
	filesFailed      int
	duration         time.Duration
	method           plugin.TransferMethod
}

// performTransfer transfers files from source to destination.
func (r *TransferResource) performTransfer(ctx context.Context, data *TransferResourceModel) (*transferResult, error) {
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
	result := &transferResult{
		method: plugin.TransferMethodStandard,
	}

	// Check if source is a file or directory
	srcInfo, err := srcBackend.Stat(ctx, data.SourcePath.ValueString())
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	opts := plugin.TransferOptions{
		BufferSize:     data.BufferSizeKB.ValueInt64() * 1024,
		Concurrency:    int(data.Concurrency.ValueInt64()),
		ChecksumVerify: data.ChecksumVerify.ValueBool(),
		PreserveMtime:  data.PreserveTimestamps.ValueBool(),
		PreserveMode:   data.PreservePermissions.ValueBool(),
		Streaming:      data.Streaming.ValueBool(),
		ZeroCopy:       data.ZeroCopy.ValueBool(),
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

			// Validate path safety
			cleanRel := filepath.Clean(relPath)
			if filepath.IsAbs(cleanRel) || strings.HasPrefix(cleanRel, "..") {
				return nil, fmt.Errorf("backend returned unsafe path: %s", relPath)
			}

			// Transfer file
			srcPath := file.Path
			dstPath := filepath.Join(data.DestinationPath.ValueString(), cleanRel)

			bytes, method, err := r.transferFile(ctx, srcBackend, srcPath, dstBackend, dstPath, opts, data.Overwrite.ValueBool())
			if err != nil {
				tflog.Warn(ctx, "Failed to transfer file", map[string]any{
					"source": srcPath,
					"error":  err.Error(),
				})
				result.filesFailed++
				continue
			}

			result.bytesTransferred += bytes
			result.filesTransferred++
			result.method = method
		}
	} else {
		// Single file transfer
		bytes, method, err := r.transferFile(ctx, srcBackend, data.SourcePath.ValueString(), dstBackend, data.DestinationPath.ValueString(), opts, data.Overwrite.ValueBool())
		if err != nil {
			return nil, err
		}
		result.bytesTransferred = bytes
		result.filesTransferred = 1
		result.method = method
	}

	result.duration = time.Since(start)

	if result.filesFailed > 0 {
		return result, fmt.Errorf("%d of %d files failed to transfer", result.filesFailed, result.filesTransferred+result.filesFailed)
	}

	return result, nil
}

// transferFile transfers a single file between backends.
func (r *TransferResource) transferFile(ctx context.Context, srcBackend plugin.Backend, srcPath string, dstBackend plugin.Backend, dstPath string, opts plugin.TransferOptions, overwrite bool) (int64, plugin.TransferMethod, error) {
	// Check if destination exists when overwrite is false
	if !overwrite {
		exists, err := dstBackend.Exists(ctx, dstPath)
		if err != nil {
			return 0, plugin.TransferMethodStandard, fmt.Errorf("failed to check destination existence: %w", err)
		}
		if exists {
			return 0, plugin.TransferMethodStandard, fmt.Errorf("destination already exists and overwrite is false: %s", dstPath)
		}
	}

	// Read from source
	reader, err := srcBackend.Read(ctx, srcPath)
	if err != nil {
		return 0, plugin.TransferMethodStandard, fmt.Errorf("failed to read source: %w", err)
	}
	defer reader.Close()

	// Create counting reader
	cr := common.NewCountingReader(reader)

	// Write to destination
	writeOpts := plugin.WriteOptions{
		CreateDirs:       true,
		Overwrite:        overwrite,
		Atomic:           true,
		VerifyAfterWrite: opts.ChecksumVerify,
	}

	// Get source file info for permissions/timestamps
	if opts.PreserveMode || opts.PreserveMtime {
		srcInfo, err := srcBackend.Stat(ctx, srcPath)
		if err == nil {
			if opts.PreserveMode {
				writeOpts.Mode = srcInfo.Mode
			}
		}
	}

	if err := dstBackend.Write(ctx, dstPath, cr, writeOpts); err != nil {
		return 0, plugin.TransferMethodStandard, fmt.Errorf("failed to write destination: %w", err)
	}

	// Determine transfer method based on options
	method := plugin.TransferMethodStandard
	if opts.Streaming {
		method = plugin.TransferMethodStreamingBuffer
	}

	return cr.Count(), method, nil
}

// getBackend returns the appropriate backend.
func (r *TransferResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// getListValues extracts string values from a types.List.
func (r *TransferResource) getListValues(list types.List) []string {
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
func (r *TransferResource) matchesFilters(name string, includes, excludes []string) bool {
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
func (r *TransferResource) updateComputedValues(data *TransferResourceModel, result *transferResult) {
	data.BytesTransferred = types.Int64Value(result.bytesTransferred)
	data.FilesTransferred = types.Int64Value(int64(result.filesTransferred))
	data.FilesFailed = types.Int64Value(int64(result.filesFailed))
	data.DurationMs = types.Int64Value(result.duration.Milliseconds())
	data.TransferMethod = types.StringValue(string(result.method))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.DestinationPath.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
