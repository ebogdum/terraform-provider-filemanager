// SPDX-License-Identifier: MIT

// Package compare implements the filemanager_compare data source.
package compare

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"time"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &CompareDataSource{}

// NewCompareDataSource creates a new compare data source.
func NewCompareDataSource() datasource.DataSource {
	return &CompareDataSource{}
}

// CompareDataSource defines the data source implementation.
type CompareDataSource struct {
	config *common.ProviderConfig
}

type compareOptions struct {
	compareContent  bool
	compareChecksum bool
	compareSize     bool
	compareMode     bool
	compareOwner    bool
	compareMtime    bool
	checksumAlgo    string
	mtimeTolerance  time.Duration
}

type compareResultState struct {
	differences []string
	identical   bool
}

// CompareDataSourceModel describes the data source data model.
type CompareDataSourceModel struct {
	ID types.String `tfsdk:"id"`

	// Input
	Source            types.String `tfsdk:"source"`
	Target            types.String `tfsdk:"target"`
	SourceBackend     types.String `tfsdk:"source_backend"`
	TargetBackend     types.String `tfsdk:"target_backend"`
	CompareContent    types.Bool   `tfsdk:"compare_content"`
	CompareChecksum   types.Bool   `tfsdk:"compare_checksum"`
	CompareSize       types.Bool   `tfsdk:"compare_size"`
	CompareMode       types.Bool   `tfsdk:"compare_mode"`
	CompareOwner      types.Bool   `tfsdk:"compare_owner"`
	CompareMtime      types.Bool   `tfsdk:"compare_mtime"`
	ChecksumAlgorithm types.String `tfsdk:"checksum_algorithm"`
	MtimeTolerance    types.String `tfsdk:"mtime_tolerance"`

	// Computed - overall result
	Identical types.Bool `tfsdk:"identical"`

	// Computed - existence
	SourceExists types.Bool `tfsdk:"source_exists"`
	TargetExists types.Bool `tfsdk:"target_exists"`

	// Computed - comparison results
	ContentMatch  types.Bool `tfsdk:"content_match"`
	ChecksumMatch types.Bool `tfsdk:"checksum_match"`
	SizeMatch     types.Bool `tfsdk:"size_match"`
	ModeMatch     types.Bool `tfsdk:"mode_match"`
	OwnerMatch    types.Bool `tfsdk:"owner_match"`
	MtimeMatch    types.Bool `tfsdk:"mtime_match"`

	// Computed - source metadata
	SourceChecksum types.String `tfsdk:"source_checksum"`
	SourceSize     types.Int64  `tfsdk:"source_size"`
	SourceMode     types.String `tfsdk:"source_mode"`
	SourceUID      types.Int64  `tfsdk:"source_uid"`
	SourceGID      types.Int64  `tfsdk:"source_gid"`
	SourceMtime    types.String `tfsdk:"source_mtime"`

	// Computed - target metadata
	TargetChecksum types.String `tfsdk:"target_checksum"`
	TargetSize     types.Int64  `tfsdk:"target_size"`
	TargetMode     types.String `tfsdk:"target_mode"`
	TargetUID      types.Int64  `tfsdk:"target_uid"`
	TargetGID      types.Int64  `tfsdk:"target_gid"`
	TargetMtime    types.String `tfsdk:"target_mtime"`

	// Computed - differences
	Differences types.List `tfsdk:"differences"`
}

// Metadata returns the data source type name.
func (d *CompareDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_compare"
}

