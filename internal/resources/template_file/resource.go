// SPDX-License-Identifier: MIT

// Package template_file implements the filemanager_template_file resource.
package template_file

import (
	"bytes"
	"context"
	"crypto/md5"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"os"
	"strconv"
	"strings"
	"text/template"

	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &TemplateFileResource{}
	_ resource.ResourceWithImportState = &TemplateFileResource{}
)

// NewTemplateFileResource creates a new template file resource.
func NewTemplateFileResource() resource.Resource {
	return &TemplateFileResource{}
}

// TemplateFileResource defines the resource implementation.
type TemplateFileResource struct {
	config *common.ProviderConfig
}

// TemplateFileResourceModel describes the resource data model.
type TemplateFileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	Template            types.String `tfsdk:"template"`
	TemplateFile        types.String `tfsdk:"template_file"`
	Source              types.String `tfsdk:"source"`
	Vars                types.Map    `tfsdk:"vars"`
	Engine              types.String `tfsdk:"engine"`
	LeftDelim           types.String `tfsdk:"left_delim"`
	RightDelim          types.String `tfsdk:"right_delim"`
	MissingKey          types.String `tfsdk:"missing_key"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	UID                 types.Int64  `tfsdk:"uid"`
	GID                 types.Int64  `tfsdk:"gid"`
	Owner               types.String `tfsdk:"owner"`
	Group               types.String `tfsdk:"group"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`
	AtomicWrite         types.Bool   `tfsdk:"atomic_write"`
	VerifyChecksum      types.Bool   `tfsdk:"verify_checksum"`

	// Computed attributes
	RenderedContent types.String `tfsdk:"rendered_content"`
	Size            types.Int64  `tfsdk:"size"`
	MD5             types.String `tfsdk:"md5"`
	SHA256          types.String `tfsdk:"sha256"`
	SHA512          types.String `tfsdk:"sha512"`
	Directory       types.String `tfsdk:"directory"`
	Filename        types.String `tfsdk:"filename"`
	Extension       types.String `tfsdk:"extension"`
	AbsolutePath    types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *TemplateFileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_template_file"
}

// Schema defines the schema for the resource.
func (r *TemplateFileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Renders a template with variables and writes the result to a file.",
		MarkdownDescription: `
Renders a template with variables and writes the result to a file.

## Template Syntax

This resource uses Go's ` + "`text/template`" + ` package. Common operations:

- ` + "`{{ .variable_name }}`" + ` - Insert a variable
- ` + "`{{ if .condition }}...{{ end }}`" + ` - Conditional
- ` + "`{{ range .list }}...{{ end }}`" + ` - Iterate over a list
- ` + "`{{ .var | upper }}`" + ` - Apply a function

## Built-in Functions

The following functions are available in templates:

