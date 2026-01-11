# =============================================================================
# DATA SOURCES - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/13-data-sources"
  source_dir = "${path.module}/../../test/output/13-data-sources/source"
}

# -----------------------------------------------------------------------------
# SOURCE FILES FOR DATA SOURCE TESTING
# -----------------------------------------------------------------------------

resource "filemanager_directory" "source" {
  path           = local.source_dir
  create_parents = true
}

# Text files
resource "filemanager_file" "text_simple" {
  path               = "${local.source_dir}/simple.txt"
  content            = "Simple text content"
  file_permission    = "0644"
  create_parent_dirs = true
}

resource "filemanager_file" "text_multiline" {
  path               = "${local.source_dir}/multiline.txt"
  content            = "Line 1\nLine 2\nLine 3\nLine 4\nLine 5"
  file_permission    = "0644"
  create_parent_dirs = true
}

resource "filemanager_file" "text_large" {
  path               = "${local.source_dir}/large.txt"
  content            = join("\n", [for i in range(100) : "Line ${i}: This is a test line with some content."])
  file_permission    = "0644"
  create_parent_dirs = true
}

# Config files
resource "filemanager_file" "config_json" {
  path               = "${local.source_dir}/config.json"
  content            = jsonencode({ key = "value", nested = { field = "data" } })
  file_permission    = "0644"
  create_parent_dirs = true
}

resource "filemanager_file" "config_yaml" {
  path               = "${local.source_dir}/config.yaml"
  content            = "key: value\nnested:\n  field: data"
  file_permission    = "0644"
  create_parent_dirs = true
}

resource "filemanager_file" "config_toml" {
  path               = "${local.source_dir}/config.toml"
  content            = "[section]\nkey = \"value\""
  file_permission    = "0644"
  create_parent_dirs = true
}

# Scripts with different permissions
resource "filemanager_file" "script_sh" {
  path               = "${local.source_dir}/scripts/run.sh"
  content            = "#!/bin/bash\necho 'Running...'"
  file_permission    = "0755"
  create_parent_dirs = true
}

resource "filemanager_file" "script_py" {
  path               = "${local.source_dir}/scripts/run.py"
  content            = "#!/usr/bin/env python3\nprint('Running...')"
  file_permission    = "0755"
  create_parent_dirs = true
}

# Nested structure
resource "filemanager_file" "nested_level1" {
  path               = "${local.source_dir}/level1/file1.txt"
  content            = "Level 1 content"
  create_parent_dirs = true
}

resource "filemanager_file" "nested_level2" {
  path               = "${local.source_dir}/level1/level2/file2.txt"
  content            = "Level 2 content"
  create_parent_dirs = true
}

resource "filemanager_file" "nested_level3" {
  path               = "${local.source_dir}/level1/level2/level3/file3.txt"
  content            = "Level 3 content"
  create_parent_dirs = true
}

# Log files
resource "filemanager_file" "log1" {
  path               = "${local.source_dir}/logs/app.log"
  content            = "[INFO] Application started\n[DEBUG] Processing..."
  create_parent_dirs = true
}

resource "filemanager_file" "log2" {
  path               = "${local.source_dir}/logs/error.log"
  content            = "[ERROR] Something went wrong"
  create_parent_dirs = true
}

# Unicode content
resource "filemanager_file" "unicode" {
  path               = "${local.source_dir}/unicode.txt"
  content            = "日本語 中文 한국어 العربية"
  create_parent_dirs = true
}

# Empty file
resource "filemanager_file" "empty" {
  path               = "${local.source_dir}/empty.txt"
  content            = ""
  create_parent_dirs = true
}

# Symlink
resource "filemanager_symlink" "link" {
  path   = "${local.source_dir}/link_to_simple.txt"
  target = filemanager_file.text_simple.path

  depends_on = [filemanager_file.text_simple]
}

# =============================================================================
# FILE DATA SOURCE
# =============================================================================

# Case 1: Read simple text file
data "filemanager_file" "simple" {
  path = filemanager_file.text_simple.path

  depends_on = [filemanager_file.text_simple]
}

# Case 2: Read multiline text file
data "filemanager_file" "multiline" {
  path = filemanager_file.text_multiline.path

  depends_on = [filemanager_file.text_multiline]
}

