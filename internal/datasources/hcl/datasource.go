// SPDX-License-Identifier: MIT

// Package hcl implements the filemanager_hcl data source.
package hcl

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/zclconf/go-cty/cty"
	ctyjson "github.com/zclconf/go-cty/cty/json"

	"github.com/ebogdum/filemanager/internal/common"
	hclformat "github.com/ebogdum/filemanager/internal/formats/hcl"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &HCLDataSource{}

// NewHCLDataSource creates a new HCL data source.
func NewHCLDataSource() datasource.DataSource {
	return &HCLDataSource{}
}

// HCLDataSource defines the data source implementation.
type HCLDataSource struct {
	config *common.ProviderConfig
}

// HCLDataSourceModel describes the data source data model.
type HCLDataSourceModel struct {
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
func (d *HCLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hcl"
}

// Schema defines the schema for the data source.
func (d *HCLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads an HCL file and returns its parsed content as a dynamic value.",
		MarkdownDescription: `
Reads an HCL (HashiCorp Configuration Language) file and returns its parsed content as a dynamic value.
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the HCL file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"query": schema.StringAttribute{
				Description: "Path query to extract specific data (e.g., 'variable.name', 'resource.aws_instance.web').",
				Optional:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "The parsed HCL content as a dynamic Terraform value.",
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
func (d *HCLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *HCLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data HCLDataSourceModel

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
			fmt.Sprintf("HCL file %s does not exist", data.Path.ValueString()),
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

	// Parse HCL
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCL(content, data.Path.ValueString())
	if diags.HasErrors() {
		resp.Diagnostics.AddError(
			"Failed to parse HCL",
			fmt.Sprintf("File %s contains invalid HCL: %s", data.Path.ValueString(), diags.Error()),
		)
		return
	}

	// Convert to map
	parsed, err := hclFileToMap(file.Body.(*hclsyntax.Body))
	if err != nil {
		resp.Diagnostics.AddError("Failed to convert HCL to map", err.Error())
		return
	}

	// Convert to Terraform dynamic type
	dynamicVal, convDiags := common.GoValueToTerraformDynamic(ctx, parsed)
	resp.Diagnostics.Append(convDiags...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Handle query if specified
	var queryResultVal types.Dynamic
	if !data.Query.IsNull() && data.Query.ValueString() != "" {
		format := hclformat.New()
		result, err := format.Query(parsed, data.Query.ValueString())
		if err != nil {
			resp.Diagnostics.AddError("Query failed", err.Error())
			return
		}
		queryResultVal, convDiags = common.GoValueToTerraformDynamic(ctx, result)
		resp.Diagnostics.Append(convDiags...)
		if resp.Diagnostics.HasError() {
			return
		}
	} else {
		queryResultVal = types.DynamicNull()
	}

	// Calculate checksums
	md5Hash := md5.Sum(content)
	sha256Hash := sha256.Sum256(content)

	// Set values
	data.ID = data.Path
	data.Data = dynamicVal
	data.QueryResult = queryResultVal
	data.Content = types.StringValue(string(content))
	data.Size = types.Int64Value(int64(len(content)))
	data.MD5 = types.StringValue(hex.EncodeToString(md5Hash[:]))
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256Hash[:]))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *HCLDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// hclFileToMap converts an HCL body to a map representation.
func hclFileToMap(body *hclsyntax.Body) (map[string]any, error) {
	result := make(map[string]any)

	// Process attributes
	for name, attr := range body.Attributes {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			result[name] = fmt.Sprintf("%v", attr.Expr)
			continue
		}
		result[name] = ctyToGo(val)
	}

	// Process blocks
	for _, block := range body.Blocks {
		blockValue, err := blockToMap(block)
		if err != nil {
			return nil, err
		}

		blockKey := block.Type
		if len(block.Labels) > 0 {
			blockKey = block.Type + "." + strings.Join(block.Labels, ".")
		}

		if existing, ok := result[blockKey]; ok {
			switch v := existing.(type) {
			case []any:
				result[blockKey] = append(v, blockValue)
			default:
				result[blockKey] = []any{v, blockValue}
			}
		} else {
			result[blockKey] = blockValue
		}
	}

	return result, nil
}

// blockToMap converts an HCL block to a map.
func blockToMap(block *hclsyntax.Block) (map[string]any, error) {
	result := make(map[string]any)

	if len(block.Labels) > 0 {
		result["__labels__"] = block.Labels
	}

	for name, attr := range block.Body.Attributes {
		val, diags := attr.Expr.Value(nil)
		if diags.HasErrors() {
			result[name] = fmt.Sprintf("%v", attr.Expr)
			continue
		}
		result[name] = ctyToGo(val)
	}

	for _, nestedBlock := range block.Body.Blocks {
		nestedValue, err := blockToMap(nestedBlock)
		if err != nil {
			return nil, err
		}

		nestedKey := nestedBlock.Type
		if len(nestedBlock.Labels) > 0 {
			nestedKey = nestedBlock.Type + "." + strings.Join(nestedBlock.Labels, ".")
		}

		if existing, ok := result[nestedKey]; ok {
			switch v := existing.(type) {
			case []any:
				result[nestedKey] = append(v, nestedValue)
			default:
				result[nestedKey] = []any{v, nestedValue}
			}
		} else {
			result[nestedKey] = nestedValue
		}
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
		jsonBytes, err := ctyjson.Marshal(val, ty)
		if err == nil {
			var result any
			if json.Unmarshal(jsonBytes, &result) == nil {
				return result
			}
		}
		return val.GoString()
	}
}
