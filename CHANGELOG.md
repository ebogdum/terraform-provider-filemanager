# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

## [1.4.1] - 2026-07-31

Security-only release. No functional or API changes.

### Security
- Bump `golang.org/x/crypto` 0.46.0 → 0.53.0 — resolves 11 SSH advisories, including
  agent key-constraint bypass, `@revoked` certificate status not enforced,
  `VerifiedPublicKeyCallback` permission skip, FIDO/U2F presence-check bypass,
  infinite loop on large channel writes, and multiple client/server panics and DoS paths
- Bump `google.golang.org/grpc` 1.79.3 → 1.82.1 — resolves xDS RBAC and HTTP/2 advisories
- Bump `golang.org/x/net` 0.48.0 → 0.56.0 — resolves HTML parser DoS and a
  `dns/dnsmessage` panic on malformed SVCB/HTTPS records
- Bump `golang.org/x/text` 0.37.0 → 0.39.0 — resolves an infinite loop on invalid input
  reachable from HCL parsing and template rendering (GO-2026-5970)
- Bump `go.opentelemetry.io/otel` and `otel/sdk` 1.40.0 → 1.44.0 — resolves `baggage`
  header allocation amplification and BSD `kenv` PATH hijacking
- Bump `github.com/aws/aws-sdk-go-v2/service/s3` 1.48.1 → 1.97.3 and
  `aws/protocol/eventstream` 1.5.4 → 1.7.8 — resolves an EventStream decoder panic (DoS)

### Changed
- Bump AWS SDK v2 core to 1.43.2, `config` to 1.32.33, `credentials` to 1.19.32,
  and `smithy-go` to 1.27.5 to match the S3 client upgrade

## [1.4.0] - 2026-04-04

### Security
- Remove `env` template function from `template_file` resource (SSTI risk)
- Remove `tls_skip_verify` from FTP service (was rejected by backend anyway)
- Fix SSH `insecure_skip_host_key` config key mismatch (flag was silently ignored)
- Mark `sha256` as sensitive in `sensitive_file` resource
- Add source path validation against `base_path` in file resource
- Simplify `ImportState` to prevent arbitrary file reads
- Validate backup directory paths against traversal attacks
- Validate XML element names before serialization
- Remove credential file paths from error messages
- Remove sensitive SHA256 hashes from drift detection warnings

### Fixed
- Cross-device atomic writes now use temp-file + rename (preserves ACID guarantee)
- SSH agent connection no longer leaked on backend close
- Connection pool race condition in `Get()` (lock-held health check)
- Pool cleanup no longer blocks on network timeouts (close outside mutex)
- SSH/FTP recursive delete now has depth limits and skips symlinks
- FTP error 550 disambiguation (not-found vs permission-denied)
- `toJSON` template function now produces actual JSON (was using `fmt.Sprintf`)
- `toInt` template function logs warning on parse failure instead of silent zero
- `ParseOctalMode` logs warning on invalid permission strings
- `ComputePathOutputs` handles `filepath.Abs` errors gracefully
- `numberValueToGo` handles int64 overflow correctly
- `CountingReader` is now thread-safe (atomic operations)
- S3 service `Update` now closes old backend before replacing
- Archive checksum computed via streaming (no longer loads entire file into memory)
- Remove double-close of gzWriter in backup
- Remove dead traversal check in lock.go
- Log backup cleanup errors instead of silently swallowing

### Added
- `~username` expansion support in `ExpandPath`
- `ReadAllLimited` utility for bounded reads
- Improved symlink `target_type` description (portability implications)

### Dependencies
- Bump `github.com/antchfx/xpath` from 1.2.4 to 1.3.6
- Bump `github.com/go-jose/go-jose/v4` from 4.1.3 to 4.1.4
- Bump `go.opentelemetry.io/otel/sdk` from 1.38.0 to 1.40.0
- Bump `google.golang.org/grpc` from 1.78.0 to 1.79.3

## [1.2.0] - 2026-01-23

