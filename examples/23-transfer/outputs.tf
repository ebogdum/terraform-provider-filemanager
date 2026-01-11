# =============================================================================
# TRANSFER RESOURCE - OUTPUTS
# =============================================================================

output "basic_transfer" {
  description = "Basic transfer test results"
  value = {
    bytes_transferred = filemanager_transfer.basic.bytes_transferred
    files_transferred = filemanager_transfer.basic.files_transferred
    transfer_method   = filemanager_transfer.basic.transfer_method
    source_path       = filemanager_transfer.basic.source_path
    destination_path  = filemanager_transfer.basic.destination_path
  }
}

output "directory_transfer" {
  description = "Directory transfer test results"
  value = {
    bytes_transferred = filemanager_transfer.directory.bytes_transferred
    files_transferred = filemanager_transfer.directory.files_transferred
  }
}

output "filtered_transfers" {
  description = "Filtered transfer test results"
  value = {
    with_includes = {
      files_transferred = filemanager_transfer.with_includes.files_transferred
    }
    with_excludes = {
      files_transferred = filemanager_transfer.with_excludes.files_transferred
    }
  }
}

output "verification_transfers" {
  description = "Verification transfer test results"
  value = {
    with_checksum = {
      bytes_transferred = filemanager_transfer.with_checksum.bytes_transferred
    }
  }
}

output "preservation_transfers" {
  description = "Preservation transfer test results"
  value = {
    preserve_timestamps = {
      files_transferred = filemanager_transfer.preserve_timestamps.files_transferred
    }
    preserve_permissions = {
      files_transferred = filemanager_transfer.preserve_permissions.files_transferred
    }
  }
}

output "transfer_modes" {
  description = "Transfer mode test results"
  value = {
    streaming = {
      transfer_method   = filemanager_transfer.streaming.transfer_method
      bytes_transferred = filemanager_transfer.streaming.bytes_transferred
    }
    zero_copy = {
      transfer_method   = filemanager_transfer.zero_copy.transfer_method
      bytes_transferred = filemanager_transfer.zero_copy.bytes_transferred
    }
  }
}

output "advanced_transfers" {
  description = "Advanced transfer test results"
  value = {
    with_overwrite = {
      files_transferred = filemanager_transfer.with_overwrite.files_transferred
    }
    buffer_size = {
      files_transferred = filemanager_transfer.buffer_size.files_transferred
    }
    with_concurrency = {
      files_transferred = filemanager_transfer.with_concurrency.files_transferred
    }
  }
}

output "full_transfer" {
  description = "Full transfer test results"
  value = {
    files_transferred = filemanager_transfer.full.files_transferred
    bytes_transferred = filemanager_transfer.full.bytes_transferred
    duration_ms       = filemanager_transfer.full.duration_ms
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_transfer_tests = 13
    categories = [
      "basic_transfer",
      "directory_transfer",
      "filtered_transfers",
      "verification",
      "preservation",
      "transfer_modes",
      "advanced_options",
      "full_transfer"
    ]
  }
}
