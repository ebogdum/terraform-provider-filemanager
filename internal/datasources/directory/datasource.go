// SPDX-License-Identifier: MIT

// Package directory implements the filemanager_directory data source.
package directory

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &DirectoryDataSource{}

// NewDirectoryDataSource creates a new directory data source.
func NewDirectoryDataSource() datasource.DataSource {
	return &DirectoryDataSource{}
}

// DirectoryDataSource defines the data source implementation.
type DirectoryDataSource struct {
	config *common.ProviderConfig
}

// DirectoryDataSourceModel describes the data source data model.
type DirectoryDataSourceModel struct {
	ID        types.String     `tfsdk:"id"`
	Path      types.String     `tfsdk:"path"`
	Service   types.String     `tfsdk:"service"`
	Pattern   types.String     `tfsdk:"pattern"`
	Recursive types.Bool       `tfsdk:"recursive"`
	Files     []FileEntryModel `tfsdk:"files"`
	TotalSize types.Int64      `tfsdk:"total_size"`
	FileCount types.Int64      `tfsdk:"file_count"`
}

// FileEntryModel describes a single file entry.
type FileEntryModel struct {
	Path    types.String `tfsdk:"path"`
	Name    types.String `tfsdk:"name"`
	Size    types.Int64  `tfsdk:"size"`
	IsDir   types.Bool   `tfsdk:"is_dir"`
	Mode    types.String `tfsdk:"mode"`
	ModTime types.String `tfsdk:"mod_time"`
}

// Metadata returns the data source type name.
func (d *DirectoryDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_directory"
}

// Schema defines the schema for the data source.
func (d *DirectoryDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists directory contents and returns file metadata.",
		MarkdownDescription: "Lists directory contents from the local filesystem or a configured backend and returns file metadata.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the directory to list.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"pattern": schema.StringAttribute{
				Description: "Glob pattern to filter files (e.g., \"*.json\").",
				Optional:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Whether to list files recursively.",
				Optional:    true,
			},
			"files": schema.ListNestedAttribute{
				Description: "List of files in the directory.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"path": schema.StringAttribute{
							Description: "Full path of the file.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Name of the file.",
							Computed:    true,
						},
						"size": schema.Int64Attribute{
							Description: "Size of the file in bytes.",
							Computed:    true,
						},
						"is_dir": schema.BoolAttribute{
							Description: "Whether the entry is a directory.",
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
			"total_size": schema.Int64Attribute{
				Description: "Total size of all files in bytes.",
				Computed:    true,
			},
			"file_count": schema.Int64Attribute{
				Description: "Number of files found.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *DirectoryDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *DirectoryDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data DirectoryDataSourceModel

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

	// Check if path exists and is a directory
	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check directory existence", err.Error())
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"Directory not found",
			fmt.Sprintf("Directory %s does not exist", data.Path.ValueString()),
		)
		return
	}

	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat directory", err.Error())
		return
	}

	if !info.IsDir {
		resp.Diagnostics.AddError(
			"Path is not a directory",
			fmt.Sprintf("%s is not a directory", data.Path.ValueString()),
		)
		return
	}

	// List directory contents
	listOpts := plugin.ListOptions{
		Recursive: data.Recursive.ValueBool(),
	}

	entries, err := backend.List(ctx, data.Path.ValueString(), listOpts)
	if err != nil {
		resp.Diagnostics.AddError("Failed to list directory", err.Error())
		return
	}

	// Filter by pattern if specified
	pattern := data.Pattern.ValueString()
	var filteredEntries []plugin.FileInfo
	for _, entry := range entries {
		if pattern != "" {
			matched, err := filepath.Match(pattern, entry.Name)
			if err != nil {
				resp.Diagnostics.AddWarning("Invalid pattern", err.Error())
				filteredEntries = entries
				break
			}
			if !matched {
				continue
			}
		}
		filteredEntries = append(filteredEntries, entry)
	}

	// Build result
	var totalSize int64
	files := make([]FileEntryModel, 0, len(filteredEntries))
	for _, entry := range filteredEntries {
		files = append(files, FileEntryModel{
			Path:    types.StringValue(entry.Path),
			Name:    types.StringValue(entry.Name),
			Size:    types.Int64Value(entry.Size),
			IsDir:   types.BoolValue(entry.IsDir),
			Mode:    types.StringValue(fmt.Sprintf("%04o", entry.Mode.Perm())),
			ModTime: types.StringValue(entry.ModTime.Format("2006-01-02T15:04:05Z07:00")),
		})
		if !entry.IsDir {
			totalSize += entry.Size
		}
	}

	data.ID = data.Path
	data.Files = files
	data.TotalSize = types.Int64Value(totalSize)
	data.FileCount = types.Int64Value(int64(len(files)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *DirectoryDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}
