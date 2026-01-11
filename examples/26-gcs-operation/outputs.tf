# =============================================================================
# GCS OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "GCS service identifier"
  value       = filemanager_gcs_service.prod.id
}

output "service_connected" {
  description = "Whether the GCS service is connected"
  value       = filemanager_gcs_service.prod.connected
}

output "head_operation" {
  description = "HEAD operation results - object metadata"
  value = {
    etag          = filemanager_gcs_operation.head.etag
    size          = filemanager_gcs_operation.head.size
    content_type  = filemanager_gcs_operation.head.content_type
    storage_class = filemanager_gcs_operation.head.computed_storage_class
    generation    = filemanager_gcs_operation.head.generation
    time_created  = filemanager_gcs_operation.head.time_created
    updated       = filemanager_gcs_operation.head.updated
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    etag       = filemanager_gcs_operation.backup.etag
    generation = filemanager_gcs_operation.backup.generation
  }
}

output "metadata_operation" {
  description = "SET_METADATA operation results"
  value = {
    current_metadata = filemanager_gcs_operation.metadata.current_metadata
    metageneration   = filemanager_gcs_operation.metadata.metageneration
  }
}

output "archive_operation" {
  description = "SET_STORAGE_CLASS operation results"
  value = {
    computed_storage_class = filemanager_gcs_operation.archive.computed_storage_class
  }
}