# Case 3: Read large file
data "filemanager_file" "large" {
  path = filemanager_file.text_large.path

  depends_on = [filemanager_file.text_large]
}

# Case 4: Read JSON config
data "filemanager_file" "json" {
  path = filemanager_file.config_json.path

  depends_on = [filemanager_file.config_json]
}

# Case 5: Read YAML config
data "filemanager_file" "yaml" {
  path = filemanager_file.config_yaml.path

  depends_on = [filemanager_file.config_yaml]
}

# Case 6: Read executable script
data "filemanager_file" "executable" {
  path = filemanager_file.script_sh.path

  depends_on = [filemanager_file.script_sh]
}

# Case 7: Read unicode file
data "filemanager_file" "unicode" {
  path = filemanager_file.unicode.path

  depends_on = [filemanager_file.unicode]
}

# Case 8: Read empty file
data "filemanager_file" "empty" {
  path = filemanager_file.empty.path

  depends_on = [filemanager_file.empty]
}

# Case 9: Read deeply nested file
data "filemanager_file" "nested" {
  path = filemanager_file.nested_level3.path

  depends_on = [filemanager_file.nested_level3]
}

# =============================================================================
# CHECKSUM DATA SOURCE
# =============================================================================

# Case 10: MD5 checksum
data "filemanager_checksum" "md5" {
  path      = filemanager_file.text_simple.path
  algorithm = "md5"

  depends_on = [filemanager_file.text_simple]
}

# Case 11: SHA1 checksum
data "filemanager_checksum" "sha1" {
  path      = filemanager_file.text_simple.path
  algorithm = "sha1"

  depends_on = [filemanager_file.text_simple]
}

# Case 12: SHA256 checksum (default)
data "filemanager_checksum" "sha256" {
  path      = filemanager_file.text_simple.path
  algorithm = "sha256"

  depends_on = [filemanager_file.text_simple]
}

# Case 13: SHA512 checksum
data "filemanager_checksum" "sha512" {
  path      = filemanager_file.text_simple.path
  algorithm = "sha512"

  depends_on = [filemanager_file.text_simple]
}

# Case 14: Checksum of large file
data "filemanager_checksum" "large_file" {
  path      = filemanager_file.text_large.path
  algorithm = "sha256"

  depends_on = [filemanager_file.text_large]
}

# Case 15: Checksum of empty file
data "filemanager_checksum" "empty_file" {
  path      = filemanager_file.empty.path
  algorithm = "sha256"

  depends_on = [filemanager_file.empty]
}

# Case 16: Checksum of JSON file
data "filemanager_checksum" "json_file" {
  path      = filemanager_file.config_json.path
  algorithm = "sha256"

  depends_on = [filemanager_file.config_json]
}

# =============================================================================
# STAT DATA SOURCE
# =============================================================================

# Case 17: Stat regular file
data "filemanager_stat" "regular_file" {
  path = filemanager_file.text_simple.path

  depends_on = [filemanager_file.text_simple]
}

# Case 18: Stat directory
data "filemanager_stat" "directory" {
  path = local.source_dir

  depends_on = [filemanager_directory.source]
}

# Case 19: Stat executable
data "filemanager_stat" "executable" {
  path = filemanager_file.script_sh.path

  depends_on = [filemanager_file.script_sh]
}

# Case 20: Stat symlink
data "filemanager_stat" "symlink" {
  path = filemanager_symlink.link.path

  depends_on = [filemanager_symlink.link]
}

# Case 21: Stat large file
data "filemanager_stat" "large" {
  path = filemanager_file.text_large.path

  depends_on = [filemanager_file.text_large]
}

# Case 22: Stat empty file
data "filemanager_stat" "empty" {
  path = filemanager_file.empty.path

  depends_on = [filemanager_file.empty]
}

# Case 23: Stat nested file
data "filemanager_stat" "nested" {
  path = filemanager_file.nested_level3.path

  depends_on = [filemanager_file.nested_level3]
}

# Case 24: Stat non-existent path
data "filemanager_stat" "nonexistent" {
  path = "${local.source_dir}/does_not_exist.txt"

  depends_on = [filemanager_directory.source]
}

