// SPDX-License-Identifier: MIT

// Package environment implements the filemanager_environment data source.
package environment

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var _ datasource.DataSource = &EnvironmentDataSource{}

// NewEnvironmentDataSource creates a new environment data source.
func NewEnvironmentDataSource() datasource.DataSource {
	return &EnvironmentDataSource{}
}

// EnvironmentDataSource defines the data source implementation.
type EnvironmentDataSource struct {
	config *common.ProviderConfig
}

// EnvironmentDataSourceModel describes the data source data model.
type EnvironmentDataSourceModel struct {
	ID       types.String `tfsdk:"id"`
	Service  types.String `tfsdk:"service"`
	Filter   types.String `tfsdk:"filter"`
	Vars     types.Map    `tfsdk:"vars"`
	VarMap   types.Map    `tfsdk:"var_map"`
	VarCount types.Int64  `tfsdk:"var_count"`
}

// Metadata returns the data source type name.
func (d *EnvironmentDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_environment"
}

// Schema defines the schema for the data source.
func (d *EnvironmentDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Reads environment variables from local or remote systems.",
		MarkdownDescription: `
Reads environment variables from the local system or from a remote system via SSH.

## Example Usage

### Local Environment Variables

` + "```hcl" + `
data "filemanager_environment" "local" {}

output "path" {
  value = data.filemanager_environment.local.vars["PATH"]
}
` + "```" + `

### Remote Environment Variables via SSH

` + "```hcl" + `
resource "filemanager_ssh_service" "server" {
  host             = "example.com"
  port             = 22
  username         = "admin"
  private_key_file = "~/.ssh/id_rsa"
}

data "filemanager_environment" "remote" {
  service = filemanager_ssh_service.server.id
}
` + "```" + `

### Filtered Environment Variables

` + "```hcl" + `
data "filemanager_environment" "aws" {
  filter = "AWS_*"
}
` + "```" + `
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the data source.",
				Computed:    true,
			},
			"service": schema.StringAttribute{
				Description: "Service alias for remote environment access (e.g., SSH service). If not specified, reads from local environment.",
				Optional:    true,
			},
			"filter": schema.StringAttribute{
				Description: "Filter environment variables by name pattern (supports * wildcard). For example, 'AWS_*' matches all AWS-related variables.",
				Optional:    true,
			},
			"vars": schema.MapAttribute{
				Description: "Map of environment variable names to their values.",
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"var_map": schema.MapAttribute{
				Description: "Alias for vars. Map of environment variable names to their values.",
				Computed:    true,
				Sensitive:   true,
				ElementType: types.StringType,
			},
			"var_count": schema.Int64Attribute{
				Description: "Number of environment variables returned.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the data source with provider data.
func (d *EnvironmentDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *EnvironmentDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data EnvironmentDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var envVars map[string]string
	var err error

	serviceName := data.Service.ValueString()
	if serviceName == "" || serviceName == "local" {
		// Read local environment
		envVars = readLocalEnvironment()
		data.ID = types.StringValue("local")
	} else {
		// Read remote environment via SSH
		envVars, err = d.readRemoteEnvironment(ctx, serviceName)
		if err != nil {
			resp.Diagnostics.AddError("Failed to read remote environment", err.Error())
			return
		}
		data.ID = types.StringValue(fmt.Sprintf("remote:%s", serviceName))
	}

	// Apply filter if specified
	filter := data.Filter.ValueString()
	if filter != "" {
		envVars = filterEnvironment(envVars, filter)
	}

	// Convert to Terraform map
	varsMap, diags := types.MapValueFrom(ctx, types.StringType, envVars)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	data.Vars = varsMap
	data.VarMap = varsMap
	data.VarCount = types.Int64Value(int64(len(envVars)))

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// readLocalEnvironment reads environment variables from the local system.
func readLocalEnvironment() map[string]string {
	envVars := make(map[string]string)

	for _, env := range os.Environ() {
		parts := strings.SplitN(env, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	return envVars
}

// readRemoteEnvironment reads environment variables from a remote system via SSH.
func (d *EnvironmentDataSource) readRemoteEnvironment(ctx context.Context, serviceName string) (map[string]string, error) {
	// Get the backend for this service
	backend, err := d.config.Registry.Backends.GetAlias(serviceName)
	if err != nil {
		return nil, fmt.Errorf("failed to get backend for service %s: %w", serviceName, err)
	}

	// Check if the backend supports command execution
	executor, ok := backend.(plugin.CommandExecutor)
	if !ok {
		return nil, fmt.Errorf("service %s does not support command execution (required for environment reading)", serviceName)
	}

	// Execute 'env' command to get environment variables
	output, err := executor.Execute(ctx, "env")
	if err != nil {
		// Try 'printenv' as fallback
		output, err = executor.Execute(ctx, "printenv")
		if err != nil {
			return nil, fmt.Errorf("failed to execute environment command: %w", err)
		}
	}

	// Parse the output
	envVars := make(map[string]string)
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := scanner.Text()
		if line == "" {
			continue
		}

		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			envVars[parts[0]] = parts[1]
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error parsing environment output: %w", err)
	}

	return envVars, nil
}

// filterEnvironment filters environment variables by pattern.
func filterEnvironment(envVars map[string]string, pattern string) map[string]string {
	filtered := make(map[string]string)

	for name, value := range envVars {
		if matchPattern(name, pattern) {
			filtered[name] = value
		}
	}

	return filtered
}

// matchPattern matches a string against a wildcard pattern using filepath.Match.
func matchPattern(s, pattern string) bool {
	if pattern == "" {
		return true
	}
	matched, err := filepath.Match(pattern, s)
	if err != nil {
		return s == pattern
	}
	return matched
}

// GetEnvironmentKeys returns sorted list of environment variable names.
// Exported for use by other packages if needed.
func GetEnvironmentKeys(envVars map[string]string) []string {
	keys := make([]string, 0, len(envVars))
	for k := range envVars {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
