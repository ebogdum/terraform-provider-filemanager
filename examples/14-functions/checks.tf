# =============================================================================
# VERIFICATION CHECKS - Provider functions validation
# =============================================================================

check "verify_source_dir" {
  data "filemanager_stat" "source_dir_check" {
    path = filemanager_directory.source.path
  }

  assert {
    condition     = data.filemanager_stat.source_dir_check.exists == true
    error_message = "Source directory should exist"
  }

  assert {
    condition     = data.filemanager_stat.source_dir_check.is_dir == true
    error_message = "Source should be a directory"
  }
}

check "verify_test_file" {
  data "filemanager_file" "test_file_check" {
    path = filemanager_file.test_file.path
  }

  assert {
    condition     = data.filemanager_file.test_file_check.content == "Test content"
    error_message = "Test file content mismatch"
  }
}

check "verify_config_json" {
  data "filemanager_stat" "config_json_check" {
    path = filemanager_file.config_json.path
  }

  assert {
    condition     = data.filemanager_stat.config_json_check.exists == true
    error_message = "Config JSON file should exist"
  }
}

check "verify_script_sh" {
  data "filemanager_stat" "script_sh_check" {
    path = filemanager_file.script_sh.path
  }

  assert {
    condition     = data.filemanager_stat.script_sh_check.exists == true
    error_message = "Script file should exist"
  }
}

check "verify_nested_file" {
  data "filemanager_file" "nested_file_check" {
    path = filemanager_file.nested_file.path
  }

  assert {
    condition     = data.filemanager_file.nested_file_check.content == "Deeply nested"
    error_message = "Nested file content mismatch"
  }
}

check "verify_empty_dir" {
  data "filemanager_stat" "empty_dir_check" {
    path = filemanager_directory.empty_dir.path
  }

  assert {
    condition     = data.filemanager_stat.empty_dir_check.exists == true
    error_message = "Empty directory should exist"
  }

  assert {
    condition     = data.filemanager_stat.empty_dir_check.is_dir == true
    error_message = "Empty dir should be a directory"
  }
}

check "verify_results_file" {
  data "filemanager_file" "results_check" {
    path = filemanager_file.results.path
  }

  assert {
    condition     = data.filemanager_file.results_check.size > 0
    error_message = "Results file should not be empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.results_check.content, "path_join")
    error_message = "Results should contain path_join results"
  }

  assert {
    condition     = strcontains(data.filemanager_file.results_check.content, "path_dirname")
    error_message = "Results should contain path_dirname results"
  }
}

# Verify function results are correct (static assertions based on expected values)
check "verify_path_join_results" {
  data "filemanager_file" "path_join_check" {
    path = filemanager_file.results.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.path_join_check.content, "/home/user")
    error_message = "path_join basic should produce /home/user"
  }
}

check "verify_file_exists_results" {
  data "filemanager_file" "file_exists_check" {
    path = filemanager_file.results.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.file_exists_check.content, "\"module\":true")
    error_message = "file_exists should return true for module main.tf"
  }

  assert {
    condition     = strcontains(data.filemanager_file.file_exists_check.content, "\"no\":false")
    error_message = "file_exists should return false for non-existent file"
  }
}

check "verify_dir_exists_results" {
  data "filemanager_file" "dir_exists_check" {
    path = filemanager_file.results.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.dir_exists_check.content, "\"root\":true")
    error_message = "dir_exists should return true for root directory"
  }

  assert {
    condition     = strcontains(data.filemanager_file.dir_exists_check.content, "\"tmp\":true")
    error_message = "dir_exists should return true for /tmp directory"
  }
}
