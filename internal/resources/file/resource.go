// SPDX-License-Identifier: MIT

// Package file implements the filemanager_file resource.
package file

import (
	"context"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"os"
	"os/user"
	"strconv"
	"strings"

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

	"github.com/ebogdum/filemanager/internal/acid"
	"github.com/ebogdum/filemanager/internal/common"
	"github.com/ebogdum/filemanager/internal/plugin"
)

// Ensure provider defined types fully satisfy framework interfaces.
var (
	_ resource.Resource                = &FileResource{}
	_ resource.ResourceWithImportState = &FileResource{}
)

// NewFileResource creates a new file resource.
func NewFileResource() resource.Resource {
	return &FileResource{}
}

// FileResource defines the resource implementation.
type FileResource struct {
	config *common.ProviderConfig
}

// FileResourceModel describes the resource data model.
type FileResourceModel struct {
	ID                  types.String `tfsdk:"id"`
	Path                types.String `tfsdk:"path"`
	Service             types.String `tfsdk:"service"`
	Content             types.String `tfsdk:"content"`
	ContentBase64       types.String `tfsdk:"content_base64"`
	Source              types.String `tfsdk:"source"`
	FilePermission      types.String `tfsdk:"file_permission"`
	DirectoryPermission types.String `tfsdk:"directory_permission"`
	UID                 types.Int64  `tfsdk:"uid"`
	GID                 types.Int64  `tfsdk:"gid"`
	Owner               types.String `tfsdk:"owner"`
	Group               types.String `tfsdk:"group"`
	CreateParentDirs    types.Bool   `tfsdk:"create_parent_dirs"`
	Force               types.Bool   `tfsdk:"force"`
	Backup              types.Bool   `tfsdk:"backup"`
	BackupRetention     types.Int64  `tfsdk:"backup_retention"`
	Encoding            types.String `tfsdk:"encoding"`
	Newline             types.String `tfsdk:"newline"`
	AtomicWrite         types.Bool   `tfsdk:"atomic_write"`
	VerifyChecksum      types.Bool   `tfsdk:"verify_checksum"`

	// Computed attributes
	Size         types.Int64  `tfsdk:"size"`
	MD5          types.String `tfsdk:"md5"`
	SHA256       types.String `tfsdk:"sha256"`
	SHA512       types.String `tfsdk:"sha512"`
	Directory    types.String `tfsdk:"directory"`
	Filename     types.String `tfsdk:"filename"`
	Extension    types.String `tfsdk:"extension"`
	AbsolutePath types.String `tfsdk:"absolute_path"`
}

// Metadata returns the resource type name.
func (r *FileResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_file"
}

