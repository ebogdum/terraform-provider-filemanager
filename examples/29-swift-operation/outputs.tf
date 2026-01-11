# =============================================================================
# SWIFT OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "Swift service identifier"
  value       = filemanager_swift_service.storage.id
}

output "service_connected" {
  description = "Whether the Swift service is connected"
  value       = filemanager_swift_service.storage.connected
}

output "head_operation" {
  description = "HEAD operation results - object metadata"
  value = {
    size             = filemanager_swift_operation.head.size
    last_modified    = filemanager_swift_operation.head.last_modified
    content_type     = filemanager_swift_operation.head.content_type
    etag             = filemanager_swift_operation.head.etag
    is_dir           = filemanager_swift_operation.head.is_dir
    name             = filemanager_swift_operation.head.name
    current_metadata = filemanager_swift_operation.head.current_metadata
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    size          = filemanager_swift_operation.backup.size
    last_modified = filemanager_swift_operation.backup.last_modified
    etag          = filemanager_swift_operation.backup.etag
    name          = filemanager_swift_operation.backup.name
  }
}

output "metadata_operation" {
  description = "SET_METADATA operation results"
  value = {
    current_metadata = filemanager_swift_operation.metadata.current_metadata
  }
}
