# =============================================================================
# DIRECTORY RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/03-directory"
}

# -----------------------------------------------------------------------------
# BASIC DIRECTORY OPERATIONS
# -----------------------------------------------------------------------------

# Case 1: Basic directory
resource "filemanager_directory" "basic" {
  path = "${local.output_dir}/basic"
}

# Case 2: Directory with specific permissions
resource "filemanager_directory" "with_perms" {
  path       = "${local.output_dir}/with-perms"
  permission = "0755"
}

# Case 3: Restricted directory (0700)
resource "filemanager_directory" "restricted" {
  path       = "${local.output_dir}/restricted"
  permission = "0700"
}

# Case 4: Read-only directory (0555)
resource "filemanager_directory" "readonly" {
  path       = "${local.output_dir}/readonly"
  permission = "0555"
}

# Case 5: Full permissions (0777)
resource "filemanager_directory" "full_perms" {
  path       = "${local.output_dir}/full-perms"
  permission = "0777"
}

# -----------------------------------------------------------------------------
# NESTED DIRECTORIES
# -----------------------------------------------------------------------------

# Case 6: Single level nesting
resource "filemanager_directory" "nested_single" {
  path           = "${local.output_dir}/nested/single"
  create_parents = true
}

# Case 7: Deep nesting
resource "filemanager_directory" "nested_deep" {
  path           = "${local.output_dir}/nested/level1/level2/level3/level4/level5"
  create_parents = true
}

# Case 8: Multiple sibling directories
resource "filemanager_directory" "siblings_a" {
  path           = "${local.output_dir}/siblings/dir-a"
  create_parents = true
}

resource "filemanager_directory" "siblings_b" {
  path           = "${local.output_dir}/siblings/dir-b"
  create_parents = true
}

resource "filemanager_directory" "siblings_c" {
  path           = "${local.output_dir}/siblings/dir-c"
  create_parents = true
}

# -----------------------------------------------------------------------------
# SPECIAL DIRECTORY NAMES
# -----------------------------------------------------------------------------

# Case 9: Directory with spaces
resource "filemanager_directory" "with_spaces" {
  path           = "${local.output_dir}/special/dir with spaces"
  create_parents = true
}

# Case 10: Directory with dots
resource "filemanager_directory" "with_dots" {
  path           = "${local.output_dir}/special/dir.with.dots"
  create_parents = true
}

# Case 11: Hidden directory
resource "filemanager_directory" "hidden" {
  path           = "${local.output_dir}/special/.hidden"
  create_parents = true
}

# Case 12: Directory with numbers
resource "filemanager_directory" "with_numbers" {
  path           = "${local.output_dir}/special/dir123"
  create_parents = true
}

# Case 13: Directory with special chars
resource "filemanager_directory" "special_chars" {
  path           = "${local.output_dir}/special/dir-with_special.chars"
  create_parents = true
}

# Case 14: Very long directory name
resource "filemanager_directory" "long_name" {
  path           = "${local.output_dir}/special/this_is_a_very_long_directory_name_that_tests_limits"
  create_parents = true
}

# -----------------------------------------------------------------------------
# DIRECTORY STRUCTURES
# -----------------------------------------------------------------------------

# Case 15: Project structure
resource "filemanager_directory" "project_root" {
  path           = "${local.output_dir}/project"
  create_parents = true
}

resource "filemanager_directory" "project_src" {
  path       = "${local.output_dir}/project/src"
  depends_on = [filemanager_directory.project_root]
}

resource "filemanager_directory" "project_tests" {
  path       = "${local.output_dir}/project/tests"
  depends_on = [filemanager_directory.project_root]
}

resource "filemanager_directory" "project_docs" {
  path       = "${local.output_dir}/project/docs"
  depends_on = [filemanager_directory.project_root]
}

resource "filemanager_directory" "project_config" {
  path       = "${local.output_dir}/project/config"
  depends_on = [filemanager_directory.project_root]
}

# Case 16: Data directory structure
resource "filemanager_directory" "data_raw" {
  path           = "${local.output_dir}/data/raw"
  create_parents = true
}

resource "filemanager_directory" "data_processed" {
  path           = "${local.output_dir}/data/processed"
  create_parents = true
}

resource "filemanager_directory" "data_output" {
  path           = "${local.output_dir}/data/output"
  create_parents = true
}

# Case 17: Logs directory structure with date-based subdirs
resource "filemanager_directory" "logs_2024" {
  path           = "${local.output_dir}/logs/2024/01"
  create_parents = true
}

resource "filemanager_directory" "logs_archive" {
  path           = "${local.output_dir}/logs/archive"
  create_parents = true
}

# -----------------------------------------------------------------------------
# PERMISSION VARIATIONS
# -----------------------------------------------------------------------------

# Parent directory for permission tests (needs execute bit for traversal)
resource "filemanager_directory" "permissions_parent" {
  path           = "${local.output_dir}/permissions"
  permission     = "0755"
  create_parents = true
}

# Case 18: Directory with various permission levels
resource "filemanager_directory" "perm_644" {
  path       = "${local.output_dir}/permissions/mode-644"
  permission = "0644"

  depends_on = [filemanager_directory.permissions_parent]
}

resource "filemanager_directory" "perm_750" {
  path       = "${local.output_dir}/permissions/mode-750"
  permission = "0750"

  depends_on = [filemanager_directory.permissions_parent]
}

resource "filemanager_directory" "perm_711" {
  path       = "${local.output_dir}/permissions/mode-711"
  permission = "0711"

  depends_on = [filemanager_directory.permissions_parent]
}

# -----------------------------------------------------------------------------
# FORCE DELETE (for cleanup)
# -----------------------------------------------------------------------------

# Case 19: Directory with force delete enabled
resource "filemanager_directory" "force_delete" {
  path           = "${local.output_dir}/force-delete"
  force_delete   = true
  create_parents = true
}

# Case 20: Nested directory with force delete
resource "filemanager_directory" "force_delete_nested" {
  path           = "${local.output_dir}/force-delete-nested/sub1/sub2"
  force_delete   = true
  create_parents = true
}
