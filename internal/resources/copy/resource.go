// SPDX-License-Identifier: MIT

// Package copy implements the filemanager_copy resource.
package copy

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/listdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &CopyResource{}
	_ resource.ResourceWithImportState = &CopyResource{}
)

// NewCopyResource creates a new copy resource.
func NewCopyResource() resource.Resource {
	return &CopyResource{}
}

// CopyResource defines the resource implementation.
type CopyResource struct {
	config *common.ProviderConfig
}

// CopyResourceModel describes the resource data model.
type CopyResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Source              types.String `tfsdk:"source"`
	Destination         types.String `tfsdk:"destination"`
	Recursive           types.Bool   `tfsdk:"recursive"`
	Overwrite           types.Bool   `tfsdk:"overwrite"`
	PreservePermissions types.Bool   `tfsdk:"preserve_permissions"`
	Excludes            types.List   `tfsdk:"excludes"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`

	// Computed
	FilesCopied  types.Int64  `tfsdk:"files_copied"`
	BytesCopied  types.Int64  `tfsdk:"bytes_copied"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *CopyResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_copy"
}

// Schema defines the schema for the resource.
func (r *CopyResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Copies files or directories.",
		MarkdownDescription: `Copies files or directories from source to destination.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"source": schema.StringAttribute{
				Description: "Source path (file or directory).",
				Required:    true,
			},
			"destination": schema.StringAttribute{
				Description: "Destination path.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Copy directories recursively.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"overwrite": schema.BoolAttribute{
				Description: "Overwrite existing files.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"preserve_permissions": schema.BoolAttribute{
				Description: "Preserve file permissions from source.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"excludes": schema.ListAttribute{
				Description: "Glob patterns to exclude from copying.",
				Optional:    true,
				Computed:    true,
				ElementType: types.StringType,
				Default:     listdefault.StaticValue(types.ListValueMust(types.StringType, []attr.Value{})),
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode (used when preserve_permissions is false).",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"files_copied": schema.Int64Attribute{
				Description: "Number of files copied.",
				Computed:    true,
			},
			"bytes_copied": schema.Int64Attribute{
				Description: "Total bytes copied.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "MD5 checksum of the destination (for single file copy).",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the destination (for single file copy).",
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
func (r *CopyResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *CopyResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CopyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating copy", map[string]any{
		"source":      data.Source.ValueString(),
		"destination": data.Destination.ValueString(),
	})

	filesCopied, bytesCopied, err := r.performCopy(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to copy", err.Error())
		return
	}

	r.updateComputedValues(&data, filesCopied, bytesCopied)
	data.ID = types.StringValue(fmt.Sprintf("%s->%s", data.Source.ValueString(), data.Destination.ValueString()))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *CopyResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CopyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if destination exists
	_, err := os.Stat(data.Destination.ValueString())
	if err != nil {
		if os.IsNotExist(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Failed to stat destination", err.Error())
		return
	}

	// Update checksums if destination is a file
	info, _ := os.Stat(data.Destination.ValueString())
	if info != nil && !info.IsDir() {
		content, err := os.ReadFile(data.Destination.ValueString())
		if err == nil {
			md5sum := md5.Sum(content)
			data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

			sha256sum := sha256.Sum256(content)
			data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))
		}
	}

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Destination.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *CopyResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data CopyResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating copy", map[string]any{
		"source":      data.Source.ValueString(),
		"destination": data.Destination.ValueString(),
	})

	filesCopied, bytesCopied, err := r.performCopy(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to copy", err.Error())
		return
	}

	r.updateComputedValues(&data, filesCopied, bytesCopied)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *CopyResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CopyResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Remove the destination
	if err := os.RemoveAll(data.Destination.ValueString()); err != nil && !os.IsNotExist(err) {
		resp.Diagnostics.AddError("Failed to delete destination", err.Error())
		return
	}
}

// ImportState imports an existing resource.
func (r *CopyResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("destination"), req, resp)
}

