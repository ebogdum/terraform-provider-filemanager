// SPDX-License-Identifier: MIT

// Package env implements the filemanager_env data source.
package env

import (
	"bufio"
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ENVDataSource{}

// NewENVDataSource creates a new ENV data source.
func NewENVDataSource() datasource.DataSource {
	return &ENVDataSource{}
}

// ENVDataSource defines the data source implementation.
type ENVDataSource struct {
	config *common.ProviderConfig
}

// ENVDataSourceModel describes the data source data model.
type ENVDataSourceModel struct {
	ID      types.String  `tfsdk:"id"`
	Path    types.String  `tfsdk:"path"`
	Service types.String  `tfsdk:"service"`
	Data    types.Dynamic `tfsdk:"data"`
	Content types.String  `tfsdk:"content"`
	Size    types.Int64   `tfsdk:"size"`
	MD5     types.String  `tfsdk:"md5"`
	SHA256  types.String  `tfsdk:"sha256"`
}

// Metadata returns the data source type name.
func (d *ENVDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_env"
}

// Schema defines the schema for the data source.
func (d *ENVDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads a .env file and returns its parsed content as key-value pairs.",
		MarkdownDescription: "Reads a .env file and returns its parsed content as a map of environment variables.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the .env file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "The parsed environment variables as key-value pairs. All values are strings.",
				Computed:    true,
			},
			"content": schema.StringAttribute{
				Description: "The raw file content as a string.",
				Computed:    true,
				Sensitive:   true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "MD5 checksum of the file content.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the file content.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *ENVDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ENVDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ENVDataSourceModel

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
			fmt.Sprintf("ENV file %s does not exist", data.Path.ValueString()),
		)
		return
	}

	// Read file
	reader, err := backend.Read(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file", err.Error())
		return
	}
	defer reader.Close()

	content, err := io.ReadAll(reader)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file content", err.Error())
		return
	}

	// Parse ENV
	parsed, err := parseEnv(content)
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse ENV",
			fmt.Sprintf("File %s contains invalid ENV format: %s", data.Path.ValueString(), err.Error()),
		)
		return
	}

	// Convert to Terraform dynamic type
	dynamicVal, diags := common.GoValueToTerraformDynamic(ctx, parsed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Calculate checksums
	md5Hash := md5.Sum(content)
	sha256Hash := sha256.Sum256(content)

	// Set values
	data.ID = data.Path
	data.Data = dynamicVal
	data.Content = types.StringValue(string(content))
	data.Size = types.Int64Value(int64(len(content)))
	data.MD5 = types.StringValue(hex.EncodeToString(md5Hash[:]))
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256Hash[:]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *ENVDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// parseEnv parses .env file content into a map.
func parseEnv(data []byte) (map[string]any, error) {
	result := make(map[string]any)

	scanner := bufio.NewScanner(bytes.NewReader(data))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Find the = sign
		idx := strings.Index(line, "=")
		if idx == -1 {
			continue
		}

		key := strings.TrimSpace(line[:idx])
		value := strings.TrimSpace(line[idx+1:])

		// Remove export prefix if present
		key = strings.TrimPrefix(key, "export ")
		key = strings.TrimSpace(key)

		// Handle quoted values
		value = unquote(value)

		result[key] = value
	}

	if err := scanner.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

// unquote removes surrounding quotes from a value.
func unquote(s string) string {
	if len(s) < 2 {
		return s
	}

	// Double quotes
	if s[0] == '"' && s[len(s)-1] == '"' {
		s = s[1 : len(s)-1]
		s = strings.ReplaceAll(s, `\"`, `"`)
		s = strings.ReplaceAll(s, `\\`, `\`)
		s = strings.ReplaceAll(s, `\n`, "\n")
		s = strings.ReplaceAll(s, `\t`, "\t")
		return s
	}

	// Single quotes
	if s[0] == '\'' && s[len(s)-1] == '\'' {
		return s[1 : len(s)-1]
	}

	return s
}
