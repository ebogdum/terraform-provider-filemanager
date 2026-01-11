# =============================================================================
# DIRECTORY RESOURCE - OUTPUTS
# =============================================================================

output "basic_directories" {
  description = "Basic directory tests"
  value = {
    basic      = filemanager_directory.basic.path
    with_perms = filemanager_directory.with_perms.path
    restricted = filemanager_directory.restricted.path
    readonly   = filemanager_directory.readonly.path
    full_perms = filemanager_directory.full_perms.path
  }
}

output "nested_directories" {
  description = "Nested directory tests"
  value = {
    single = filemanager_directory.nested_single.path
    deep   = filemanager_directory.nested_deep.path
  }
}

output "sibling_directories" {
  description = "Sibling directories"
  value = {
    a = filemanager_directory.siblings_a.path
    b = filemanager_directory.siblings_b.path
    c = filemanager_directory.siblings_c.path
  }
}

output "special_name_directories" {
  description = "Directories with special names"
  value = {
    with_spaces   = filemanager_directory.with_spaces.path
    with_dots     = filemanager_directory.with_dots.path
    hidden        = filemanager_directory.hidden.path
    with_numbers  = filemanager_directory.with_numbers.path
    special_chars = filemanager_directory.special_chars.path
    long_name     = filemanager_directory.long_name.path
  }
}

output "project_structure" {
  description = "Project-like directory structure"
  value = {
    root   = filemanager_directory.project_root.path
    src    = filemanager_directory.project_src.path
    tests  = filemanager_directory.project_tests.path
    docs   = filemanager_directory.project_docs.path
    config = filemanager_directory.project_config.path
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_directories = 26
    categories = [
      "basic",
      "nested",
      "special_names",
      "project_structures",
      "permissions",
      "force_delete"
    ]
  }
}
