// SPDX-License-Identifier: MIT

// Package gcs_service implements the filemanager_gcs_service resource.
package gcs_service

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/gcs"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &GCSServiceResource{}
	_ resource.ResourceWithImportState = &GCSServiceResource{}
)

// NewGCSServiceResource creates a new GCS service resource.
func NewGCSServiceResource() resource.Resource {
	return &GCSServiceResource{}
}

// GCSServiceResource defines the resource implementation.
type GCSServiceResource struct {
	config *common.ProviderConfig
}

// GCSServiceResourceModel describes the resource data model.
type GCSServiceResourceModel struct {
	// Inputs
	Bucket              types.String `tfsdk:"bucket"`
	Project             types.String `tfsdk:"project"`
	BasePath            types.String `tfsdk:"base_path"`
	CredentialsFile     types.String `tfsdk:"credentials_file"`
	CredentialsJSON     types.String `tfsdk:"credentials_json"`
	CredentialsJSONFile types.String `tfsdk:"credentials_json_file"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *GCSServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_gcs_service"
}

// Schema defines the schema for the resource.
func (r *GCSServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures a Google Cloud Storage service for use with other filemanager resources.",
		MarkdownDescription: "Configures a Google Cloud Storage service for use with other filemanager resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: gcs:bucket).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Description: "GCS bucket name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"project": schema.StringAttribute{
				Description: "GCP project ID. Used for bucket operations that require a project.",
				Optional:    true,
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations within the bucket.",
				Optional:    true,
			},
			"credentials_file": schema.StringAttribute{
				Description: "Path to GCP service account credentials JSON file.",
				Optional:    true,
			},
			"credentials_json": schema.StringAttribute{
				Description: "GCP service account credentials as JSON string.",
				Optional:    true,
				Sensitive:   true,
			},
			"credentials_json_file": schema.StringAttribute{
				Description: "Path to file containing GCP service account credentials JSON.",
				Optional:    true,
			},
			"connected": schema.BoolAttribute{
				Description: "Whether the service is currently connected.",
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
func (r *GCSServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *GCSServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data GCSServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating GCS service", map[string]any{
		"bucket": data.Bucket.ValueString(),
	})

	// Read credentials from files if specified
	credentialsJSON, err := common.ReadCredential(
		data.CredentialsJSON.ValueString(),
		data.CredentialsJSONFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read credentials JSON", err.Error())
		return
	}

	// Expand credentials file path
	credentialsFile, err := common.ExpandPath(data.CredentialsFile.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to expand credentials file path", err.Error())
		return
	}

	// Create and connect the backend
	backend := gcs.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":           data.Bucket.ValueString(),
			"project":          data.Project.ValueString(),
			"credentials_file": credentialsFile,
			"credentials_json": credentialsJSON,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to GCS", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("gcs:%s", data.Bucket.ValueString())

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register GCS service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.ServiceType = types.StringValue("gcs")

	tflog.Info(ctx, "Created GCS service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *GCSServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data GCSServiceResourceModel

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
func (r *GCSServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data GCSServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	credentialsJSON, err := common.ReadCredential(
		data.CredentialsJSON.ValueString(),
		data.CredentialsJSONFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read credentials JSON", err.Error())
		return
	}

	credentialsFile, err := common.ExpandPath(data.CredentialsFile.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to expand credentials file path", err.Error())
		return
	}

	backend := gcs.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":           data.Bucket.ValueString(),
			"project":          data.Project.ValueString(),
			"credentials_file": credentialsFile,
			"credentials_json": credentialsJSON,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to GCS", err.Error())
		return
	}

	serviceID := data.ID.ValueString()
	r.config.Registry.Backends.RemoveAlias(serviceID)
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update GCS service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *GCSServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data GCSServiceResourceModel

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

	tflog.Info(ctx, "Deleted GCS service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *GCSServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