// performCopy copies files from source to destination.
func (r *CopyResource) performCopy(ctx context.Context, data *CopyResourceModel) (int, int64, error) {
	source := data.Source.ValueString()
	destination := data.Destination.ValueString()

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

	// Check source
	srcInfo, err := os.Stat(source)
	if err != nil {
		return 0, 0, fmt.Errorf("source not found: %w", err)
	}

	if srcInfo.IsDir() {
		return r.copyDir(ctx, source, destination, data, excludes)
	}
	return r.copyFile(ctx, source, destination, data)
}

// copyFile copies a single file.
func (r *CopyResource) copyFile(ctx context.Context, src, dst string, data *CopyResourceModel) (int, int64, error) {
	srcInfo, err := os.Stat(src)
	if err != nil {
		return 0, 0, err
	}

	// Check if destination exists
	if !data.Overwrite.ValueBool() {
		if _, err := os.Stat(dst); err == nil {
			return 0, 0, fmt.Errorf("destination exists and overwrite is false: %s", dst)
		}
	}

	// Create destination directory
	dstDir := filepath.Dir(dst)
	dirMode := common.ParseDirMode(data.DirectoryPermission.ValueString())
	if err := os.MkdirAll(dstDir, dirMode); err != nil {
		return 0, 0, fmt.Errorf("failed to create destination directory: %w", err)
	}

	// Open source
	srcFile, err := os.Open(src)
	if err != nil {
		return 0, 0, err
	}
	defer srcFile.Close()

	// Determine file mode
	var fileMode os.FileMode
	if data.PreservePermissions.ValueBool() {
		fileMode = srcInfo.Mode()
	} else {
		fileMode = common.ParseFileMode(data.FilePermission.ValueString())
	}

	// Create destination
	dstFile, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, fileMode)
	if err != nil {
		return 0, 0, err
	}
	defer dstFile.Close()

	// Copy content
	written, err := io.Copy(dstFile, srcFile)
	if err != nil {
		return 0, 0, err
	}

	return 1, written, nil
}

// copyDir copies a directory recursively.
func (r *CopyResource) copyDir(ctx context.Context, src, dst string, data *CopyResourceModel, excludes []string) (int, int64, error) {
	if !data.Recursive.ValueBool() {
		return 0, 0, fmt.Errorf("source is a directory but recursive is false")
	}

	filesCopied := 0
	var bytesCopied int64

	err := filepath.Walk(src, func(srcPath string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Get relative path
		relPath, err := filepath.Rel(src, srcPath)
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
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
			// Also try matching against the filename
			matched, _ = filepath.Match(pattern, info.Name())
			if matched {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		dstPath := filepath.Join(dst, relPath)

		if info.IsDir() {
			dirMode := common.ParseDirMode(data.DirectoryPermission.ValueString())
			if data.PreservePermissions.ValueBool() {
				dirMode = info.Mode()
			}
			return os.MkdirAll(dstPath, dirMode)
		}

		// Copy file
		count, bytes, err := r.copyFile(ctx, srcPath, dstPath, data)
		if err != nil {
			return err
		}
		filesCopied += count
		bytesCopied += bytes

		return nil
	})

	return filesCopied, bytesCopied, err
}

// updateComputedValues updates the computed values in the model.
func (r *CopyResource) updateComputedValues(data *CopyResourceModel, filesCopied int, bytesCopied int64) {
	data.FilesCopied = types.Int64Value(int64(filesCopied))
	data.BytesCopied = types.Int64Value(bytesCopied)

	// Calculate checksums if destination is a file
	info, err := os.Stat(data.Destination.ValueString())
	if err == nil && !info.IsDir() {
		content, err := os.ReadFile(data.Destination.ValueString())
		if err == nil {
			md5sum := md5.Sum(content)
			data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

			sha256sum := sha256.Sum256(content)
			data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))
		}
	} else {
		data.MD5 = types.StringNull()
		data.SHA256 = types.StringNull()
	}

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Destination.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