### Added

#### Resources
- `filemanager_tfvars_file` - Manage Terraform `.tfvars` files with native type support
  - Native dynamic types: pass maps, lists, numbers, booleans directly
  - Internal interpolation: self-referencing variables via `{{ .var_name }}` syntax
  - External interpolation: use outputs from other Terraform resources/data sources
  - Variable-level operations: set/delete individual variables
  - Merge with existing files: preserve unmanaged variables
  - Dual format output: HCL (`.tfvars`) or JSON (`.tfvars.json`)
  - Template functions: upper, lower, trim, replace, split, join, default, env
  - Topological dependency resolution for interpolation order
  - Deep interpolation: template references resolved in nested maps and lists

#### Data Sources
- `filemanager_tfvars` - Read and parse Terraform `.tfvars` files
  - Auto-detects HCL vs JSON format by file extension
  - Returns all variables as native Terraform dynamic types
  - Query support: extract specific variables by name
  - Reports variable names list and count
  - Supports all backends (local, SSH, S3, Azure, GCS, etc.)

#### Internal
- `common.TerraformDynamicToGoValue()` - Reverse conversion from Terraform dynamic types to Go native values

## [1.0.0] - 2026-01-11

### Added

#### Resources
- `filemanager_file` - Manage plain text files with ACID guarantees
- `filemanager_sensitive_file` - Manage files with sensitive content (state encryption)
- `filemanager_directory` - Create and manage directories
- `filemanager_symlink` - Create symbolic links
- `filemanager_json_file` - Manage JSON files with deep merge and path queries
- `filemanager_yaml_file` - Manage YAML files with deep merge and path queries
- `filemanager_toml_file` - Manage TOML configuration files
- `filemanager_ini_file` - Manage INI configuration files
- `filemanager_env_file` - Manage .env environment files
- `filemanager_xml_file` - Manage XML files with XPath queries
- `filemanager_hcl_file` - Manage HCL configuration files
- `filemanager_template_file` - Render Go templates to files
- `filemanager_app_config` - Manage application-specific configurations (nginx, consul, prometheus)
- `filemanager_archive` - Create tar, tar.gz, and zip archives
- `filemanager_copy` - Copy files and directories with exclusion patterns
- `filemanager_upload` - Upload files to remote backends
- `filemanager_download` - Download files from remote backends
- `filemanager_sync` - Synchronize directories between backends
- `filemanager_transfer` - Transfer files between any backends
- `filemanager_s3_operation` - S3 object operations (metadata, tags, storage class, restore)
- `filemanager_azure_operation` - Azure Blob operations (metadata, tags, access tier, lease)
- `filemanager_gcs_operation` - GCS object operations (metadata, storage class, temporary hold)
- `filemanager_b2_operation` - Backblaze B2 operations

#### Data Sources
- `filemanager_file` - Read file content and metadata
- `filemanager_files` - List files matching a glob pattern
- `filemanager_directory` - List directory contents
- `filemanager_stat` - Get file/directory metadata
- `filemanager_checksum` - Calculate file checksums (MD5, SHA256, SHA512)
- `filemanager_validate` - Validate structured content against schemas
- `filemanager_compare` - Compare files or directories

#### Provider Functions
- `file_exists(path)` - Check if a file exists
- `dir_exists(path)` - Check if a directory exists
- `path_join(parts...)` - Join path components
- `path_dirname(path)` - Get directory name from path
- `path_basename(path)` - Get base name from path
- `path_ext(path)` - Get file extension
- `path_expand(path)` - Expand ~ and environment variables
- `glob(pattern)` - Find files matching a glob pattern

#### Features
- Multiple backend support: local filesystem, SSH/SFTP, S3, Azure Blob, GCS, Backblaze B2
- ACID guarantees with atomic writes, file locking, and checksum verification
- Structured content formats with deep merge and schema validation
- Zero-copy file transfers using sendfile, splice, and copy_file_range
- Template rendering with Go templates and custom delimiters
