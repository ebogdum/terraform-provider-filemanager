// SPDX-License-Identifier: MIT

// Package provider implements the Terraform provider for file management.
package provider

import (
	"context"
	"time"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/function"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/provider"
	"github.com/hashicorp/terraform-plugin-framework/provider/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"

	"github.com/ebogdum/filemanager/internal/backends/ftp"
	"github.com/ebogdum/filemanager/internal/backends/local"
	"github.com/ebogdum/filemanager/internal/backends/ssh"
	"github.com/ebogdum/filemanager/internal/backends/swift"
	"github.com/ebogdum/filemanager/internal/common"
	checksumDataSource "github.com/ebogdum/filemanager/internal/datasources/checksum"
	compareDataSource "github.com/ebogdum/filemanager/internal/datasources/compare"
	directoryDataSource "github.com/ebogdum/filemanager/internal/datasources/directory"
	envDataSource "github.com/ebogdum/filemanager/internal/datasources/env"
	environmentDataSource "github.com/ebogdum/filemanager/internal/datasources/environment"
	fileDataSource "github.com/ebogdum/filemanager/internal/datasources/file"
	filesDataSource "github.com/ebogdum/filemanager/internal/datasources/files"
	groupsDataSource "github.com/ebogdum/filemanager/internal/datasources/groups"
	hclDataSource "github.com/ebogdum/filemanager/internal/datasources/hcl"
	iniDataSource "github.com/ebogdum/filemanager/internal/datasources/ini"
	jsonDataSource "github.com/ebogdum/filemanager/internal/datasources/json"
	statDataSource "github.com/ebogdum/filemanager/internal/datasources/stat"
	tomlDataSource "github.com/ebogdum/filemanager/internal/datasources/toml"
	usersDataSource "github.com/ebogdum/filemanager/internal/datasources/users"
	validateDataSource "github.com/ebogdum/filemanager/internal/datasources/validate"
	xmlDataSource "github.com/ebogdum/filemanager/internal/datasources/xml"
	yamlDataSource "github.com/ebogdum/filemanager/internal/datasources/yaml"
	"github.com/ebogdum/filemanager/internal/functions"
	"github.com/ebogdum/filemanager/internal/plugin"
	appConfigResource "github.com/ebogdum/filemanager/internal/resources/app_config"
	archiveResource "github.com/ebogdum/filemanager/internal/resources/archive"
	azureOperationResource "github.com/ebogdum/filemanager/internal/resources/azure_operation"
	azureServiceResource "github.com/ebogdum/filemanager/internal/resources/azure_service"
	b2OperationResource "github.com/ebogdum/filemanager/internal/resources/b2_operation"
	b2ServiceResource "github.com/ebogdum/filemanager/internal/resources/b2_service"
	copyResource "github.com/ebogdum/filemanager/internal/resources/copy"
	directoryResource "github.com/ebogdum/filemanager/internal/resources/directory"
	downloadResource "github.com/ebogdum/filemanager/internal/resources/download"
	envFileResource "github.com/ebogdum/filemanager/internal/resources/env_file"
	fileResource "github.com/ebogdum/filemanager/internal/resources/file"
	ftpOperationResource "github.com/ebogdum/filemanager/internal/resources/ftp_operation"
	ftpServiceResource "github.com/ebogdum/filemanager/internal/resources/ftp_service"
	gcsOperationResource "github.com/ebogdum/filemanager/internal/resources/gcs_operation"
	gcsServiceResource "github.com/ebogdum/filemanager/internal/resources/gcs_service"
	hclFileResource "github.com/ebogdum/filemanager/internal/resources/hcl_file"
	iniFileResource "github.com/ebogdum/filemanager/internal/resources/ini_file"
	jsonFileResource "github.com/ebogdum/filemanager/internal/resources/json_file"
	s3OperationResource "github.com/ebogdum/filemanager/internal/resources/s3_operation"
	s3ServiceResource "github.com/ebogdum/filemanager/internal/resources/s3_service"
	sensitiveFileResource "github.com/ebogdum/filemanager/internal/resources/sensitive_file"
	sshServiceResource "github.com/ebogdum/filemanager/internal/resources/ssh_service"
	swiftOperationResource "github.com/ebogdum/filemanager/internal/resources/swift_operation"
	swiftServiceResource "github.com/ebogdum/filemanager/internal/resources/swift_service"
	symlinkResource "github.com/ebogdum/filemanager/internal/resources/symlink"
	syncResource "github.com/ebogdum/filemanager/internal/resources/sync"
	templateFileResource "github.com/ebogdum/filemanager/internal/resources/template_file"
	tomlFileResource "github.com/ebogdum/filemanager/internal/resources/toml_file"
	transferResource "github.com/ebogdum/filemanager/internal/resources/transfer"
	uploadResource "github.com/ebogdum/filemanager/internal/resources/upload"
	xmlFileResource "github.com/ebogdum/filemanager/internal/resources/xml_file"
	yamlFileResource "github.com/ebogdum/filemanager/internal/resources/yaml_file"
)

