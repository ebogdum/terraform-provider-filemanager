# PROVIDER FUNCTIONS - OUTPUTS

# =============================================================================
# PATH_JOIN FUNCTION OUTPUTS
# =============================================================================

output "path_join" {
  value = {
    basic    = local.path_join_basic
    three    = local.path_join_three
    many     = local.path_join_many
    relative = local.path_join_relative
    trailing = local.path_join_trailing
    empty    = local.path_join_empty
    single   = local.path_join_single
    dots     = local.path_join_dots
  }
  description = "Results of path_join function tests"
}

# =============================================================================
# PATH_DIRNAME FUNCTION OUTPUTS
# =============================================================================

output "path_dirname" {
  value = {
    basic    = local.dirname_basic
    dir      = local.dirname_dir
    root     = local.dirname_root
    deep     = local.dirname_deep
    relative = local.dirname_relative
    dots     = local.dirname_dots
  }
  description = "Results of path_dirname function tests"
}

# =============================================================================
# PATH_BASENAME FUNCTION OUTPUTS
# =============================================================================

output "path_basename" {
  value = {
    basic      = local.basename_basic
    dir        = local.basename_dir
    ext        = local.basename_ext
    deep       = local.basename_deep
    relative   = local.basename_relative
    multi_dots = local.basename_multi_dots
  }
  description = "Results of path_basename function tests"
}

# =============================================================================
# PATH_EXT FUNCTION OUTPUTS
# =============================================================================

output "path_ext" {
  value = {
    txt         = local.ext_txt
    json        = local.ext_json
    sh          = local.ext_sh
    tar_gz      = local.ext_tar_gz
    none        = local.ext_none
    hidden      = local.ext_hidden
    hidden_none = local.ext_hidden_none
    multi       = local.ext_multi
  }
  description = "Results of path_ext function tests"
}

# =============================================================================
# PATH_EXPAND FUNCTION OUTPUTS
# =============================================================================

output "path_expand" {
  value = {
    tilde         = local.expand_tilde
    tilde_sub     = local.expand_tilde_sub
    absolute      = local.expand_absolute
    relative      = local.expand_relative
    tilde_only    = local.expand_tilde_only
    tilde_complex = local.expand_tilde_complex
  }
  description = "Results of path_expand function tests"
}

# =============================================================================
# FILE_EXISTS FUNCTION OUTPUTS
# =============================================================================

output "file_exists" {
  value = {
    module   = local.file_exists_module
    no       = local.file_exists_no
    root     = local.file_exists_root
    relative = local.file_exists_relative
  }
  description = "Results of file_exists function tests"
}

# =============================================================================
# DIR_EXISTS FUNCTION OUTPUTS
# =============================================================================

output "dir_exists" {
  value = {
    module   = local.dir_exists_module
    no       = local.dir_exists_no
    root     = local.dir_exists_root
    tmp      = local.dir_exists_tmp
    relative = local.dir_exists_relative
  }
  description = "Results of dir_exists function tests"
}

# =============================================================================
# GLOB FUNCTION OUTPUTS
# =============================================================================

output "glob" {
  value = {
    tf_files   = local.glob_tf
    tf_count   = length(local.glob_tf)
    none_files = local.glob_none
    none_count = length(local.glob_none)
    dirs       = local.glob_dirs
    all_files  = local.glob_all
    all_count  = length(local.glob_all)
  }
  description = "Results of glob function tests"
}

# =============================================================================
# COMBINED USAGE OUTPUTS
# =============================================================================

output "combined" {
  value = {
    join_dirname    = local.combined_join_dirname
    join_basename   = local.combined_join_basename
    join_ext        = local.combined_join_ext
    built_path      = local.built_path
    computed_exists = local.computed_exists
  }
  description = "Results of combined function usage tests"
}

# =============================================================================
# RESULTS FILE
# =============================================================================

output "results_file" {
  value = {
    path = filemanager_file.results.path
  }
  description = "Path to the results JSON file"
}

# =============================================================================
# SUMMARY
# =============================================================================

output "summary" {
  value = {
    total_test_cases = 52
    functions = {
      path_join     = 8
      path_dirname  = 6
      path_basename = 6
      path_ext      = 8
      path_expand   = 6
      file_exists   = 4
      dir_exists    = 5
      glob          = 4
      combined      = 5
    }
    categories = [
      "path_manipulation",
      "existence_checks",
      "file_globbing",
      "combined_usage"
    ]
  }
}
