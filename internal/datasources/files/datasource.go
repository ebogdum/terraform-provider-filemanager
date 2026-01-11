// SPDX-License-Identifier: MIT

// Package files implements the filemanager_files data source for glob patterns.
package files

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FilesDataSource{}

// NewFilesDataSource creates a new files data source.
func NewFilesDataSource() datasource.DataSource {
	return &FilesDataSource{}
}

// FilesDataSource defines the data source implementation.
type FilesDataSource struct {
	config *common.ProviderConfig
}

// FilesDataSourceModel describes the data source data model.
type FilesDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Path      types.String `tfsdk:"path"`
	Pattern   types.String `tfsdk:"pattern"`
	Recursive types.Bool   `tfsdk:"recursive"`
	Service   types.String `tfsdk:"service"`
	Files     types.List   `tfsdk:"files"`
	FileCount types.Int64  `tfsdk:"file_count"`
}

// FileInfoModel describes file info in the result.
type FileInfoModel struct {
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	Size    types.Int64  `tfsdk:"size"`
	IsDir   types.Bool   `tfsdk:"is_dir"`
	Mode    types.String `tfsdk:"mode"`
	ModTime types.String `tfsdk:"mod_time"`
}

// Metadata returns the data source type name.
func (d *FilesDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_files"
}

// Schema defines the schema for the data source.
func (d *FilesDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists files matching a pattern.",
		MarkdownDescription: "Lists files in a directory matching a glob pattern.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The directory path to search in.",
				Required:    true,
			},
			"pattern": schema.StringAttribute{
				Description: "Glob pattern to match files (e.g., '*.txt', '*.conf').",
				Optional:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Search subdirectories recursively.",
				Optional:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"files": schema.ListNestedAttribute{
				Description: "List of files matching the pattern.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Full path to the file.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Base name of the file.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "Size of the file in bytes.",
							Computed:    true,
						},
						"is_dir": schema.BoolAttribute{
							Description: "Whether the path is a directory.",
							Computed:    true,
						},
						"mode": schema.StringAttribute{
							Description: "File permission mode in octal format.",
							Computed:    true,
						},
						"mod_time": schema.StringAttribute{
							Description: "File modification time in RFC3339 format.",
							Computed:    true,
						},
					},
				},
			},
			"file_count": schema.Int64Attribute{
				Description: "Number of files matching the pattern.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *FilesDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*common.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *common.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	d.config = config
}

// Read reads the data source.
func (d *FilesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FilesDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get backend
	backend, err := d.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Check if path exists
	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check path existence", err.Error())
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"Path not found",
			fmt.Sprintf("Path %s does not exist", data.Path.ValueString()),
		)
		return
	}

	// List files
	listOpts := plugin.ListOptions{
		Recursive:     data.Recursive.ValueBool(),
		Pattern:       data.Pattern.ValueString(),
		IncludeHidden: false,
	}

	files, err := backend.List(ctx, data.Path.ValueString(), listOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list files", err.Error())
		return
	}

	// Build result
	pattern := data.Pattern.ValueString()

	fileAttrTypes := map[string]attr.Type{
		"path":     types.StringType,
		"name":     types.StringType,
		"size":     types.Int64Type,
		"is_dir":   types.BoolType,
		"mode":     types.StringType,
		"mod_time": types.StringType,
	}

	var fileObjects []attr.Value
	for _, f := range files {
		// Apply pattern matching if specified
		if pattern != "" {
			matched, err := filepath.Match(pattern, f.Name)
			if err != nil || !matched {
				continue
			}
		}

		fileObj, diags := types.ObjectValue(fileAttrTypes, map[string]attr.Value{
			"path":     types.StringValue(f.Path),
			"name":     types.StringValue(f.Name),
			"size":     types.Int64Value(f.Size),
			"is_dir":   types.BoolValue(f.IsDir),
			"mode":     types.StringValue(fmt.Sprintf("%04o", f.Mode.Perm())),
			"mod_time": types.StringValue(f.ModTime.Format("2006-01-02T15:04:05Z07:00")),
		})
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		fileObjects = append(fileObjects, fileObj)
	}

	// Convert to types.List
	filesList, diags := types.ListValue(types.ObjectType{AttrTypes: fileAttrTypes}, fileObjects)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.ID = data.Path
	data.Files = filesList
	data.FileCount = types.Int64Value(int64(len(fileObjects)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *FilesDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}
