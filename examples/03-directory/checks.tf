# =============================================================================
# VERIFICATION CHECKS - Directory resource validation
# =============================================================================

check "verify_basic_directory" {
  data "filemanager_stat" "basic_check" {
    path = filemanager_directory.basic.path
  }

  assert {
    condition     = data.filemanager_stat.basic_check.exists == true
    error_message = "Basic directory does not exist"
  }

  assert {
    condition     = data.filemanager_stat.basic_check.is_dir == true
    error_message = "Basic path should be a directory"
  }
}

check "verify_restricted_permissions" {
  data "filemanager_stat" "restricted_check" {
    path = filemanager_directory.restricted.path
  }

  assert {
    condition     = data.filemanager_stat.restricted_check.mode == "0700"
    error_message = "Restricted directory should have mode 0700, got ${data.filemanager_stat.restricted_check.mode}"
  }
}

check "verify_readonly_permissions" {
  data "filemanager_stat" "readonly_check" {
    path = filemanager_directory.readonly.path
  }

  assert {
    condition     = data.filemanager_stat.readonly_check.mode == "0555"
    error_message = "Readonly directory should have mode 0555, got ${data.filemanager_stat.readonly_check.mode}"
  }
}

check "verify_full_permissions" {
  data "filemanager_stat" "full_perms_check" {
    path = filemanager_directory.full_perms.path
  }

  assert {
    condition     = data.filemanager_stat.full_perms_check.mode == "0777"
    error_message = "Full perms directory should have mode 0777, got ${data.filemanager_stat.full_perms_check.mode}"
  }
}

check "verify_deeply_nested" {
  data "filemanager_stat" "nested_deep_check" {
    path = filemanager_directory.nested_deep.path
  }

  assert {
    condition     = data.filemanager_stat.nested_deep_check.exists == true
    error_message = "Deeply nested directory does not exist"
  }

  assert {
    condition     = data.filemanager_stat.nested_deep_check.is_dir == true
    error_message = "Deeply nested path should be a directory"
  }
}

check "verify_sibling_a" {
  data "filemanager_stat" "sibling_a_check" {
    path = filemanager_directory.siblings_a.path
  }

  assert {
    condition     = data.filemanager_stat.sibling_a_check.exists == true
    error_message = "Sibling directory A does not exist"
  }
}

check "verify_sibling_b" {
  data "filemanager_stat" "sibling_b_check" {
    path = filemanager_directory.siblings_b.path
  }

  assert {
    condition     = data.filemanager_stat.sibling_b_check.exists == true
    error_message = "Sibling directory B does not exist"
  }
}

check "verify_sibling_c" {
  data "filemanager_stat" "sibling_c_check" {
    path = filemanager_directory.siblings_c.path
  }

  assert {
    condition     = data.filemanager_stat.sibling_c_check.exists == true
    error_message = "Sibling directory C does not exist"
  }
}

check "verify_spaces_in_name" {
  data "filemanager_stat" "spaces_check" {
    path = filemanager_directory.with_spaces.path
  }

  assert {
    condition     = data.filemanager_stat.spaces_check.exists == true
    error_message = "Directory with spaces in name does not exist"
  }
}

check "verify_hidden_directory" {
  data "filemanager_stat" "hidden_check" {
    path = filemanager_directory.hidden.path
  }

  assert {
    condition     = data.filemanager_stat.hidden_check.exists == true
    error_message = "Hidden directory does not exist"
  }
}

check "verify_project_structure" {
  data "filemanager_stat" "project_src_check" {
    path = filemanager_directory.project_src.path
  }

  assert {
    condition     = data.filemanager_stat.project_src_check.exists == true
    error_message = "Project src directory does not exist"
  }
}

check "verify_permission_750" {
  data "filemanager_stat" "perm_750_check" {
    path = filemanager_directory.perm_750.path
  }

  assert {
    condition     = data.filemanager_stat.perm_750_check.mode == "0750"
    error_message = "Directory should have mode 0750, got ${data.filemanager_stat.perm_750_check.mode}"
  }
}
