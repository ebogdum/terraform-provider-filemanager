# =============================================================================
# VERIFICATION CHECKS - Using data sources to validate created resources
# =============================================================================

# -----------------------------------------------------------------------------
# BASIC FILE CHECKS
# -----------------------------------------------------------------------------

check "verify_basic_file" {
  data "filemanager_file" "basic_check" {
    path = filemanager_file.basic.path
  }

  assert {
    condition     = data.filemanager_file.basic_check.content == "Hello, World!"
    error_message = "Basic file content mismatch: expected 'Hello, World!'"
  }
}

check "verify_empty_file" {
  data "filemanager_stat" "empty_check" {
    path = filemanager_file.empty.path
  }

  assert {
    condition     = data.filemanager_stat.empty_check.exists == true
    error_message = "Empty file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.empty_check.size == 0
    error_message = "Empty file should have size 0"
  }
}

check "verify_multiline_file" {
  data "filemanager_file" "multiline_check" {
    path = filemanager_file.multiline.path
  }

  assert {
    condition     = length(data.filemanager_file.multiline_check.content) > 0
    error_message = "Multiline file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.multiline_check.content, "Line 1")
    error_message = "Multiline file missing 'Line 1'"
  }
}

check "verify_large_file" {
  data "filemanager_stat" "large_check" {
    path = filemanager_file.large.path
  }

  assert {
    condition     = data.filemanager_stat.large_check.exists == true
    error_message = "Large file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.large_check.size > 1000
    error_message = "Large file should be bigger than 1000 bytes"
  }
}

check "verify_special_chars_file" {
  data "filemanager_file" "special_chars_check" {
    path = filemanager_file.special_chars.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.special_chars_check.content, "Unicode:")
    error_message = "Special chars file missing unicode content"
  }
}

# -----------------------------------------------------------------------------
# PERMISSION CHECKS
# -----------------------------------------------------------------------------

check "verify_readonly_permissions" {
  data "filemanager_stat" "readonly_check" {
    path = filemanager_file.readonly.path
  }

  assert {
    condition     = data.filemanager_stat.readonly_check.exists == true
    error_message = "Readonly file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.readonly_check.mode == "0444"
    error_message = "Readonly file should have mode 0444, got ${data.filemanager_stat.readonly_check.mode}"
  }
}

check "verify_owner_only_permissions" {
  data "filemanager_stat" "owner_only_check" {
    path = filemanager_file.owner_only.path
  }

  assert {
    condition     = data.filemanager_stat.owner_only_check.mode == "0600"
    error_message = "Owner-only file should have mode 0600, got ${data.filemanager_stat.owner_only_check.mode}"
  }
}

check "verify_full_permissions" {
  data "filemanager_stat" "full_perms_check" {
    path = filemanager_file.full_perms.path
  }

  assert {
    condition     = data.filemanager_stat.full_perms_check.mode == "0777"
    error_message = "Full perms file should have mode 0777, got ${data.filemanager_stat.full_perms_check.mode}"
  }
}

check "verify_executable_permissions" {
  data "filemanager_stat" "executable_check" {
    path = filemanager_file.executable.path
  }

  assert {
    condition     = data.filemanager_stat.executable_check.mode == "0755"
    error_message = "Executable file should have mode 0755, got ${data.filemanager_stat.executable_check.mode}"
  }
}

# -----------------------------------------------------------------------------
# ENCODING CHECKS
# -----------------------------------------------------------------------------

check "verify_base64_content" {
  data "filemanager_file" "base64_check" {
    path = filemanager_file.base64_content.path
  }

  assert {
    condition     = data.filemanager_file.base64_check.content == "This was base64 encoded"
    error_message = "Base64 decoded content mismatch"
  }
}

check "verify_binary_file" {
  data "filemanager_stat" "binary_check" {
    path = filemanager_file.binary_via_base64.path
  }

  assert {
    condition     = data.filemanager_stat.binary_check.exists == true
    error_message = "Binary file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.binary_check.size > 0
    error_message = "Binary file should not be empty"
  }
}

# -----------------------------------------------------------------------------
# NESTED DIRECTORY CHECKS
# -----------------------------------------------------------------------------

check "verify_deeply_nested" {
  data "filemanager_file" "deeply_nested_check" {
    path = filemanager_file.deeply_nested.path
  }

  assert {
    condition     = data.filemanager_file.deeply_nested_check.content == "Deep in the directory tree"
    error_message = "Deeply nested file content mismatch"
  }
}

check "verify_nested_file1" {
  data "filemanager_stat" "nested1_check" {
    path = filemanager_file.nested_file1.path
  }

  assert {
    condition     = data.filemanager_stat.nested1_check.exists == true
    error_message = "Nested file 1 does not exist"
  }
}

check "verify_nested_file2" {
  data "filemanager_stat" "nested2_check" {
    path = filemanager_file.nested_file2.path
  }

  assert {
    condition     = data.filemanager_stat.nested2_check.exists == true
    error_message = "Nested file 2 does not exist"
  }
}

check "verify_nested_file3" {
  data "filemanager_stat" "nested3_check" {
    path = filemanager_file.nested_file3.path
  }

  assert {
    condition     = data.filemanager_stat.nested3_check.exists == true
    error_message = "Nested file 3 does not exist"
  }
}

# -----------------------------------------------------------------------------
# EDGE CASE CHECKS
# -----------------------------------------------------------------------------

check "verify_spaces_in_name" {
  data "filemanager_stat" "spaces_check" {
    path = filemanager_file.spaces_in_name.path
  }

  assert {
    condition     = data.filemanager_stat.spaces_check.exists == true
    error_message = "File with spaces in name does not exist"
  }
}

check "verify_hidden_file" {
  data "filemanager_stat" "hidden_check" {
    path = filemanager_file.hidden_file.path
  }

  assert {
    condition     = data.filemanager_stat.hidden_check.exists == true
    error_message = "Hidden file does not exist"
  }
}

check "verify_single_char" {
  data "filemanager_file" "single_char_check" {
    path = filemanager_file.single_char.path
  }

  assert {
    condition     = data.filemanager_file.single_char_check.content == "X"
    error_message = "Single char file content mismatch"
  }

  assert {
    condition     = data.filemanager_file.single_char_check.size == 1
    error_message = "Single char file should be 1 byte"
  }
}

# -----------------------------------------------------------------------------
# CHECKSUM VERIFICATION
# -----------------------------------------------------------------------------

check "verify_checksums" {
  data "filemanager_file" "basic_checksums" {
    path = filemanager_file.basic.path
  }

  assert {
    condition     = data.filemanager_file.basic_checksums.md5 != ""
    error_message = "MD5 checksum should not be empty"
  }

  assert {
    condition     = data.filemanager_file.basic_checksums.sha256 != ""
    error_message = "SHA256 checksum should not be empty"
  }
}
