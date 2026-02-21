# Terraform Provider FileManager

A comprehensive Terraform provider for file and directory management with ACID guarantees, multiple storage backends, and structured content handling.

## Features

- **ACID Guarantees**: Atomic writes, file locking, checksum verification
- **Multiple Backends**: Local filesystem, SSH/SFTP, FTP/FTPS, S3, Azure Blob, GCS, Backblaze B2, OpenStack Swift
- **Structured Content**: JSON, YAML, TOML, INI, ENV, XML, HCL with deep merge and schema validation
- **Template Rendering**: Go templates with custom delimiters
- **Application Configs**: Native support for nginx, consul, prometheus, and other applications
- **Cloud Operations**: S3, Azure, GCS, and B2 object operations (metadata, tags, storage class, restore)
- **Zero-Copy Transfers**: Optimized file transfers using sendfile, splice, and copy_file_range

## Installation

```hcl
terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {
  base_path               = "/opt/app"
  default_file_permission = "0644"
  atomic_writes           = true
}
```

## Resources

| Resource | Description |
|----------|-------------|
| `filemanager_file` | Manage plain text files |
| `filemanager_sensitive_file` | Manage files with sensitive content |
| `filemanager_directory` | Create and manage directories |
| `filemanager_symlink` | Create symbolic links |
| `filemanager_json_file` | Manage JSON files with deep merge |
| `filemanager_yaml_file` | Manage YAML files with deep merge |
| `filemanager_toml_file` | Manage TOML files |
| `filemanager_ini_file` | Manage INI configuration files |
| `filemanager_env_file` | Manage .env environment files |
| `filemanager_xml_file` | Manage XML files |
| `filemanager_hcl_file` | Manage HCL configuration files |
| `filemanager_tfvars_file` | Manage Terraform .tfvars files with native types and interpolation |
| `filemanager_template_file` | Render Go templates to files |
| `filemanager_app_config` | Manage application-specific configs |
| `filemanager_archive` | Create tar, tar.gz, and zip archives |
| `filemanager_copy` | Copy files and directories |
| `filemanager_upload` | Upload files to remote backends |
| `filemanager_download` | Download files from remote backends |
| `filemanager_sync` | Synchronize directories between backends |
| `filemanager_transfer` | Transfer files between any backends |
| `filemanager_s3_operation` | S3 object operations (metadata, tags, restore) |
| `filemanager_azure_operation` | Azure Blob operations (metadata, tags, tier) |
| `filemanager_gcs_operation` | GCS object operations (metadata, storage class) |
| `filemanager_b2_operation` | Backblaze B2 operations |
| `filemanager_s3_service` | Configure S3 backend service |
| `filemanager_azure_service` | Configure Azure Blob backend service |
| `filemanager_gcs_service` | Configure GCS backend service |
| `filemanager_b2_service` | Configure Backblaze B2 backend service |
| `filemanager_ssh_service` | Configure SSH/SFTP backend service |
| `filemanager_ftp_service` | Configure FTP/FTPS backend service |
| `filemanager_ftp_operation` | FTP file operations |
| `filemanager_swift_service` | Configure OpenStack Swift backend service |
| `filemanager_swift_operation` | Swift object operations |

## Data Sources

| Data Source | Description |
|-------------|-------------|
| `filemanager_file` | Read file content |
| `filemanager_files` | List files matching a pattern |
| `filemanager_directory` | List directory contents |
| `filemanager_stat` | Get file/directory metadata |
| `filemanager_checksum` | Calculate file checksums |
| `filemanager_validate` | Validate structured content |
| `filemanager_compare` | Compare files or directories |
| `filemanager_json` | Read and parse JSON files |
| `filemanager_yaml` | Read and parse YAML files |
| `filemanager_toml` | Read and parse TOML files |
| `filemanager_ini` | Read and parse INI files |
| `filemanager_xml` | Read and parse XML files |
| `filemanager_hcl` | Read and parse HCL files |
| `filemanager_tfvars` | Read and parse Terraform .tfvars files |
| `filemanager_env` | Read and parse .env files |
| `filemanager_environment` | Read environment variables |
| `filemanager_users` | Query system users |
| `filemanager_groups` | Query system groups |

## Functions

| Function | Description |
|----------|-------------|
| `file_exists(path)` | Check if a file exists |
| `dir_exists(path)` | Check if a directory exists |
| `path_join(parts...)` | Join path components |
| `path_dirname(path)` | Get directory name from path |
| `path_basename(path)` | Get base name from path |
| `path_ext(path)` | Get file extension |
| `path_expand(path)` | Expand ~ and environment variables |
| `glob(pattern)` | Find files matching a glob pattern |

## Quick Start

### Create a file

