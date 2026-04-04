// SPDX-License-Identifier: MIT

// Package file implements the filemanager_file data source.
package file

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// maxReadBytes is the maximum file size (10 MB) that can be read by this datasource.
const maxReadBytes = 10 * 1024 * 1024

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &FileDataSource{}

// NewFileDataSource creates a new file data source.
func NewFileDataSource() datasource.DataSource {
	return &FileDataSource{}
}

// FileDataSource defines the data source implementation.
type FileDataSource struct {
	config *common.ProviderConfig
}

// FileDataSourceModel describes the data source data model.
type FileDataSourceModel struct {
	ID            types.String `tfsdk:"id"`
	Path          types.String `tfsdk:"path"`
	Service       types.String `tfsdk:"service"`
	Content       types.String `tfsdk:"content"`
	ContentBase64 types.String `tfsdk:"content_base64"`
	Size          types.Int64  `tfsdk:"size"`
	MD5           types.String `tfsdk:"md5"`
	SHA256        types.String `tfsdk:"sha256"`
	IsDir         types.Bool   `tfsdk:"is_dir"`
	Mode          types.String `tfsdk:"mode"`
	ModTime       types.String `tfsdk:"mod_time"`
}

// Metadata returns the data source type name.
func (d *FileDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Schema defines the schema for the data source.
func (d *FileDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads a file and returns its content and metadata.",
		MarkdownDescription: `Reads a file from the local filesystem or a configured backend and returns its content and metadata.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"content": schema.StringAttribute{
				Description: "The content of the file as a string. Empty if file is binary or too large.",
				Computed:    true,
			},
			"content_base64": schema.StringAttribute{
				Description: "The content of the file as base64 encoded string.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "Deprecated insecure checksum field. Always null.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the file content.",
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
	}
}

// Configure configures the data source with provider data.
func (d *FileDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *FileDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data FileDataSourceModel

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
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.Diagnostics.AddError(
			"File not found",
			fmt.Sprintf("File %s does not exist", data.Path.ValueString()),
		)
		return
	}

	// Get file info
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat file", err.Error())
		return
	}

	data.ID = data.Path
	data.Size = types.Int64Value(info.Size)
	data.IsDir = types.BoolValue(info.IsDir)
	data.Mode = types.StringValue(fmt.Sprintf("%04o", info.Mode.Perm()))
	data.ModTime = types.StringValue(info.ModTime.Format("2006-01-02T15:04:05Z07:00"))

	// Read content if it's a file
	if !info.IsDir {
		reader, err := backend.Read(ctx, data.Path.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Failed to read file", err.Error())
			return
		}
		defer reader.Close()

		limited := io.LimitReader(reader, maxReadBytes+1)
		content, err := io.ReadAll(limited)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read file content", err.Error())
			return
		}
		if int64(len(content)) > maxReadBytes {
			resp.Diagnostics.AddError("File too large",
				fmt.Sprintf("File exceeds maximum read size of %d bytes", maxReadBytes))
			return
		}

		// Calculate checksums
		data.MD5 = types.StringNull()

		sha256sum := sha256.Sum256(content)
		data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

		// Check if content is text (valid UTF-8)
		if isTextContent(content) {
			data.Content = types.StringValue(string(content))
		} else {
			data.Content = types.StringNull()
		}

		data.ContentBase64 = types.StringValue(base64.StdEncoding.EncodeToString(content))
	} else {
		data.Content = types.StringNull()
		data.ContentBase64 = types.StringNull()
		data.MD5 = types.StringNull()
		data.SHA256 = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *FileDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// isTextContent checks if the content is valid UTF-8 text.
func isTextContent(content []byte) bool {
	// Check for null bytes which indicate binary
	for _, b := range content {
		if b == 0 {
			return false
		}
	}
	return true
}
