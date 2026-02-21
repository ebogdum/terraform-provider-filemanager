// SPDX-License-Identifier: MIT

// Package ssh_service implements the filemanager_ssh_service resource.
package ssh_service

import (
	"context"
	"fmt"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/ssh"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SSHServiceResource{}
	_ resource.ResourceWithImportState = &SSHServiceResource{}
)

// NewSSHServiceResource creates a new SSH service resource.
func NewSSHServiceResource() resource.Resource {
	return &SSHServiceResource{}
}

// SSHServiceResource defines the resource implementation.
type SSHServiceResource struct {
	config *common.ProviderConfig
}

// SSHServiceResourceModel describes the resource data model.
type SSHServiceResourceModel struct {
	// Inputs
	Host                types.String `tfsdk:"host"`
	Port                types.Int64  `tfsdk:"port"`
	Username            types.String `tfsdk:"username"`
	Password            types.String `tfsdk:"password"`
	PasswordFile        types.String `tfsdk:"password_file"`
	PrivateKey          types.String `tfsdk:"private_key"`
	PrivateKeyFile      types.String `tfsdk:"private_key_file"`
	Passphrase          types.String `tfsdk:"passphrase"`
	PassphraseFile      types.String `tfsdk:"passphrase_file"`
	BasePath            types.String `tfsdk:"base_path"`
	Timeout             types.String `tfsdk:"timeout"`
	InsecureSkipHostKey types.Bool   `tfsdk:"insecure_skip_host_key"`
	KnownHostsFile      types.String `tfsdk:"known_hosts_file"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	Endpoint    types.String `tfsdk:"endpoint"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *SSHServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_ssh_service"
}

// Schema defines the schema for the resource.
func (r *SSHServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures an SSH/SFTP service for use with other filemanager resources.",
		MarkdownDescription: "Configures an SSH/SFTP service for remote file operations via SFTP.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: ssh:host:port).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"host": schema.StringAttribute{
				Description: "SSH server hostname or IP address.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"port": schema.Int64Attribute{
				Description: "SSH server port. Defaults to 22.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(22),
			},
			"username": schema.StringAttribute{
				Description: "SSH username.",
				Required:    true,
			},
			"password": schema.StringAttribute{
				Description: "SSH password. Use private_key for key-based authentication.",
				Optional:    true,
				Sensitive:   true,
			},
			"password_file": schema.StringAttribute{
				Description: "Path to file containing SSH password.",
				Optional:    true,
			},
			"private_key": schema.StringAttribute{
				Description: "SSH private key content (PEM format).",
				Optional:    true,
				Sensitive:   true,
			},
			"private_key_file": schema.StringAttribute{
				Description: "Path to SSH private key file.",
				Optional:    true,
			},
			"passphrase": schema.StringAttribute{
				Description: "Passphrase for encrypted private key.",
				Optional:    true,
				Sensitive:   true,
			},
			"passphrase_file": schema.StringAttribute{
				Description: "Path to file containing passphrase for encrypted private key.",
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
			"insecure_skip_host_key": schema.BoolAttribute{
				Description: "Deprecated insecure option. Setting this to true is rejected for security reasons.",
				Optional:    true,
			},
			"known_hosts_file": schema.StringAttribute{
				Description: "Path to known_hosts file for host key verification. Defaults to ~/.ssh/known_hosts.",
				Optional:    true,
			},
			"connected": schema.BoolAttribute{
				Description: "Whether the service is currently connected.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The SSH endpoint in host:port format.",
				Computed:    true,
			},
			"service_type": schema.StringAttribute{
				Description: "The type of service (s3, gcs, azure, b2, or ssh).",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *SSHServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SSHServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SSHServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	host := data.Host.ValueString()
	port := data.Port.ValueInt64()

	tflog.Debug(ctx, "Creating SSH service", map[string]any{
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

	privateKey, err := common.ReadCredential(
		data.PrivateKey.ValueString(),
		data.PrivateKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read private key", err.Error())
		return
	}

	passphrase, err := common.ReadCredential(
		data.Passphrase.ValueString(),
		data.PassphraseFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read passphrase", err.Error())
		return
	}

	// Parse timeout
	timeout, err := time.ParseDuration(data.Timeout.ValueString())
	if err != nil {
		timeout = 30 * time.Second
	}

	// Create and connect the backend
	backend := ssh.New()
	backendConfig := plugin.BackendConfig{
		Host:       host,
		Port:       int(port),
		Username:   data.Username.ValueString(),
		Password:   password,
		PrivateKey: []byte(privateKey),
		BasePath:   data.BasePath.ValueString(),
		Timeout:    timeout,
		Extra: map[string]any{
			"passphrase":             passphrase,
			"insecure_skip_host_key": data.InsecureSkipHostKey.ValueBool(),
			"known_hosts_file":       data.KnownHostsFile.ValueString(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect via SSH", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("ssh:%s:%d", host, port)
	endpoint := fmt.Sprintf("%s:%d", host, port)

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register SSH service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(endpoint)
	data.ServiceType = types.StringValue("ssh")

	tflog.Info(ctx, "Created SSH service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *SSHServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SSHServiceResourceModel

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
func (r *SSHServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SSHServiceResourceModel

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

	privateKey, err := common.ReadCredential(
		data.PrivateKey.ValueString(),
		data.PrivateKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read private key", err.Error())
		return
	}

	passphrase, err := common.ReadCredential(
		data.Passphrase.ValueString(),
		data.PassphraseFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read passphrase", err.Error())
		return
	}

	timeout, err := time.ParseDuration(data.Timeout.ValueString())
	if err != nil {
		timeout = 30 * time.Second
	}

	backend := ssh.New()
	backendConfig := plugin.BackendConfig{
		Host:       host,
		Port:       int(port),
		Username:   data.Username.ValueString(),
		Password:   password,
		PrivateKey: []byte(privateKey),
		BasePath:   data.BasePath.ValueString(),
		Timeout:    timeout,
		Extra: map[string]any{
			"passphrase":             passphrase,
			"insecure_skip_host_key": data.InsecureSkipHostKey.ValueBool(),
			"known_hosts_file":       data.KnownHostsFile.ValueString(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect via SSH", err.Error())
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
		resp.Diagnostics.AddError("Failed to update SSH service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(endpoint)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *SSHServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SSHServiceResourceModel

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

	tflog.Info(ctx, "Deleted SSH service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *SSHServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