// Schema defines the schema for the data source.
func (d *CompareDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Compares two files for equality across multiple dimensions.",
		MarkdownDescription: `Compares two files for equality across multiple dimensions including content, checksum, size, permissions, ownership, and modification time.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},

			// Input attributes
			"source": schema.StringAttribute{
				Description: "Path to the source file.",
				Required:    true,
			},
			"target": schema.StringAttribute{
				Description: "Path to the target file.",
				Required:    true,
			},
			"source_backend": schema.StringAttribute{
				Description: "Backend for source file. Defaults to local.",
				Optional:    true,
			},
			"target_backend": schema.StringAttribute{
				Description: "Backend for target file. Defaults to local.",
				Optional:    true,
			},
			"compare_content": schema.BoolAttribute{
				Description: "Compare file content byte-by-byte. Defaults to true.",
				Optional:    true,
			},
			"compare_checksum": schema.BoolAttribute{
				Description: "Compare file checksums. Defaults to true.",
				Optional:    true,
			},
			"compare_size": schema.BoolAttribute{
				Description: "Compare file sizes. Defaults to true.",
				Optional:    true,
			},
			"compare_mode": schema.BoolAttribute{
				Description: "Compare file permissions. Defaults to true.",
				Optional:    true,
			},
			"compare_owner": schema.BoolAttribute{
				Description: "Compare UID/GID ownership. Defaults to false.",
				Optional:    true,
			},
			"compare_mtime": schema.BoolAttribute{
				Description: "Compare modification times. Defaults to false.",
				Optional:    true,
			},
			"checksum_algorithm": schema.StringAttribute{
				Description: "Checksum algorithm: sha256, sha512. Defaults to sha256.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.OneOf("sha256", "sha512"),
				},
			},
			"mtime_tolerance": schema.StringAttribute{
				Description: "Tolerance for modification time comparison as Go duration (e.g., '5s', '1m'). Defaults to 0 (exact match).",
				Optional:    true,
			},

			// Computed - overall result
			"identical": schema.BoolAttribute{
				Description: "True if all enabled comparisons match.",
				Computed:    true,
			},

			// Computed - existence
			"source_exists": schema.BoolAttribute{
				Description: "Whether the source file exists.",
				Computed:    true,
			},
			"target_exists": schema.BoolAttribute{
				Description: "Whether the target file exists.",
				Computed:    true,
			},

			// Computed - comparison results
			"content_match": schema.BoolAttribute{
				Description: "Content comparison result. Null if not compared.",
				Computed:    true,
			},
			"checksum_match": schema.BoolAttribute{
				Description: "Checksum comparison result. Null if not compared.",
				Computed:    true,
			},
			"size_match": schema.BoolAttribute{
				Description: "Size comparison result. Null if not compared.",
				Computed:    true,
			},
			"mode_match": schema.BoolAttribute{
				Description: "Permission mode comparison result. Null if not compared.",
				Computed:    true,
			},
			"owner_match": schema.BoolAttribute{
				Description: "UID/GID comparison result. Null if not compared.",
				Computed:    true,
			},
			"mtime_match": schema.BoolAttribute{
				Description: "Modification time comparison result. Null if not compared.",
				Computed:    true,
			},

			// Computed - source metadata
			"source_checksum": schema.StringAttribute{
				Description: "Checksum of source file.",
				Computed:    true,
			},
			"source_size": schema.Int64Attribute{
				Description: "Size of source file in bytes.",
				Computed:    true,
			},
			"source_mode": schema.StringAttribute{
				Description: "Permission mode of source file in octal.",
				Computed:    true,
			},
			"source_uid": schema.Int64Attribute{
				Description: "UID of source file.",
				Computed:    true,
			},
			"source_gid": schema.Int64Attribute{
				Description: "GID of source file.",
				Computed:    true,
			},
			"source_mtime": schema.StringAttribute{
				Description: "Modification time of source file in RFC3339 format.",
				Computed:    true,
			},

			// Computed - target metadata
			"target_checksum": schema.StringAttribute{
				Description: "Checksum of target file.",
				Computed:    true,
			},
			"target_size": schema.Int64Attribute{
				Description: "Size of target file in bytes.",
				Computed:    true,
			},
			"target_mode": schema.StringAttribute{
				Description: "Permission mode of target file in octal.",
				Computed:    true,
			},
			"target_uid": schema.Int64Attribute{
				Description: "UID of target file.",
				Computed:    true,
			},
			"target_gid": schema.Int64Attribute{
				Description: "GID of target file.",
				Computed:    true,
			},
			"target_mtime": schema.StringAttribute{
				Description: "Modification time of target file in RFC3339 format.",
				Computed:    true,
			},

			// Computed - differences
			"differences": schema.ListAttribute{
				Description: "List of attributes that differ between source and target.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *CompareDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *CompareDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data CompareDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	options, err := compareOptionsFromModel(&data)
	if err != nil {
		resp.Diagnostics.AddError("Invalid mtime_tolerance", err.Error())
		return
	}

	sourceBackend, err := d.getBackend(ctx, data.SourceBackend.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get source backend", err.Error())
		return
	}

	targetBackend, err := d.getBackend(ctx, data.TargetBackend.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get target backend", err.Error())
		return
	}

	sourceExists, targetExists, err := checkCompareExistence(ctx, sourceBackend, targetBackend, data.Source.ValueString(), data.Target.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check source/target existence", err.Error())
		return
	}

	data.ID = types.StringValue(fmt.Sprintf("%s:%s", data.Source.ValueString(), data.Target.ValueString()))
	data.SourceExists = types.BoolValue(sourceExists)
	data.TargetExists = types.BoolValue(targetExists)

	if !sourceExists || !targetExists {
		setCompareMissingState(ctx, &data, resp, sourceExists, targetExists)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	sourceInfo, targetInfo, err := statComparedFiles(ctx, sourceBackend, targetBackend, data.Source.ValueString(), data.Target.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat source/target", err.Error())
		return
	}

	if err := ensureComparedFiles(sourceInfo, targetInfo); err != nil {
		resp.Diagnostics.AddError("Invalid comparison target", err.Error())
		return
	}

	populateComparedMetadata(&data, sourceInfo, targetInfo)
	state := compareResultState{differences: make([]string, 0), identical: true}
	compareFileMetadata(&data, sourceInfo, targetInfo, options, &state)

	if err := d.compareFileContent(ctx, sourceBackend, targetBackend, &data, options, &state); err != nil {
		resp.Diagnostics.AddError("Content comparison failed", err.Error())
		return
	}

	data.Identical = types.BoolValue(state.identical)

	diffList, diags := types.ListValueFrom(ctx, types.StringType, state.differences)
	resp.Diagnostics.Append(diags...)
	data.Differences = diffList

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func compareOptionsFromModel(data *CompareDataSourceModel) (compareOptions, error) {
	opts := compareOptions{
		compareContent:  true,
		compareChecksum: true,
		compareSize:     true,
		compareMode:     true,
		compareOwner:    false,
		compareMtime:    false,
		checksumAlgo:    "sha256",
	}
	if !data.CompareContent.IsNull() {
		opts.compareContent = data.CompareContent.ValueBool()
	}
	if !data.CompareChecksum.IsNull() {
		opts.compareChecksum = data.CompareChecksum.ValueBool()
	}
	if !data.CompareSize.IsNull() {
		opts.compareSize = data.CompareSize.ValueBool()
	}
	if !data.CompareMode.IsNull() {
		opts.compareMode = data.CompareMode.ValueBool()
	}
	if !data.CompareOwner.IsNull() {
		opts.compareOwner = data.CompareOwner.ValueBool()
	}
	if !data.CompareMtime.IsNull() {
		opts.compareMtime = data.CompareMtime.ValueBool()
	}
	if !data.ChecksumAlgorithm.IsNull() && data.ChecksumAlgorithm.ValueString() != "" {
		opts.checksumAlgo = data.ChecksumAlgorithm.ValueString()
	}
	if !data.MtimeTolerance.IsNull() && data.MtimeTolerance.ValueString() != "" {
		tolerance, err := time.ParseDuration(data.MtimeTolerance.ValueString())
		if err != nil {
			return compareOptions{}, err
		}
		opts.mtimeTolerance = tolerance
	}
	return opts, nil
}

func checkCompareExistence(ctx context.Context, sourceBackend, targetBackend plugin.Backend, sourcePath, targetPath string) (bool, bool, error) {
	sourceExists, err := sourceBackend.Exists(ctx, sourcePath)
	if err != nil {
		return false, false, err
	}
	targetExists, err := targetBackend.Exists(ctx, targetPath)
	if err != nil {
		return false, false, err
	}
	return sourceExists, targetExists, nil
}

func setCompareMissingState(ctx context.Context, data *CompareDataSourceModel, resp *datasource.ReadResponse, sourceExists, targetExists bool) {
	data.Identical = types.BoolValue(false)
	data.ContentMatch = types.BoolNull()
	data.ChecksumMatch = types.BoolNull()
	data.SizeMatch = types.BoolNull()
	data.ModeMatch = types.BoolNull()
	data.OwnerMatch = types.BoolNull()
	data.MtimeMatch = types.BoolNull()
	data.SourceChecksum = types.StringNull()
	data.SourceSize = types.Int64Null()
	data.SourceMode = types.StringNull()
	data.SourceUID = types.Int64Null()
	data.SourceGID = types.Int64Null()
	data.SourceMtime = types.StringNull()
	data.TargetChecksum = types.StringNull()
	data.TargetSize = types.Int64Null()
	data.TargetMode = types.StringNull()
	data.TargetUID = types.Int64Null()
	data.TargetGID = types.Int64Null()
	data.TargetMtime = types.StringNull()

	differences := make([]string, 0, 2)
	if !sourceExists {
		differences = append(differences, "source_missing")
	}
	if !targetExists {
		differences = append(differences, "target_missing")
	}
	diffList, diags := types.ListValueFrom(ctx, types.StringType, differences)
	resp.Diagnostics.Append(diags...)
	data.Differences = diffList
}

func statComparedFiles(ctx context.Context, sourceBackend, targetBackend plugin.Backend, sourcePath, targetPath string) (*plugin.FileInfo, *plugin.FileInfo, error) {
	sourceInfo, err := sourceBackend.Stat(ctx, sourcePath)
	if err != nil {
		return nil, nil, err
	}
	targetInfo, err := targetBackend.Stat(ctx, targetPath)
	if err != nil {
		return nil, nil, err
	}
	return sourceInfo, targetInfo, nil
}

func ensureComparedFiles(sourceInfo, targetInfo *plugin.FileInfo) error {
	if sourceInfo.IsDir {
		return fmt.Errorf("source path is a directory, not a file")
	}
	if targetInfo.IsDir {
		return fmt.Errorf("target path is a directory, not a file")
	}
	return nil
}

func populateComparedMetadata(data *CompareDataSourceModel, sourceInfo, targetInfo *plugin.FileInfo) {
	data.SourceSize = types.Int64Value(sourceInfo.Size)
	data.SourceMode = types.StringValue(fmt.Sprintf("%04o", sourceInfo.Mode.Perm()))
	data.SourceUID = types.Int64Value(int64(sourceInfo.UID))
	data.SourceGID = types.Int64Value(int64(sourceInfo.GID))
	data.SourceMtime = types.StringValue(sourceInfo.ModTime.Format(time.RFC3339))

	data.TargetSize = types.Int64Value(targetInfo.Size)
	data.TargetMode = types.StringValue(fmt.Sprintf("%04o", targetInfo.Mode.Perm()))
	data.TargetUID = types.Int64Value(int64(targetInfo.UID))
	data.TargetGID = types.Int64Value(int64(targetInfo.GID))
	data.TargetMtime = types.StringValue(targetInfo.ModTime.Format(time.RFC3339))
}

func compareFileMetadata(data *CompareDataSourceModel, sourceInfo, targetInfo *plugin.FileInfo, opts compareOptions, state *compareResultState) {
	if opts.compareSize {
		sizeMatch := sourceInfo.Size == targetInfo.Size
		data.SizeMatch = types.BoolValue(sizeMatch)
		if !sizeMatch {
			state.differences = append(state.differences, "size")
			state.identical = false
		}
	} else {
		data.SizeMatch = types.BoolNull()
	}

	if opts.compareMode {
		modeMatch := sourceInfo.Mode.Perm() == targetInfo.Mode.Perm()
		data.ModeMatch = types.BoolValue(modeMatch)
		if !modeMatch {
			state.differences = append(state.differences, "mode")
			state.identical = false
		}
	} else {
		data.ModeMatch = types.BoolNull()
	}

	if opts.compareOwner {
		ownerMatch := sourceInfo.UID == targetInfo.UID && sourceInfo.GID == targetInfo.GID
		data.OwnerMatch = types.BoolValue(ownerMatch)
		if !ownerMatch {
			if sourceInfo.UID != targetInfo.UID {
				state.differences = append(state.differences, "uid")
			}
			if sourceInfo.GID != targetInfo.GID {
				state.differences = append(state.differences, "gid")
			}
			state.identical = false
		}
	} else {
		data.OwnerMatch = types.BoolNull()
	}

	if opts.compareMtime {
		mtimeMatch := compareMtimeWithTolerance(sourceInfo.ModTime, targetInfo.ModTime, opts.mtimeTolerance)
		data.MtimeMatch = types.BoolValue(mtimeMatch)
		if !mtimeMatch {
			state.differences = append(state.differences, "mtime")
			state.identical = false
		}
	} else {
		data.MtimeMatch = types.BoolNull()
	}
}

func compareMtimeWithTolerance(source, target time.Time, tolerance time.Duration) bool {
	if tolerance <= 0 {
		return source.Equal(target)
	}
	diff := source.Sub(target)
	if diff < 0 {
		diff = -diff
	}
	return diff <= tolerance
}

func (d *CompareDataSource) compareFileContent(ctx context.Context, sourceBackend, targetBackend plugin.Backend, data *CompareDataSourceModel, opts compareOptions, state *compareResultState) error {
	if !opts.compareContent && !opts.compareChecksum {
		data.ContentMatch = types.BoolNull()
		data.ChecksumMatch = types.BoolNull()
		data.SourceChecksum = types.StringNull()
		data.TargetChecksum = types.StringNull()
		return nil
	}

	sourceContent, err := readCompareContent(ctx, sourceBackend, data.Source.ValueString())
	if err != nil {
		return err
	}
	targetContent, err := readCompareContent(ctx, targetBackend, data.Target.ValueString())
	if err != nil {
		return err
	}

	sourceChecksum := calculateChecksum(sourceContent, opts.checksumAlgo)
	targetChecksum := calculateChecksum(targetContent, opts.checksumAlgo)
	data.SourceChecksum = types.StringValue(sourceChecksum)
	data.TargetChecksum = types.StringValue(targetChecksum)

	if opts.compareChecksum {
		checksumMatch := sourceChecksum == targetChecksum
		data.ChecksumMatch = types.BoolValue(checksumMatch)
		if !checksumMatch {
			state.differences = append(state.differences, "checksum")
			state.identical = false
		}
	} else {
		data.ChecksumMatch = types.BoolNull()
	}

	if opts.compareContent {
		contentMatch := bytes.Equal(sourceContent, targetContent)
		data.ContentMatch = types.BoolValue(contentMatch)
		if !contentMatch {
			state.differences = append(state.differences, "content")
			state.identical = false
		}
	} else {
		data.ContentMatch = types.BoolNull()
	}

	return nil
}

func readCompareContent(ctx context.Context, backend plugin.Backend, filePath string) ([]byte, error) {
	reader, err := backend.Read(ctx, filePath)
	if err != nil {
		return nil, err
	}
	defer reader.Close()
	return io.ReadAll(reader)
}

// getBackend returns the appropriate backend.
func (d *CompareDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}

// calculateChecksum calculates a checksum using the specified algorithm.
func calculateChecksum(content []byte, algorithm string) string {
	var h hash.Hash
	switch algorithm {
	case "sha256":
		h = sha256.New()
	case "sha512":
		h = sha512.New()
	default:
		h = sha256.New()
	}
	h.Write(content)
	return hex.EncodeToString(h.Sum(nil))
}
