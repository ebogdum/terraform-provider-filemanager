# =============================================================================
# PROVIDER FUNCTIONS - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/14-functions"
  source_dir = "${path.module}/../../test/output/14-functions/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR FUNCTION TESTING
# -----------------------------------------------------------------------------

resource "filemanager_directory" "source" {
  path           = local.source_dir
  create_parents = true
}

resource "filemanager_file" "test_file" {
  path               = "${local.source_dir}/test.txt"
  content            = "Test content"
  create_parent_dirs = true
}

resource "filemanager_file" "config_json" {
  path               = "${local.source_dir}/config.json"
  content            = jsonencode({ key = "value" })
  create_parent_dirs = true
}

resource "filemanager_file" "script_sh" {
  path               = "${local.source_dir}/scripts/run.sh"
  content            = "#!/bin/bash\necho 'Hello'"
  create_parent_dirs = true
}

resource "filemanager_file" "nested_file" {
  path               = "${local.source_dir}/level1/level2/level3/deep.txt"
  content            = "Deeply nested"
  create_parent_dirs = true
}

resource "filemanager_directory" "empty_dir" {
  path           = "${local.source_dir}/empty"
  create_parents = true
}

# =============================================================================
# PATH_JOIN FUNCTION
# =============================================================================

# Uses static paths (provider functions are evaluated during plan)

locals {
  # Case 1: Basic path join - two parts
  path_join_basic = provider::filemanager::path_join(["/home", "user"])

  # Case 2: Path join - three parts
  path_join_three = provider::filemanager::path_join(["/home", "user", "documents"])

  # Case 3: Path join - many parts
  path_join_many = provider::filemanager::path_join(["/", "var", "lib", "app", "data", "file.txt"])

  # Case 4: Path join - relative path
  path_join_relative = provider::filemanager::path_join(["config", "app.json"])

  # Case 5: Path join - with trailing slash
  path_join_trailing = provider::filemanager::path_join(["/home/", "user/", "file.txt"])

  # Case 6: Path join - empty parts
  path_join_empty = provider::filemanager::path_join(["home", "", "user"])

  # Case 7: Path join - single part
  path_join_single = provider::filemanager::path_join(["/home"])

  # Case 8: Path join - with dots
  path_join_dots = provider::filemanager::path_join(["/home", "user", "..", "admin"])
}

# =============================================================================
# PATH_DIRNAME FUNCTION
# =============================================================================

locals {
  # Case 9: Basic dirname
  dirname_basic = provider::filemanager::path_dirname("/home/user/file.txt")

  # Case 10: Dirname of directory path
  dirname_dir = provider::filemanager::path_dirname("/home/user/documents/")

  # Case 11: Dirname of root-level file
  dirname_root = provider::filemanager::path_dirname("/file.txt")

  # Case 12: Dirname of deeply nested path
  dirname_deep = provider::filemanager::path_dirname("/a/b/c/d/e/f/g/file.txt")

  # Case 13: Dirname of relative path
  dirname_relative = provider::filemanager::path_dirname("config/app.json")

  # Case 14: Dirname with dots
  dirname_dots = provider::filemanager::path_dirname("/home/user/../admin/file.txt")
}

# =============================================================================
# PATH_BASENAME FUNCTION
# =============================================================================

locals {
  # Case 15: Basic basename
  basename_basic = provider::filemanager::path_basename("/home/user/file.txt")

  # Case 16: Basename of directory path
  basename_dir = provider::filemanager::path_basename("/home/user/documents")

  # Case 17: Basename with extension
  basename_ext = provider::filemanager::path_basename("/path/to/script.sh")

  # Case 18: Basename of deeply nested path
  basename_deep = provider::filemanager::path_basename("/a/b/c/d/e/f/g/file.txt")

  # Case 19: Basename of relative path
  basename_relative = provider::filemanager::path_basename("config/app.json")

  # Case 20: Basename with multiple dots
  basename_multi_dots = provider::filemanager::path_basename("/path/to/archive.tar.gz")
}

# =============================================================================
# PATH_EXT FUNCTION
# =============================================================================

locals {
  # Case 21: Basic extension
  ext_txt = provider::filemanager::path_ext("/path/to/file.txt")

  # Case 22: JSON extension
  ext_json = provider::filemanager::path_ext("/path/to/config.json")

  # Case 23: Script extension
  ext_sh = provider::filemanager::path_ext("/path/to/script.sh")

  # Case 24: Double extension
  ext_tar_gz = provider::filemanager::path_ext("/path/to/archive.tar.gz")

  # Case 25: No extension
  ext_none = provider::filemanager::path_ext("/path/to/Makefile")

  # Case 26: Hidden file with extension
  ext_hidden = provider::filemanager::path_ext("/path/to/.gitignore")

  # Case 27: Hidden file no extension
  ext_hidden_none = provider::filemanager::path_ext("/path/to/.env")

  # Case 28: Multiple dots
  ext_multi = provider::filemanager::path_ext("/path/to/file.backup.2024.txt")
}

# =============================================================================
# PATH_EXPAND FUNCTION
# =============================================================================

locals {
  # Case 29: Expand tilde
  expand_tilde = provider::filemanager::path_expand("~/documents")

  # Case 30: Expand tilde with subdirectory
  expand_tilde_sub = provider::filemanager::path_expand("~/projects/app")

  # Case 31: Absolute path (no expansion needed)
  expand_absolute = provider::filemanager::path_expand("/etc/config")

  # Case 32: Relative path (no expansion)
  expand_relative = provider::filemanager::path_expand("./config")

  # Case 33: Tilde only
  expand_tilde_only = provider::filemanager::path_expand("~")

  # Case 34: Complex tilde path
  expand_tilde_complex = provider::filemanager::path_expand("~/a/b/c/d/e")
}

