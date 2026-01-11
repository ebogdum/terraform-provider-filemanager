# =============================================================================
# UPLOAD RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/20-upload"
  source_dir = "${path.module}/../../test/output/20-upload/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR UPLOAD
# -----------------------------------------------------------------------------

# Create source directory structure
resource "filemanager_directory" "source" {
  path = local.source_dir

  create_parents = true
}

resource "filemanager_file" "source_file1" {
  path    = "${local.source_dir}/file1.txt"
  content = "Source file 1 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file2" {
  path    = "${local.source_dir}/file2.txt"
  content = "Source file 2 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file3" {
  path    = "${local.source_dir}/subdir/file3.txt"
  content = "Source file 3 in subdir"

  create_parent_dirs = true
}

resource "filemanager_file" "source_js" {
  path    = "${local.source_dir}/app.js"
  content = "console.log('Hello World');"

  create_parent_dirs = true
}

resource "filemanager_file" "source_css" {
  path    = "${local.source_dir}/style.css"
  content = "body { margin: 0; }"

  create_parent_dirs = true
}

resource "filemanager_file" "source_log" {
  path    = "${local.source_dir}/debug.log"
  content = "Debug log content - should be excluded"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# BASIC UPLOAD OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic upload (local to local)
resource "filemanager_upload" "basic" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/basic/file1.txt"

  recursive = false
}

# Case 2: Directory upload with recursion
resource "filemanager_upload" "directory" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/directory/"

  recursive = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_js,
    filemanager_file.source_css,
    filemanager_file.source_log,
  ]
}

# Case 3: Upload with include patterns
resource "filemanager_upload" "with_includes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/includes/"

  recursive = true
  includes  = ["*.js", "*.css"]

  depends_on = [
    filemanager_file.source_js,
    filemanager_file.source_css,
  ]
}

# Case 4: Upload with exclude patterns
resource "filemanager_upload" "with_excludes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/excludes/"

  recursive = true
  excludes  = ["*.log", "subdir/**"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_log,
    filemanager_file.source_file3,
  ]
}

# Case 5: Upload with checksum verification
resource "filemanager_upload" "with_checksum" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file2.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/checksum/file2.txt"

  checksum_verify = true
}

# Case 6: Upload with timestamp preservation
resource "filemanager_upload" "preserve_timestamps" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/timestamps/"

  recursive           = true
  preserve_timestamps = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 7: Upload with overwrite enabled
resource "filemanager_upload" "with_overwrite" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/overwrite/file.txt"

  overwrite = true
}

# Case 8: Upload with concurrency settings
resource "filemanager_upload" "with_concurrency" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/uploads/concurrent/"

  recursive   = true
  concurrency = 2

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
  ]
}

