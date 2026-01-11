// SPDX-License-Identifier: MIT

// Package archive implements the filemanager_archive resource.
package archive

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &ArchiveResource{}
	_ resource.ResourceWithImportState = &ArchiveResource{}
)

// NewArchiveResource creates a new archive resource.
func NewArchiveResource() resource.Resource {
	return &ArchiveResource{}
}

// ArchiveResource defines the resource implementation.
type ArchiveResource struct {
	config *common.ProviderConfig
}

// ArchiveResourceModel describes the resource data model.
type ArchiveResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Type                types.String `tfsdk:"type"`
	SourceDir           types.String `tfsdk:"source_dir"`
	SourceFiles         types.List   `tfsdk:"source_files"`
	Excludes            types.List   `tfsdk:"excludes"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`

	// Computed
	Size         types.Int64  `tfsdk:"size"`
	FileCount    types.Int64  `tfsdk:"file_count"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *ArchiveResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_archive"
}

// Schema defines the schema for the resource.
func (r *ArchiveResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Creates an archive file (zip, tar, tar.gz).",
		MarkdownDescription: `Creates an archive file from a source directory or list of files.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path where the archive will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				Description: "Archive type: zip, tar, tar.gz.",
				Required:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("zip", "tar", "tar.gz"),
				},
			},
			"source_dir": schema.StringAttribute{
				Description: "Source directory to archive.",
				Optional:    true,
			},
			"source_files": schema.ListAttribute{
				Description: "List of source file paths to include in the archive.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from the archive.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode for the archive.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode for parent directories.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"create_parent_dirs": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"size": schema.Int64Attribute{
				Description: "Size of the archive in bytes.",
				Computed:    true,
			},
			"file_count": schema.Int64Attribute{
				Description: "Number of files in the archive.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "MD5 checksum of the archive.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the archive.",
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
func (r *ArchiveResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *ArchiveResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data ArchiveResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating archive", map[string]any{
		"path": data.Path.ValueString(),
		"type": data.Type.ValueString(),
	})

	// Collect files to archive
	files, err := r.collectFiles(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to collect files", err.Error())
		return
	}

	// Create parent directories if needed
	if data.CreateParentDirs.ValueBool() {
		dir := filepath.Dir(data.Path.ValueString())
		mode := common.ParseDirMode(data.DirectoryPermission.ValueString())
		if err := os.MkdirAll(dir, mode); err != nil {
			resp.Diagnostics.AddError("Failed to create parent directories", err.Error())
			return
		}
	}

	// Create the archive
	fileCount, err := r.createArchive(ctx, &data, files)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create archive", err.Error())
		return
	}

	// Update computed values
	r.updateComputedValues(&data, fileCount)
	data.ID = data.Path

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *ArchiveResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data ArchiveResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if archive exists
	info, err := os.Stat(data.Path.ValueString())
	if err != nil {
		if os.IsNotExist(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to stat archive", err.Error())
		return
	}

	data.Size = types.Int64Value(info.Size())

	// Calculate checksums
	content, err := os.ReadFile(data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read archive", err.Error())
		return
	}

	md5sum := md5.Sum(content)
	data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *ArchiveResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data ArchiveResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating archive", map[string]any{
		"path": data.Path.ValueString(),
	})

	files, err := r.collectFiles(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to collect files", err.Error())
		return
	}

	fileCount, err := r.createArchive(ctx, &data, files)
	if err != nil {
		resp.Diagnostics.AddError("Failed to create archive", err.Error())
		return
	}

	r.updateComputedValues(&data, fileCount)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *ArchiveResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data ArchiveResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if err := os.Remove(data.Path.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("Failed to delete archive", err.Error())
		return
	}
}

