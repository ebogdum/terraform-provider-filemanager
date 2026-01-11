# =============================================================================
# SYNC RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/22-sync"
  source_dir = "${path.module}/../../test/output/22-sync/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR SYNC
# -----------------------------------------------------------------------------

resource "filemanager_file" "source_file1" {
  path    = "${local.source_dir}/file1.txt"
  content = "Sync source file 1 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file2" {
  path    = "${local.source_dir}/file2.txt"
  content = "Sync source file 2 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file3" {
  path    = "${local.source_dir}/file3.txt"
  content = "Sync source file 3 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_nested" {
  path    = "${local.source_dir}/subdir/nested.txt"
  content = "Nested file for sync"

  create_parent_dirs = true
}

resource "filemanager_file" "source_config" {
  path    = "${local.source_dir}/config.json"
  content = jsonencode({ env = "production", version = "1.0" })

  create_parent_dirs = true
}

resource "filemanager_file" "source_log" {
  path    = "${local.source_dir}/app.log"
  content = "Log content - should be excluded"

  create_parent_dirs = true
}

resource "filemanager_file" "source_tmp" {
  path    = "${local.source_dir}/cache.tmp"
  content = "Temp content - should be excluded"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# BASIC SYNC OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic sync (local to local)
resource "filemanager_sync" "basic" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/basic/"

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
  ]
}

# Case 2: Sync with delete orphans
resource "filemanager_sync" "delete_orphans" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/delete_orphans/"

  delete_orphans = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 3: Sync with size-only comparison
resource "filemanager_sync" "size_only" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/size_only/"

  comparison_mode = "size_only"

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 4: Sync with checksum comparison
resource "filemanager_sync" "checksum" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/checksum/"

  comparison_mode = "checksum"

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 5: Sync with mtime comparison (default)
resource "filemanager_sync" "mtime" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/mtime/"

  comparison_mode = "mtime"

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 6: Sync with include patterns
resource "filemanager_sync" "with_includes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/includes/"

  includes = ["*.txt", "*.json"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_config,
  ]
}

# Case 7: Sync with exclude patterns
resource "filemanager_sync" "with_excludes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/excludes/"

  excludes = ["*.log", "*.tmp", "subdir/**"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_nested,
  ]
}

# Case 8: Sync with timestamp preservation
resource "filemanager_sync" "preserve_timestamps" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/timestamps/"

  preserve_timestamps = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 9: Sync with concurrency settings
resource "filemanager_sync" "with_concurrency" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/concurrent/"

  concurrency = 4

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_nested,
  ]
}

# Case 10: Full sync with all options
resource "filemanager_sync" "full" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/sync/full/"

  delete_orphans      = true
  comparison_mode     = "checksum"
  excludes            = ["*.log", "*.tmp"]
  preserve_timestamps = true
  concurrency         = 2

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_nested,
    filemanager_file.source_config,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
  ]
}

