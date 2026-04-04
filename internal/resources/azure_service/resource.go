// SPDX-License-Identifier: MIT

// Package azure_service implements the filemanager_azure_service resource.
package azure_service

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/azure"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &AzureServiceResource{}
	_ resource.ResourceWithImportState = &AzureServiceResource{}
)

// NewAzureServiceResource creates a new Azure service resource.
func NewAzureServiceResource() resource.Resource {
	return &AzureServiceResource{}
}

// AzureServiceResource defines the resource implementation.
type AzureServiceResource struct {
	config *common.ProviderConfig
}

// AzureServiceResourceModel describes the resource data model.
type AzureServiceResourceModel struct {
	// Inputs
	AccountName          types.String `tfsdk:"account_name"`
	Container            types.String `tfsdk:"container"`
	BasePath             types.String `tfsdk:"base_path"`
	AccountKey           types.String `tfsdk:"account_key"`
	AccountKeyFile       types.String `tfsdk:"account_key_file"`
	ConnectionString     types.String `tfsdk:"connection_string"`
	ConnectionStringFile types.String `tfsdk:"connection_string_file"`
	SASToken             types.String `tfsdk:"sas_token"`
	SASTokenFile         types.String `tfsdk:"sas_token_file"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *AzureServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_azure_service"
}

// Schema defines the schema for the resource.
func (r *AzureServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures an Azure Blob Storage service for use with other filemanager resources.",
		MarkdownDescription: `Configures an Azure Blob Storage service for use with other filemanager resources.`,
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: azure:account:container).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"account_name": schema.StringAttribute{
				Description: "Azure Storage account name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"container": schema.StringAttribute{
				Description: "Azure Blob container name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations within the container.",
				Optional:    true,
			},
			"account_key": schema.StringAttribute{
				Description: "Azure Storage account key. One of account_key, connection_string, or sas_token is required.",
				Optional:    true,
				Sensitive:   true,
			},
			"account_key_file": schema.StringAttribute{
				Description: "Path to file containing Azure Storage account key.",
				Optional:    true,
			},
			"connection_string": schema.StringAttribute{
				Description: "Azure Storage connection string.",
				Optional:    true,
				Sensitive:   true,
			},
			"connection_string_file": schema.StringAttribute{
				Description: "Path to file containing Azure Storage connection string.",
				Optional:    true,
			},
			"sas_token": schema.StringAttribute{
				Description: "Azure SAS token for delegated access.",
				Optional:    true,
				Sensitive:   true,
			},
			"sas_token_file": schema.StringAttribute{
				Description: "Path to file containing Azure SAS token.",
				Optional:    true,
			},
			"connected": schema.BoolAttribute{
				Description: "Whether the service is currently connected.",
				Computed:    true,
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.UseStateForUnknown(),
				},
			},
			"service_type": schema.StringAttribute{
				Description: "The type of service (s3, gcs, azure, b2, or ssh).",
				Computed:    true,
			},
		},
	}
}

// Configure configures the resource with provider data.
func (r *AzureServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *AzureServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data AzureServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating Azure service", map[string]any{
		"account_name": data.AccountName.ValueString(),
		"container":    data.Container.ValueString(),
	})

	// Read credentials from files if specified
	accountKey, err := common.ReadCredential(
		data.AccountKey.ValueString(),
		data.AccountKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read account key", err.Error())
		return
	}

	connectionString, err := common.ReadCredential(
		data.ConnectionString.ValueString(),
		data.ConnectionStringFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read connection string", err.Error())
		return
	}

	sasToken, err := common.ReadCredential(
		data.SASToken.ValueString(),
		data.SASTokenFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAS token", err.Error())
		return
	}

	// Create and connect the backend
	backend := azure.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"account_name":      data.AccountName.ValueString(),
			"container":         data.Container.ValueString(),
			"account_key":       accountKey,
			"connection_string": connectionString,
			"sas_token":         sasToken,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to Azure", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("azure:%s:%s", data.AccountName.ValueString(), data.Container.ValueString())

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register Azure service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.ServiceType = types.StringValue("azure")

	tflog.Info(ctx, "Created Azure service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *AzureServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data AzureServiceResourceModel

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
func (r *AzureServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data AzureServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Read credentials from files if specified
	accountKey, err := common.ReadCredential(
		data.AccountKey.ValueString(),
		data.AccountKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read account key", err.Error())
		return
	}

	connectionString, err := common.ReadCredential(
		data.ConnectionString.ValueString(),
		data.ConnectionStringFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read connection string", err.Error())
		return
	}

	sasToken, err := common.ReadCredential(
		data.SASToken.ValueString(),
		data.SASTokenFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read SAS token", err.Error())
		return
	}

	backend := azure.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"account_name":      data.AccountName.ValueString(),
			"container":         data.Container.ValueString(),
			"account_key":       accountKey,
			"connection_string": connectionString,
			"sas_token":         sasToken,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to Azure", err.Error())
		return
	}

	// Register new backend first to avoid window with no registered backend
	serviceID := data.ID.ValueString()
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update Azure service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *AzureServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data AzureServiceResourceModel

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

	tflog.Info(ctx, "Deleted Azure service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *AzureServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
