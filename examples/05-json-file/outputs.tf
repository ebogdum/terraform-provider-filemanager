# =============================================================================
# JSON FILE RESOURCE - OUTPUTS
# =============================================================================

output "basic_json_files" {
  description = "Basic JSON file tests"
  value = {
    simple = {
      path     = filemanager_json_file.simple.path
      rendered = filemanager_json_file.simple.rendered
    }
    nested = {
      path = filemanager_json_file.nested.path
    }
    array_primitives = {
      path = filemanager_json_file.array_primitives.path
    }
    array_objects = {
      path = filemanager_json_file.array_objects.path
    }
    mixed_types = {
      path = filemanager_json_file.mixed_types.path
    }
    empty_object = {
      path = filemanager_json_file.empty_object.path
    }
    empty_array = {
      path = filemanager_json_file.empty_array.path
    }
  }
}

output "merge_results" {
  description = "Merge strategy test results"
  value = {
    deep = {
      path     = filemanager_json_file.merge_deep.path
      rendered = filemanager_json_file.merge_deep.rendered
    }
    replace = {
      path     = filemanager_json_file.merge_replace.path
      rendered = filemanager_json_file.merge_replace.rendered
    }
    append = {
      path     = filemanager_json_file.merge_append.path
      rendered = filemanager_json_file.merge_append.rendered
    }
    complex = {
      path     = filemanager_json_file.merge_complex.path
      rendered = filemanager_json_file.merge_complex.rendered
    }
  }
}

output "formatting_results" {
  description = "Formatting option test results"
  value = {
    sorted   = filemanager_json_file.sorted.path
    unsorted = filemanager_json_file.unsorted.path
    indent_4 = filemanager_json_file.indent_4.path
    indent_1 = filemanager_json_file.indent_1.path
    compact  = filemanager_json_file.compact.path
  }
}

output "config_files" {
  description = "Real-world config file tests"
  value = {
    package_json    = filemanager_json_file.package_json.path
    tsconfig        = filemanager_json_file.tsconfig.path
    api_response    = filemanager_json_file.api_response.path
    vscode_settings = filemanager_json_file.vscode_settings.path
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_json_files = 25
    categories = [
      "basic_structures",
      "merge_strategies",
      "formatting_options",
      "real_world_configs",
      "edge_cases"
    ]
  }
}
