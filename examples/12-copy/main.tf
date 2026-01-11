# =============================================================================
# COPY RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/12-copy"
  source_dir = "${path.module}/../../test/output/12-copy/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR COPYING
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

resource "filemanager_file" "source_script" {
  path               = "${local.source_dir}/script.sh"
  content            = "#!/bin/bash\necho 'Hello'"
  file_permission    = "0755"
  create_parent_dirs = true
}

resource "filemanager_file" "source_config" {
  path               = "${local.source_dir}/config/app.conf"
  content            = "setting=value"
  create_parent_dirs = true
}

resource "filemanager_file" "source_log" {
  path               = "${local.source_dir}/logs/app.log"
  content            = "Log content - should be excluded"
  create_parent_dirs = true
}

resource "filemanager_file" "source_tmp" {
  path               = "${local.source_dir}/cache/temp.tmp"
  content            = "Temporary file - should be excluded"
  create_parent_dirs = true
}

resource "filemanager_file" "source_nested" {
  path               = "${local.source_dir}/deep/level1/level2/nested.txt"
  content            = "Deeply nested file"
  create_parent_dirs = true
}

resource "filemanager_file" "source_large" {
  path               = "${local.source_dir}/large.bin"
  content            = join("", [for i in range(1000) : "Line ${i}: ${uuid()}\n"])
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# SINGLE FILE COPY
# -----------------------------------------------------------------------------

# Case 1: Basic single file copy
resource "filemanager_copy" "single_basic" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/single/file1_copy.txt"

  depends_on = [filemanager_file.source_file1]
}

# Case 2: Single file with different name
resource "filemanager_copy" "single_rename" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/single/renamed.txt"

  depends_on = [filemanager_file.source_file1]
}

# Case 3: Copy executable with preserved permissions
resource "filemanager_copy" "single_executable" {
  source               = filemanager_file.source_script.path
  destination          = "${local.output_dir}/single/script_copy.sh"
  preserve_permissions = true

  depends_on = [filemanager_file.source_script]
}

# Case 4: Copy with custom permissions
resource "filemanager_copy" "single_custom_perms" {
  source               = filemanager_file.source_file1.path
  destination          = "${local.output_dir}/single/custom_perms.txt"
  preserve_permissions = false
  file_permission      = "0600"

  depends_on = [filemanager_file.source_file1]
}

# Case 5: Copy large file
resource "filemanager_copy" "single_large" {
  source      = filemanager_file.source_large.path
  destination = "${local.output_dir}/single/large_copy.bin"

  depends_on = [filemanager_file.source_large]
}

# -----------------------------------------------------------------------------
# DIRECTORY COPY
# -----------------------------------------------------------------------------

# Case 6: Basic directory copy (recursive)
resource "filemanager_copy" "dir_basic" {
  source      = local.source_dir
  destination = "${local.output_dir}/directories/full_copy"
  recursive   = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_file2,
    filemanager_file.source_script,
    filemanager_file.source_config,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_nested,
  ]
}

# Case 7: Directory copy with excludes
resource "filemanager_copy" "dir_excludes" {
  source      = local.source_dir
  destination = "${local.output_dir}/directories/with_excludes"
  recursive   = true
  excludes    = ["*.log", "*.tmp"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
  ]
}

# Case 8: Directory copy with multiple exclude patterns
resource "filemanager_copy" "dir_multi_excludes" {
  source      = local.source_dir
  destination = "${local.output_dir}/directories/multi_excludes"
  recursive   = true
  excludes    = ["*.log", "*.tmp", "*.bin", "cache"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_log,
    filemanager_file.source_tmp,
    filemanager_file.source_large,
  ]
}

# Case 9: Directory copy preserving permissions
resource "filemanager_copy" "dir_preserve_perms" {
  source               = local.source_dir
  destination          = "${local.output_dir}/directories/preserved_perms"
  recursive            = true
  preserve_permissions = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_script,
  ]
}

# Case 10: Directory copy with custom permissions
resource "filemanager_copy" "dir_custom_perms" {
  source               = local.source_dir
  destination          = "${local.output_dir}/directories/custom_perms"
  recursive            = true
  preserve_permissions = false
  file_permission      = "0644"
  directory_permission = "0755"

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_script,
  ]
}

# -----------------------------------------------------------------------------
# OVERWRITE SCENARIOS
# -----------------------------------------------------------------------------

# First create a file to be overwritten
resource "filemanager_file" "existing_file" {
  path               = "${local.output_dir}/overwrite/existing.txt"
  content            = "Original content"
  create_parent_dirs = true
}

# Case 11: Copy with overwrite enabled (default)
resource "filemanager_copy" "overwrite_enabled" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/overwrite/enabled.txt"
  overwrite   = true

  depends_on = [filemanager_file.source_file1]
}

# Case 12: Copy to new location with overwrite disabled
resource "filemanager_copy" "overwrite_disabled_new" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/overwrite/new_file.txt"
  overwrite   = false

  depends_on = [filemanager_file.source_file1]
}

