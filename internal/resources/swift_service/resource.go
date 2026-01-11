// SPDX-License-Identifier: MIT

// Package swift_service implements the filemanager_swift_service resource.
package swift_service

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

	"github.com/ebogdum/filemanager/internal/backends/swift"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SwiftServiceResource{}
	_ resource.ResourceWithImportState = &SwiftServiceResource{}
)

// NewSwiftServiceResource creates a new Swift service resource.
func NewSwiftServiceResource() resource.Resource {
	return &SwiftServiceResource{}
}

// SwiftServiceResource defines the resource implementation.
type SwiftServiceResource struct {
	config *common.ProviderConfig
}

// SwiftServiceResourceModel describes the resource data model.
type SwiftServiceResourceModel struct {
	// Inputs
	AuthURL                     types.String `tfsdk:"auth_url"`
	Container                   types.String `tfsdk:"container"`
	Username                    types.String `tfsdk:"username"`
	Password                    types.String `tfsdk:"password"`
	PasswordFile                types.String `tfsdk:"password_file"`
	Tenant                      types.String `tfsdk:"tenant"`
	TenantID                    types.String `tfsdk:"tenant_id"`
	Domain                      types.String `tfsdk:"domain"`
	DomainID                    types.String `tfsdk:"domain_id"`
	Region                      types.String `tfsdk:"region"`
	AuthVersion                 types.Int64  `tfsdk:"auth_version"`
	Token                       types.String `tfsdk:"token"`
	StorageURL                  types.String `tfsdk:"storage_url"`
	ApplicationCredentialID     types.String `tfsdk:"application_credential_id"`
	ApplicationCredentialSecret types.String `tfsdk:"application_credential_secret"`
	BasePath                    types.String `tfsdk:"base_path"`
	Timeout                     types.String `tfsdk:"timeout"`
	CreateContainer             types.Bool   `tfsdk:"create_container"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	Endpoint    types.String `tfsdk:"endpoint"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *SwiftServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_swift_service"
}

