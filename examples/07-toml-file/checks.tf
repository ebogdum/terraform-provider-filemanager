# =============================================================================
# VERIFICATION CHECKS - TOML file resource validation
# =============================================================================

check "verify_simple_toml" {
  data "filemanager_file" "simple_check" {
    path = filemanager_toml_file.simple.path
  }

  assert {
    condition     = data.filemanager_file.simple_check.size > 0
    error_message = "Simple TOML file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "title")
    error_message = "Simple TOML should contain 'title'"
  }
}

check "verify_dotted_toml" {
  data "filemanager_file" "dotted_check" {
    path = filemanager_toml_file.dotted.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.dotted_check.content, "server")
    error_message = "Dotted TOML should contain 'server'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.dotted_check.content, "database")
    error_message = "Dotted TOML should contain 'database'"
  }
}

check "verify_cargo_simple" {
  data "filemanager_file" "cargo_simple_check" {
    path = filemanager_toml_file.cargo_simple.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.cargo_simple_check.content, "package")
    error_message = "Cargo.toml should contain 'package'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.cargo_simple_check.content, "name")
    error_message = "Cargo.toml should contain 'name'"
  }
}

check "verify_cargo_deps" {
  data "filemanager_file" "cargo_deps_check" {
    path = filemanager_toml_file.cargo_deps.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.cargo_deps_check.content, "dependencies")
    error_message = "Cargo.toml should contain 'dependencies'"
  }
}

check "verify_pyproject" {
  data "filemanager_file" "pyproject_check" {
    path = filemanager_toml_file.pyproject_simple.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.pyproject_check.content, "project")
    error_message = "pyproject.toml should contain 'project'"
  }
}

check "verify_hugo_config" {
  data "filemanager_file" "hugo_check" {
    path = filemanager_toml_file.hugo_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.hugo_check.content, "baseURL") || strcontains(data.filemanager_file.hugo_check.content, "base")
    error_message = "Hugo config should contain base URL"
  }
}

# =============================================================================
# FORMAT VALIDATION CHECKS (filemanager_validate)
# =============================================================================

check "validate_simple_toml_format" {
  data "filemanager_validate" "simple_toml" {
    path   = filemanager_toml_file.simple.path
    format = "toml"
  }

  assert {
    condition     = data.filemanager_validate.simple_toml.is_valid == true
    error_message = "Simple TOML should be valid"
  }
}

check "validate_cargo_format" {
  data "filemanager_validate" "cargo_toml" {
    path = filemanager_toml_file.cargo_simple.path
  }

  assert {
    condition     = data.filemanager_validate.cargo_toml.is_valid == true
    error_message = "Cargo.toml should be valid TOML"
  }
}

check "validate_pyproject_format" {
  data "filemanager_validate" "pyproject_toml" {
    path = filemanager_toml_file.pyproject_simple.path
  }

  assert {
    condition     = data.filemanager_validate.pyproject_toml.is_valid == true
    error_message = "pyproject.toml should be valid TOML"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# =============================================================================

check "compare_toml_same_file" {
  data "filemanager_compare" "toml_same" {
    source = filemanager_toml_file.simple.path
    target = filemanager_toml_file.simple.path
  }

  assert {
    condition     = data.filemanager_compare.toml_same.identical == true
    error_message = "Same TOML file compared to itself should be identical"
  }

  assert {
    condition     = data.filemanager_compare.toml_same.checksum_match == true
    error_message = "Same TOML file should have matching checksum"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_toml_time_check" {
  data "filemanager_stat" "toml_time_check" {
    path            = filemanager_toml_file.simple.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.toml_time_check.is_modified_within == true
    error_message = "Newly created TOML file should be modified within last hour"
  }

  assert {
    condition     = data.filemanager_stat.toml_time_check.owner_name != null
    error_message = "TOML file owner name should be resolved"
  }
}
