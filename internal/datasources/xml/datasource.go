// SPDX-License-Identifier: MIT

// Package xml implements the filemanager_xml data source.
package xml

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"strings"

	"github.com/antchfx/xmlquery"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &XMLDataSource{}

// NewXMLDataSource creates a new XML data source.
func NewXMLDataSource() datasource.DataSource {
	return &XMLDataSource{}
}

// XMLDataSource defines the data source implementation.
type XMLDataSource struct {
	config *common.ProviderConfig
}

// XMLDataSourceModel describes the data source data model.
type XMLDataSourceModel struct {
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
func (d *XMLDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_xml"
}

// Schema defines the schema for the data source.
func (d *XMLDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Reads an XML file and returns its parsed content as a dynamic value.",
		MarkdownDescription: `Reads an XML file and returns its parsed content as a dynamic value representing the XML structure.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the XML file to read.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"data": schema.DynamicAttribute{
				Description: "The parsed XML content as a dynamic Terraform value. Attributes are prefixed with '@', text content uses '#text'.",
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
func (d *XMLDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *XMLDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data XMLDataSourceModel

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
			fmt.Sprintf("XML file %s does not exist", data.Path.ValueString()),
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

	// Parse XML
	doc, err := xmlquery.Parse(bytes.NewReader(content))
	if err != nil {
		resp.Diagnostics.AddError(
			"Failed to parse XML",
			fmt.Sprintf("File %s contains invalid XML: %s", data.Path.ValueString(), err.Error()),
		)
		return
	}

	// Convert to map
	parsed := xmlNodeToMap(doc)

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
func (d *XMLDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// xmlNodeToMap converts an xmlquery.Node to a map representation.
func xmlNodeToMap(node *xmlquery.Node) any {
	if node == nil {
		return nil
	}

	switch node.Type {
	case xmlquery.DocumentNode:
		for child := node.FirstChild; child != nil; child = child.NextSibling {
			if child.Type == xmlquery.ElementNode {
				result := make(map[string]any)
				result[child.Data] = elementToMap(child)
				return result
			}
		}
		return nil

	case xmlquery.ElementNode:
		result := make(map[string]any)
		result[node.Data] = elementToMap(node)
		return result

	case xmlquery.TextNode:
		return strings.TrimSpace(node.Data)

	default:
		return nil
	}
}

// elementToMap converts an XML element to a map.
func elementToMap(node *xmlquery.Node) any {
	if node == nil {
		return nil
	}

	result := make(map[string]any)

	// Handle attributes
	for _, attr := range node.Attr {
		attrKey := "@" + attr.Name.Local
		if attr.Name.Space != "" {
			attrKey = "@" + attr.Name.Space + ":" + attr.Name.Local
		}
		result[attrKey] = attr.Value
	}

	// Collect children by tag name
	childMap := make(map[string][]any)
	var textContent strings.Builder
	hasElementChildren := false

	for child := node.FirstChild; child != nil; child = child.NextSibling {
		switch child.Type {
		case xmlquery.ElementNode:
			hasElementChildren = true
			childValue := elementToMap(child)
			childMap[child.Data] = append(childMap[child.Data], childValue)

		case xmlquery.TextNode:
			text := strings.TrimSpace(child.Data)
			if text != "" {
				textContent.WriteString(text)
			}
		}
	}

	// Add children to result
	for name, children := range childMap {
		if len(children) == 1 {
			result[name] = children[0]
		} else {
			result[name] = children
		}
	}

	// Handle text content
	if textContent.Len() > 0 {
		if hasElementChildren || len(result) > 0 {
			result["#text"] = textContent.String()
		} else {
			return textContent.String()
		}
	}

	if len(result) == 0 {
		return make(map[string]any)
	}

	return result
}
