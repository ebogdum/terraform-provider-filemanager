// SPDX-License-Identifier: MIT

// Package ftp_service implements the filemanager_ftp_service resource.
package ftp_service

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/ftp"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &FTPServiceResource{}
	_ resource.ResourceWithImportState = &FTPServiceResource{}
)

// NewFTPServiceResource creates a new FTP service resource.
func NewFTPServiceResource() resource.Resource {
	return &FTPServiceResource{}
}

// FTPServiceResource defines the resource implementation.
type FTPServiceResource struct {
	config *common.ProviderConfig
}

// FTPServiceResourceModel describes the resource data model.
type FTPServiceResourceModel struct {
	// Inputs
	Host          types.String `tfsdk:"host"`
	Port          types.Int64  `tfsdk:"port"`
	Username      types.String `tfsdk:"username"`
	Password      types.String `tfsdk:"password"`
	PasswordFile  types.String `tfsdk:"password_file"`
	BasePath      types.String `tfsdk:"base_path"`
	Timeout       types.String `tfsdk:"timeout"`
	TLSEnabled    types.Bool   `tfsdk:"tls_enabled"`
	ExplicitTLS   types.Bool   `tfsdk:"explicit_tls"`
	TLSSkipVerify types.Bool   `tfsdk:"tls_skip_verify"`
	PassiveMode   types.Bool   `tfsdk:"passive_mode"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	Endpoint    types.String `tfsdk:"endpoint"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *FTPServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ftp_service"
}

// Schema defines the schema for the resource.
func (r *FTPServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures an FTP/FTPS service for use with other filemanager resources.",
		MarkdownDescription: "Configures an FTP/FTPS service for remote file operations.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: ftp:host:port).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Description: "FTP server hostname or IP address.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "FTP server port. Defaults to 21.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(21),
			},
			"username": schema.StringAttribute{
				Description: "FTP username. Defaults to 'anonymous'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("anonymous"),
			},
			"password": schema.StringAttribute{
				Description: "FTP password.",
				Optional:    true,
				Sensitive:   true,
			},
			"password_file": schema.StringAttribute{
				Description: "Path to file containing FTP password.",
				Optional:    true,
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations.",
				Optional:    true,
			},
			"timeout": schema.StringAttribute{
				Description: "Connection timeout (e.g., '30s', '1m'). Defaults to 30s.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("30s"),
			},
			"tls_enabled": schema.BoolAttribute{
				Description: "Enable TLS/SSL for secure FTP (FTPS).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"explicit_tls": schema.BoolAttribute{
				Description: "Use explicit TLS (AUTH TLS) instead of implicit TLS. Only used when tls_enabled is true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"tls_skip_verify": schema.BoolAttribute{
				Description: "Skip TLS certificate verification. WARNING: This is insecure and should only be used for testing.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"passive_mode": schema.BoolAttribute{
				Description: "Use passive mode for data connections. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"connected": schema.BoolAttribute{
				Description: "Whether the service is currently connected.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The FTP endpoint in host:port format.",
				Computed:    true,
			},
			"service_type": schema.StringAttribute{
				Description: "The type of service (ftp).",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *FTPServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

// Create creates the resource and sets the initial Terraform state.
func (r *FTPServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FTPServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := data.Host.ValueString()
	port := data.Port.ValueInt64()

	tflog.Debug(ctx, "Creating FTP service", map[string]any{
		"host": host,
		"port": port,
	})

	// Read credentials from files if specified
	password, err := common.ReadCredential(
		data.Password.ValueString(),
		data.PasswordFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read password", err.Error())
		return
	}

	// Parse timeout
	timeout, err := time.ParseDuration(data.Timeout.ValueString())
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create and connect the backend
	backend := ftp.New()
	backendConfig := plugin.BackendConfig{
		Host:          host,
		Port:          int(port),
		Username:      data.Username.ValueString(),
		Password:      password,
		BasePath:      data.BasePath.ValueString(),
		Timeout:       timeout,
		TLSEnabled:    data.TLSEnabled.ValueBool(),
		TLSSkipVerify: data.TLSSkipVerify.ValueBool(),
		Extra: map[string]any{
			"explicit_tls": data.ExplicitTLS.ValueBool(),
			"passive_mode": data.PassiveMode.ValueBool(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect via FTP", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("ftp:%s:%d", host, port)
	endpoint := fmt.Sprintf("%s:%d", host, port)

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register FTP service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(endpoint)
	data.ServiceType = types.StringValue("ftp")

	tflog.Info(ctx, "Created FTP service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *FTPServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FTPServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.ID.ValueString()
	backend, err := r.config.Registry.Backends.GetAlias(serviceID)
	if err != nil {
		data.Connected = types.BoolValue(false)
	} else {
		if err := backend.Ping(ctx); err != nil {
			data.Connected = types.BoolValue(false)
		} else {
			data.Connected = types.BoolValue(true)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *FTPServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FTPServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := data.Host.ValueString()
	port := data.Port.ValueInt64()

	password, err := common.ReadCredential(
		data.Password.ValueString(),
		data.PasswordFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read password", err.Error())
		return
	}

	timeout, err := time.ParseDuration(data.Timeout.ValueString())
	if err != nil {
		timeout = 30 * time.Second
	}

	backend := ftp.New()
	backendConfig := plugin.BackendConfig{
		Host:          host,
		Port:          int(port),
		Username:      data.Username.ValueString(),
		Password:      password,
		BasePath:      data.BasePath.ValueString(),
		Timeout:       timeout,
		TLSEnabled:    data.TLSEnabled.ValueBool(),
		TLSSkipVerify: data.TLSSkipVerify.ValueBool(),
		Extra: map[string]any{
			"explicit_tls": data.ExplicitTLS.ValueBool(),
			"passive_mode": data.PassiveMode.ValueBool(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect via FTP", err.Error())
		return
	}

	serviceID := data.ID.ValueString()
	endpoint := fmt.Sprintf("%s:%d", host, port)

	// Close old connection
	oldBackend, oldErr := r.config.Registry.Backends.GetAlias(serviceID)
	if oldErr == nil {
		_ = oldBackend.Close()
	}

	r.config.Registry.Backends.RemoveAlias(serviceID)
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update FTP service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(endpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *FTPServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FTPServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.ID.ValueString()
	backend, err := r.config.Registry.Backends.GetAlias(serviceID)
	if err == nil {
		_ = backend.Close()
	}
	r.config.Registry.Backends.RemoveAlias(serviceID)

	tflog.Info(ctx, "Deleted FTP service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *FTPServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
