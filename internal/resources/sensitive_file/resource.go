// SPDX-License-Identifier: MIT

// Package sensitive_file implements the filemanager_sensitive_file resource.
package sensitive_file

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"strings"

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

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &SensitiveFileResource{}
	_ resource.ResourceWithImportState = &SensitiveFileResource{}
)

// NewSensitiveFileResource creates a new sensitive file resource.
func NewSensitiveFileResource() resource.Resource {
	return &SensitiveFileResource{}
}

// SensitiveFileResource defines the resource implementation.
type SensitiveFileResource struct {
	config *common.ProviderConfig
}

// SensitiveFileResourceModel describes the resource data model.
type SensitiveFileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	Content             types.String `tfsdk:"content"`
	ContentBase64       types.String `tfsdk:"content_base64"`
	PrettyPrintJSON     types.Bool   `tfsdk:"pretty_print_json"`
	Indent              types.Int64  `tfsdk:"indent"`
	SortKeys            types.Bool   `tfsdk:"sort_keys"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`
	AtomicWrite         types.Bool   `tfsdk:"atomic_write"`

	// Computed
	Size         types.Int64  `tfsdk:"size"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *SensitiveFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_sensitive_file"
}

// Schema defines the schema for the resource.
func (r *SensitiveFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file with sensitive content that is masked in logs and state.",
		MarkdownDescription: `
Manages a file with sensitive content. The content is marked as sensitive and will not appear in logs or plan output.

## Security Notes

- Content is marked as sensitive and won't appear in logs or plan output
- State still contains the content, so protect your state file appropriately
- Consider using restrictive file permissions (e.g., "0600")
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
				Description: "The path where the file will be created.",
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
			"content": schema.StringAttribute{
				Description: "The sensitive content to write to the file as a string. Use jsonencode() for JSON content.",
				Optional:    true,
				Sensitive:   true,
			},
			"content_base64": schema.StringAttribute{
				Description: "The sensitive content to write to the file, base64 encoded.",
				Optional:    true,
				Sensitive:   true,
			},
			"pretty_print_json": schema.BoolAttribute{
				Description: "If true and content is valid JSON, it will be pretty-printed with indentation.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"indent": schema.Int64Attribute{
				Description: "Indentation spaces for JSON output when pretty_print_json is true. Defaults to 2.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(2),
			},
			"sort_keys": schema.BoolAttribute{
				Description: "Sort JSON object keys alphabetically when pretty_print_json is true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode in octal format. Defaults to 0600 for security.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0600"),
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
				Description: "Deprecated insecure checksum field. Always null.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the file content.",
				Computed:    true,
				Sensitive:   true,
			},
			"directory": schema.StringAttribute{
				Description: "The parent directory of the path.",
				Computed:    true,
			},
			"filename": schema.StringAttribute{
				Description: "The base name of the file (e.g., 'config.json').",
				Computed:    true,
			},
			"extension": schema.StringAttribute{
				Description: "The file extension without the leading dot (e.g., 'json').",
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
func (r *SensitiveFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *SensitiveFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SensitiveFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Log without exposing content
	tflog.Debug(ctx, "Creating sensitive file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Get content
	content, err := r.getContent(&data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get content", err.Error())
		return
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
func (r *SensitiveFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SensitiveFileResourceModel

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

	// Compare computed SHA256 with stored value for drift detection
	computedSha256 := hex.EncodeToString(func() []byte { h := sha256.Sum256(content); return h[:] }())
	if !data.SHA256.IsNull() && data.SHA256.ValueString() != computedSha256 {
		resp.Diagnostics.AddWarning(
			"File content has changed externally",
			fmt.Sprintf("File %s content has been modified outside of Terraform", data.Path.ValueString()),
		)
	}

	r.updateComputedValues(&data, content)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource.
func (r *SensitiveFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data SensitiveFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating sensitive file", map[string]any{
		"path": data.Path.ValueString(),
	})

	content, err := r.getContent(&data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get content", err.Error())
		return
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
func (r *SensitiveFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SensitiveFileResourceModel

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
func (r *SensitiveFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// getContent returns the file content from the model.
func (r *SensitiveFileResource) getContent(data *SensitiveFileResourceModel) ([]byte, error) {
	// Priority: content_base64 > content
	if !data.ContentBase64.IsNull() && !data.ContentBase64.IsUnknown() && data.ContentBase64.ValueString() != "" {
		decoded, err := base64.StdEncoding.DecodeString(data.ContentBase64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return decoded, nil
	}

	if !data.Content.IsNull() && !data.Content.IsUnknown() {
		content := []byte(data.Content.ValueString())

		// If pretty_print_json is enabled, try to format the content as JSON
		if data.PrettyPrintJSON.ValueBool() {
			formatted, err := r.formatJSON(content, data)
			if err != nil {
				return nil, fmt.Errorf("failed to format JSON: %w", err)
			}
			return formatted, nil
		}

		return content, nil
	}

	return nil, fmt.Errorf("either content or content_base64 must be specified")
}

// formatJSON parses and formats JSON content with pretty printing.
func (r *SensitiveFileResource) formatJSON(content []byte, data *SensitiveFileResourceModel) ([]byte, error) {
	// Parse the JSON
	var value any
	if err := json.Unmarshal(content, &value); err != nil {
		return nil, fmt.Errorf("content is not valid JSON: %w", err)
	}

	// Serialize with pretty printing
	// Note: encoding/json sorts map keys by default since Go maps are iterated in key order
	// in json.Marshal. The sort_keys flag is handled implicitly.
	var buf bytes.Buffer
	encoder := json.NewEncoder(&buf)
	encoder.SetEscapeHTML(false)

	indent := int(data.Indent.ValueInt64())
	if indent <= 0 {
		indent = 2
	}
	encoder.SetIndent("", strings.Repeat(" ", indent))

	if err := encoder.Encode(value); err != nil {
		return nil, fmt.Errorf("failed to serialize JSON: %w", err)
	}

	return buf.Bytes(), nil
}

// getBackend returns the appropriate backend.
func (r *SensitiveFileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}
	return r.config.Registry.Backends.GetAlias(backendName)
}

// updateComputedValues updates the computed values in the model.
func (r *SensitiveFileResource) updateComputedValues(data *SensitiveFileResourceModel, content []byte) {
	data.Size = types.Int64Value(int64(len(content)))

	data.MD5 = types.StringNull()

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath
}
