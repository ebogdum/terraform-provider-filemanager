# =============================================================================
# FILE RESOURCE - OUTPUTS
# =============================================================================

# -----------------------------------------------------------------------------
# BASIC FILE OUTPUTS
# -----------------------------------------------------------------------------

output "basic_file" {
  description = "Basic file details"
  value = {
    path   = filemanager_file.basic.path
    size   = filemanager_file.basic.size
    md5    = filemanager_file.basic.md5
    sha256 = filemanager_file.basic.sha256
  }
}

output "empty_file" {
  description = "Empty file details"
  value = {
    path = filemanager_file.empty.path
    size = filemanager_file.empty.size
    md5  = filemanager_file.empty.md5
  }
}

output "multiline_file" {
  description = "Multiline file details"
  value = {
    path = filemanager_file.multiline.path
    size = filemanager_file.multiline.size
  }
}

output "large_file" {
  description = "Large file details"
  value = {
    path = filemanager_file.large.path
    size = filemanager_file.large.size
  }
}

output "special_chars_file" {
  description = "Special characters file details"
  value = {
    path = filemanager_file.special_chars.path
    size = filemanager_file.special_chars.size
  }
}

# -----------------------------------------------------------------------------
# PERMISSION OUTPUTS
# -----------------------------------------------------------------------------

output "permission_files" {
  description = "Files with various permissions"
  value = {
    readonly = {
      path = filemanager_file.readonly.path
      mode = filemanager_file.readonly.file_permission
    }
    owner_only = {
      path = filemanager_file.owner_only.path
      mode = filemanager_file.owner_only.file_permission
    }
    full_perms = {
      path = filemanager_file.full_perms.path
      mode = filemanager_file.full_perms.file_permission
    }
    executable = {
      path = filemanager_file.executable.path
      mode = filemanager_file.executable.file_permission
    }
    custom_dir = {
      path      = filemanager_file.custom_dir_perms.path
      file_mode = filemanager_file.custom_dir_perms.file_permission
      dir_mode  = filemanager_file.custom_dir_perms.directory_permission
    }
  }
}

# -----------------------------------------------------------------------------
# ENCODING OUTPUTS
# -----------------------------------------------------------------------------

output "encoded_files" {
  description = "Base64 encoded files"
  value = {
    from_base64 = {
      path = filemanager_file.base64_content.path
      size = filemanager_file.base64_content.size
    }
    binary = {
      path = filemanager_file.binary_via_base64.path
      size = filemanager_file.binary_via_base64.size
    }
  }
}

# -----------------------------------------------------------------------------
# ATOMIC & CHECKSUM OUTPUTS
# -----------------------------------------------------------------------------

output "atomic_files" {
  description = "Atomic write and checksum verified files"
  value = {
    atomic = {
      path = filemanager_file.atomic.path
      md5  = filemanager_file.atomic.md5
    }
    verified = {
      path   = filemanager_file.verified.path
      sha256 = filemanager_file.verified.sha256
    }
    atomic_verified = {
      path   = filemanager_file.atomic_verified.path
      md5    = filemanager_file.atomic_verified.md5
      sha256 = filemanager_file.atomic_verified.sha256
    }
  }
}

# -----------------------------------------------------------------------------
# NEWLINE OUTPUTS
# -----------------------------------------------------------------------------

output "newline_files" {
  description = "Files with different newline styles"
  value = {
    lf = {
      path = filemanager_file.newline_lf.path
      size = filemanager_file.newline_lf.size
    }
    crlf = {
      path = filemanager_file.newline_crlf.path
      size = filemanager_file.newline_crlf.size
    }
  }
}

# -----------------------------------------------------------------------------
# EXTENSION OUTPUTS
# -----------------------------------------------------------------------------

output "extension_files" {
  description = "Files with various extensions"
  value = {
    json = filemanager_file.ext_json.path
    yaml = filemanager_file.ext_yaml.path
    xml  = filemanager_file.ext_xml.path
    html = filemanager_file.ext_html.path
    css  = filemanager_file.ext_css.path
    js   = filemanager_file.ext_js.path
    md   = filemanager_file.ext_md.path
    none = filemanager_file.no_extension.path
  }
}

# -----------------------------------------------------------------------------
# EDGE CASE OUTPUTS
# -----------------------------------------------------------------------------

output "edge_case_files" {
  description = "Edge case file details"
  value = {
    spaces_in_name = filemanager_file.spaces_in_name.path
    dots_in_name   = filemanager_file.dots_in_name.path
    hidden         = filemanager_file.hidden_file.path
    long_filename  = filemanager_file.long_filename.path
    whitespace     = filemanager_file.whitespace_only.path
    single_char    = filemanager_file.single_char.path
    newline_only   = filemanager_file.newline_only.path
  }
}

# -----------------------------------------------------------------------------
# SUMMARY
# -----------------------------------------------------------------------------

output "summary" {
  description = "Test summary"
  value = {
    total_files_created = 28
    test_categories = [
      "basic_operations",
      "permissions",
      "encoding",
      "atomic_write",
      "newline_handling",
      "extensions",
      "edge_cases"
    ]
  }
}