// Schema defines the schema for the resource.
func (r *SwiftServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures an OpenStack Swift service for use with other filemanager resources.",
		MarkdownDescription: `Configures an OpenStack Swift object storage service for remote file operations.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: swift:region:container).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"auth_url": schema.StringAttribute{
				Description: "OpenStack identity (Keystone) authentication URL.",
				Required:    true,
			},
			"container": schema.StringAttribute{
				Description: "Swift container name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"username": schema.StringAttribute{
				Description: "OpenStack username.",
				Optional:    true,
			},
			"password": schema.StringAttribute{
				Description: "OpenStack password or API key.",
				Optional:    true,
				Sensitive:   true,
			},
			"password_file": schema.StringAttribute{
				Description: "Path to file containing OpenStack password.",
				Optional:    true,
			},
			"tenant": schema.StringAttribute{
				Description: "OpenStack tenant/project name.",
				Optional:    true,
			},
			"tenant_id": schema.StringAttribute{
				Description: "OpenStack tenant/project ID.",
				Optional:    true,
			},
			"domain": schema.StringAttribute{
				Description: "OpenStack domain name. Defaults to 'Default'.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("Default"),
			},
			"domain_id": schema.StringAttribute{
				Description: "OpenStack domain ID.",
				Optional:    true,
			},
			"region": schema.StringAttribute{
				Description: "OpenStack region name.",
				Optional:    true,
			},
			"auth_version": schema.Int64Attribute{
				Description: "OpenStack identity API version (1, 2, or 3). Auto-detected if not specified.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(0),
			},
			"token": schema.StringAttribute{
				Description: "Pre-authenticated token. Use with storage_url for token-based auth.",
				Optional:    true,
				Sensitive:   true,
			},
			"storage_url": schema.StringAttribute{
				Description: "Swift storage URL. Required when using token authentication.",
				Optional:    true,
			},
			"application_credential_id": schema.StringAttribute{
				Description: "OpenStack application credential ID.",
				Optional:    true,
			},
			"application_credential_secret": schema.StringAttribute{
				Description: "OpenStack application credential secret.",
				Optional:    true,
				Sensitive:   true,
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations within the container.",
				Optional:    true,
			},
			"timeout": schema.StringAttribute{
				Description: "Connection timeout (e.g., '30s', '1m'). Defaults to 30s.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("30s"),
			},
			"create_container": schema.BoolAttribute{
				Description: "Create the container if it doesn't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"connected": schema.BoolAttribute{
				Description: "Whether the service is currently connected.",
				Computed:    true,
			},
			"endpoint": schema.StringAttribute{
				Description: "The Swift endpoint URL.",
				Computed:    true,
			},
			"service_type": schema.StringAttribute{
				Description: "The type of service (swift).",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *SwiftServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SwiftServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SwiftServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	container := data.Container.ValueString()
	region := data.Region.ValueString()

	tflog.Debug(ctx, "Creating Swift service", map[string]any{
		"container": container,
		"region":    region,
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
	backend := swift.New()
	backendConfig := plugin.BackendConfig{
		Endpoint: data.AuthURL.ValueString(),
		Username: data.Username.ValueString(),
		Password: password,
		Token:    data.Token.ValueString(),
		BasePath: data.BasePath.ValueString(),
		Timeout:  timeout,
		Extra: map[string]any{
			"container":                     container,
			"auth_url":                      data.AuthURL.ValueString(),
			"tenant":                        data.Tenant.ValueString(),
			"tenant_id":                     data.TenantID.ValueString(),
			"domain":                        data.Domain.ValueString(),
			"domain_id":                     data.DomainID.ValueString(),
			"region":                        region,
			"auth_version":                  int(data.AuthVersion.ValueInt64()),
			"storage_url":                   data.StorageURL.ValueString(),
			"application_credential_id":     data.ApplicationCredentialID.ValueString(),
			"application_credential_secret": data.ApplicationCredentialSecret.ValueString(),
			"create_container":              data.CreateContainer.ValueBool(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to Swift", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("swift:%s:%s", region, container)
	if region == "" {
		serviceID = fmt.Sprintf("swift:default:%s", container)
	}

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register Swift service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(data.AuthURL.ValueString())
	data.ServiceType = types.StringValue("swift")

	tflog.Info(ctx, "Created Swift service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *SwiftServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SwiftServiceResourceModel

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
func (r *SwiftServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SwiftServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	container := data.Container.ValueString()
	region := data.Region.ValueString()

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

	backend := swift.New()
	backendConfig := plugin.BackendConfig{
		Endpoint: data.AuthURL.ValueString(),
		Username: data.Username.ValueString(),
		Password: password,
		Token:    data.Token.ValueString(),
		BasePath: data.BasePath.ValueString(),
		Timeout:  timeout,
		Extra: map[string]any{
			"container":                     container,
			"auth_url":                      data.AuthURL.ValueString(),
			"tenant":                        data.Tenant.ValueString(),
			"tenant_id":                     data.TenantID.ValueString(),
			"domain":                        data.Domain.ValueString(),
			"domain_id":                     data.DomainID.ValueString(),
			"region":                        region,
			"auth_version":                  int(data.AuthVersion.ValueInt64()),
			"storage_url":                   data.StorageURL.ValueString(),
			"application_credential_id":     data.ApplicationCredentialID.ValueString(),
			"application_credential_secret": data.ApplicationCredentialSecret.ValueString(),
			"create_container":              data.CreateContainer.ValueBool(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to Swift", err.Error())
		return
	}

	serviceID := data.ID.ValueString()

	// Close old connection
	oldBackend, oldErr := r.config.Registry.Backends.GetAlias(serviceID)
	if oldErr == nil {
		_ = oldBackend.Close()
	}

	r.config.Registry.Backends.RemoveAlias(serviceID)
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update Swift service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	data.Endpoint = types.StringValue(data.AuthURL.ValueString())
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *SwiftServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SwiftServiceResourceModel

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

	tflog.Info(ctx, "Deleted Swift service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *SwiftServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
