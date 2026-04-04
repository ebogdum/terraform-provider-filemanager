// SPDX-License-Identifier: MIT

// Package sync implements the filemanager_sync resource.
package sync

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
	_ resource.Resource                = &SyncResource{}
	_ resource.ResourceWithImportState = &SyncResource{}
)

// NewSyncResource creates a new sync resource.
func NewSyncResource() resource.Resource {
	return &SyncResource{}
}

// SyncResource defines the resource implementation.
type SyncResource struct {
	config *common.ProviderConfig
}

// SyncResourceModel describes the resource data model.
type SyncResourceModel struct {
	ID                 types.String `tfsdk:"id"`
	SourceBackend      types.String `tfsdk:"source_backend"`
	SourcePath         types.String `tfsdk:"source_path"`
	DestinationBackend types.String `tfsdk:"destination_backend"`
	DestinationPath    types.String `tfsdk:"destination_path"`
	DeleteOrphans      types.Bool   `tfsdk:"delete_orphans"`
	ComparisonMode     types.String `tfsdk:"comparison_mode"`
	Includes           types.List   `tfsdk:"includes"`
	Excludes           types.List   `tfsdk:"excludes"`
	Concurrency        types.Int64  `tfsdk:"concurrency"`
	PreserveTimestamps types.Bool   `tfsdk:"preserve_timestamps"`
	Triggers           types.Map    `tfsdk:"triggers"`

	// Computed
	FilesTransferred types.Int64 `tfsdk:"files_transferred"`
	FilesDeleted     types.Int64 `tfsdk:"files_deleted"`
	FilesSkipped     types.Int64 `tfsdk:"files_skipped"`
	BytesTransferred types.Int64 `tfsdk:"bytes_transferred"`
	DurationMs       types.Int64 `tfsdk:"duration_ms"`
}

// Metadata returns the resource type name.
func (r *SyncResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sync"
}

