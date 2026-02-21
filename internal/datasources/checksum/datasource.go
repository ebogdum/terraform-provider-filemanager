// SPDX-License-Identifier: MIT

// Package checksum implements the filemanager_checksum data source.
package checksum

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &ChecksumDataSource{}

// NewChecksumDataSource creates a new checksum data source.
func NewChecksumDataSource() datasource.DataSource {
	return &ChecksumDataSource{}
}

// ChecksumDataSource defines the data source implementation.
type ChecksumDataSource struct {
	config *common.ProviderConfig
}

// ChecksumDataSourceModel describes the data source data model.
type ChecksumDataSourceModel struct {
	ID        types.String `tfsdk:"id"`
	Path      types.String `tfsdk:"path"`
	Algorithm types.String `tfsdk:"algorithm"`
	Service   types.String `tfsdk:"service"`
	Checksum  types.String `tfsdk:"checksum"`
	Size      types.Int64  `tfsdk:"size"`
}

// Metadata returns the data source type name.
func (d *ChecksumDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_checksum"
}

// Schema defines the schema for the data source.
func (d *ChecksumDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Calculates checksum of a file.",
		MarkdownDescription: "Calculates the checksum of a file using the specified algorithm.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path of the file to calculate checksum for.",
				Required:    true,
			},
			"algorithm": schema.StringAttribute{
				Description: "Checksum algorithm: sha256, sha512.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("sha256", "sha512"),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"checksum": schema.StringAttribute{
				Description: "The calculated checksum as a hex string.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the file in bytes.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *ChecksumDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *ChecksumDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data ChecksumDataSourceModel

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

	if info.IsDir {
		resp.Diagnostics.AddError(
			"Path is a directory",
			fmt.Sprintf("Path %s is a directory, not a file", data.Path.ValueString()),
		)
		return
	}

	// Read file and calculate checksum
	reader, err := backend.Read(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file", err.Error())
		return
	}
	defer reader.Close()

	// Get hash algorithm
	algorithm := data.Algorithm.ValueString()
	if algorithm == "" {
		algorithm = "sha256"
	}

	var h hash.Hash
	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		resp.Diagnostics.AddError(
			"Invalid algorithm",
			fmt.Sprintf("Unknown algorithm: %s", algorithm),
		)
		return
	}

	// Calculate checksum while reading
	n, err := io.Copy(h, reader)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file content", err.Error())
		return
	}

	data.ID = data.Path
	data.Checksum = types.StringValue(hex.EncodeToString(h.Sum(nil)))
	data.Size = types.Int64Value(n)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// getBackend returns the appropriate backend.
func (d *ChecksumDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}