# -----------------------------------------------------------------------------
# DEEPLY NESTED STRUCTURES
# -----------------------------------------------------------------------------

# Case 13: Copy deeply nested directory
resource "filemanager_copy" "nested_deep" {
  source      = "${local.source_dir}/deep"
  destination = "${local.output_dir}/nested/deep_copy"
  recursive   = true

  depends_on = [filemanager_file.source_nested]
}

# Case 14: Copy specific nested file
resource "filemanager_copy" "nested_single" {
  source      = filemanager_file.source_nested.path
  destination = "${local.output_dir}/nested/single_nested.txt"

  depends_on = [filemanager_file.source_nested]
}

# -----------------------------------------------------------------------------
# MULTIPLE COPIES
# -----------------------------------------------------------------------------

# Case 15-17: Multiple copies of same source
resource "filemanager_copy" "multi_copy_1" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/multi/copy1.txt"

  depends_on = [filemanager_file.source_file1]
}

resource "filemanager_copy" "multi_copy_2" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/multi/copy2.txt"

  depends_on = [filemanager_file.source_file1]
}

resource "filemanager_copy" "multi_copy_3" {
  source      = filemanager_file.source_file1.path
  destination = "${local.output_dir}/multi/copy3.txt"

  depends_on = [filemanager_file.source_file1]
}

# -----------------------------------------------------------------------------
# SELECTIVE SUBDIRECTORY COPY
# -----------------------------------------------------------------------------

# Case 18: Copy only config subdirectory
resource "filemanager_copy" "subdir_config" {
  source      = "${local.source_dir}/config"
  destination = "${local.output_dir}/subdir/config_only"
  recursive   = true

  depends_on = [filemanager_file.source_config]
}

# Case 19: Copy only logs subdirectory
resource "filemanager_copy" "subdir_logs" {
  source      = "${local.source_dir}/logs"
  destination = "${local.output_dir}/subdir/logs_only"
  recursive   = true

  depends_on = [filemanager_file.source_log]
}

# -----------------------------------------------------------------------------
# PERMISSION VARIATIONS
# -----------------------------------------------------------------------------

# Case 20: Copy with restrictive permissions
resource "filemanager_copy" "perms_restrictive" {
  source               = local.source_dir
  destination          = "${local.output_dir}/permissions/restrictive"
  recursive            = true
  preserve_permissions = false
  file_permission      = "0600"
  directory_permission = "0700"
  excludes             = ["*.log", "*.tmp", "*.bin"]

  depends_on = [filemanager_file.source_file1]
}

# Case 21: Copy with permissive permissions
resource "filemanager_copy" "perms_permissive" {
  source               = local.source_dir
  destination          = "${local.output_dir}/permissions/permissive"
  recursive            = true
  preserve_permissions = false
  file_permission      = "0666"
  directory_permission = "0777"
  excludes             = ["*.log", "*.tmp", "*.bin"]

  depends_on = [filemanager_file.source_file1]
}

# -----------------------------------------------------------------------------
# BACKUP/DEPLOYMENT SCENARIOS
# -----------------------------------------------------------------------------

# Case 22: Simulate deployment copy
resource "filemanager_copy" "deploy" {
  source      = local.source_dir
  destination = "${local.output_dir}/deploy/www"
  recursive   = true
  overwrite   = true
  excludes    = ["*.log", "*.tmp", "cache"]

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_config,
  ]
}

# Case 23: Simulate backup copy
resource "filemanager_copy" "backup" {
  source               = local.source_dir
  destination          = "${local.output_dir}/backup/data"
  recursive            = true
  preserve_permissions = true

  depends_on = [
    filemanager_file.source_file1,
    filemanager_file.source_config,
  ]
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Source file with special characters in name
resource "filemanager_file" "source_special_name" {
  path               = "${local.source_dir}/special/file with spaces.txt"
  content            = "File with spaces in name"
  create_parent_dirs = true
}

# Case 24: Copy file with spaces in name
resource "filemanager_copy" "special_spaces" {
  source      = filemanager_file.source_special_name.path
  destination = "${local.output_dir}/special/copied with spaces.txt"

  depends_on = [filemanager_file.source_special_name]
}

# Source with unicode name
resource "filemanager_file" "source_unicode" {
  path               = "${local.source_dir}/special/unicode_文件.txt"
  content            = "Unicode filename content"
  create_parent_dirs = true
}

# Case 25: Copy file with unicode name
resource "filemanager_copy" "special_unicode" {
  source      = filemanager_file.source_unicode.path
  destination = "${local.output_dir}/special/copied_unicode_文件.txt"

  depends_on = [filemanager_file.source_unicode]
}

# Source with dots in name
resource "filemanager_file" "source_dotted" {
  path               = "${local.source_dir}/special/config.backup.old.txt"
  content            = "File with multiple dots"
  create_parent_dirs = true
}

# Case 26: Copy file with multiple dots
resource "filemanager_copy" "special_dots" {
  source      = filemanager_file.source_dotted.path
  destination = "${local.output_dir}/special/copied.backup.old.txt"

  depends_on = [filemanager_file.source_dotted]
}

