// SPDX-License-Identifier: MIT

// Package s3_service implements the filemanager_s3_service resource.
package s3_service

import (
	"context"
	"fmt"

	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/s3"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &S3ServiceResource{}
	_ resource.ResourceWithImportState = &S3ServiceResource{}
)

// NewS3ServiceResource creates a new S3 service resource.
func NewS3ServiceResource() resource.Resource {
	return &S3ServiceResource{}
}

// S3ServiceResource defines the resource implementation.
type S3ServiceResource struct {
	config *common.ProviderConfig
}

// S3ServiceResourceModel describes the resource data model.
type S3ServiceResourceModel struct {
	// Inputs
	Bucket        types.String `tfsdk:"bucket"`
	Region        types.String `tfsdk:"region"`
	Endpoint      types.String `tfsdk:"endpoint"`
	BasePath      types.String `tfsdk:"base_path"`
	AccessKey     types.String `tfsdk:"access_key"`
	AccessKeyFile types.String `tfsdk:"access_key_file"`
	SecretKey     types.String `tfsdk:"secret_key"`
	SecretKeyFile types.String `tfsdk:"secret_key_file"`
	SessionToken  types.String `tfsdk:"session_token"`

	// Outputs (Computed)
	ID          types.String `tfsdk:"id"`
	Connected   types.Bool   `tfsdk:"connected"`
	ServiceType types.String `tfsdk:"service_type"`
}

// Metadata returns the resource type name.
func (r *S3ServiceResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_service"
}

// Schema defines the schema for the resource.
func (r *S3ServiceResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:         "Configures an S3-compatible storage service for use with other filemanager resources.",
		MarkdownDescription: "Configures an S3-compatible storage service for use with other filemanager resources.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Unique identifier for the service (format: s3:region:bucket).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"bucket": schema.StringAttribute{
				Description: "The S3 bucket name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"region": schema.StringAttribute{
				Description: "AWS region for the bucket. Defaults to us-east-1.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("us-east-1"),
			},
			"endpoint": schema.StringAttribute{
				Description: "Custom endpoint URL for S3-compatible services (e.g., MinIO, DigitalOcean Spaces).",
				Optional:    true,
			},
			"base_path": schema.StringAttribute{
				Description: "Base path prefix for all operations within the bucket.",
				Optional:    true,
			},
			"access_key": schema.StringAttribute{
				Description: "AWS access key ID. If not provided, uses default AWS credential chain.",
				Optional:    true,
				Sensitive:   true,
			},
			"access_key_file": schema.StringAttribute{
				Description: "Path to file containing AWS access key ID. Alternative to access_key.",
				Optional:    true,
			},
			"secret_key": schema.StringAttribute{
				Description: "AWS secret access key. If not provided, uses default AWS credential chain.",
				Optional:    true,
				Sensitive:   true,
			},
			"secret_key_file": schema.StringAttribute{
				Description: "Path to file containing AWS secret access key. Alternative to secret_key.",
				Optional:    true,
			},
			"session_token": schema.StringAttribute{
				Description: "AWS session token for temporary credentials.",
				Optional:    true,
				Sensitive:   true,
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
func (r *S3ServiceResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *S3ServiceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data S3ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating S3 service", map[string]any{
		"bucket": data.Bucket.ValueString(),
		"region": data.Region.ValueString(),
	})

	// Read credentials from files if specified
	accessKey, err := common.ReadCredential(
		data.AccessKey.ValueString(),
		data.AccessKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read access key", err.Error())
		return
	}

	secretKey, err := common.ReadCredential(
		data.SecretKey.ValueString(),
		data.SecretKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read secret key", err.Error())
		return
	}

	// Create and connect the backend
	backend := s3.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":        data.Bucket.ValueString(),
			"region":        data.Region.ValueString(),
			"endpoint":      data.Endpoint.ValueString(),
			"access_key":    accessKey,
			"secret_key":    secretKey,
			"session_token": data.SessionToken.ValueString(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to S3", err.Error())
		return
	}

	// Generate service ID
	serviceID := fmt.Sprintf("s3:%s:%s", data.Region.ValueString(), data.Bucket.ValueString())

	// Register with the registry
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to register S3 service", err.Error())
		return
	}

	// Set computed values
	data.ID = types.StringValue(serviceID)
	data.Connected = types.BoolValue(true)
	data.ServiceType = types.StringValue("s3")

	tflog.Info(ctx, "Created S3 service", map[string]any{
		"id": serviceID,
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *S3ServiceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data S3ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Check if service is still registered
	serviceID := data.ID.ValueString()
	backend, err := r.config.Registry.Backends.GetAlias(serviceID)
	if err != nil {
		// Service no longer registered, mark as disconnected
		data.Connected = types.BoolValue(false)
	} else {
		// Ping to verify connection
		if err := backend.Ping(ctx); err != nil {
			data.Connected = types.BoolValue(false)
		} else {
			data.Connected = types.BoolValue(true)
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *S3ServiceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data S3ServiceResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating S3 service", map[string]any{
		"bucket": data.Bucket.ValueString(),
	})

	// Read credentials from files if specified
	accessKey, err := common.ReadCredential(
		data.AccessKey.ValueString(),
		data.AccessKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read access key", err.Error())
		return
	}

	secretKey, err := common.ReadCredential(
		data.SecretKey.ValueString(),
		data.SecretKeyFile.ValueString(),
	)
	if err != nil {
		resp.Diagnostics.AddError("Failed to read secret key", err.Error())
		return
	}

	// Create and connect new backend
	backend := s3.New()
	backendConfig := plugin.BackendConfig{
		BasePath: data.BasePath.ValueString(),
		Extra: map[string]any{
			"bucket":        data.Bucket.ValueString(),
			"region":        data.Region.ValueString(),
			"endpoint":      data.Endpoint.ValueString(),
			"access_key":    accessKey,
			"secret_key":    secretKey,
			"session_token": data.SessionToken.ValueString(),
		},
	}

	if err := backend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddError("Failed to connect to S3", err.Error())
		return
	}

	// Close old backend and replace alias
	serviceID := data.ID.ValueString()
	oldBackend, oldErr := r.config.Registry.Backends.GetAlias(serviceID)
	if nil == oldErr {
		_ = oldBackend.Close()
	}
	r.config.Registry.Backends.RemoveAlias(serviceID)
	if err := r.config.Registry.Backends.SetAlias(serviceID, backend); err != nil {
		resp.Diagnostics.AddError("Failed to update S3 service registration", err.Error())
		return
	}

	data.Connected = types.BoolValue(true)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource and removes the Terraform state.
func (r *S3ServiceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data S3ServiceResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	serviceID := data.ID.ValueString()

	tflog.Debug(ctx, "Deleting S3 service", map[string]any{
		"id": serviceID,
	})

	// Close the backend connection
	backend, err := r.config.Registry.Backends.GetAlias(serviceID)
	if err == nil {
		_ = backend.Close()
	}

	// Remove from registry
	r.config.Registry.Backends.RemoveAlias(serviceID)

	tflog.Info(ctx, "Deleted S3 service", map[string]any{
		"id": serviceID,
	})
}

// ImportState imports an existing resource.
func (r *S3ServiceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
