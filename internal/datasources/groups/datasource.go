// SPDX-License-Identifier: MIT

// Package groups implements the filemanager_groups data source.
package groups

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"os/user"
	"runtime"
	"strconv"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &GroupsDataSource{}

// NewGroupsDataSource creates a new groups data source.
func NewGroupsDataSource() datasource.DataSource {
	return &GroupsDataSource{}
}

// GroupsDataSource defines the data source implementation.
type GroupsDataSource struct {
	config *common.ProviderConfig
}

// GroupModel represents a single group.
type GroupModel struct {
	Name    types.String `tfsdk:"name"`
	GID     types.Int64  `tfsdk:"gid"`
	Members types.List   `tfsdk:"members"`
}

// GroupsDataSourceModel describes the data source data model.
type GroupsDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Filter   types.String `tfsdk:"filter"`
	MinGID   types.Int64  `tfsdk:"min_gid"`
	MaxGID   types.Int64  `tfsdk:"max_gid"`
	Groups   []GroupModel `tfsdk:"groups"`
	GroupMap types.Map    `tfsdk:"group_map"`
	GIDMap   types.Map    `tfsdk:"gid_map"`
}

// Metadata returns the data source type name.
func (d *GroupsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_groups"
}

// Schema defines the schema for the data source.
func (d *GroupsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists system groups. Unix/Linux only.",
		MarkdownDescription: `Lists all groups on the system. This data source reads from /etc/group on Unix/Linux systems.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"filter": schema.StringAttribute{
				Description: "Filter groups by name pattern (supports * wildcard).",
				Optional:    true,
			},
			"min_gid": schema.Int64Attribute{
				Description: "Minimum GID to include (e.g., 1000 to exclude system groups).",
				Optional:    true,
			},
			"max_gid": schema.Int64Attribute{
				Description: "Maximum GID to include.",
				Optional:    true,
			},
			"groups": schema.ListNestedAttribute{
				Description: "List of groups.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							Description: "Group name.",
							Computed:    true,
						},
						"gid": schema.Int64Attribute{
							Description: "Group ID.",
							Computed:    true,
						},
						"members": schema.ListAttribute{
							Description: "List of usernames that are members of this group.",
							Computed:    true,
							ElementType: types.StringType,
						},
					},
				},
			},
			"group_map": schema.MapAttribute{
				Description: "Map of group name to GID for easy lookup.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"gid_map": schema.MapAttribute{
				Description: "Map of GID (as string) to group name for reverse lookup.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *GroupsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *GroupsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data GroupsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check platform
	if runtime.GOOS == "windows" {
		resp.Diagnostics.AddError(
			"Unsupported Platform",
			"The groups data source is only supported on Unix/Linux systems.",
		)
		return
	}

	// Read groups
	groups, err := readGroup()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read groups", err.Error())
		return
	}

	// Apply filters
	filter := data.Filter.ValueString()
	minGID := int64(-1)
	maxGID := int64(-1)

	if !data.MinGID.IsNull() {
		minGID = data.MinGID.ValueInt64()
	}
	if !data.MaxGID.IsNull() {
		maxGID = data.MaxGID.ValueInt64()
	}

	var filteredGroups []GroupModel
	groupMap := make(map[string]int64)
	gidMap := make(map[string]string)

	for _, g := range groups {
		// Apply GID filter
		if minGID >= 0 && g.GID < minGID {
			continue
		}
		if maxGID >= 0 && g.GID > maxGID {
			continue
		}

		// Apply name filter
		if filter != "" && !matchPattern(g.Name, filter) {
			continue
		}

		// Convert members to Terraform list
		membersList, diags := types.ListValueFrom(ctx, types.StringType, g.Members)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}

		filteredGroups = append(filteredGroups, GroupModel{
			Name:    types.StringValue(g.Name),
			GID:     types.Int64Value(g.GID),
			Members: membersList,
		})

		groupMap[g.Name] = g.GID
		gidMap[strconv.FormatInt(g.GID, 10)] = g.Name
	}

	data.ID = types.StringValue("groups")
	data.Groups = filteredGroups

	// Convert maps to Terraform types
	groupMapValue, diags := types.MapValueFrom(ctx, types.Int64Type, groupMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.GroupMap = groupMapValue

	gidMapValue, diags := types.MapValueFrom(ctx, types.StringType, gidMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.GIDMap = gidMapValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// groupEntry represents a parsed /etc/group entry.
type groupEntry struct {
	Name    string
	GID     int64
	Members []string
}

// readGroup reads and parses /etc/group.
func readGroup() ([]groupEntry, error) {
	file, err := os.Open("/etc/group")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/group: %w", err)
	}
	defer file.Close()

	var entries []groupEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 4 {
			continue
		}

		gid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}

		var members []string
		if parts[3] != "" {
			members = strings.Split(parts[3], ",")
		}

		entries = append(entries, groupEntry{
			Name:    parts[0],
			GID:     gid,
			Members: members,
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /etc/group: %w", err)
	}

	return entries, nil
}

// LookupGroup looks up a group by name using the os/user package.
// This is exported for use by resources that need to resolve group names.
func LookupGroup(groupname string) (gid int, err error) {
	g, err := user.LookupGroup(groupname)
	if err != nil {
		return -1, err
	}
	gidInt, err := strconv.Atoi(g.Gid)
	if err != nil {
		return -1, err
	}
	return gidInt, nil
}

// matchPattern matches a string against a simple wildcard pattern.
func matchPattern(s, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if !strings.Contains(pattern, "*") {
		return s == pattern
	}

	parts := strings.Split(pattern, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(s, parts[0]) && strings.HasSuffix(s, parts[1])
	}

	return s == pattern
}
