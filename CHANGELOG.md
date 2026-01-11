# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [Unreleased]

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