// Schema defines the schema for the resource.
func (r *SyncResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Synchronizes files between two backends.",
		MarkdownDescription: `
Synchronizes files between two backends with support for delete orphans and comparison modes.

## Comparison Modes

- **mtime** (default): Compare by modification time and size
- **size_only**: Compare by size only (faster but less accurate)
- **checksum**: Compare by checksum (slower but most accurate)
`,
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
				Description: "Source path (directory) within the source backend.",
				Required:    true,
			},
			"destination_backend": schema.StringAttribute{
				Description: "Destination backend alias.",
				Required:    true,
			},
			"destination_path": schema.StringAttribute{
				Description: "Destination path (directory) within the destination backend.",
				Required:    true,
			},
			"delete_orphans": schema.BoolAttribute{
				Description: "Delete files at destination that don't exist at source.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"comparison_mode": schema.StringAttribute{
				Description: "How to compare files: 'mtime' (default), 'size_only', or 'checksum'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("mtime"),
			},
			"includes": schema.ListAttribute{
				Description: "Glob patterns to include in the sync.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from the sync.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"concurrency": schema.Int64Attribute{
				Description: "Number of parallel sync workers.",
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
			"triggers": schema.MapAttribute{
				Description: "Map of values that, when changed, trigger a re-sync.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"files_transferred": schema.Int64Attribute{
				Description: "Number of files transferred.",
				Computed:    true,
			},
			"files_deleted": schema.Int64Attribute{
				Description: "Number of files deleted (orphans).",
				Computed:    true,
			},
			"files_skipped": schema.Int64Attribute{
				Description: "Number of files skipped (unchanged).",
				Computed:    true,
			},
			"bytes_transferred": schema.Int64Attribute{
				Description: "Total bytes transferred.",
				Computed:    true,
			},
			"duration_ms": schema.Int64Attribute{
				Description: "Duration of the sync operation in milliseconds.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *SyncResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SyncResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SyncResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating sync", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performSync(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to sync", err.Error())
		return
	}

	r.updateComputedValues(&data, result)
	data.ID = types.StringValue(fmt.Sprintf("%s:%s->%s:%s",
		data.SourceBackend.ValueString(), data.SourcePath.ValueString(),
		data.DestinationBackend.ValueString(), data.DestinationPath.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *SyncResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SyncResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// For sync resources, we just verify the destination exists
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

	// Count files in destination for drift detection
	dstFiles, listErr := backend.List(ctx, data.DestinationPath.ValueString(), plugin.ListOptions{Recursive: true})
	if listErr == nil {
		fileCount := 0
		for _, e := range dstFiles {
			if !e.IsDir {
				fileCount++
			}
		}
		data.FilesTransferred = types.Int64Value(int64(fileCount))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *SyncResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SyncResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating sync", map[string]any{
		"source":      data.SourcePath.ValueString(),
		"destination": data.DestinationPath.ValueString(),
	})

	result, err := r.performSync(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to sync", err.Error())
		return
	}

	r.updateComputedValues(&data, result)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *SyncResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SyncResourceModel

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

	// Delete the synced content
	if err := backend.Delete(ctx, data.DestinationPath.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete synced content", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *SyncResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// syncResult holds the result of a sync operation.
type syncResult struct {
	filesTransferred int
	filesDeleted     int
	filesSkipped     int
	bytesTransferred int64
	duration         time.Duration
}

// performSync synchronizes files between source and destination.
func (r *SyncResource) performSync(ctx context.Context, data *SyncResourceModel) (*syncResult, error) {
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
	result := &syncResult{}

	// List source files
	srcFiles, err := srcBackend.List(ctx, data.SourcePath.ValueString(), plugin.ListOptions{Recursive: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list source: %w", err)
	}

	// List destination files for comparison
	dstFiles, err := dstBackend.List(ctx, data.DestinationPath.ValueString(), plugin.ListOptions{Recursive: true})
	if err != nil && err != plugin.ErrPathNotFound {
		return nil, fmt.Errorf("failed to list destination: %w", err)
	}

	// Build destination file map for quick lookup
	dstMap := make(map[string]plugin.FileInfo)
	for _, f := range dstFiles {
		if !f.IsDir {
			dstMap[f.Name] = f
		}
	}

	// Process source files
	for _, srcFile := range srcFiles {
		if srcFile.IsDir {
			continue
		}

		// Apply filters
		if !r.matchesFilters(srcFile.Name, includes, excludes) {
			result.filesSkipped++
			continue
		}

		dstFile, exists := dstMap[srcFile.Name]

		// Check if file needs to be transferred
		needsTransfer := !exists
		if exists {
			needsTransfer = r.needsUpdate(srcFile, dstFile, data.ComparisonMode.ValueString())
		}

		if needsTransfer {
			srcPath := srcFile.Path
			dstPath := filepath.Join(data.DestinationPath.ValueString(), srcFile.Name)

			bytes, err := r.transferFile(ctx, srcBackend, srcPath, dstBackend, dstPath)
			if err != nil {
				return nil, fmt.Errorf("failed to transfer %s: %w", srcPath, err)
			}

			result.bytesTransferred += bytes
			result.filesTransferred++
		} else {
			result.filesSkipped++
		}

		// Remove from dst map to track orphans
		delete(dstMap, srcFile.Name)
	}

	// Handle orphans (files in dst but not in src)
	if data.DeleteOrphans.ValueBool() {
		for name := range dstMap {
			dstPath := filepath.Join(data.DestinationPath.ValueString(), name)
			if err := dstBackend.Delete(ctx, dstPath); err != nil {
				// Log but continue
				tflog.Warn(ctx, "Failed to delete orphan", map[string]any{
					"path":  dstPath,
					"error": err.Error(),
				})
			} else {
				result.filesDeleted++
			}
		}
	}

	result.duration = time.Since(start)
	return result, nil
}

// needsUpdate determines if a file needs to be updated based on comparison mode.
func (r *SyncResource) needsUpdate(src, dst plugin.FileInfo, mode string) bool {
	switch mode {
	case "checksum":
		return src.SHA256 != dst.SHA256
	case "size_only":
		return src.Size != dst.Size
	default: // "mtime"
		return src.ModTime.After(dst.ModTime) || src.Size != dst.Size
	}
}

// transferFile transfers a single file from source to destination.
func (r *SyncResource) transferFile(ctx context.Context, srcBackend plugin.Backend, srcPath string, dstBackend plugin.Backend, dstPath string) (int64, error) {
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
		CreateDirs: true,
		Overwrite:  true,
		Atomic:     true,
	}

	if err := dstBackend.Write(ctx, dstPath, cr, writeOpts); err != nil {
		return 0, fmt.Errorf("failed to write destination: %w", err)
	}

	return cr.Count(), nil
}

// getBackend returns the appropriate backend.
func (r *SyncResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// getListValues extracts string values from a types.List.
func (r *SyncResource) getListValues(list types.List) []string {
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
func (r *SyncResource) matchesFilters(name string, includes, excludes []string) bool {
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
func (r *SyncResource) updateComputedValues(data *SyncResourceModel, result *syncResult) {
	data.FilesTransferred = types.Int64Value(int64(result.filesTransferred))
	data.FilesDeleted = types.Int64Value(int64(result.filesDeleted))
	data.FilesSkipped = types.Int64Value(int64(result.filesSkipped))
	data.BytesTransferred = types.Int64Value(result.bytesTransferred)
	data.DurationMs = types.Int64Value(result.duration.Milliseconds())
}