// Ensure FileManagerProvider satisfies various provider interfaces.
var (
	_ provider.Provider              = &FileManagerProvider{}
	_ provider.ProviderWithFunctions = &FileManagerProvider{}
)

// FileManagerProvider defines the provider implementation.
type FileManagerProvider struct {
	version  string
	registry *plugin.Registry
}

// FileManagerProviderModel describes the provider data model.
type FileManagerProviderModel struct {
	BasePath                   types.String `tfsdk:"base_path"`
	DefaultFilePermission      types.String `tfsdk:"default_file_permission"`
	DefaultDirectoryPermission types.String `tfsdk:"default_directory_permission"`
	AtomicWrites               types.Bool   `tfsdk:"atomic_writes"`
	VerifyChecksum             types.Bool   `tfsdk:"verify_checksum"`
	EnableLocking              types.Bool   `tfsdk:"enable_locking"`
	LockTimeout                types.String `tfsdk:"lock_timeout"`
	BackupEnabled              types.Bool   `tfsdk:"backup_enabled"`
	BackupRetention            types.Int64  `tfsdk:"backup_retention"`
	BackupDir                  types.String `tfsdk:"backup_dir"`
}

// New creates a new provider instance.
func New(version string) func() provider.Provider {
	return func() provider.Provider {
		return &FileManagerProvider{
			version:  version,
			registry: plugin.NewRegistry(),
		}
	}
}

// Metadata returns the provider type name.
func (p *FileManagerProvider) Metadata(ctx context.Context, req provider.MetadataRequest, resp *provider.MetadataResponse) {
	resp.TypeName = "filemanager"
	resp.Version = p.version
}

// Schema defines the provider-level schema.
func (p *FileManagerProvider) Schema(ctx context.Context, req provider.SchemaRequest, resp *provider.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "The FileManager provider enables comprehensive file and directory management with ACID guarantees, " +
			"supporting multiple storage backends, structured content formats, and application-specific configurations.",
		MarkdownDescription: `
The FileManager provider enables comprehensive file and directory management with ACID guarantees.

## Features

- **ACID Guarantees**: Atomic writes, file locking, checksum verification
- **Multiple Backends**: Local filesystem, SSH/SFTP, S3, Azure, GCS, and more
- **Structured Content**: JSON, YAML, TOML with deep merge, path queries, and schema validation
- **Application Configs**: Native support for nginx, consul, prometheus, and other application configurations
- **Zero-Copy Transfers**: Optimized file transfers using sendfile, splice, and copy_file_range
`,
		Attributes: map[string]schema.Attribute{
			"base_path": schema.StringAttribute{
				Description: "Base path for relative file paths. If set, all relative paths will be resolved from this directory.",
				Optional:    true,
			},
			"default_file_permission": schema.StringAttribute{
				Description: "Default permission mode for created files in octal format (e.g., \"0644\").",
				Optional:    true,
			},
			"default_directory_permission": schema.StringAttribute{
				Description: "Default permission mode for created directories in octal format (e.g., \"0755\").",
				Optional:    true,
			},
			"atomic_writes": schema.BoolAttribute{
				Description: "Enable atomic writes for all file operations. Uses temp file + rename for safe writes.",
				Optional:    true,
			},
			"verify_checksum": schema.BoolAttribute{
				Description: "Verify file checksums after write operations.",
				Optional:    true,
			},
			"enable_locking": schema.BoolAttribute{
				Description: "Enable file locking for concurrent access protection.",
				Optional:    true,
			},
			"lock_timeout": schema.StringAttribute{
				Description: "Timeout for acquiring file locks (e.g., \"30s\", \"1m\").",
				Optional:    true,
			},
			"backup_enabled": schema.BoolAttribute{
				Description: "Enable automatic backups before file modifications.",
				Optional:    true,
			},
			"backup_retention": schema.Int64Attribute{
				Description: "Number of backup copies to retain per file.",
				Optional:    true,
			},
			"backup_dir": schema.StringAttribute{
				Description: "Directory for storing backup files. If empty, backups are stored alongside originals.",
				Optional:    true,
			},
		},
	}
}

