# DATA SOURCES - OUTPUTS

# =============================================================================
# FILE DATA SOURCE OUTPUTS
# =============================================================================

output "file_data" {
  value = {
    simple = {
      path    = data.filemanager_file.simple.path
      content = data.filemanager_file.simple.content
      size    = data.filemanager_file.simple.size
      md5     = data.filemanager_file.simple.md5
      sha256  = data.filemanager_file.simple.sha256
      mode    = data.filemanager_file.simple.mode
    }
    multiline = {
      path    = data.filemanager_file.multiline.path
      content = data.filemanager_file.multiline.content
      size    = data.filemanager_file.multiline.size
    }
    large = {
      path   = data.filemanager_file.large.path
      size   = data.filemanager_file.large.size
      sha256 = data.filemanager_file.large.sha256
    }
    json = {
      path    = data.filemanager_file.json.path
      content = data.filemanager_file.json.content
      size    = data.filemanager_file.json.size
    }
    yaml = {
      path    = data.filemanager_file.yaml.path
      content = data.filemanager_file.yaml.content
    }
    executable = {
      path = data.filemanager_file.executable.path
      mode = data.filemanager_file.executable.mode
    }
    unicode = {
      path    = data.filemanager_file.unicode.path
      content = data.filemanager_file.unicode.content
    }
    empty = {
      path   = data.filemanager_file.empty.path
      size   = data.filemanager_file.empty.size
      sha256 = data.filemanager_file.empty.sha256
    }
    nested = {
      path    = data.filemanager_file.nested.path
      content = data.filemanager_file.nested.content
    }
  }
}

# =============================================================================
# CHECKSUM DATA SOURCE OUTPUTS
# =============================================================================

output "checksum_data" {
  value = {
    md5 = {
      path     = data.filemanager_checksum.md5.path
      checksum = data.filemanager_checksum.md5.checksum
      size     = data.filemanager_checksum.md5.size
    }
    sha1 = {
      path     = data.filemanager_checksum.sha1.path
      checksum = data.filemanager_checksum.sha1.checksum
    }
    sha256 = {
      path     = data.filemanager_checksum.sha256.path
      checksum = data.filemanager_checksum.sha256.checksum
    }
    sha512 = {
      path     = data.filemanager_checksum.sha512.path
      checksum = data.filemanager_checksum.sha512.checksum
    }
    large_file = {
      path     = data.filemanager_checksum.large_file.path
      checksum = data.filemanager_checksum.large_file.checksum
      size     = data.filemanager_checksum.large_file.size
    }
    empty_file = {
      path     = data.filemanager_checksum.empty_file.path
      checksum = data.filemanager_checksum.empty_file.checksum
    }
    json_file = {
      path     = data.filemanager_checksum.json_file.path
      checksum = data.filemanager_checksum.json_file.checksum
    }
  }
}

# =============================================================================
# STAT DATA SOURCE OUTPUTS
# =============================================================================

output "stat_data" {
  value = {
    regular_file = {
      path       = data.filemanager_stat.regular_file.path
      exists     = data.filemanager_stat.regular_file.exists
      size       = data.filemanager_stat.regular_file.size
      is_file    = data.filemanager_stat.regular_file.is_file
      is_dir     = data.filemanager_stat.regular_file.is_dir
      is_symlink = data.filemanager_stat.regular_file.is_symlink
      mode       = data.filemanager_stat.regular_file.mode
      mod_time   = data.filemanager_stat.regular_file.mod_time
      uid        = data.filemanager_stat.regular_file.uid
      gid        = data.filemanager_stat.regular_file.gid
    }
    directory = {
      path    = data.filemanager_stat.directory.path
      exists  = data.filemanager_stat.directory.exists
      is_dir  = data.filemanager_stat.directory.is_dir
      is_file = data.filemanager_stat.directory.is_file
      mode    = data.filemanager_stat.directory.mode
    }
    executable = {
      path   = data.filemanager_stat.executable.path
      exists = data.filemanager_stat.executable.exists
      mode   = data.filemanager_stat.executable.mode
    }
    symlink = {
      path        = data.filemanager_stat.symlink.path
      exists      = data.filemanager_stat.symlink.exists
      is_symlink  = data.filemanager_stat.symlink.is_symlink
      link_target = data.filemanager_stat.symlink.link_target
    }
    large = {
      path = data.filemanager_stat.large.path
      size = data.filemanager_stat.large.size
    }
    empty = {
      path = data.filemanager_stat.empty.path
      size = data.filemanager_stat.empty.size
    }
    nonexistent = {
      path   = data.filemanager_stat.nonexistent.path
      exists = data.filemanager_stat.nonexistent.exists
    }
  }
}

