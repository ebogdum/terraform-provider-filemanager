# =============================================================================
# DOWNLOAD RESOURCE - OUTPUTS
# =============================================================================

output "basic_download" {
  description = "Basic download test results"
  value = {
    bytes_transferred = filemanager_download.basic.bytes_transferred
    files_transferred = filemanager_download.basic.files_transferred
    md5               = filemanager_download.basic.md5
    sha256            = filemanager_download.basic.sha256
    source_path       = filemanager_download.basic.source_path
    destination_path  = filemanager_download.basic.destination_path
  }
}

output "directory_download" {
  description = "Directory download test results"
  value = {
    bytes_transferred = filemanager_download.directory.bytes_transferred
    files_transferred = filemanager_download.directory.files_transferred
  }
}

output "filtered_downloads" {
  description = "Filtered download test results"
  value = {
    with_includes = {
      files_transferred = filemanager_download.with_includes.files_transferred
    }
    with_excludes = {
      files_transferred = filemanager_download.with_excludes.files_transferred
    }
  }
}

output "advanced_downloads" {
  description = "Advanced download test results"
  value = {
    with_checksum = {
      bytes_transferred = filemanager_download.with_checksum.bytes_transferred
    }
    preserve_timestamps = {
      files_transferred = filemanager_download.preserve_timestamps.files_transferred
    }
    with_overwrite = {
      files_transferred = filemanager_download.with_overwrite.files_transferred
    }
    with_permissions = {
      files_transferred = filemanager_download.with_permissions.files_transferred
    }
    with_concurrency = {
      files_transferred = filemanager_download.with_concurrency.files_transferred
    }
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_download_tests = 9
    categories = [
      "basic_download",
      "directory_download",
      "filtered_downloads",
      "advanced_options"
    ]
  }
}
