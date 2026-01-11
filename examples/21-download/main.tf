# =============================================================================
# DOWNLOAD RESOURCE - ALL USE CASES
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {}

locals {
  output_dir = "${path.module}/../../test/output/21-download"
  source_dir = "${path.module}/../../test/output/21-download/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR DOWNLOAD
# -----------------------------------------------------------------------------

resource "filemanager_file" "source_file1" {
  path    = "${local.source_dir}/file1.txt"
  content = "Download source file 1 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file2" {
  path    = "${local.source_dir}/file2.txt"
  content = "Download source file 2 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_nested" {
  path    = "${local.source_dir}/level1/level2/nested.txt"
  content = "Nested file content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_data_json" {
  path    = "${local.source_dir}/data.json"
  content = jsonencode({ key = "value", count = 42 })

  create_parent_dirs = true
}

resource "filemanager_file" "source_data_yaml" {
  path    = "${local.source_dir}/data.yaml"
  content = "key: value\ncount: 42"

  create_parent_dirs = true
}

resource "filemanager_file" "source_temp" {
  path    = "${local.source_dir}/temp.tmp"
  content = "Temporary file - should be excluded"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# BASIC DOWNLOAD OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic download (local to local)
resource "filemanager_download" "basic" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/basic/file1.txt"
}

# Case 2: Directory download with recursion
resource "filemanager_download" "directory" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/directory/"

  recursive = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_nested,
    filemanager_file.source_data_json,
    filemanager_file.source_data_yaml,
    filemanager_file.source_temp,
  ]
}

# Case 3: Download with include patterns
resource "filemanager_download" "with_includes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/includes/"

  recursive = true
  includes  = ["*.json", "*.yaml"]

  depends_on = [
    filemanager_file.source_data_json,
    filemanager_file.source_data_yaml,
  ]
}

# Case 4: Download with exclude patterns
resource "filemanager_download" "with_excludes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/excludes/"

  recursive = true
  excludes  = ["*.tmp", "level1/**"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_temp,
    filemanager_file.source_nested,
  ]
}

# Case 5: Download with checksum verification
resource "filemanager_download" "with_checksum" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file2.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/checksum/file2.txt"

  # No expected_checksum - just verify the download
}

# Case 6: Download with timestamp preservation
resource "filemanager_download" "preserve_timestamps" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/timestamps/"

  recursive           = true
  preserve_timestamps = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 7: Download with overwrite enabled
resource "filemanager_download" "with_overwrite" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/overwrite/file.txt"

  overwrite = true
}

# Case 8: Download with custom permissions
resource "filemanager_download" "with_permissions" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/permissions/file.txt"

  file_permission      = "0644"
  directory_permission = "0755"
}

# Case 9: Download with concurrency settings
resource "filemanager_download" "with_concurrency" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/downloads/concurrent/"

  recursive   = true
  concurrency = 2

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_nested,
  ]
}