# =============================================================================
# FILES DATA SOURCE OUTPUTS
# =============================================================================

output "files_data" {
  value = {
    all = {
      path       = data.filemanager_files.all.path
      file_count = data.filemanager_files.all.file_count
    }
    txt_files = {
      path       = data.filemanager_files.txt_files.path
      pattern    = data.filemanager_files.txt_files.pattern
      file_count = data.filemanager_files.txt_files.file_count
    }
    json_files = {
      path       = data.filemanager_files.json_files.path
      pattern    = data.filemanager_files.json_files.pattern
      file_count = data.filemanager_files.json_files.file_count
    }
    yaml_files = {
      path       = data.filemanager_files.yaml_files.path
      pattern    = data.filemanager_files.yaml_files.pattern
      file_count = data.filemanager_files.yaml_files.file_count
    }
    recursive = {
      path       = data.filemanager_files.recursive.path
      recursive  = data.filemanager_files.recursive.recursive
      file_count = data.filemanager_files.recursive.file_count
    }
    recursive_txt = {
      path       = data.filemanager_files.recursive_txt.path
      pattern    = data.filemanager_files.recursive_txt.pattern
      recursive  = data.filemanager_files.recursive_txt.recursive
      file_count = data.filemanager_files.recursive_txt.file_count
    }
    scripts = {
      path       = data.filemanager_files.scripts.path
      file_count = data.filemanager_files.scripts.file_count
    }
    shell_scripts = {
      path       = data.filemanager_files.shell_scripts.path
      pattern    = data.filemanager_files.shell_scripts.pattern
      file_count = data.filemanager_files.shell_scripts.file_count
    }
    logs = {
      path       = data.filemanager_files.logs.path
      pattern    = data.filemanager_files.logs.pattern
      file_count = data.filemanager_files.logs.file_count
    }
  }
}

# =============================================================================
# DIRECTORY DATA SOURCE OUTPUTS
# =============================================================================

output "directory_data" {
  value = {
    source = {
      path       = data.filemanager_directory.source.path
      file_count = data.filemanager_directory.source.file_count
      total_size = data.filemanager_directory.source.total_size
    }
    txt = {
      path       = data.filemanager_directory.txt.path
      pattern    = data.filemanager_directory.txt.pattern
      file_count = data.filemanager_directory.txt.file_count
    }
    json = {
      path       = data.filemanager_directory.json.path
      pattern    = data.filemanager_directory.json.pattern
      file_count = data.filemanager_directory.json.file_count
    }
    recursive = {
      path       = data.filemanager_directory.recursive.path
      recursive  = data.filemanager_directory.recursive.recursive
      file_count = data.filemanager_directory.recursive.file_count
      total_size = data.filemanager_directory.recursive.total_size
    }
    recursive_txt = {
      path       = data.filemanager_directory.recursive_txt.path
      pattern    = data.filemanager_directory.recursive_txt.pattern
      recursive  = data.filemanager_directory.recursive_txt.recursive
      file_count = data.filemanager_directory.recursive_txt.file_count
    }
    scripts = {
      path       = data.filemanager_directory.scripts.path
      file_count = data.filemanager_directory.scripts.file_count
    }
    logs = {
      path       = data.filemanager_directory.logs.path
      pattern    = data.filemanager_directory.logs.pattern
      file_count = data.filemanager_directory.logs.file_count
    }
    level1 = {
      path       = data.filemanager_directory.level1.path
      file_count = data.filemanager_directory.level1.file_count
    }
    level2 = {
      path       = data.filemanager_directory.level2.path
      file_count = data.filemanager_directory.level2.file_count
    }
    level3 = {
      path       = data.filemanager_directory.level3.path
      file_count = data.filemanager_directory.level3.file_count
    }
  }
}

# =============================================================================
# SUMMARY
# =============================================================================

output "summary" {
  value = {
    total_test_cases = 43
    data_sources = {
      file      = 9
      checksum  = 7
      stat      = 8
      files     = 9
      directory = 10
    }
    categories = [
      "file_reading",
      "checksum_algorithms",
      "file_stat_metadata",
      "glob_patterns",
      "directory_listing",
      "recursive_search",
      "edge_cases"
    ]
  }
}
