// SPDX-License-Identifier: MIT

// Package stat implements the filemanager_stat data source.
package stat

import (
	"context"
	"fmt"
	"os/user"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &StatDataSource{}

// NewStatDataSource creates a new stat data source.
func NewStatDataSource() datasource.DataSource {
	return &StatDataSource{}
}

// StatDataSource defines the data source implementation.
type StatDataSource struct {
	config *common.ProviderConfig
}

// StatDataSourceModel describes the data source data model.
type StatDataSourceModel struct {
	ID           types.String `tfsdk:"id"`
	Path         types.String `tfsdk:"path"`
	Service      types.String `tfsdk:"service"`
	Exists       types.Bool   `tfsdk:"exists"`
	Size         types.Int64  `tfsdk:"size"`
	IsDir        types.Bool   `tfsdk:"is_dir"`
	IsFile       types.Bool   `tfsdk:"is_file"`
	IsSymlink    types.Bool   `tfsdk:"is_symlink"`
	LinkTarget   types.String `tfsdk:"link_target"`
	Mode         types.String `tfsdk:"mode"`
	ModTime      types.String `tfsdk:"mod_time"`
	AccessTime   types.String `tfsdk:"access_time"`
	CreationTime types.String `tfsdk:"creation_time"`
	UID          types.Int64  `tfsdk:"uid"`
	GID          types.Int64  `tfsdk:"gid"`
	ContentType  types.String `tfsdk:"content_type"`

	// Time-based check inputs
	ModifiedWithin types.String `tfsdk:"modified_within"`
	AccessedWithin types.String `tfsdk:"accessed_within"`

	// Time-based check outputs
	IsModifiedWithin types.Bool   `tfsdk:"is_modified_within"`
	IsAccessedWithin types.Bool   `tfsdk:"is_accessed_within"`
	Age              types.String `tfsdk:"age"`

	// Owner name resolution
	OwnerName types.String `tfsdk:"owner_name"`
	GroupName types.String `tfsdk:"group_name"`
}

// Metadata returns the data source type name.
func (d *StatDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_stat"
}

// Schema defines the schema for the data source.
func (d *StatDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Returns detailed file metadata.",
		MarkdownDescription: `Returns detailed file or directory metadata including permissions, ownership, and timestamps.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"path": schema.StringAttribute{
				Description: "The path to get metadata for.",
				Required:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
			},
			"exists": schema.BoolAttribute{
				Description: "Whether the path exists.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size in bytes (0 for directories).",
				Computed:    true,
			},
			"is_dir": schema.BoolAttribute{
				Description: "Whether the path is a directory.",
				Computed:    true,
			},
			"is_file": schema.BoolAttribute{
				Description: "Whether the path is a regular file.",
				Computed:    true,
			},
			"is_symlink": schema.BoolAttribute{
				Description: "Whether the path is a symbolic link.",
				Computed:    true,
			},
			"link_target": schema.StringAttribute{
				Description: "Target of the symbolic link (empty if not a symlink).",
				Computed:    true,
			},
			"mode": schema.StringAttribute{
				Description: "File permission mode in octal format.",
				Computed:    true,
			},
			"mod_time": schema.StringAttribute{
				Description: "Last modification time in RFC3339 format.",
				Computed:    true,
			},
			"access_time": schema.StringAttribute{
				Description: "Last access time in RFC3339 format.",
				Computed:    true,
			},
			"creation_time": schema.StringAttribute{
				Description: "Creation time in RFC3339 format (if available).",
				Computed:    true,
			},
			"uid": schema.Int64Attribute{
				Description: "User ID of the file owner (Unix only).",
				Computed:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "Group ID of the file owner (Unix only).",
				Computed:    true,
			},
			"content_type": schema.StringAttribute{
				Description: "Content type (MIME type) if available.",
				Computed:    true,
			},

			// Time-based check inputs
			"modified_within": schema.StringAttribute{
				Description: "Check if file was modified within this duration (e.g., '30m', '24h', '7d'). Sets is_modified_within.",
				Optional:    true,
			},
			"accessed_within": schema.StringAttribute{
				Description: "Check if file was accessed within this duration (e.g., '30m', '24h', '7d'). Sets is_accessed_within.",
				Optional:    true,
			},

			// Time-based check outputs
			"is_modified_within": schema.BoolAttribute{
				Description: "True if file was modified within the duration specified by modified_within. Null if modified_within not set.",
				Computed:    true,
			},
			"is_accessed_within": schema.BoolAttribute{
				Description: "True if file was accessed within the duration specified by accessed_within. Null if accessed_within not set.",
				Computed:    true,
			},
			"age": schema.StringAttribute{
				Description: "Duration since the file was last modified (e.g., '2h30m15s').",
				Computed:    true,
			},

			// Owner name resolution
			"owner_name": schema.StringAttribute{
				Description: "Username of the file owner (resolved from UID, Unix only).",
				Computed:    true,
			},
			"group_name": schema.StringAttribute{
				Description: "Group name of the file owner (resolved from GID, Unix only).",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *StatDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *StatDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data StatDataSourceModel

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

	data.ID = data.Path
	data.Exists = types.BoolValue(exists)

	if !exists {
		setMissingStatFields(&data)
		resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
		return
	}

	// Get file info
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to stat path", err.Error())
		return
	}

	now := time.Now()
	age := populateStatFieldsFromInfo(&data, info, now)
	if err := populateModifiedWithin(&data, age); err != nil {
		resp.Diagnostics.AddError("Invalid modified_within duration", err.Error())
		return
	}
	if err := populateAccessedWithin(&data, info, now); err != nil {
		resp.Diagnostics.AddError("Invalid accessed_within duration", err.Error())
		return
	}
	populateOwnerGroupNames(&data, info)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func setMissingStatFields(data *StatDataSourceModel) {
	data.Size = types.Int64Null()
	data.IsDir = types.BoolNull()
	data.IsFile = types.BoolNull()
	data.IsSymlink = types.BoolNull()
	data.LinkTarget = types.StringNull()
	data.Mode = types.StringNull()
	data.ModTime = types.StringNull()
	data.AccessTime = types.StringNull()
	data.CreationTime = types.StringNull()
	data.UID = types.Int64Null()
	data.GID = types.Int64Null()
	data.ContentType = types.StringNull()
	data.IsModifiedWithin = types.BoolNull()
	data.IsAccessedWithin = types.BoolNull()
	data.Age = types.StringNull()
	data.OwnerName = types.StringNull()
	data.GroupName = types.StringNull()
}

func populateStatFieldsFromInfo(data *StatDataSourceModel, info *plugin.FileInfo, now time.Time) time.Duration {
	data.Size = types.Int64Value(info.Size)
	data.IsDir = types.BoolValue(info.IsDir)
	data.IsFile = types.BoolValue(!info.IsDir && !info.IsSymlink)
	data.IsSymlink = types.BoolValue(info.IsSymlink)
	data.Mode = types.StringValue(fmt.Sprintf("%04o", info.Mode.Perm()))
	data.ModTime = types.StringValue(info.ModTime.Format("2006-01-02T15:04:05Z07:00"))
	data.LinkTarget = optionalSymlinkTarget(info)
	data.AccessTime = optionalTimeString(info.LastAccessTime)
	data.CreationTime = optionalTimeString(info.CreationTime)
	data.UID = types.Int64Value(int64(info.UID))
	data.GID = types.Int64Value(int64(info.GID))
	data.ContentType = optionalString(info.ContentType)

	age := now.Sub(info.ModTime)
	data.Age = types.StringValue(age.Truncate(time.Second).String())
	return age
}

func optionalSymlinkTarget(info *plugin.FileInfo) types.String {
	if info.IsSymlink && info.LinkTarget != "" {
		return types.StringValue(info.LinkTarget)
	}
	return types.StringNull()
}

func optionalTimeString(t time.Time) types.String {
	if t.IsZero() {
		return types.StringNull()
	}
	return types.StringValue(t.Format("2006-01-02T15:04:05Z07:00"))
}

func optionalString(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func populateModifiedWithin(data *StatDataSourceModel, age time.Duration) error {
	if data.ModifiedWithin.IsNull() || data.ModifiedWithin.ValueString() == "" {
		data.IsModifiedWithin = types.BoolNull()
		return nil
	}
	duration, err := parseDuration(data.ModifiedWithin.ValueString())
	if err != nil {
		return err
	}
	data.IsModifiedWithin = types.BoolValue(age <= duration)
	return nil
}

func populateAccessedWithin(data *StatDataSourceModel, info *plugin.FileInfo, now time.Time) error {
	if data.AccessedWithin.IsNull() || data.AccessedWithin.ValueString() == "" {
		data.IsAccessedWithin = types.BoolNull()
		return nil
	}
	duration, err := parseDuration(data.AccessedWithin.ValueString())
	if err != nil {
		return err
	}
	if info.LastAccessTime.IsZero() {
		data.IsAccessedWithin = types.BoolNull()
		return nil
	}
	accessAge := now.Sub(info.LastAccessTime)
	data.IsAccessedWithin = types.BoolValue(accessAge <= duration)
	return nil
}

func populateOwnerGroupNames(data *StatDataSourceModel, info *plugin.FileInfo) {
	isLocal := data.Service.IsNull() || data.Service.ValueString() == "" || data.Service.ValueString() == "local"
	data.OwnerName = resolveIdentityName(isLocal, info.UID, true)
	data.GroupName = resolveIdentityName(isLocal, info.GID, false)
}

func resolveIdentityName(isLocal bool, id int, isUser bool) types.String {
	idStr := fmt.Sprintf("%d", id)
	if !isLocal {
		return types.StringValue(idStr)
	}
	if isUser {
		if u, err := user.LookupId(idStr); err == nil {
			return types.StringValue(u.Username)
		}
		return types.StringValue(idStr)
	}
	if g, err := user.LookupGroupId(idStr); err == nil {
		return types.StringValue(g.Name)
	}
	return types.StringValue(idStr)
}

// parseDuration parses a duration string with support for days (d).
func parseDuration(s string) (time.Duration, error) {
	// Handle day suffix (e.g., "7d", "30d")
	if len(s) > 1 && s[len(s)-1] == 'd' {
		var days int
		if _, err := fmt.Sscanf(s, "%dd", &days); err == nil {
			return time.Duration(days) * 24 * time.Hour, nil
		}
	}
	return time.ParseDuration(s)
}

// getBackend returns the appropriate backend.
func (d *StatDataSource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return d.config.LocalBackend, nil
	}
	return d.config.Registry.Backends.GetAlias(backendName)
}