- ` + "`upper`" + `, ` + "`lower`" + ` - Case conversion
- ` + "`title`" + ` - Title case
- ` + "`trim`" + `, ` + "`trimPrefix`" + `, ` + "`trimSuffix`" + ` - String trimming
- ` + "`replace`" + ` - String replacement
- ` + "`split`" + `, ` + "`join`" + ` - String splitting/joining
- ` + "`contains`" + `, ` + "`hasPrefix`" + `, ` + "`hasSuffix`" + ` - String matching
- ` + "`default`" + ` - Default value if empty
- ` + "`indent`" + ` - Indent text
- ` + "`env`" + ` - Read environment variable
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
				Description: "The path where the rendered file will be created.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"service": schema.StringAttribute{
				Description: "Service to use for file operations. Defaults to the local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"template": schema.StringAttribute{
				Description: "The template content as a string. Conflicts with source and template_file.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("source"), path.MatchRoot("template_file")),
					stringvalidator.AtLeastOneOf(path.MatchRoot("template"), path.MatchRoot("source"), path.MatchRoot("template_file")),
				},
			},
			"template_file": schema.StringAttribute{
				Description:         "Path to a template file to use. Conflicts with template and source. Deprecated: use source instead.",
				MarkdownDescription: "Path to a template file to use. Conflicts with `template` and `source`. **Deprecated:** use `source` instead.",
				Optional:            true,
				DeprecationMessage:  "Use 'source' instead.",
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("template"), path.MatchRoot("source")),
				},
			},
			"source": schema.StringAttribute{
				Description:         "Path to a template file to read. The template will be rendered with the provided variables. Conflicts with template.",
				MarkdownDescription: "Path to a template file to read. The template will be rendered with the provided variables. Conflicts with `template`.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("template"), path.MatchRoot("template_file")),
				},
			},
			"vars": schema.MapAttribute{
				Description: "Variables to substitute in the template.",
				Optional:    true,
				ElementType: types.StringType,
			},
			"engine": schema.StringAttribute{
				Description: "Template engine to use. Options: go (default), mustache.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("go"),
				Validators: []validator.String{
					stringvalidator.OneOf("go", "mustache"),
				},
			},
			"left_delim": schema.StringAttribute{
				Description: "Left delimiter for template actions. Default: \"{{\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("{{"),
			},
			"right_delim": schema.StringAttribute{
				Description: "Right delimiter for template actions. Default: \"}}\".",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("}}"),
			},
			"missing_key": schema.StringAttribute{
				Description: "Behavior when a map is indexed with a missing key. Options: invalid (error), zero (use zero value), error (return error). Default: error.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("error"),
				Validators: []validator.String{
					stringvalidator.OneOf("invalid", "zero", "error"),
				},
			},
			"file_permission": schema.StringAttribute{
				Description: "File permission mode in octal format (e.g., \"0644\").",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0644"),
			},
			"directory_permission": schema.StringAttribute{
				Description: "Directory permission mode for parent directories in octal format (e.g., \"0755\").",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("0755"),
			},
			"uid": schema.Int64Attribute{
				Description: "User ID for file ownership (Unix only).",
				Optional:    true,
			},
			"gid": schema.Int64Attribute{
				Description: "Group ID for file ownership (Unix only).",
				Optional:    true,
			},
			"owner": schema.StringAttribute{
				Description: "User name for file ownership. Alternative to uid.",
				Optional:    true,
			},
			"group": schema.StringAttribute{
				Description: "Group name for file ownership. Alternative to gid.",
				Optional:    true,
			},
			"create_parent_dirs": schema.BoolAttribute{
				Description: "Create parent directories if they don't exist.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"atomic_write": schema.BoolAttribute{
				Description: "Use atomic write operations (temp file + rename).",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"verify_checksum": schema.BoolAttribute{
				Description: "Verify file checksum after writing.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},

			// Computed outputs
			"rendered_content": schema.StringAttribute{
				Description: "The rendered template content.",
				Computed:    true,
			},
			"size": schema.Int64Attribute{
				Description: "Size of the rendered file in bytes.",
				Computed:    true,
			},
			"md5": schema.StringAttribute{
				Description: "MD5 checksum of the rendered content.",
				Computed:    true,
			},
			"sha256": schema.StringAttribute{
				Description: "SHA-256 checksum of the rendered content.",
				Computed:    true,
			},
			"sha512": schema.StringAttribute{
				Description: "SHA-512 checksum of the rendered content.",
				Computed:    true,
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
func (r *TemplateFileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *TemplateFileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data TemplateFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating template file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Render the template
	rendered, err := r.renderTemplate(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to render template", err.Error())
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
		Mode:             common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:          common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs:       data.CreateParentDirs.ValueBool(),
		Overwrite:        true,
		Atomic:           data.AtomicWrite.ValueBool(),
		VerifyAfterWrite: data.VerifyChecksum.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(rendered), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	// Set ownership if specified
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	// Set computed values
	content := []byte(rendered)
	r.computeChecksums(&data, content)
	data.ID = data.Path
	data.RenderedContent = types.StringValue(rendered)
	data.Size = types.Int64Value(int64(len(content)))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	tflog.Debug(ctx, "Template file created successfully", map[string]any{
		"path": data.Path.ValueString(),
		"size": len(content),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *TemplateFileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data TemplateFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Check if file exists
	exists, err := backend.Exists(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to check file existence", err.Error())
		return
	}

	if !exists {
		// File was deleted outside of Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	// Read file content and update checksums
	reader, err := backend.Read(ctx, data.Path.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to read file", err.Error())
		return
	}

	var buf bytes.Buffer
	if _, err := buf.ReadFrom(reader); err != nil {
		reader.Close()
		resp.Diagnostics.AddError("Failed to read file content", err.Error())
		return
	}
	reader.Close()

	content := buf.Bytes()

	// Update computed values
	r.computeChecksums(&data, content)
	data.Size = types.Int64Value(int64(len(content)))
	data.RenderedContent = types.StringValue(string(content))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *TemplateFileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data TemplateFileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating template file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Render the template
	rendered, err := r.renderTemplate(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to render template", err.Error())
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
		Mode:             common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:          common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs:       data.CreateParentDirs.ValueBool(),
		Overwrite:        true,
		Atomic:           data.AtomicWrite.ValueBool(),
		VerifyAfterWrite: data.VerifyChecksum.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(rendered), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	// Set ownership if specified
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	// Set computed values
	content := []byte(rendered)
	r.computeChecksums(&data, content)
	data.RenderedContent = types.StringValue(rendered)
	data.Size = types.Int64Value(int64(len(content)))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	tflog.Debug(ctx, "Template file updated successfully", map[string]any{
		"path": data.Path.ValueString(),
		"size": len(content),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *TemplateFileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data TemplateFileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting template file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Delete file
	if err := backend.Delete(ctx, data.Path.ValueString()); err != nil {
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete file", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "Template file deleted successfully", map[string]any{
		"path": data.Path.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *TemplateFileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// renderTemplate renders the template with the provided variables.
func (r *TemplateFileResource) renderTemplate(ctx context.Context, data *TemplateFileResourceModel) (string, error) {
	// Get template content
	var templateContent string
	if !data.Template.IsNull() && data.Template.ValueString() != "" {
		templateContent = data.Template.ValueString()
	} else if !data.Source.IsNull() && data.Source.ValueString() != "" {
		content, err := os.ReadFile(data.Source.ValueString())
		if err != nil {
			return "", fmt.Errorf("failed to read template source file %q: %w", data.Source.ValueString(), err)
		}
		templateContent = string(content)
	} else if !data.TemplateFile.IsNull() && data.TemplateFile.ValueString() != "" {
		// Deprecated: use source instead
		content, err := os.ReadFile(data.TemplateFile.ValueString())
		if err != nil {
			return "", fmt.Errorf("failed to read template file %q: %w", data.TemplateFile.ValueString(), err)
		}
		templateContent = string(content)
	} else {
		return "", fmt.Errorf("either template or source must be specified")
	}

	// Get variables
	vars := make(map[string]string)
	if !data.Vars.IsNull() {
		diags := data.Vars.ElementsAs(ctx, &vars, false)
		if diags.HasError() {
			return "", fmt.Errorf("failed to parse vars")
		}
	}

	// Render based on engine
	engine := data.Engine.ValueString()
	switch engine {
	case "go", "":
		return r.renderGoTemplate(templateContent, vars, data)
	case "mustache":
		return r.renderMustacheTemplate(templateContent, vars)
	default:
		return "", fmt.Errorf("unknown template engine: %s", engine)
	}
}

// renderGoTemplate renders a Go text/template.
func (r *TemplateFileResource) renderGoTemplate(templateContent string, vars map[string]string, data *TemplateFileResourceModel) (string, error) {
	// Create template with custom delimiters
	leftDelim := data.LeftDelim.ValueString()
	rightDelim := data.RightDelim.ValueString()

	tmpl := template.New("template").Delims(leftDelim, rightDelim)

	// Set missing key option
	missingKey := data.MissingKey.ValueString()
	switch missingKey {
	case "invalid":
		tmpl = tmpl.Option("missingkey=invalid")
	case "zero":
		tmpl = tmpl.Option("missingkey=zero")
	case "error":
		tmpl = tmpl.Option("missingkey=error")
	}

	// Add custom functions
	tmpl = tmpl.Funcs(templateFuncs())

	// Parse template
	tmpl, err := tmpl.Parse(templateContent)
	if err != nil {
		return "", fmt.Errorf("failed to parse template: %w", err)
	}

	// Execute template
	var buf bytes.Buffer
	if err := tmpl.Execute(&buf, vars); err != nil {
		return "", fmt.Errorf("failed to execute template: %w", err)
	}

	return buf.String(), nil
}

// renderMustacheTemplate renders a Mustache template.
func (r *TemplateFileResource) renderMustacheTemplate(templateContent string, vars map[string]string) (string, error) {
	// Simple mustache-like variable substitution
	// For full mustache support, we'd need a proper mustache library
	result := templateContent
	for key, value := range vars {
		// Replace {{{key}}} first (unescaped) - must be before {{key}} to avoid partial matches
		unescapedPlaceholder := "{{{" + key + "}}}"
		result = strings.ReplaceAll(result, unescapedPlaceholder, value)

		// Then replace {{key}} with value
		placeholder := "{{" + key + "}}"
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result, nil
}

// templateFuncs returns custom template functions.
func templateFuncs() template.FuncMap {
	return template.FuncMap{
		// String case functions
		"upper": strings.ToUpper,
		"lower": strings.ToLower,
		"title": strings.Title,

		// String trimming
		"trim": strings.TrimSpace,
		"trimPrefix": func(prefix, s string) string {
			return strings.TrimPrefix(s, prefix)
		},
		"trimSuffix": func(suffix, s string) string {
			return strings.TrimSuffix(s, suffix)
		},

		// String replacement
		"replace": func(old, new, s string) string {
			return strings.ReplaceAll(s, old, new)
		},

		// String splitting/joining
		"split": strings.Split,
		"join": func(sep string, parts []string) string {
			return strings.Join(parts, sep)
		},

		// String matching - args swapped for pipeline use: {{ .text | contains "substr" }}
		"contains": func(substr, s string) bool {
			return strings.Contains(s, substr)
		},
		"hasPrefix": func(prefix, s string) bool {
			return strings.HasPrefix(s, prefix)
		},
		"hasSuffix": func(suffix, s string) bool {
			return strings.HasSuffix(s, suffix)
		},

		// Default value
		"default": func(def, val string) string {
			if val == "" {
				return def
			}
			return val
		},

		// Indent text
		"indent": func(spaces int, s string) string {
			pad := strings.Repeat(" ", spaces)
			lines := strings.Split(s, "\n")
			for i, line := range lines {
				if line != "" {
					lines[i] = pad + line
				}
			}
			return strings.Join(lines, "\n")
		},

		// Environment variable
		"env": os.Getenv,

		// Quote string
		"quote": func(s string) string {
			return fmt.Sprintf("%q", s)
		},

		// Base64 encoding
		"base64": func(s string) string {
			return base64Encode(s)
		},

		// JSON encoding (simple)
		"toJSON": func(v any) string {
			return fmt.Sprintf("%v", v)
		},

		// Coalesce - return first non-empty value
		"coalesce": func(values ...string) string {
			for _, v := range values {
				if v != "" {
					return v
				}
			}
			return ""
		},

		// Ternary operator
		"ternary": func(trueVal, falseVal string, condition bool) string {
			if condition {
				return trueVal
			}
			return falseVal
		},

		// Boolean conversion
		"toBool": func(s string) bool {
			s = strings.ToLower(strings.TrimSpace(s))
			return s == "true" || s == "yes" || s == "1" || s == "on"
		},

		// Integer conversion
		"toInt": func(s string) int {
			i, _ := strconv.Atoi(s)
			return i
		},
	}
}

// base64Encode encodes a string to base64.
func base64Encode(s string) string {
	return base64.StdEncoding.EncodeToString([]byte(s))
}

// getBackend returns the appropriate backend for the resource.
func (r *TemplateFileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}

	return r.config.Registry.Backends.GetAlias(backendName)
}

// setOwnership sets file ownership if specified.
func (r *TemplateFileResource) setOwnership(ctx context.Context, backend plugin.Backend, path string, data *TemplateFileResourceModel) error {
	uid := -1
	gid := -1

	if !data.UID.IsNull() {
		uid = int(data.UID.ValueInt64())
	}

	if !data.GID.IsNull() {
		gid = int(data.GID.ValueInt64())
	}

	if uid >= 0 || gid >= 0 {
		if err := backend.Chown(ctx, path, uid, gid); err != nil {
			if err != plugin.ErrNotSupported {
				return err
			}
		}
	}

	return nil
}

// computeChecksums computes MD5, SHA256, and SHA512 checksums.
func (r *TemplateFileResource) computeChecksums(data *TemplateFileResourceModel, content []byte) {
	md5sum := md5.Sum(content)
	data.MD5 = types.StringValue(hex.EncodeToString(md5sum[:]))

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	sha512sum := sha512.Sum512(content)
	data.SHA512 = types.StringValue(hex.EncodeToString(sha512sum[:]))
}
