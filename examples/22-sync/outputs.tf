# =============================================================================
# SYNC RESOURCE - OUTPUTS
# =============================================================================

output "basic_sync" {
  description = "Basic sync test results"
  value = {
    files_transferred = filemanager_sync.basic.files_transferred
    bytes_transferred = filemanager_sync.basic.bytes_transferred
    files_deleted     = filemanager_sync.basic.files_deleted
    files_skipped     = filemanager_sync.basic.files_skipped
    source_path       = filemanager_sync.basic.source_path
    destination_path  = filemanager_sync.basic.destination_path
  }
}

output "delete_orphans_sync" {
  description = "Delete orphans sync test results"
  value = {
    files_transferred = filemanager_sync.delete_orphans.files_transferred
    files_deleted     = filemanager_sync.delete_orphans.files_deleted
  }
}

output "comparison_modes" {
  description = "Comparison mode test results"
  value = {
    size_only = {
      files_transferred = filemanager_sync.size_only.files_transferred
    }
    checksum = {
      files_transferred = filemanager_sync.checksum.files_transferred
    }
    mtime = {
      files_transferred = filemanager_sync.mtime.files_transferred
    }
  }
}

output "filtered_syncs" {
  description = "Filtered sync test results"
  value = {
    with_includes = {
      files_transferred = filemanager_sync.with_includes.files_transferred
    }
    with_excludes = {
      files_transferred = filemanager_sync.with_excludes.files_transferred
    }
  }
}

output "advanced_syncs" {
  description = "Advanced sync test results"
  value = {
    preserve_timestamps = {
      files_transferred = filemanager_sync.preserve_timestamps.files_transferred
    }
    with_concurrency = {
      files_transferred = filemanager_sync.with_concurrency.files_transferred
    }
  }
}

output "full_sync" {
  description = "Full sync test results"
  value = {
    files_transferred = filemanager_sync.full.files_transferred
    bytes_transferred = filemanager_sync.full.bytes_transferred
    files_deleted     = filemanager_sync.full.files_deleted
    duration_ms       = filemanager_sync.full.duration_ms
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_sync_tests = 10
    categories = [
      "basic_sync",
      "delete_orphans",
      "comparison_modes",
      "filtered_syncs",
      "advanced_options",
      "full_sync"
    ]
  }
}
