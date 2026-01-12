// SPDX-License-Identifier: MIT

// Package app_config implements the filemanager_app_config resource.
package app_config

import (
	"context"
	"crypto/md5"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/apps/consul"
	"github.com/ebogdum/filemanager/internal/apps/docker"
	"github.com/ebogdum/filemanager/internal/apps/elasticsearch"
	"github.com/ebogdum/filemanager/internal/apps/envoy"
	"github.com/ebogdum/filemanager/internal/apps/grafana"
	"github.com/ebogdum/filemanager/internal/apps/haproxy"
	"github.com/ebogdum/filemanager/internal/apps/httpd"
	"github.com/ebogdum/filemanager/internal/apps/kubernetes"
	"github.com/ebogdum/filemanager/internal/apps/mysql"
	"github.com/ebogdum/filemanager/internal/apps/nginx"
	"github.com/ebogdum/filemanager/internal/apps/nomad"
	"github.com/ebogdum/filemanager/internal/apps/postgresql"
	"github.com/ebogdum/filemanager/internal/apps/prometheus"
	"github.com/ebogdum/filemanager/internal/apps/redis"
	"github.com/ebogdum/filemanager/internal/apps/ssh_client"
	"github.com/ebogdum/filemanager/internal/apps/sshd"
	"github.com/ebogdum/filemanager/internal/apps/systemd"
	"github.com/ebogdum/filemanager/internal/apps/traefik"
	"github.com/ebogdum/filemanager/internal/apps/vault"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &AppConfigResource{}
	_ resource.ResourceWithImportState = &AppConfigResource{}
)

// NewAppConfigResource creates a new app config resource.
func NewAppConfigResource() resource.Resource {
	return &AppConfigResource{
		apps: map[string]plugin.AppPlugin{
			"nginx":         nginx.New(),
			"prometheus":    prometheus.New(),
			"consul":        consul.New(),
			"redis":         redis.New(),
			"kubernetes":    kubernetes.New(),
			"vault":         vault.New(),
			"docker":        docker.New(),
			"systemd":       systemd.New(),
			"nomad":         nomad.New(),
			"grafana":       grafana.New(),
			"haproxy":       haproxy.New(),
			"traefik":       traefik.New(),
			"envoy":         envoy.New(),
			"elasticsearch": elasticsearch.New(),
			"sshd":          sshd.New(),
			"ssh_client":    ssh_client.New(),
			"httpd":         httpd.New(),
			"mysql":         mysql.New(),
			"postgresql":    postgresql.New(),
		},
	}
}

// AppConfigResource defines the resource implementation.
type AppConfigResource struct {
	config *common.ProviderConfig
	apps   map[string]plugin.AppPlugin
}

// AppConfigResourceModel describes the resource data model.
type AppConfigResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	App                 types.String `tfsdk:"app"`
	Config              types.String `tfsdk:"config"`
	MergeWith           types.String `tfsdk:"merge_with"`
	Validate            types.Bool   `tfsdk:"validate"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`
	AtomicWrite         types.Bool   `tfsdk:"atomic_write"`

	// Computed
	Size             types.Int64  `tfsdk:"size"`
	MD5              types.String `tfsdk:"md5"`
	SHA256           types.String `tfsdk:"sha256"`
	Rendered         types.String `tfsdk:"rendered"`
	ValidationErrors types.List   `tfsdk:"validation_errors"`
	Directory        types.String `tfsdk:"directory"`
	Filename         types.String `tfsdk:"filename"`
	Extension        types.String `tfsdk:"extension"`
	AbsolutePath     types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *AppConfigResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_app_config"
}

// Schema defines the schema for the resource.
func (r *AppConfigResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages application-specific configuration files with semantic validation.",
		MarkdownDescription: `
Manages application-specific configuration files with semantic validation and native format output.

Supported applications:
- **nginx** - Nginx web server configuration
- **prometheus** - Prometheus monitoring configuration
`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the resource.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"path": schema.StringAttribute{
				Description: "The path where the configuration file will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"app": schema.StringAttribute{
				Description: "Application type: nginx, prometheus.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"config": schema.StringAttribute{
				Description: "Configuration content as JSON (use jsonencode()). Will be converted to native format.",
				Required:    true,
			},
			"merge_with": schema.StringAttribute{
				Description: "Additional configuration to merge (as JSON string).",
				Optional:    true,
			},
			"validate": schema.BoolAttribute{
				Description: "Validate configuration using app-specific rules.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode in octal format.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode for parent directories.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"create_parent_dirs": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"atomic_write": schema.BoolAttribute{
				Description: "Use atomic write operations.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
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
			"rendered": schema.StringAttribute{
				Description: "The rendered configuration content in native format.",
				Computed:    true,
			},
			"validation_errors": schema.ListAttribute{
				Description: "List of validation errors/warnings (if any).",
				Computed:    true,
				ElementType: types.StringType,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the file (e.g., 'config.conf').",
				Computed:    true,
			},
			"extension": schema.StringAttribute{
				Description: "The file extension without the leading dot (e.g., 'conf').",
				Computed:    true,
			},
			"absolute_path": schema.StringAttribute{
				Description: "The absolute resolved path.",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *AppConfigResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	config, ok := req.ProviderData.(*common.ProviderConfig)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected *common.ProviderConfig, got: %T", req.ProviderData),
		)
		return
	}

	r.config = config
}