```hcl
resource "filemanager_file" "config" {
  path    = "/etc/myapp/config.txt"
  content = "Hello, World!"
}
```

### Create a JSON config

```hcl
resource "filemanager_json_file" "settings" {
  path = "/etc/myapp/settings.json"
  content = {
    database = {
      host = "localhost"
      port = 5432
    }
    logging = {
      level = "info"
    }
  }
  indent = 2
}
```

### Template rendering

```hcl
resource "filemanager_template_file" "nginx" {
  path         = "/etc/nginx/sites-available/myapp"
  template     = file("${path.module}/templates/nginx.conf.tpl")
  variables = {
    server_name = "example.com"
    port        = 8080
  }
}
```

### Copy directory with exclusions

```hcl
resource "filemanager_copy" "deploy" {
  source      = "${path.module}/app"
  destination = "/var/www/html"
  recursive   = true
  excludes    = ["*.log", "*.tmp", ".git"]
}
```

### Create an archive

```hcl
resource "filemanager_archive" "backup" {
  source      = "/var/www/html"
  destination = "/backups/www-backup.tar.gz"
  format      = "tar.gz"
  excludes    = ["*.log"]
}
```

### S3 operations

```hcl
resource "filemanager_s3_operation" "archive" {
  backend       = "s3-prod"
  key           = "data/old-logs.tar.gz"
  operation     = "set_storage_class"
  storage_class = "GLACIER"
}

resource "filemanager_s3_operation" "metadata" {
  backend   = "s3-prod"
  key       = "data/config.json"
  operation = "set_metadata"
  metadata = {
    "x-amz-meta-version" = "1.0"
    "x-amz-meta-author"  = "terraform"
  }
}
```

## Provider Configuration

```hcl
provider "filemanager" {
  # Base path for relative file paths
  base_path = "/opt/app"

  # Default permissions
  default_file_permission      = "0644"
  default_directory_permission = "0755"

  # ACID features
  atomic_writes   = true
  verify_checksum = true
  enable_locking  = true
  lock_timeout    = "30s"

  # Backup settings
  backup_enabled   = true
  backup_retention = 5
  backup_dir       = ".backups"
}
```

## Examples

See the [examples](./examples) directory for comprehensive usage examples:

- `01-file` - Basic file operations
- `02-sensitive-file` - Sensitive file handling
- `03-directory` - Directory management
- `04-symlink` - Symbolic links
- `05-json-file` - JSON file management
- `06-yaml-file` - YAML file management
- `07-toml-file` - TOML file management
- `08-ini-file` - INI file management
- `09-env-file` - Environment file management
- `10-template-file` - Template rendering
- `11-archive` - Archive creation
- `12-copy` - File/directory copying
- `13-data-sources` - Data source usage
- `14-functions` - Provider functions
- `15-integration` - Integration scenarios
- `16-multi-service` - Multi-service configuration
- `17-xml-file` - XML file management
- `18-hcl-file` - HCL file management
- `19-app-config` - Application configs
- `20-upload` - File uploads
- `21-download` - File downloads
- `22-sync` - Directory synchronization
- `23-transfer` - Backend-to-backend transfers
- `24-s3-operation` - S3 operations
- `25-azure-operation` - Azure operations
- `26-gcs-operation` - GCS operations
- `27-b2-operation` - B2 operations
- `28-ftp-operation` - FTP operations
- `29-swift-operation` - Swift operations
- `30-json-data` - JSON data source
- `31-yaml-data` - YAML data source
- `32-toml-data` - TOML data source
- `33-ini-data` - INI data source
- `34-xml-data` - XML data source
- `35-hcl-data` - HCL data source
- `36-env-data` - ENV data source
- `37-tfvars-file` - Terraform .tfvars file management
- `38-tfvars-data` - Reading .tfvars files

## Building from Source

```bash
# Clone the repository
git clone https://github.com/ebogdum/filemanager.git
cd filemanager

# Build
go build -o terraform-provider-filemanager

# Install locally
mkdir -p ~/.terraform.d/plugins/ebogdum/filemanager/1.0.0/$(go env GOOS)_$(go env GOARCH)
mv terraform-provider-filemanager ~/.terraform.d/plugins/ebogdum/filemanager/1.0.0/$(go env GOOS)_$(go env GOARCH)/
```

## Local Git Hooks

Install repository-managed hooks:

```bash
./tools/install-git-hooks.sh
```

Installed hooks:

- `pre-commit`: Regenerates docs with `tfplugindocs` and blocks commits if `docs/` changes are not staged.
- `pre-push` (tags only): Mirrors `.github/workflows/test.yml` checks (`go mod download`, `go build`, `go vet`, `go test`, docs freshness) and runs `goreleaser check` from `.github/workflows/release.yml`.

## License

MIT License - see [LICENSE](LICENSE) for details.