// Configure prepares the provider for data source and resource operations.
func (p *FileManagerProvider) Configure(ctx context.Context, req provider.ConfigureRequest, resp *provider.ConfigureResponse) {
	var config FileManagerProviderModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Register built-in backends
	if err := p.registry.Backends.Register("file", func() plugin.Backend {
		return local.New()
	}); err != nil {
		// Log but don't fail - backend may already be registered from previous configure
		tflog.Debug(ctx, "Backend registration note", map[string]any{
			"scheme": "file",
			"error":  err.Error(),
		})
	}

	// Register SSH/SFTP backend
	if err := p.registry.Backends.Register("ssh", func() plugin.Backend {
		return ssh.New()
	}); err != nil {
		// Log but don't fail - backend may already be registered
		tflog.Debug(ctx, "Backend registration note", map[string]any{
			"scheme": "ssh",
			"error":  err.Error(),
		})
	}

	// Register FTP/FTPS backend
	if err := p.registry.Backends.Register("ftp", func() plugin.Backend {
		return ftp.New()
	}); err != nil {
		tflog.Debug(ctx, "Backend registration note", map[string]any{
			"scheme": "ftp",
			"error":  err.Error(),
		})
	}

	// Register OpenStack Swift backend
	if err := p.registry.Backends.Register("swift", func() plugin.Backend {
		return swift.New()
	}); err != nil {
		tflog.Debug(ctx, "Backend registration note", map[string]any{
			"scheme": "swift",
			"error":  err.Error(),
		})
	}

	// Create provider configuration for resources
	providerConfig := &common.ProviderConfig{
		Registry:                   p.registry,
		BasePath:                   stringValueOrDefault(config.BasePath, ""),
		DefaultFilePermission:      stringValueOrDefault(config.DefaultFilePermission, "0644"),
		DefaultDirectoryPermission: stringValueOrDefault(config.DefaultDirectoryPermission, "0755"),
		AtomicWrites:               boolValueOrDefault(config.AtomicWrites, true),
		VerifyChecksum:             boolValueOrDefault(config.VerifyChecksum, true),
		EnableLocking:              boolValueOrDefault(config.EnableLocking, true),
		LockTimeout:                parseDuration(stringValueOrDefault(config.LockTimeout, "30s")),
		BackupEnabled:              boolValueOrDefault(config.BackupEnabled, false),
		BackupRetention:            int(int64ValueOrDefault(config.BackupRetention, 5)),
		BackupDir:                  stringValueOrDefault(config.BackupDir, ".backups"),
	}

	// Initialize the local backend
	localBackend := local.New()
	backendConfig := plugin.BackendConfig{
		BasePath: providerConfig.BasePath,
	}

	if err := localBackend.Connect(ctx, backendConfig); err != nil {
		resp.Diagnostics.AddAttributeError(
			path.Root("base_path"),
			"Failed to initialize local backend",
			err.Error(),
		)
		return
	}

	// Store configured backend
	if err := p.registry.Backends.SetAlias("local", localBackend); err != nil {
		// Log but don't fail - alias may already be set from previous configure
		tflog.Debug(ctx, "Backend alias note", map[string]any{
			"alias": "local",
			"error": err.Error(),
		})
	}

	providerConfig.LocalBackend = localBackend

	// Make configuration available to resources and data sources
	resp.DataSourceData = providerConfig
	resp.ResourceData = providerConfig
}

