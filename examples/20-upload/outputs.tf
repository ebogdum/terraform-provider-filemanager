# =============================================================================
# UPLOAD RESOURCE - OUTPUTS
# =============================================================================

output "basic_upload" {
  description = "Basic upload test results"
  value = {
    bytes_transferred = filemanager_upload.basic.bytes_transferred
    files_transferred = filemanager_upload.basic.files_transferred
    source_path       = filemanager_upload.basic.source_path
    destination_path  = filemanager_upload.basic.destination_path
  }
}

output "directory_upload" {
  description = "Directory upload test results"
  value = {
    bytes_transferred = filemanager_upload.directory.bytes_transferred
    files_transferred = filemanager_upload.directory.files_transferred
  }
}

output "filtered_uploads" {
  description = "Filtered upload test results"
  value = {
    with_includes = {
      files_transferred = filemanager_upload.with_includes.files_transferred
    }
    with_excludes = {
      files_transferred = filemanager_upload.with_excludes.files_transferred
    }
  }
}

output "advanced_uploads" {
  description = "Advanced upload test results"
  value = {
    with_checksum = {
      bytes_transferred = filemanager_upload.with_checksum.bytes_transferred
    }
    preserve_timestamps = {
      files_transferred = filemanager_upload.preserve_timestamps.files_transferred
    }
    with_overwrite = {
      files_transferred = filemanager_upload.with_overwrite.files_transferred
    }
    with_concurrency = {
      files_transferred = filemanager_upload.with_concurrency.files_transferred
    }
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_upload_tests = 8
    categories = [
      "basic_upload",
      "directory_upload",
      "filtered_uploads",
      "advanced_options"
    ]
  }
}
