// SPDX-License-Identifier: MIT

// Package b2_service implements the filemanager_b2_service resource.
package b2_service

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

	"github.com/ebogdum/filemanager/internal/backends/b2"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &B2ServiceResource{}
	_ resource.ResourceWithImportState = &B2ServiceResource{}
)

// NewB2ServiceResource creates a new B2 service resource.
func NewB2ServiceResource() resource.Resource {
	return &B2ServiceResource{}
}

// B2ServiceResource defines the resource implementation.
type B2ServiceResource struct {
	config *common.ProviderConfig
}

// B2ServiceResourceModel describes the resource data model.
type B2ServiceResourceModel struct {
	// Inputs
	Bucket               types.String `tfsdk:"bucket"`
	BasePath             types.String `tfsdk:"base_path"`
	ApplicationKeyID     types.String `tfsdk:"application_key_id"`
	ApplicationKeyIDFile types.String `tfsdk:"application_key_id_file"`
	ApplicationKey       types.String `tfsdk:"application_key"`
	ApplicationKeyFile   types.String `tfsdk:"application_key_file"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *B2ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_b2_service"
}

// Schema defines the schema for the resource.
func (r *B2ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures a Backblaze B2 Cloud Storage service for use with other filemanager resources.",
		MarkdownDescription: "Configures a Backblaze B2 Cloud Storage service for use with other filemanager resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: b2:bucket).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Description: "B2 bucket name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations within the bucket.",
				Optional:    true,
			},
			"application_key_id": schema.StringAttribute{
				Description: "B2 application key ID.",
				Optional:    true,
				Sensitive:   true,
			},
			"application_key_id_file": schema.StringAttribute{
				Description: "Path to file containing B2 application key ID.",
				Optional:    true,
			},
			"application_key": schema.StringAttribute{
				Description: "B2 application key.",
				Optional:    true,
				Sensitive:   true,
			},
			"application_key_file": schema.StringAttribute{
				Description: "Path to file containing B2 application key.",
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
func (r *B2ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *B2ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data B2ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating B2 service", map[string]any{
		"bucket": data.Bucket.ValueString(),
	})

	// Read credentials from files if specified
	applicationKeyID, err := common.ReadCredential(
		data.ApplicationKeyID.ValueString(),
		data.ApplicationKeyIDFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application key ID", err.Error())
		return
	}

	applicationKey, err := common.ReadCredential(
		data.ApplicationKey.ValueString(),
		data.ApplicationKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application key", err.Error())
		return
	}

	// Create and connect the backend
	backend := b2.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":             data.Bucket.ValueString(),
			"application_key_id": applicationKeyID,
			"application_key":    applicationKey,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to B2", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("b2:%s", data.Bucket.ValueString())

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register B2 service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.ServiceType = types.StringValue("b2")

	tflog.Info(ctx, "Created B2 service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *B2ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data B2ServiceResourceModel

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
func (r *B2ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data B2ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	applicationKeyID, err := common.ReadCredential(
		data.ApplicationKeyID.ValueString(),
		data.ApplicationKeyIDFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application key ID", err.Error())
		return
	}

	applicationKey, err := common.ReadCredential(
		data.ApplicationKey.ValueString(),
		data.ApplicationKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read application key", err.Error())
		return
	}

	backend := b2.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":             data.Bucket.ValueString(),
			"application_key_id": applicationKeyID,
			"application_key":    applicationKey,
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to B2", err.Error())
		return
	}

	serviceID := data.ID.ValueString()
	r.config.Registry.Backends.RemoveAlias(serviceID)
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update B2 service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)
	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *B2ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data B2ServiceResourceModel

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

	tflog.Info(ctx, "Deleted B2 service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *B2ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
