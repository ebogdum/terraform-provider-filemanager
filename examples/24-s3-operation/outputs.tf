# =============================================================================
# S3 OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "S3 service identifier"
  value       = filemanager_s3_service.prod.id
}

output "service_connected" {
  description = "Whether the S3 service is connected"
  value       = filemanager_s3_service.prod.connected
}

output "head_operation" {
  description = "HEAD operation results - object metadata"
  value = {
    etag          = filemanager_s3_operation.head.etag
    size          = filemanager_s3_operation.head.size
    content_type  = filemanager_s3_operation.head.content_type
    storage_class = filemanager_s3_operation.head.current_storage_class
    last_modified = filemanager_s3_operation.head.last_modified
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    etag = filemanager_s3_operation.backup.etag
    size = filemanager_s3_operation.backup.size
  }
}

output "metadata_operation" {
  description = "SET_METADATA operation results"
  value = {
    current_metadata = filemanager_s3_operation.metadata.current_metadata
  }
}

output "tags_operation" {
  description = "SET_TAGS operation results"
  value = {
    current_tags = filemanager_s3_operation.tags.current_tags
  }
}

output "archive_operation" {
  description = "SET_STORAGE_CLASS operation results"
  value = {
    current_storage_class = filemanager_s3_operation.archive.current_storage_class
  }
}