# =============================================================================
# FILES DATA SOURCE (Glob)
# =============================================================================

# Case 25: List all files in directory (no pattern)
data "filemanager_files" "all" {
  path = local.source_dir

  depends_on = [
    filemanager_file.text_simple,
    filemanager_file.text_multiline,
    filemanager_file.config_json,
  ]
}

# Case 26: List with pattern - text files
data "filemanager_files" "txt_files" {
  path    = local.source_dir
  pattern = "*.txt"

  depends_on = [filemanager_file.text_simple, filemanager_file.text_multiline]
}

# Case 27: List with pattern - config files
data "filemanager_files" "json_files" {
  path    = local.source_dir
  pattern = "*.json"

  depends_on = [filemanager_file.config_json]
}

# Case 28: List with pattern - YAML files
data "filemanager_files" "yaml_files" {
  path    = local.source_dir
  pattern = "*.yaml"

  depends_on = [filemanager_file.config_yaml]
}

# Case 29: List recursively
data "filemanager_files" "recursive" {
  path      = local.source_dir
  recursive = true

  depends_on = [
    filemanager_file.nested_level1,
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 30: List recursively with pattern
data "filemanager_files" "recursive_txt" {
  path      = local.source_dir
  pattern   = "*.txt"
  recursive = true

  depends_on = [
    filemanager_file.text_simple,
    filemanager_file.nested_level1,
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 31: List scripts subdirectory
data "filemanager_files" "scripts" {
  path = "${local.source_dir}/scripts"

  depends_on = [filemanager_file.script_sh, filemanager_file.script_py]
}

# Case 32: List with shell pattern - *.sh
data "filemanager_files" "shell_scripts" {
  path    = "${local.source_dir}/scripts"
  pattern = "*.sh"

  depends_on = [filemanager_file.script_sh]
}

# Case 33: List logs subdirectory
data "filemanager_files" "logs" {
  path    = "${local.source_dir}/logs"
  pattern = "*.log"

  depends_on = [filemanager_file.log1, filemanager_file.log2]
}

# =============================================================================
# DIRECTORY DATA SOURCE
# =============================================================================

# Case 34: List root source directory
data "filemanager_directory" "source" {
  path = local.source_dir

  depends_on = [filemanager_file.text_simple, filemanager_file.config_json]
}

# Case 35: List with pattern - text files
data "filemanager_directory" "txt" {
  path    = local.source_dir
  pattern = "*.txt"

  depends_on = [filemanager_file.text_simple, filemanager_file.text_multiline]
}

# Case 36: List with pattern - JSON
data "filemanager_directory" "json" {
  path    = local.source_dir
  pattern = "*.json"

  depends_on = [filemanager_file.config_json]
}

# Case 37: List recursively
data "filemanager_directory" "recursive" {
  path      = local.source_dir
  recursive = true

  depends_on = [
    filemanager_file.text_simple,
    filemanager_file.nested_level1,
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 38: List recursively with pattern
data "filemanager_directory" "recursive_txt" {
  path      = local.source_dir
  pattern   = "*.txt"
  recursive = true

  depends_on = [
    filemanager_file.text_simple,
    filemanager_file.nested_level1,
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 39: List scripts directory
data "filemanager_directory" "scripts" {
  path = "${local.source_dir}/scripts"

  depends_on = [filemanager_file.script_sh, filemanager_file.script_py]
}

# Case 40: List logs directory with pattern
data "filemanager_directory" "logs" {
  path    = "${local.source_dir}/logs"
  pattern = "*.log"

  depends_on = [filemanager_file.log1, filemanager_file.log2]
}

# Case 41: List deeply nested level1
data "filemanager_directory" "level1" {
  path      = "${local.source_dir}/level1"
  recursive = true

  depends_on = [
    filemanager_file.nested_level1,
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 42: List deeply nested level2
data "filemanager_directory" "level2" {
  path      = "${local.source_dir}/level1/level2"
  recursive = true

  depends_on = [
    filemanager_file.nested_level2,
    filemanager_file.nested_level3,
  ]
}

# Case 43: List deeply nested level3
data "filemanager_directory" "level3" {
  path = "${local.source_dir}/level1/level2/level3"

  depends_on = [filemanager_file.nested_level3]
}