# =============================================================================
# FILE_EXISTS FUNCTION
# =============================================================================

# Using static paths that exist before apply

locals {
  # Case 35: Check existing file (using module path which always exists)
  file_exists_module = provider::filemanager::file_exists("${path.module}/main.tf")

  # Case 36: Check non-existent file
  file_exists_no = provider::filemanager::file_exists("/nonexistent/path/file.txt")

  # Case 37: Check root path (should be false - it's a directory)
  file_exists_root = provider::filemanager::file_exists("/")

  # Case 38: Check with relative path
  file_exists_relative = provider::filemanager::file_exists("main.tf")
}

# =============================================================================
# DIR_EXISTS FUNCTION
# =============================================================================

locals {
  # Case 39: Check existing directory (module directory)
  dir_exists_module = provider::filemanager::dir_exists(path.module)

  # Case 40: Check non-existent directory
  dir_exists_no = provider::filemanager::dir_exists("/nonexistent/path/dir")

  # Case 41: Check root directory
  dir_exists_root = provider::filemanager::dir_exists("/")

  # Case 42: Check tmp directory
  dir_exists_tmp = provider::filemanager::dir_exists("/tmp")

  # Case 43: Check with relative path
  dir_exists_relative = provider::filemanager::dir_exists(".")
}

# =============================================================================
# GLOB FUNCTION
# =============================================================================

locals {
  # Case 44: Glob for current directory terraform files
  glob_tf = provider::filemanager::glob("${path.module}/*.tf")

  # Case 45: Glob for non-matching pattern
  glob_none = provider::filemanager::glob("${path.module}/*.xyz")

  # Case 46: Glob with directory pattern
  glob_dirs = provider::filemanager::glob("${path.module}/../*")

  # Case 47: Glob for all files in module
  glob_all = provider::filemanager::glob("${path.module}/*")
}

# =============================================================================
# COMBINED USAGE EXAMPLES
# =============================================================================

locals {
  # Case 48: Combine path_join and path_dirname
  combined_join_dirname = provider::filemanager::path_dirname(
    provider::filemanager::path_join(["/home", "user", "documents", "file.txt"])
  )

  # Case 49: Combine path_join and path_basename
  combined_join_basename = provider::filemanager::path_basename(
    provider::filemanager::path_join(["/home", "user", "documents", "file.txt"])
  )

  # Case 50: Combine path_join and path_ext
  combined_join_ext = provider::filemanager::path_ext(
    provider::filemanager::path_join(["/home", "user", "archive.tar.gz"])
  )

  # Case 51: Build path from components
  built_path = provider::filemanager::path_join([
    provider::filemanager::path_dirname("/etc/app/config.json"),
    "new_config.yaml"
  ])

  # Case 52: Check if computed path exists
  computed_exists = provider::filemanager::file_exists(
    provider::filemanager::path_join([path.module, "main.tf"])
  )
}

# =============================================================================
# OUTPUT FILE - Store results as file for verification
# =============================================================================

resource "filemanager_file" "results" {
  path = "${local.output_dir}/function_results.json"
  content = jsonencode({
    path_join = {
      basic    = local.path_join_basic
      three    = local.path_join_three
      many     = local.path_join_many
      relative = local.path_join_relative
      trailing = local.path_join_trailing
      empty    = local.path_join_empty
      single   = local.path_join_single
      dots     = local.path_join_dots
    }
    path_dirname = {
      basic    = local.dirname_basic
      dir      = local.dirname_dir
      root     = local.dirname_root
      deep     = local.dirname_deep
      relative = local.dirname_relative
      dots     = local.dirname_dots
    }
    path_basename = {
      basic      = local.basename_basic
      dir        = local.basename_dir
      ext        = local.basename_ext
      deep       = local.basename_deep
      relative   = local.basename_relative
      multi_dots = local.basename_multi_dots
    }
    path_ext = {
      txt         = local.ext_txt
      json        = local.ext_json
      sh          = local.ext_sh
      tar_gz      = local.ext_tar_gz
      none        = local.ext_none
      hidden      = local.ext_hidden
      hidden_none = local.ext_hidden_none
      multi       = local.ext_multi
    }
    path_expand = {
      tilde         = local.expand_tilde
      tilde_sub     = local.expand_tilde_sub
      absolute      = local.expand_absolute
      relative      = local.expand_relative
      tilde_only    = local.expand_tilde_only
      tilde_complex = local.expand_tilde_complex
    }
    file_exists = {
      module   = local.file_exists_module
      no       = local.file_exists_no
      root     = local.file_exists_root
      relative = local.file_exists_relative
    }
    dir_exists = {
      module   = local.dir_exists_module
      no       = local.dir_exists_no
      root     = local.dir_exists_root
      tmp      = local.dir_exists_tmp
      relative = local.dir_exists_relative
    }
    glob = {
      tf_count   = length(local.glob_tf)
      none_count = length(local.glob_none)
      all_count  = length(local.glob_all)
    }
    combined = {
      join_dirname    = local.combined_join_dirname
      join_basename   = local.combined_join_basename
      join_ext        = local.combined_join_ext
      built_path      = local.built_path
      computed_exists = local.computed_exists
    }
  })
  create_parent_dirs = true

  depends_on = [filemanager_directory.source]
}

