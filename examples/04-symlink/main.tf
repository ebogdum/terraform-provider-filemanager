# =============================================================================
# SYMLINK RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/04-symlink"
}

# -----------------------------------------------------------------------------
# PREREQUISITE FILES AND DIRECTORIES
# -----------------------------------------------------------------------------

resource "filemanager_file" "target_file" {
  path               = "${local.output_dir}/targets/file.txt"
  content            = "This is the target file"
  create_parent_dirs = true
}

resource "filemanager_file" "target_file2" {
  path               = "${local.output_dir}/targets/another.txt"
  content            = "Another target file"
  create_parent_dirs = true
}

resource "filemanager_directory" "target_dir" {
  path           = "${local.output_dir}/targets/directory"
  create_parents = true
}

resource "filemanager_file" "file_in_target_dir" {
  path       = "${local.output_dir}/targets/directory/inside.txt"
  content    = "File inside target directory"
  depends_on = [filemanager_directory.target_dir]
}

# -----------------------------------------------------------------------------
# BASIC SYMLINKS
# -----------------------------------------------------------------------------

# Case 1: Basic file symlink (absolute path)
resource "filemanager_symlink" "basic_absolute" {
  path   = "${local.output_dir}/links/basic_absolute.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 2: Symlink to directory (absolute)
resource "filemanager_symlink" "dir_absolute" {
  path   = "${local.output_dir}/links/dir_link"
  target = filemanager_directory.target_dir.path

  create_parent_dirs = true
  depends_on         = [filemanager_directory.target_dir]
}

# Case 3: Relative symlink - same directory
resource "filemanager_symlink" "relative_same_dir" {
  path        = "${local.output_dir}/targets/link_to_file.txt"
  target      = "file.txt"
  target_type = "relative"

  depends_on = [filemanager_file.target_file]
}

# Case 4: Relative symlink - parent directory
resource "filemanager_symlink" "relative_parent" {
  path        = "${local.output_dir}/targets/subdir/link_to_parent.txt"
  target      = "../file.txt"
  target_type = "relative"

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 5: Relative symlink - nested
resource "filemanager_symlink" "relative_nested" {
  path        = "${local.output_dir}/links/nested/deep/link.txt"
  target      = "../../../targets/file.txt"
  target_type = "relative"

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# -----------------------------------------------------------------------------
# MULTIPLE SYMLINKS TO SAME TARGET
# -----------------------------------------------------------------------------

# Case 6: Multiple links to same file
resource "filemanager_symlink" "multi_link_1" {
  path   = "${local.output_dir}/multi/link1.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

resource "filemanager_symlink" "multi_link_2" {
  path   = "${local.output_dir}/multi/link2.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

resource "filemanager_symlink" "multi_link_3" {
  path   = "${local.output_dir}/multi/link3.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# -----------------------------------------------------------------------------
# SYMLINK CHAINS (link to link)
# -----------------------------------------------------------------------------

# Case 7: Chain of symlinks
resource "filemanager_symlink" "chain_1" {
  path   = "${local.output_dir}/chain/link1.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

resource "filemanager_symlink" "chain_2" {
  path   = "${local.output_dir}/chain/link2.txt"
  target = filemanager_symlink.chain_1.path

  depends_on = [filemanager_symlink.chain_1]
}

resource "filemanager_symlink" "chain_3" {
  path   = "${local.output_dir}/chain/link3.txt"
  target = filemanager_symlink.chain_2.path

  depends_on = [filemanager_symlink.chain_2]
}

# -----------------------------------------------------------------------------
# SPECIAL NAMES
# -----------------------------------------------------------------------------

# Case 8: Symlink with spaces in name
resource "filemanager_symlink" "spaces_in_name" {
  path   = "${local.output_dir}/special/link with spaces.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 9: Hidden symlink
resource "filemanager_symlink" "hidden" {
  path   = "${local.output_dir}/special/.hidden_link"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 10: Symlink with extension
resource "filemanager_symlink" "with_extension" {
  path   = "${local.output_dir}/special/link.lnk"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 11: No extension symlink to file with extension
resource "filemanager_symlink" "no_ext_to_ext" {
  path   = "${local.output_dir}/special/no_extension"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# -----------------------------------------------------------------------------
# CROSS-DIRECTORY SYMLINKS
# -----------------------------------------------------------------------------

# Case 12: Link in different directory
resource "filemanager_symlink" "cross_dir" {
  path   = "${local.output_dir}/other_location/link.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# Case 13: Deep path link to shallow target
resource "filemanager_symlink" "deep_to_shallow" {
  path   = "${local.output_dir}/very/deep/nested/path/link.txt"
  target = filemanager_file.target_file.path

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}

# -----------------------------------------------------------------------------
# DIRECTORY SYMLINKS
# -----------------------------------------------------------------------------

# Case 14: Multiple directory symlinks
resource "filemanager_symlink" "dir_link_1" {
  path   = "${local.output_dir}/dir_links/dir1"
  target = filemanager_directory.target_dir.path

  create_parent_dirs = true
  depends_on         = [filemanager_directory.target_dir]
}

resource "filemanager_symlink" "dir_link_2" {
  path   = "${local.output_dir}/dir_links/dir2"
  target = filemanager_directory.target_dir.path

  create_parent_dirs = true
  depends_on         = [filemanager_directory.target_dir]
}

# Case 15: Symlink to parent (be careful - can create issues if not handled)
resource "filemanager_symlink" "link_to_sibling" {
  path   = "${local.output_dir}/siblings/link_a"
  target = "${local.output_dir}/targets"

  create_parent_dirs = true
  depends_on         = [filemanager_file.target_file]
}
