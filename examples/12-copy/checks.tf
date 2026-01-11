# =============================================================================
# VERIFICATION CHECKS - Copy resource validation
# =============================================================================

check "verify_copy_single" {
  data "filemanager_file" "copy_single_check" {
    path = filemanager_copy.single_basic.destination
  }

  assert {
    condition     = data.filemanager_file.copy_single_check.content == "Content of file 1"
    error_message = "Copied file content should match source"
  }
}

check "verify_copy_rename" {
  data "filemanager_stat" "copy_rename_check" {
    path = filemanager_copy.single_rename.destination
  }

  assert {
    condition     = data.filemanager_stat.copy_rename_check.exists == true
    error_message = "Renamed copy should exist"
  }
}

check "verify_copy_preserve_perms" {
  data "filemanager_stat" "copy_perms_check" {
    path = filemanager_copy.single_executable.destination
  }

  assert {
    condition     = data.filemanager_stat.copy_perms_check.exists == true
    error_message = "Permission-preserved copy should exist"
  }

  assert {
    condition     = data.filemanager_stat.copy_perms_check.mode == "0755"
    error_message = "Copied file should preserve 0755 permissions"
  }
}

check "verify_copy_recursive" {
  data "filemanager_stat" "copy_recursive_check" {
    path = "${filemanager_copy.dir_basic.destination}/file1.txt"
  }

  assert {
    condition     = data.filemanager_stat.copy_recursive_check.exists == true
    error_message = "Recursively copied file1.txt should exist"
  }
}

check "verify_copy_filtered" {
  data "filemanager_stat" "copy_filtered_check" {
    path = "${filemanager_copy.dir_excludes.destination}/file1.txt"
  }

  assert {
    condition     = data.filemanager_stat.copy_filtered_check.exists == true
    error_message = "Filtered copy should include file1.txt"
  }
}

check "verify_copy_nested" {
  data "filemanager_stat" "copy_nested_check" {
    path = "${filemanager_copy.nested_deep.destination}/level1/level2/nested.txt"
  }

  assert {
    condition     = data.filemanager_stat.copy_nested_check.exists == true
    error_message = "Nested file should be copied"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# Verify that copies match their sources
# =============================================================================

check "compare_single_file_copy" {
  data "filemanager_compare" "single_copy" {
    source = filemanager_file.source_file1.path
    target = filemanager_copy.single_basic.destination
  }

  assert {
    condition     = data.filemanager_compare.single_copy.content_match == true
    error_message = "Copied file content should match source"
  }

  assert {
    condition     = data.filemanager_compare.single_copy.checksum_match == true
    error_message = "Copied file checksum should match source"
  }

  assert {
    condition     = data.filemanager_compare.single_copy.size_match == true
    error_message = "Copied file size should match source"
  }
}

check "compare_large_file_copy" {
  data "filemanager_compare" "large_copy" {
    source             = filemanager_file.source_large.path
    target             = filemanager_copy.single_large.destination
    checksum_algorithm = "sha256"
  }

  assert {
    condition     = data.filemanager_compare.large_copy.checksum_match == true
    error_message = "Large file copy checksum should match"
  }

  assert {
    condition     = data.filemanager_compare.large_copy.size_match == true
    error_message = "Large file copy size should match"
  }
}

check "compare_script_copy_permissions" {
  data "filemanager_compare" "script_copy" {
    source       = filemanager_file.source_script.path
    target       = filemanager_copy.single_executable.destination
    compare_mode = true
  }

  assert {
    condition     = data.filemanager_compare.script_copy.mode_match == true
    error_message = "Script file permissions should be preserved after copy"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_copy_time_check" {
  data "filemanager_stat" "copy_time_check" {
    path            = filemanager_copy.single_basic.destination
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.copy_time_check.is_modified_within == true
    error_message = "Newly copied file should be modified within last hour"
  }
}
