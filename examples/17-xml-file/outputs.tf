# =============================================================================
# XML FILE RESOURCE - OUTPUTS
# =============================================================================

output "basic_xml_files" {
  description = "Basic XML file tests"
  value = {
    simple = {
      path     = filemanager_xml_file.simple.path
      md5      = filemanager_xml_file.simple.md5
      rendered = filemanager_xml_file.simple.rendered
    }
    with_attributes = {
      path = filemanager_xml_file.with_attributes.path
      md5  = filemanager_xml_file.with_attributes.md5
    }
    nested = {
      path = filemanager_xml_file.nested.path
      md5  = filemanager_xml_file.nested.md5
    }
  }
}

output "config_files" {
  description = "Configuration file tests"
  value = {
    pom = {
      path = filemanager_xml_file.pom.path
      md5  = filemanager_xml_file.pom.md5
    }
    spring_beans = {
      path = filemanager_xml_file.spring_beans.path
      md5  = filemanager_xml_file.spring_beans.md5
    }
  }
}

output "formatting_results" {
  description = "Formatting option test results"
  value = {
    compact       = filemanager_xml_file.compact.path
    custom_indent = filemanager_xml_file.custom_indent.path
  }
}

output "edge_cases" {
  description = "Edge case tests"
  value = {
    empty_root = {
      path = filemanager_xml_file.empty_root.path
      md5  = filemanager_xml_file.empty_root.md5
    }
    special_chars = {
      path = filemanager_xml_file.special_chars.path
      md5  = filemanager_xml_file.special_chars.md5
    }
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_xml_files = 9
    categories = [
      "basic_structures",
      "config_files",
      "formatting_options",
      "edge_cases"
    ]
  }
}
