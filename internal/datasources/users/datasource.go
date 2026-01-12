// SPDX-License-Identifier: MIT

// Package users implements the filemanager_users data source.
package users

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
var _ datasource.DataSource = &UsersDataSource{}

// NewUsersDataSource creates a new users data source.
func NewUsersDataSource() datasource.DataSource {
	return &UsersDataSource{}
}

// UsersDataSource defines the data source implementation.
type UsersDataSource struct {
	config *common.ProviderConfig
}

// UserModel represents a single user.
type UserModel struct {
	Username types.String `tfsdk:"username"`
	UID      types.Int64  `tfsdk:"uid"`
	GID      types.Int64  `tfsdk:"gid"`
	Name     types.String `tfsdk:"name"`
	HomeDir  types.String `tfsdk:"home_dir"`
	Shell    types.String `tfsdk:"shell"`
}

// UsersDataSourceModel describes the data source data model.
type UsersDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Filter   types.String `tfsdk:"filter"`
	MinUID   types.Int64  `tfsdk:"min_uid"`
	MaxUID   types.Int64  `tfsdk:"max_uid"`
	Users    []UserModel  `tfsdk:"users"`
	UserMap  types.Map    `tfsdk:"user_map"`
	UIDMap   types.Map    `tfsdk:"uid_map"`
}

// Metadata returns the data source type name.
func (d *UsersDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_users"
}

// Schema defines the schema for the data source.
func (d *UsersDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Lists system users. Unix/Linux only.",
		MarkdownDescription: `Lists all users on the system. This data source reads from /etc/passwd on Unix/Linux systems.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"filter": schema.StringAttribute{
				Description: "Filter users by username pattern (supports * wildcard).",
				Optional:    true,
			},
			"min_uid": schema.Int64Attribute{
				Description: "Minimum UID to include (e.g., 1000 to exclude system users).",
				Optional:    true,
			},
			"max_uid": schema.Int64Attribute{
				Description: "Maximum UID to include.",
				Optional:    true,
			},
			"users": schema.ListNestedAttribute{
				Description: "List of users.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"username": schema.StringAttribute{
							Description: "Username.",
							Computed:    true,
						},
						"uid": schema.Int64Attribute{
							Description: "User ID.",
							Computed:    true,
						},
						"gid": schema.Int64Attribute{
							Description: "Primary group ID.",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Full name (GECOS field).",
							Computed:    true,
						},
						"home_dir": schema.StringAttribute{
							Description: "Home directory.",
							Computed:    true,
						},
						"shell": schema.StringAttribute{
							Description: "Login shell.",
							Computed:    true,
						},
					},
				},
			},
			"user_map": schema.MapAttribute{
				Description: "Map of username to UID for easy lookup.",
				Computed:    true,
				ElementType: types.Int64Type,
			},
			"uid_map": schema.MapAttribute{
				Description: "Map of UID (as string) to username for reverse lookup.",
				Computed:    true,
				ElementType: types.StringType,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *UsersDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *UsersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data UsersDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check platform
	if runtime.GOOS == "windows" {
		resp.Diagnostics.AddError(
			"Unsupported Platform",
			"The users data source is only supported on Unix/Linux systems.",
		)
		return
	}

	// Read users
	users, err := readPasswd()
	if err != nil {
		resp.Diagnostics.AddError("Failed to read users", err.Error())
		return
	}

	// Apply filters
	filter := data.Filter.ValueString()
	minUID := int64(-1)
	maxUID := int64(-1)

	if !data.MinUID.IsNull() {
		minUID = data.MinUID.ValueInt64()
	}
	if !data.MaxUID.IsNull() {
		maxUID = data.MaxUID.ValueInt64()
	}

	var filteredUsers []UserModel
	userMap := make(map[string]int64)
	uidMap := make(map[string]string)

	for _, u := range users {
		// Apply UID filter
		if minUID >= 0 && u.UID < minUID {
			continue
		}
		if maxUID >= 0 && u.UID > maxUID {
			continue
		}

		// Apply name filter
		if filter != "" && !matchPattern(u.Username, filter) {
			continue
		}

		filteredUsers = append(filteredUsers, UserModel{
			Username: types.StringValue(u.Username),
			UID:      types.Int64Value(u.UID),
			GID:      types.Int64Value(u.GID),
			Name:     types.StringValue(u.Name),
			HomeDir:  types.StringValue(u.HomeDir),
			Shell:    types.StringValue(u.Shell),
		})

		userMap[u.Username] = u.UID
		uidMap[strconv.FormatInt(u.UID, 10)] = u.Username
	}

	data.ID = types.StringValue("users")
	data.Users = filteredUsers

	// Convert maps to Terraform types
	userMapValue, diags := types.MapValueFrom(ctx, types.Int64Type, userMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UserMap = userMapValue

	uidMapValue, diags := types.MapValueFrom(ctx, types.StringType, uidMap)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	data.UIDMap = uidMapValue

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// passwdEntry represents a parsed /etc/passwd entry.
type passwdEntry struct {
	Username string
	UID      int64
	GID      int64
	Name     string
	HomeDir  string
	Shell    string
}

// readPasswd reads and parses /etc/passwd.
func readPasswd() ([]passwdEntry, error) {
	file, err := os.Open("/etc/passwd")
	if err != nil {
		return nil, fmt.Errorf("failed to open /etc/passwd: %w", err)
	}
	defer file.Close()

	var entries []passwdEntry
	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		// Skip empty lines and comments
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		parts := strings.Split(line, ":")
		if len(parts) < 7 {
			continue
		}

		uid, err := strconv.ParseInt(parts[2], 10, 64)
		if err != nil {
			continue
		}

		gid, err := strconv.ParseInt(parts[3], 10, 64)
		if err != nil {
			continue
		}

		// GECOS field may contain commas, take first part as name
		name := parts[4]
		if idx := strings.Index(name, ","); idx != -1 {
			name = name[:idx]
		}

		entries = append(entries, passwdEntry{
			Username: parts[0],
			UID:      uid,
			GID:      gid,
			Name:     name,
			HomeDir:  parts[5],
			Shell:    parts[6],
		})
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading /etc/passwd: %w", err)
	}

	return entries, nil
}

// LookupUser looks up a user by name using the os/user package.
// This is exported for use by resources that need to resolve usernames.
func LookupUser(username string) (uid int, err error) {
	u, err := user.Lookup(username)
	if err != nil {
		return -1, err
	}
	uidInt, err := strconv.Atoi(u.Uid)
	if err != nil {
		return -1, err
	}
	return uidInt, nil
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
