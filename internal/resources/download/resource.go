// SPDX-License-Identifier: MIT

// Package download implements the filemanager_download resource.
package download

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
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
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &DownloadResource{}
	_ resource.ResourceWithImportState = &DownloadResource{}
)

// NewDownloadResource creates a new download resource.
func NewDownloadResource() resource.Resource {
	return &DownloadResource{}
}

// DownloadResource defines the resource implementation.
type DownloadResource struct {
	config *common.ProviderConfig
}

// DownloadResourceModel describes the resource data model.
type DownloadResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	SourceBackend       types.String `tfsdk:"source_backend"`
	SourcePath          types.String `tfsdk:"source_path"`
	DestinationBackend  types.String `tfsdk:"destination_backend"`
	DestinationPath     types.String `tfsdk:"destination_path"`
	Recursive           types.Bool   `tfsdk:"recursive"`
	Includes            types.List   `tfsdk:"includes"`
	Excludes            types.List   `tfsdk:"excludes"`
	ExpectedChecksum    types.String `tfsdk:"expected_checksum"`
	Concurrency         types.Int64  `tfsdk:"concurrency"`
	PreserveTimestamps  types.Bool   `tfsdk:"preserve_timestamps"`
	Overwrite           types.Bool   `tfsdk:"overwrite"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	Triggers            types.Map    `tfsdk:"triggers"`

	// Computed
	BytesTransferred types.Int64  `tfsdk:"bytes_transferred"`
	FilesTransferred types.Int64  `tfsdk:"files_transferred"`
	DurationMs       types.Int64  `tfsdk:"duration_ms"`
	MD5              types.String `tfsdk:"md5"`
	SHA256           types.String `tfsdk:"sha256"`
	Directory        types.String `tfsdk:"directory"`
	Filename         types.String `tfsdk:"filename"`
	Extension        types.String `tfsdk:"extension"`
	AbsolutePath     types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *DownloadResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_download"
}

// Schema defines the schema for the resource.
func (r *DownloadResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Downloads files or directories from a source backend to a destination backend.",
		MarkdownDescription: "Downloads files or directories from a remote backend to local or another backend.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source_backend": schema.StringAttribute{
				Description: "Source backend alias (required for download).",
				Required:    true,
			},
			"source_path": schema.StringAttribute{
				Description: "Source path (file or directory) within the source backend.",
				Required:    true,
			},
			"destination_backend": schema.StringAttribute{
				Description: "Destination backend alias. Defaults to 'local'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path within the destination backend.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Download directories recursively.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"includes": schema.ListAttribute{
				Description: "Glob patterns to include in the download.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from the download.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"expected_checksum": schema.StringAttribute{
				Description: "Expected checksum for verification (format: algorithm:hash, supported algorithm: sha256).",
				Optional:    true,
			},
			"concurrency": schema.Int64Attribute{
				Description: "Number of parallel download workers.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(4),
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
			"file_permission": schema.StringAttribute{
				Description: "File permission mode in octal format (e.g., '0644').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode in octal format (e.g., '0755').",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger a re-download.",
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
				Description: "Duration of the download operation in milliseconds.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "Deprecated insecure checksum field. Always null.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the downloaded file (for single file download).",
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
func (r *DownloadResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *DownloadResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data DownloadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating download", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performDownload(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to download", err.Error())
		return
	}

	r.updateComputedValues(&data, result)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s->%s:%s",
		data.SourceBackend.ValueString(), data.SourcePath.ValueString(),
		data.DestinationBackend.ValueString(), data.DestinationPath.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *DownloadResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data DownloadResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Verify the destination exists
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

	// Update checksums for single file
	info, err := backend.Stat(ctx, data.DestinationPath.ValueString())
	if err == nil && !info.IsDir {
		r.updateFileChecksums(ctx, &data, backend)
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
func (r *DownloadResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data DownloadResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating download", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performDownload(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to download", err.Error())
		return
	}

	r.updateComputedValues(&data, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *DownloadResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data DownloadResourceModel

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

	// Delete the downloaded content
	if err := backend.Delete(ctx, data.DestinationPath.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete downloaded content", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *DownloadResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// downloadResult holds the result of a download operation.
type downloadResult struct {
	bytesTransferred int64
	filesTransferred int
	duration         time.Duration
	sha256           string
}

// performDownload downloads files from source to destination.
func (r *DownloadResource) performDownload(ctx context.Context, data *DownloadResourceModel) (*downloadResult, error) {
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
	result := &downloadResult{}

	// Check if source is a file or directory
	srcInfo, err := srcBackend.Stat(ctx, data.SourcePath.ValueString())
	if err != nil {
		return nil, fmt.Errorf("source not found: %w", err)
	}

	fileMode := common.ParseFileMode(data.FilePermission.ValueString())
	dirMode := common.ParseDirMode(data.DirectoryPermission.ValueString())

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

			// Download file
			srcPath := file.Path
			dstPath := filepath.Join(data.DestinationPath.ValueString(), relPath)

			bytes, err := r.downloadFile(ctx, srcBackend, srcPath, dstBackend, dstPath, fileMode, dirMode)
			if err != nil {
				return nil, fmt.Errorf("failed to download %s: %w", srcPath, err)
			}

			result.bytesTransferred += bytes
			result.filesTransferred++
		}
	} else {
		// Single file download
		bytes, err := r.downloadFile(ctx, srcBackend, data.SourcePath.ValueString(), dstBackend, data.DestinationPath.ValueString(), fileMode, dirMode)
		if err != nil {
			return nil, err
		}
		result.bytesTransferred = bytes
		result.filesTransferred = 1

		// Calculate checksums for single file
		r.calculateFileChecksums(ctx, data, dstBackend, result)

		// Verify expected checksum if provided
		if !data.ExpectedChecksum.IsNull() && data.ExpectedChecksum.ValueString() != "" {
			if err := r.verifyChecksum(data.ExpectedChecksum.ValueString(), result); err != nil {
				return nil, err
			}
		}
	}

	result.duration = time.Since(start)
	return result, nil
}

// downloadFile downloads a single file.
func (r *DownloadResource) downloadFile(ctx context.Context, srcBackend plugin.Backend, srcPath string, dstBackend plugin.Backend, dstPath string, fileMode, dirMode os.FileMode) (int64, error) {
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
		Mode:       fileMode,
		DirMode:    dirMode,
		CreateDirs: true,
		Overwrite:  true,
		Atomic:     true,
	}

	if err := dstBackend.Write(ctx, dstPath, cr, writeOpts); err != nil {
		return 0, fmt.Errorf("failed to write destination: %w", err)
	}

	return cr.Count, nil
}

// calculateFileChecksums calculates checksums for a downloaded file.
func (r *DownloadResource) calculateFileChecksums(ctx context.Context, data *DownloadResourceModel, backend plugin.Backend, result *downloadResult) {
	reader, err := backend.Read(ctx, data.DestinationPath.ValueString())
	if err != nil {
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	sha256sum := sha256.Sum256(content)
	result.sha256 = hex.EncodeToString(sha256sum[:])
}

// updateFileChecksums updates checksums in the data model.
func (r *DownloadResource) updateFileChecksums(ctx context.Context, data *DownloadResourceModel, backend plugin.Backend) {
	reader, err := backend.Read(ctx, data.DestinationPath.ValueString())
	if err != nil {
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		return
	}

	data.MD5 = types.StringNull()

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))
}

// verifyChecksum verifies the downloaded file matches the expected checksum.
func (r *DownloadResource) verifyChecksum(expected string, result *downloadResult) error {
	parts := strings.SplitN(expected, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid checksum format, expected algorithm:hash")
	}

	algo := strings.ToLower(parts[0])
	expectedHash := strings.ToLower(parts[1])

	var actualHash string
	switch algo {
	case "sha256":
		actualHash = result.sha256
	default:
		return fmt.Errorf("unsupported checksum algorithm: %s", algo)
	}

	if actualHash != expectedHash {
		return fmt.Errorf("checksum mismatch: expected %s:%s, got %s:%s", algo, expectedHash, algo, actualHash)
	}

	return nil
}

// getBackend returns the appropriate backend.
func (r *DownloadResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// getListValues extracts string values from a types.List.
func (r *DownloadResource) getListValues(list types.List) []string {
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
func (r *DownloadResource) matchesFilters(name string, includes, excludes []string) bool {
	// If includes are specified, file must match at least one
	if len(includes) > 0 {
		matched := false
		for _, pattern := range includes {
			if m, _ := filepath.Match(pattern, name); m {
				matched = true
				break
			}
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
func (r *DownloadResource) updateComputedValues(data *DownloadResourceModel, result *downloadResult) {
	data.BytesTransferred = types.Int64Value(result.bytesTransferred)
	data.FilesTransferred = types.Int64Value(int64(result.filesTransferred))
	data.DurationMs = types.Int64Value(result.duration.Milliseconds())

	data.MD5 = types.StringNull()

	if result.sha256 != "" {
		data.SHA256 = types.StringValue(result.sha256)
	} else {
		data.SHA256 = types.StringNull()
	}

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.DestinationPath.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