// Resources defines the resources implemented in the provider.
func (p *FileManagerProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		fileResource.NewFileResource,
		sensitiveFileResource.NewSensitiveFileResource,
		jsonFileResource.NewJSONFileResource,
		yamlFileResource.NewYAMLFileResource,
		tomlFileResource.NewTOMLFileResource,
		iniFileResource.NewINIFileResource,
		envFileResource.NewEnvFileResource,
		xmlFileResource.NewXMLFileResource,
		hclFileResource.NewHCLFileResource,
		templateFileResource.NewTemplateFileResource,
		appConfigResource.NewAppConfigResource,
		directoryResource.NewDirectoryResource,
		symlinkResource.NewSymlinkResource,
		archiveResource.NewArchiveResource,
		copyResource.NewCopyResource,
		uploadResource.NewUploadResource,
		downloadResource.NewDownloadResource,
		syncResource.NewSyncResource,
		transferResource.NewTransferResource,
		s3OperationResource.NewS3OperationResource,
		azureOperationResource.NewAzureOperationResource,
		gcsOperationResource.NewGCSOperationResource,
		b2OperationResource.NewB2OperationResource,
		s3ServiceResource.NewS3ServiceResource,
		azureServiceResource.NewAzureServiceResource,
		gcsServiceResource.NewGCSServiceResource,
		b2ServiceResource.NewB2ServiceResource,
		sshServiceResource.NewSSHServiceResource,
		ftpServiceResource.NewFTPServiceResource,
		ftpOperationResource.NewFTPOperationResource,
		swiftServiceResource.NewSwiftServiceResource,
		swiftOperationResource.NewSwiftOperationResource,
	}
}

// DataSources defines the data sources implemented in the provider.
func (p *FileManagerProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		fileDataSource.NewFileDataSource,
		filesDataSource.NewFilesDataSource,
		directoryDataSource.NewDirectoryDataSource,
		checksumDataSource.NewChecksumDataSource,
		statDataSource.NewStatDataSource,
		validateDataSource.NewValidateDataSource,
		compareDataSource.NewCompareDataSource,
		jsonDataSource.NewJSONDataSource,
		yamlDataSource.NewYAMLDataSource,
		tomlDataSource.NewTOMLDataSource,
		iniDataSource.NewINIDataSource,
		xmlDataSource.NewXMLDataSource,
		hclDataSource.NewHCLDataSource,
		envDataSource.NewENVDataSource,
		environmentDataSource.NewEnvironmentDataSource,
		usersDataSource.NewUsersDataSource,
		groupsDataSource.NewGroupsDataSource,
	}
}

// Functions defines the provider functions.
func (p *FileManagerProvider) Functions(ctx context.Context) []func() function.Function {
	return []func() function.Function{
		functions.NewFileExistsFunction,
		functions.NewDirExistsFunction,
		functions.NewPathJoinFunction,
		functions.NewPathDirnameFunction,
		functions.NewPathBasenameFunction,
		functions.NewPathExtFunction,
		functions.NewPathExpandFunction,
		functions.NewGlobFunction,
	}
}

// Helper functions for extracting values from framework types.

func stringValueOrDefault(v types.String, def string) string {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueString()
}

func boolValueOrDefault(v types.Bool, def bool) bool {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueBool()
}

func int64ValueOrDefault(v types.Int64, def int64) int64 {
	if v.IsNull() || v.IsUnknown() {
		return def
	}
	return v.ValueInt64()
}

func parseDuration(s string) time.Duration {
	d, err := time.ParseDuration(s)
	if err != nil {
		return 30 * time.Second
	}
	return d
}