// ImportState imports an existing resource.
func (r *ArchiveResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// collectFiles collects files to include in the archive.
func (r *ArchiveResource) collectFiles(ctx context.Context, data *ArchiveResourceModel) ([]string, error) {
	var files []string

	// Get excludes
	var excludes []string
	if !data.Excludes.IsNull() && !data.Excludes.IsUnknown() {
		elements := data.Excludes.Elements()
		for _, e := range elements {
			if strVal, ok := e.(types.String); ok {
				excludes = append(excludes, strVal.ValueString())
			}
		}
	}

	// If source_dir is specified, walk the directory
	if !data.SourceDir.IsNull() && !data.SourceDir.IsUnknown() && data.SourceDir.ValueString() != "" {
		sourceDir := data.SourceDir.ValueString()
		err := filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			// Skip directories
			if info.IsDir() {
				return nil
			}

			// Get relative path for matching
			relPath, err := filepath.Rel(sourceDir, path)
			if err != nil {
				return err
			}

			// Check excludes
			for _, pattern := range excludes {
				matched, err := filepath.Match(pattern, relPath)
				if err != nil {
					continue
				}
				if matched {
					return nil
				}
				// Also try matching against the filename
				matched, _ = filepath.Match(pattern, info.Name())
				if matched {
					return nil
				}
			}

			files = append(files, path)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}

	// If source_files is specified, add those
	if !data.SourceFiles.IsNull() && !data.SourceFiles.IsUnknown() {
		elements := data.SourceFiles.Elements()
		for _, e := range elements {
			if strVal, ok := e.(types.String); ok {
				files = append(files, strVal.ValueString())
			}
		}
	}

	return files, nil
}

// createArchive creates the archive file.
func (r *ArchiveResource) createArchive(ctx context.Context, data *ArchiveResourceModel, files []string) (int, error) {
	archivePath := data.Path.ValueString()
	archiveType := data.Type.ValueString()

	// Create output file
	mode := common.ParseFileMode(data.FilePermission.ValueString())
	outFile, err := os.OpenFile(archivePath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
	if err != nil {
		return 0, fmt.Errorf("failed to create archive file: %w", err)
	}
	defer outFile.Close()

	switch archiveType {
	case "zip":
		return r.createZipArchive(outFile, files, data)
	case "tar":
		return r.createTarArchive(outFile, files, data)
	case "tar.gz":
		return r.createTarGzArchive(outFile, files, data)
	default:
		return 0, fmt.Errorf("unknown archive type: %s", archiveType)
	}
}

// createZipArchive creates a zip archive.
func (r *ArchiveResource) createZipArchive(w io.Writer, files []string, data *ArchiveResourceModel) (int, error) {
	zw := zip.NewWriter(w)
	defer zw.Close()

	sourceDir := data.SourceDir.ValueString()
	count := 0

	for _, file := range files {
		// Determine archive path
		archivePath := file
		if sourceDir != "" {
			relPath, err := filepath.Rel(sourceDir, file)
			if err == nil {
				archivePath = relPath
			}
		} else {
			archivePath = filepath.Base(file)
		}

		// Read file
		content, err := os.ReadFile(file)
		if err != nil {
			return count, fmt.Errorf("failed to read %s: %w", file, err)
		}

		// Create file in archive
		fw, err := zw.Create(archivePath)
		if err != nil {
			return count, fmt.Errorf("failed to create archive entry %s: %w", archivePath, err)
		}

		if _, err := fw.Write(content); err != nil {
			return count, fmt.Errorf("failed to write to archive entry %s: %w", archivePath, err)
		}

		count++
	}

	return count, nil
}

// createTarArchive creates a tar archive.
func (r *ArchiveResource) createTarArchive(w io.Writer, files []string, data *ArchiveResourceModel) (int, error) {
	tw := tar.NewWriter(w)
	defer tw.Close()

	sourceDir := data.SourceDir.ValueString()
	count := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			return count, fmt.Errorf("failed to stat %s: %w", file, err)
		}

		// Determine archive path
		archivePath := file
		if sourceDir != "" {
			relPath, err := filepath.Rel(sourceDir, file)
			if err == nil {
				archivePath = relPath
			}
		} else {
			archivePath = filepath.Base(file)
		}

		// Create tar header
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return count, fmt.Errorf("failed to create header for %s: %w", file, err)
		}
		header.Name = archivePath

		if err := tw.WriteHeader(header); err != nil {
			return count, fmt.Errorf("failed to write header for %s: %w", file, err)
		}

		// Write file content
		f, err := os.Open(file)
		if err != nil {
			return count, fmt.Errorf("failed to open %s: %w", file, err)
		}

		if _, err := io.Copy(tw, f); err != nil {
			f.Close()
			return count, fmt.Errorf("failed to write content for %s: %w", file, err)
		}
		f.Close()

		count++
	}

	return count, nil
}

// createTarGzArchive creates a gzip-compressed tar archive.
func (r *ArchiveResource) createTarGzArchive(w io.Writer, files []string, data *ArchiveResourceModel) (int, error) {
	gw := gzip.NewWriter(w)
	defer gw.Close()

	return r.createTarArchive(gw, files, data)
}

// updateComputedValues updates the computed values in the model.
func (r *ArchiveResource) updateComputedValues(data *ArchiveResourceModel, fileCount int) {
	data.FileCount = types.Int64Value(int64(fileCount))

	// Get file info
	info, err := os.Stat(data.Path.ValueString())
	if err == nil {
		data.Size = types.Int64Value(info.Size())
	}

	// Calculate checksums
	content, err := os.ReadFile(data.Path.ValueString())
	if err == nil {
		md5sum := md5.Sum(content)
		data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

		sha256sum := sha256.Sum256(content)
		data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))
	}

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
