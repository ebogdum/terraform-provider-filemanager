// SPDX-License-Identifier: MIT

// Package yaml implements the filemanager_yaml data source.
package yaml

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"gopkg.in/yaml.v3"

	"github.com/ebogdum/filemanager/internal/common"
	yamlformat "github.com/ebogdum/filemanager/internal/formats/yaml"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &YAMLDataSource{}

// NewYAMLDataSource creates a new YAML data source.
func NewYAMLDataSource() datasource.DataSource {
	return &YAMLDataSource{}
}

// YAMLDataSource defines the data source implementation.
type YAMLDataSource struct {
	config *common.ProviderConfig
}

// YAMLDataSourceModel describes the data source data model.
type YAMLDataSourceModel struct {
	ID          types.String  `tfsdk:"id"`
	Path        types.String  `tfsdk:"path"`
	Service     types.String  `tfsdk:"service"`
	Query       types.String  `tfsdk:"query"`
	Data        types.Dynamic `tfsdk:"data"`
	QueryResult types.Dynamic `tfsdk:"query_result"`
	Content     types.String  `tfsdk:"content"`
	Size        types.Int64   `tfsdk:"size"`
	MD5         types.String  `tfsdk:"md5"`
	SHA256      types.String  `tfsdk:"sha256"`
}

// Metadata returns the data source type name.
func (d *YAMLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_yaml"
}

// Schema defines the schema for the data source.
func (d *YAMLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads a YAML file and returns its parsed content as a dynamic value.",
		MarkdownDescription: "Reads a YAML file and returns its parsed content as a dynamic value that can be accessed like a native Terraform object.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the YAML file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"query": schema.StringAttribute{
				Description: "Path query to extract specific data (e.g., 'config.database.host', 'users[0].name').",
				Optional:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "The parsed YAML content as a dynamic Terraform value. Access nested values using dot notation or bracket syntax.",
				Computed:    true,
			},
			"query_result": schema.DynamicAttribute{
				Description: "Result of the query if specified. Returns the extracted value as a dynamic type.",
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
				Description: "Deprecated insecure checksum field. Always null.",
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
func (d *YAMLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *YAMLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data YAMLDataSourceModel

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
			fmt.Sprintf("YAML file %s does not exist", data.Path.ValueString()),
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

	// Parse YAML
	var parsed any
	if err := yaml.Unmarshal(content, &parsed); err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse YAML",
			fmt.Sprintf("File %s contains invalid YAML: %s", data.Path.ValueString(), err.Error()),
		)
		return
	}

	// Normalize YAML data (convert map[any]any to map[string]any)
	parsed = normalizeYAML(parsed)

	// Convert to Terraform dynamic type
	dynamicVal, diags := common.GoValueToTerraformDynamic(ctx, parsed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle query if specified
	var queryResultVal types.Dynamic
	if !data.Query.IsNull() && data.Query.ValueString() != "" {
		format := yamlformat.New()
		result, err := format.Query(parsed, data.Query.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Query failed", err.Error())
			return
		}
		queryResultVal, diags = common.GoValueToTerraformDynamic(ctx, result)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		queryResultVal = types.DynamicNull()
	}

	// Calculate checksums
	sha256Hash := sha256.Sum256(content)

	// Set values
	data.ID = data.Path
	data.Data = dynamicVal
	data.QueryResult = queryResultVal
	data.Content = types.StringValue(string(content))
	data.Size = types.Int64Value(int64(len(content)))
	data.MD5 = types.StringNull()
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256Hash[:]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *YAMLDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// normalizeYAML converts yaml.v3 types to standard Go types.
func normalizeYAML(v any) any {
	switch val := v.(type) {
	case map[string]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[k] = normalizeYAML(v)
		}
		return result

	case map[any]any:
		result := make(map[string]any, len(val))
		for k, v := range val {
			result[fmt.Sprintf("%v", k)] = normalizeYAML(v)
		}
		return result

	case []any:
		result := make([]any, len(val))
		for i, v := range val {
			result[i] = normalizeYAML(v)
		}
		return result

	default:
		return v
	}
}
