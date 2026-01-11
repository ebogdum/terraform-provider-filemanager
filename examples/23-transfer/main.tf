# =============================================================================
# TRANSFER RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/23-transfer"
  source_dir = "${path.module}/../../test/output/23-transfer/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR TRANSFER
# -----------------------------------------------------------------------------

resource "filemanager_file" "source_file1" {
  path    = "${local.source_dir}/file1.txt"
  content = "Transfer source file 1 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_file2" {
  path    = "${local.source_dir}/file2.txt"
  content = "Transfer source file 2 content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_json" {
  path    = "${local.source_dir}/data.json"
  content = jsonencode({ name = "transfer", version = "1.0" })

  create_parent_dirs = true
}

resource "filemanager_file" "source_csv" {
  path    = "${local.source_dir}/data.csv"
  content = "id,name,value\n1,foo,100\n2,bar,200"

  create_parent_dirs = true
}

resource "filemanager_file" "source_nested" {
  path    = "${local.source_dir}/level1/level2/deep.txt"
  content = "Deep nested file content"

  create_parent_dirs = true
}

resource "filemanager_file" "source_log" {
  path    = "${local.source_dir}/debug.log"
  content = "Debug log - should be excluded"

  create_parent_dirs = true
}

resource "filemanager_file" "source_tmp" {
  path    = "${local.source_dir}/temp.tmp"
  content = "Temp file - should be excluded"

  create_parent_dirs = true
}

# Create a larger file for transfer testing
resource "filemanager_file" "source_large" {
  path    = "${local.source_dir}/large.txt"
  content = join("\n", [for i in range(100) : "Line ${i}: This is test data for transfer verification"])

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# BASIC TRANSFER OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic transfer (local to local)
resource "filemanager_transfer" "basic" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/basic/file1.txt"

  recursive = false
}

# Case 2: Directory transfer with recursion
resource "filemanager_transfer" "directory" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/directory/"

  recursive = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_json,
    filemanager_file.source_csv,
    filemanager_file.source_nested,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_large,
  ]
}

# Case 3: Transfer with include patterns
resource "filemanager_transfer" "with_includes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/includes/"

  recursive = true
  includes  = ["*.json", "*.csv"]

  depends_on = [
    filemanager_file.source_json,
    filemanager_file.source_csv,
  ]
}

# Case 4: Transfer with exclude patterns
resource "filemanager_transfer" "with_excludes" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/excludes/"

  recursive = true
  excludes  = ["*.log", "*.tmp", "level1/**"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_nested,
  ]
}

# Case 5: Transfer with checksum verification
resource "filemanager_transfer" "with_checksum" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file2.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/checksum/file2.txt"

  checksum_verify = true
}

# Case 6: Transfer with timestamp preservation
resource "filemanager_transfer" "preserve_timestamps" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/timestamps/"

  recursive           = true
  preserve_timestamps = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 7: Transfer with permission preservation
resource "filemanager_transfer" "preserve_permissions" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/permissions/"

  recursive            = true
  preserve_permissions = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 8: Transfer with overwrite enabled
resource "filemanager_transfer" "with_overwrite" {
  source_backend      = "local"
  source_path         = filemanager_file.source_file1.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/overwrite/file.txt"

  overwrite = true
}

# Case 9: Transfer with streaming mode
resource "filemanager_transfer" "streaming" {
  source_backend      = "local"
  source_path         = filemanager_file.source_large.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/streaming/large.txt"

  streaming = true
}

# Case 10: Transfer with zero-copy mode
resource "filemanager_transfer" "zero_copy" {
  source_backend      = "local"
  source_path         = filemanager_file.source_large.path
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/zerocopy/large.txt"

  zero_copy = true
}

# Case 11: Transfer with buffer size settings
resource "filemanager_transfer" "buffer_size" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/buffer/"

  recursive      = true
  buffer_size_kb = 64

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
  ]
}

# Case 12: Transfer with concurrency settings
resource "filemanager_transfer" "with_concurrency" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/concurrent/"

  recursive   = true
  concurrency = 4

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_json,
    filemanager_file.source_csv,
  ]
}

# Case 13: Full transfer with all options
resource "filemanager_transfer" "full" {
  source_backend      = "local"
  source_path         = local.source_dir
  destination_backend = "local"
  destination_path    = "${local.output_dir}/transfers/full/"

  recursive            = true
  excludes             = ["*.log", "*.tmp"]
  checksum_verify      = true
  concurrency          = 2
  buffer_size_kb       = 32
  preserve_timestamps  = true
  preserve_permissions = true
  overwrite            = true
  streaming            = true
  zero_copy            = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_json,
    filemanager_file.source_csv,
    filemanager_file.source_nested,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_large,
  ]
}