// Create creates the resource.
func (r *AppConfigResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AppConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating app config file", map[string]any{
		"path": data.Path.ValueString(),
		"app":  data.App.ValueString(),
	})

	// Get the app plugin
	appPlugin, err := r.getAppPlugin(data.App.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unknown application type", err.Error())
		return
	}

	// Generate native content
	content, validationErrors, err := r.generateContent(ctx, &data, appPlugin)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate configuration", err.Error())
		return
	}

	// Store validation errors
	if len(validationErrors) > 0 {
		errorStrings := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorStrings[i] = fmt.Sprintf("%s: %s", ve.Path, ve.Message)
		}
		errorList, diags := types.ListValueFrom(ctx, types.StringType, errorStrings)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.ValidationErrors = errorList

		// If validation is enabled and there are errors, fail
		if data.Validate.ValueBool() {
			for _, ve := range validationErrors {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Validation error at %s", ve.Path),
					ve.Message,
				)
			}
			return
		}
	} else {
		data.ValidationErrors, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Write file
	writeOpts := plugin.WriteOptions{
		Mode:       common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:    common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs: data.CreateParentDirs.ValueBool(),
		Overwrite:  true,
		Atomic:     data.AtomicWrite.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	// Update computed values
	r.updateComputedValues(&data, content)
	data.ID = data.Path

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read reads the resource.
func (r *AppConfigResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AppConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		resp.State.RemoveResource(ctx)
		return
	}

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

	r.updateComputedValues(&data, content)

	// Keep validation_errors as is from state if not empty
	if data.ValidationErrors.IsNull() {
		data.ValidationErrors, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *AppConfigResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AppConfigResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating app config file", map[string]any{
		"path": data.Path.ValueString(),
		"app":  data.App.ValueString(),
	})

	// Get the app plugin
	appPlugin, err := r.getAppPlugin(data.App.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Unknown application type", err.Error())
		return
	}

	// Generate native content
	content, validationErrors, err := r.generateContent(ctx, &data, appPlugin)
	if err != nil {
		resp.Diagnostics.AddError("Failed to generate configuration", err.Error())
		return
	}

	// Store validation errors
	if len(validationErrors) > 0 {
		errorStrings := make([]string, len(validationErrors))
		for i, ve := range validationErrors {
			errorStrings[i] = fmt.Sprintf("%s: %s", ve.Path, ve.Message)
		}
		errorList, diags := types.ListValueFrom(ctx, types.StringType, errorStrings)
		resp.Diagnostics.Append(diags...)
		if resp.Diagnostics.HasError() {
			return
		}
		data.ValidationErrors = errorList

		if data.Validate.ValueBool() {
			for _, ve := range validationErrors {
				resp.Diagnostics.AddError(
					fmt.Sprintf("Validation error at %s", ve.Path),
					ve.Message,
				)
			}
			return
		}
	} else {
		data.ValidationErrors, _ = types.ListValueFrom(ctx, types.StringType, []string{})
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	writeOpts := plugin.WriteOptions{
		Mode:       common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:    common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs: data.CreateParentDirs.ValueBool(),
		Overwrite:  true,
		Atomic:     data.AtomicWrite.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	r.updateComputedValues(&data, content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *AppConfigResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AppConfigResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	if err := backend.Delete(ctx, data.Path.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete file", err.Error())
			return
		}
	}
}

// ImportState imports an existing resource.
func (r *AppConfigResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// getAppPlugin returns the app plugin by name.
func (r *AppConfigResource) getAppPlugin(name string) (plugin.AppPlugin, error) {
	appPlugin, ok := r.apps[name]
	if !ok {
		available := make([]string, 0, len(r.apps))
		for k := range r.apps {
			available = append(available, k)
		}
		return nil, fmt.Errorf("unknown app type %q, available: %v", name, available)
	}
	return appPlugin, nil
}

// generateContent generates the final native content.
func (r *AppConfigResource) generateContent(ctx context.Context, data *AppConfigResourceModel, appPlugin plugin.AppPlugin) ([]byte, []plugin.ValidationError, error) {
	// Parse the JSON config string into a map
	var config any
	if err := json.Unmarshal([]byte(data.Config.ValueString()), &config); err != nil {
		return nil, nil, fmt.Errorf("invalid JSON in config: %w", err)
	}

	// Merge if merge_with is specified
	if !data.MergeWith.IsNull() && !data.MergeWith.IsUnknown() && data.MergeWith.ValueString() != "" {
		var mergeWith any
		if err := json.Unmarshal([]byte(data.MergeWith.ValueString()), &mergeWith); err != nil {
			return nil, nil, fmt.Errorf("invalid JSON in merge_with: %w", err)
		}

		var err error
		config, err = appPlugin.Merge(config, mergeWith)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to merge config: %w", err)
		}
	}

	// Normalize the configuration
	config, err := appPlugin.Normalize(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to normalize config: %w", err)
	}

	// Validate the configuration
	var validationErrors []plugin.ValidationError
	if data.Validate.ValueBool() {
		structErrors, err := appPlugin.Validate(config)
		if err != nil {
			return nil, nil, fmt.Errorf("validation failed: %w", err)
		}
		validationErrors = append(validationErrors, structErrors...)

		semanticErrors, err := appPlugin.ValidateSemantic(config)
		if err != nil {
			return nil, nil, fmt.Errorf("semantic validation failed: %w", err)
		}
		validationErrors = append(validationErrors, semanticErrors...)
	}

	// Convert to native format
	content, err := appPlugin.ToNative(config)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to convert to native format: %w", err)
	}

	return content, validationErrors, nil
}

// getBackend returns the appropriate backend.
func (r *AppConfigResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *AppConfigResource) updateComputedValues(data *AppConfigResourceModel, content []byte) {
	data.Size = types.Int64Value(int64(len(content)))
	data.Rendered = types.StringValue(string(content))

	md5sum := md5.Sum(content)
	data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