// Schema defines the schema for the resource.
func (r *FileResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a file with ACID guarantees.",
		MarkdownDescription: `
Manages a file on the filesystem or remote backend with ACID guarantees.

## Content Sources

You can specify file content using one of three methods (mutually exclusive):

- ` + "`content`" + ` - Plain text content
- ` + "`content_base64`" + ` - Base64-encoded content (for binary files)
- ` + "`source`" + ` - Path to a local file to copy
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
				Description: "Service to use for file operations. Defaults to the local filesystem.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("local"),
			},
			"content": schema.StringAttribute{
				Description: "The content of the file as a string. Conflicts with content_base64 and source.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("content_base64"), path.MatchRoot("source")),
				},
			},
			"content_base64": schema.StringAttribute{
				Description: "The base64-encoded content of the file. Conflicts with content and source.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("content"), path.MatchRoot("source")),
				},
			},
			"source": schema.StringAttribute{
				Description: "Path to a local file to use as the source. Conflicts with content and content_base64.",
				Optional:    true,
				Validators: []validator.String{
					stringvalidator.ConflictsWith(path.MatchRoot("content"), path.MatchRoot("content_base64")),
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
			"force": schema.BoolAttribute{
				Description: "Force overwrite if file already exists.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"backup": schema.BoolAttribute{
				Description: "Create a backup before modifying the file.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"backup_retention": schema.Int64Attribute{
				Description: "Number of backup copies to retain.",
				Optional:    true,
			},
			"encoding": schema.StringAttribute{
				Description: "Character encoding for the file content. Options: utf-8, utf-16, latin1, base64.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("utf-8"),
			},
			"newline": schema.StringAttribute{
				Description: "Newline style for the file. Options: lf, crlf, cr.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("lf"),
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
			},
			"sha512": schema.StringAttribute{
				Description: "SHA-512 checksum of the file content.",
				Computed:    true,
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
func (r *FileResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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
func (r *FileResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Creating file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Get content to write
	content, err := r.getContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get content", err.Error())
		return
	}

	// Convert newlines if needed
	content = r.convertNewlines(content, data.Newline.ValueString())

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

	if exists && !data.Force.ValueBool() {
		resp.Diagnostics.AddError(
			"File already exists",
			fmt.Sprintf("File %s already exists. Use force = true to overwrite.", data.Path.ValueString()),
		)
		return
	}

	// Write file
	writeOpts := plugin.WriteOptions{
		Mode:             common.ParseFileMode(data.FilePermission.ValueString()),
		DirMode:          common.ParseDirMode(data.DirectoryPermission.ValueString()),
		CreateDirs:       data.CreateParentDirs.ValueBool(),
		Overwrite:        data.Force.ValueBool(),
		Atomic:           data.AtomicWrite.ValueBool(),
		VerifyAfterWrite: data.VerifyChecksum.ValueBool(),
	}

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	// Set ownership if specified
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	// Compute checksums
	r.computeChecksums(&data, content)

	// Set ID and size
	data.ID = data.Path
	data.Size = types.Int64Value(int64(len(content)))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	tflog.Debug(ctx, "File created successfully", map[string]any{
		"path": data.Path.ValueString(),
		"size": len(content),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Read refreshes the Terraform state with the latest data.
func (r *FileResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data FileResourceModel

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

	// Read file content
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

	// Update state with file info
	info, err := backend.Stat(ctx, data.Path.ValueString())
	if err == nil {
		data.Size = types.Int64Value(info.Size)
	}

	// Compute checksums
	r.computeChecksums(&data, content)

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	// Update content if it was originally provided as plain text
	if !data.Content.IsNull() {
		data.Content = types.StringValue(string(content))
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Update updates the resource and sets the updated Terraform state.
func (r *FileResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Updating file", map[string]any{
		"path": data.Path.ValueString(),
	})

	// Get content to write
	content, err := r.getContent(ctx, &data)
	if err != nil {
		resp.Diagnostics.AddError("Failed to get content", err.Error())
		return
	}

	// Convert newlines if needed
	content = r.convertNewlines(content, data.Newline.ValueString())

	// Get backend
	backend, err := r.getBackend(ctx, data.Service.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Failed to get backend", err.Error())
		return
	}

	// Create backup if enabled (only for local backend)
	if data.Backup.ValueBool() {
		// Check if we're using local backend
		if data.Service.IsNull() || data.Service.ValueString() == "" || data.Service.ValueString() == "local" {
			// Check if file exists before creating backup
			exists, err := backend.Exists(ctx, data.Path.ValueString())
			if err != nil {
				resp.Diagnostics.AddWarning("Failed to check file existence for backup", err.Error())
			} else if exists {
				backupMgr := acid.NewFileBackup()
				backupOpts := acid.BackupOptions{
					IncludeTimestamp: true,
					MaxBackups:       int(data.BackupRetention.ValueInt64()),
				}

				backupPath, err := backupMgr.Create(ctx, data.Path.ValueString(), backupOpts)
				if err != nil {
					resp.Diagnostics.AddWarning("Failed to create backup", err.Error())
				} else {
					tflog.Debug(ctx, "Backup created", map[string]any{
						"original_path": data.Path.ValueString(),
						"backup_path":   backupPath,
					})
				}
			}
		} else {
			tflog.Debug(ctx, "Backup not supported for non-local services", map[string]any{
				"service": data.Service.ValueString(),
			})
		}
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

	if err := backend.Write(ctx, data.Path.ValueString(), strings.NewReader(string(content)), writeOpts); err != nil {
		resp.Diagnostics.AddError("Failed to write file", err.Error())
		return
	}

	// Set ownership if specified
	if err := r.setOwnership(ctx, backend, data.Path.ValueString(), &data); err != nil {
		resp.Diagnostics.AddError("Failed to set ownership", err.Error())
		return
	}

	// Compute checksums
	r.computeChecksums(&data, content)

	// Update size
	data.Size = types.Int64Value(int64(len(content)))

	// Compute path outputs
	pathOutputs := common.ComputePathOutputs(data.Path.ValueString())
	data.Directory = pathOutputs.Directory
	data.Filename = pathOutputs.Filename
	data.Extension = pathOutputs.Extension
	data.AbsolutePath = pathOutputs.AbsolutePath

	tflog.Debug(ctx, "File updated successfully", map[string]any{
		"path": data.Path.ValueString(),
		"size": len(content),
	})

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

// Delete deletes the resource.
func (r *FileResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data FileResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "Deleting file", map[string]any{
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
		// Ignore not found errors during delete
		if err != plugin.ErrPathNotFound {
			resp.Diagnostics.AddError("Failed to delete file", err.Error())
			return
		}
	}

	tflog.Debug(ctx, "File deleted successfully", map[string]any{
		"path": data.Path.ValueString(),
	})
}

// ImportState imports an existing resource.
func (r *FileResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("path"), req, resp)
}

// getContent retrieves the content to write from the appropriate source.
func (r *FileResource) getContent(ctx context.Context, data *FileResourceModel) ([]byte, error) {
	// Check content sources in priority order
	if !data.Content.IsNull() && data.Content.ValueString() != "" {
		return []byte(data.Content.ValueString()), nil
	}

	if !data.ContentBase64.IsNull() && data.ContentBase64.ValueString() != "" {
		decoded, err := base64.StdEncoding.DecodeString(data.ContentBase64.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to decode base64 content: %w", err)
		}
		return decoded, nil
	}

	if !data.Source.IsNull() && data.Source.ValueString() != "" {
		content, err := os.ReadFile(data.Source.ValueString())
		if err != nil {
			return nil, fmt.Errorf("failed to read source file: %w", err)
		}
		return content, nil
	}

	// No content specified - create empty file
	return []byte{}, nil
}

// convertNewlines converts newlines to the specified style.
func (r *FileResource) convertNewlines(content []byte, style string) []byte {
	if style == "" || style == "lf" {
		return content
	}

	s := string(content)
	// Normalize to LF first
	s = strings.ReplaceAll(s, "\r\n", "\n")
	s = strings.ReplaceAll(s, "\r", "\n")

	switch style {
	case "crlf":
		s = strings.ReplaceAll(s, "\n", "\r\n")
	case "cr":
		s = strings.ReplaceAll(s, "\n", "\r")
	}

	return []byte(s)
}

// getBackend returns the appropriate backend for the resource.
func (r *FileResource) getBackend(ctx context.Context, backendName string) (plugin.Backend, error) {
	if backendName == "" || backendName == "local" {
		return r.config.LocalBackend, nil
	}

	return r.config.Registry.Backends.GetAlias(backendName)
}

// setOwnership sets file ownership if specified.
func (r *FileResource) setOwnership(ctx context.Context, backend plugin.Backend, path string, data *FileResourceModel) error {
	uid := -1
	gid := -1

	// Numeric UID takes precedence
	if !data.UID.IsNull() {
		uid = int(data.UID.ValueInt64())
	} else if !data.Owner.IsNull() && data.Owner.ValueString() != "" {
		// Lookup by name
		u, err := user.Lookup(data.Owner.ValueString())
		if err != nil {
			return fmt.Errorf("failed to lookup user %s: %w", data.Owner.ValueString(), err)
		}
		uidInt, _ := strconv.Atoi(u.Uid)
		uid = uidInt
	}

	// Numeric GID takes precedence
	if !data.GID.IsNull() {
		gid = int(data.GID.ValueInt64())
	} else if !data.Group.IsNull() && data.Group.ValueString() != "" {
		// Lookup by name
		g, err := user.LookupGroup(data.Group.ValueString())
		if err != nil {
			return fmt.Errorf("failed to lookup group %s: %w", data.Group.ValueString(), err)
		}
		gidInt, _ := strconv.Atoi(g.Gid)
		gid = gidInt
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
func (r *FileResource) computeChecksums(data *FileResourceModel, content []byte) {
	data.MD5 = types.StringNull()

	sha256sum := sha256.Sum256(content)
	data.SHA256 = types.StringValue(hex.EncodeToString(sha256sum[:]))

	sha512sum := sha512.Sum512(content)
	data.SHA512 = types.StringValue(hex.EncodeToString(sha512sum[:]))
}
