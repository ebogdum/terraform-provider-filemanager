// SPDX-License-Identifier: MIT

// Package tfvars implements the filemanager_tfvars data source.
package tfvars

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sort"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &TfvarsDataSource{}

// NewTfvarsDataSource creates a new tfvars data source.
func NewTfvarsDataSource() datasource.DataSource {
	return &TfvarsDataSource{}
}

// TfvarsDataSource defines the data source implementation.
type TfvarsDataSource struct {
	config *common.ProviderConfig
}

// TfvarsDataSourceModel describes the data source data model.
type TfvarsDataSourceModel struct {
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
	VarCount    types.Int64   `tfsdk:"var_count"`
	VarNames    types.List    `tfsdk:"var_names"`
}

// Metadata returns the data source type name.
func (d *TfvarsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_tfvars"
}

// Schema defines the schema for the data source.
func (d *TfvarsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads a Terraform .tfvars file and returns its parsed content as native dynamic types.",
		MarkdownDescription: "Reads a Terraform `.tfvars` or `.tfvars.json` file and returns its parsed content as native dynamic types. Supports querying individual variables by name.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the tfvars file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"query": schema.StringAttribute{
				Description: "Variable name to extract from the tfvars file. Returns the value of the specified variable.",
				Optional:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "The parsed tfvars content as a dynamic Terraform value. Access individual variables using dot notation.",
				Computed:    true,
			},
			"query_result": schema.DynamicAttribute{
				Description: "Result of the query if specified. Returns the value of the queried variable as a dynamic type.",
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
			"var_count": schema.Int64Attribute{
				Description: "Number of top-level variables in the tfvars file.",
				Computed:    true,
			},
			"var_names": schema.ListAttribute{
				Description: "Sorted list of top-level variable names in the tfvars file.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *TfvarsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *TfvarsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data TfvarsDataSourceModel

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
			fmt.Sprintf("Tfvars file %s does not exist", data.Path.ValueString()),
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

	// Parse based on format
	filePath := data.Path.ValueString()
	format := detectFormat(filePath)

	var parsed map[string]any
	if "json" == format {
		parsed, err = parseTfvarsJSON(content)
	} else {
		parsed, err = parseTfvarsHCL(content)
	}
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse tfvars file",
			fmt.Sprintf("File %s: %s", filePath, err.Error()),
		)
		return
	}

	// Convert to Terraform dynamic type
	dynamicVal, diags := common.GoValueToTerraformDynamic(ctx, parsed)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Extract var_names (sorted)
	varNames := make([]string, 0, len(parsed))
	for k := range parsed {
		varNames = append(varNames, k)
	}
	sort.Strings(varNames)

	varNameValues := make([]attr.Value, len(varNames))
	for i, name := range varNames {
		varNameValues[i] = types.StringValue(name)
	}
	varNamesList, listDiags := types.ListValue(types.StringType, varNameValues)
	resp.Diagnostics.Append(listDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle query if specified
	var queryResultVal types.Dynamic
	if !data.Query.IsNull() && data.Query.ValueString() != "" {
		queryKey := data.Query.ValueString()
		result, queryExists := parsed[queryKey]
		if !queryExists {
			resp.Diagnostics.AddError(
				"Query variable not found",
				fmt.Sprintf("Variable %q does not exist in %s", queryKey, filePath),
			)
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
	data.VarCount = types.Int64Value(int64(len(parsed)))
	data.VarNames = varNamesList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *TfvarsDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if "" == backendName || "local" == backendName {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// detectFormat determines if a file is HCL or JSON based on extension.
func detectFormat(path string) string {
	if len(path) > 11 && ".tfvars.json" == path[len(path)-12:] {
		return "json"
	}
	return "hcl"
}

// parseTfvarsHCL parses a .tfvars file (HCL format) into map[string]any.
func parseTfvarsHCL(data []byte) (map[string]any, error) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(data, "input.tfvars")
	if diags.HasErrors() {
		return nil, fmt.Errorf("HCL parse error: %s", diags.Error())
	}

	body, ok := file.Body.(*hclsyntax.Body)
	if !ok {
		return nil, fmt.Errorf("unexpected body type: %T", file.Body)
	}

	result := make(map[string]any, len(body.Attributes))
	for name, attr := range body.Attributes {
		val, valDiags := attr.Expr.Value(nil)
		if valDiags.HasErrors() {
			return nil, fmt.Errorf("error evaluating attribute %q: %s", name, valDiags.Error())
		}
		result[name] = ctyToGo(val)
	}

	return result, nil
}

// parseTfvarsJSON parses a .tfvars.json file into map[string]any.
func parseTfvarsJSON(data []byte) (map[string]any, error) {
	var result map[string]any
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, fmt.Errorf("JSON parse error: %w", err)
	}
	return result, nil
}

// ctyToGo converts a cty.Value to a Go value.
func ctyToGo(val cty.Value) any {
	if val.IsNull() {
		return nil
	}

	ty := val.Type()

	switch {
	case ty == cty.String:
		return val.AsString()

	case ty == cty.Number:
		bf := val.AsBigFloat()
		if bf.IsInt() {
			i, _ := bf.Int64()
			return i
		}
		f, _ := bf.Float64()
		return f

	case ty == cty.Bool:
		return val.True()

	case ty.IsListType() || ty.IsTupleType() || ty.IsSetType():
		var result []any
		for it := val.ElementIterator(); it.Next(); {
			_, v := it.Element()
			result = append(result, ctyToGo(v))
		}
		return result

	case ty.IsMapType() || ty.IsObjectType():
		result := make(map[string]any)
		for it := val.ElementIterator(); it.Next(); {
			k, v := it.Element()
			result[k.AsString()] = ctyToGo(v)
		}
		return result

	default:
		return val.GoString()
	}
}
