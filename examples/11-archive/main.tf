# =============================================================================
# ARCHIVE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/11-archive"
  source_dir = "${path.module}/../../test/output/11-archive/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR ARCHIVING
# -----------------------------------------------------------------------------

resource "filemanager_directory" "source" {
  path           = local.source_dir
  create_parents = true
}

resource "filemanager_file" "source_file1" {
  path               = "${local.source_dir}/file1.txt"
  content            = "Content of file 1"
  create_parent_dirs = true
}

resource "filemanager_file" "source_file2" {
  path               = "${local.source_dir}/file2.txt"
  content            = "Content of file 2"
  create_parent_dirs = true
}

resource "filemanager_file" "source_file3" {
  path               = "${local.source_dir}/subdir/file3.txt"
  content            = "Content of file 3 in subdirectory"
  create_parent_dirs = true
}

resource "filemanager_file" "source_log" {
  path               = "${local.source_dir}/app.log"
  content            = "Log file content - should be excluded"
  create_parent_dirs = true
}

resource "filemanager_file" "source_tmp" {
  path               = "${local.source_dir}/temp.tmp"
  content            = "Temporary file - should be excluded"
  create_parent_dirs = true
}

resource "filemanager_file" "source_json" {
  path               = "${local.source_dir}/config.json"
  content            = jsonencode({ key = "value" })
  create_parent_dirs = true
}

resource "filemanager_file" "source_large" {
  path               = "${local.source_dir}/large.txt"
  content            = join("\n", [for i in range(1000) : "Line ${i}: ${uuid()}"])
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# ZIP ARCHIVES
# -----------------------------------------------------------------------------

# Case 1: Basic ZIP archive
resource "filemanager_archive" "zip_basic" {
  path       = "${local.output_dir}/archives/basic.zip"
  type       = "zip"
  source_dir = local.source_dir

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_json,
  ]
}

# Case 2: ZIP with excludes
resource "filemanager_archive" "zip_excludes" {
  path       = "${local.output_dir}/archives/with_excludes.zip"
  type       = "zip"
  source_dir = local.source_dir

  excludes = ["*.log", "*.tmp"]

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
  ]
}

# Case 3: ZIP with multiple exclude patterns
resource "filemanager_archive" "zip_multi_excludes" {
  path       = "${local.output_dir}/archives/multi_excludes.zip"
  type       = "zip"
  source_dir = local.source_dir

  excludes = [
    "*.log",
    "*.tmp",
    "*.json",
    "subdir/*"
  ]

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_json,
  ]
}

# -----------------------------------------------------------------------------
# TAR ARCHIVES
# -----------------------------------------------------------------------------

# Case 4: Basic TAR archive
resource "filemanager_archive" "tar_basic" {
  path       = "${local.output_dir}/archives/basic.tar"
  type       = "tar"
  source_dir = local.source_dir

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
  ]
}

# Case 5: TAR with excludes
resource "filemanager_archive" "tar_excludes" {
  path       = "${local.output_dir}/archives/with_excludes.tar"
  type       = "tar"
  source_dir = local.source_dir

  excludes = ["*.log", "*.tmp"]

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
  ]
}

# -----------------------------------------------------------------------------
# TAR.GZ (GZIPPED) ARCHIVES
# -----------------------------------------------------------------------------

# Case 6: Basic tar.gz archive
resource "filemanager_archive" "targz_basic" {
  path       = "${local.output_dir}/archives/basic.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
  ]
}

# Case 7: tar.gz with excludes
resource "filemanager_archive" "targz_excludes" {
  path       = "${local.output_dir}/archives/with_excludes.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  excludes = ["*.log", "*.tmp", "large.txt"]

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_log,
    filemanager_file.source_large,
  ]
}

# Case 8: tar.gz backup
resource "filemanager_archive" "backup" {
  path       = "${local.output_dir}/backups/backup.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  excludes = ["*.log", "*.tmp"]

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_file3,
    filemanager_file.source_json,
  ]
}

# -----------------------------------------------------------------------------
# COMPARISON - SAME SOURCE, DIFFERENT FORMATS
# -----------------------------------------------------------------------------

# Case 9-11: Compare compression formats
resource "filemanager_archive" "compare_zip" {
  path       = "${local.output_dir}/compare/data.zip"
  type       = "zip"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_large]
}

resource "filemanager_archive" "compare_tar" {
  path       = "${local.output_dir}/compare/data.tar"
  type       = "tar"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_large]
}

resource "filemanager_archive" "compare_targz" {
  path       = "${local.output_dir}/compare/data.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_large]
}

# -----------------------------------------------------------------------------
# EMPTY DIRECTORY ARCHIVE
# -----------------------------------------------------------------------------

resource "filemanager_directory" "empty_source" {
  path           = "${local.source_dir}/empty_subdir"
  create_parents = true
  depends_on     = [filemanager_directory.source]
}

# Case 12: Archive with empty subdirectory
resource "filemanager_archive" "with_empty" {
  path       = "${local.output_dir}/special/with_empty.zip"
  type       = "zip"
  source_dir = local.source_dir

  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_directory.empty_source,
  ]
}

# -----------------------------------------------------------------------------
# NESTED DIRECTORY STRUCTURE
# -----------------------------------------------------------------------------

resource "filemanager_file" "nested_deep" {
  path               = "${local.source_dir}/level1/level2/level3/deep.txt"
  content            = "Deeply nested file"
  create_parent_dirs = true
}

# Case 13: Archive with deeply nested structure
resource "filemanager_archive" "nested" {
  path       = "${local.output_dir}/special/nested.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp"]
  create_parent_dirs = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.nested_deep,
  ]
}

# -----------------------------------------------------------------------------
# MULTIPLE ARCHIVES FROM SAME SOURCE
# -----------------------------------------------------------------------------

# Case 14-16: Multiple archives from same source
resource "filemanager_archive" "multi_1" {
  path       = "${local.output_dir}/multi/archive1.zip"
  type       = "zip"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp", "large.txt", "subdir/*", "level1/*"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_file1]
}

resource "filemanager_archive" "multi_2" {
  path       = "${local.output_dir}/multi/archive2.zip"
  type       = "zip"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp", "large.txt", "level1/*"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_file1, filemanager_file.source_file3]
}

resource "filemanager_archive" "multi_3" {
  path       = "${local.output_dir}/multi/archive3.tar.gz"
  type       = "tar.gz"
  source_dir = local.source_dir

  excludes           = ["*.log", "*.tmp"]
  create_parent_dirs = true

  depends_on = [filemanager_file.source_file1]
}
