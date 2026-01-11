# =============================================================================
# VERIFICATION CHECKS - Data sources validation
# =============================================================================

# The data sources test is unique - it creates files then reads them via data sources
# We verify the data sources return correct information

check "verify_file_datasource" {
  data "filemanager_file" "verify_simple" {
    path = filemanager_file.text_simple.path
  }

  assert {
    condition     = data.filemanager_file.verify_simple.content == "Simple text content"
    error_message = "File data source should return correct content"
  }

  assert {
    condition     = data.filemanager_file.verify_simple.size == 19
    error_message = "File data source should return correct size"
  }
}

check "verify_stat_datasource" {
  data "filemanager_stat" "verify_script" {
    path = filemanager_file.script_sh.path
  }

  assert {
    condition     = data.filemanager_stat.verify_script.exists == true
    error_message = "Stat should report file exists"
  }

  assert {
    condition     = data.filemanager_stat.verify_script.is_file == true
    error_message = "Stat should report it's a file"
  }

  assert {
    condition     = data.filemanager_stat.verify_script.mode == "0755"
    error_message = "Stat should report correct permissions"
  }
}

check "verify_checksum_datasource" {
  data "filemanager_checksum" "verify_checksum" {
    path      = filemanager_file.text_simple.path
    algorithm = "sha256"
  }

  assert {
    condition     = data.filemanager_checksum.verify_checksum.checksum != ""
    error_message = "Checksum data source should return non-empty checksum"
  }
}

check "verify_directory_datasource" {
  data "filemanager_stat" "verify_dir" {
    path = filemanager_directory.source.path
  }

  assert {
    condition     = data.filemanager_stat.verify_dir.is_dir == true
    error_message = "Directory should be recognized as directory"
  }
}

check "verify_json_content" {
  data "filemanager_file" "verify_json" {
    path = filemanager_file.config_json.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.verify_json.content, "\"key\"")
    error_message = "JSON file content should be accessible"
  }
}
