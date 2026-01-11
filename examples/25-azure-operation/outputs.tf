# =============================================================================
# AZURE OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "Azure service identifier"
  value       = filemanager_azure_service.prod.id
}

output "service_connected" {
  description = "Whether the Azure service is connected"
  value       = filemanager_azure_service.prod.connected
}

output "head_operation" {
  description = "HEAD operation results - blob properties"
  value = {
    etag          = filemanager_azure_operation.head.etag
    content_type  = filemanager_azure_operation.head.content_type
    blob_type     = filemanager_azure_operation.head.blob_type
    access_tier   = filemanager_azure_operation.head.current_access_tier
    last_modified = filemanager_azure_operation.head.last_modified
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    etag = filemanager_azure_operation.backup.etag
  }
}

output "metadata_operation" {
  description = "SET_METADATA operation results"
  value = {
    current_metadata = filemanager_azure_operation.metadata.current_metadata
  }
}

output "tags_operation" {
  description = "SET_TAGS operation results"
  value = {
    current_tags = filemanager_azure_operation.tags.current_tags
  }
}

output "archive_operation" {
  description = "SET_ACCESS_TIER operation results"
  value = {
    current_access_tier = filemanager_azure_operation.archive.current_access_tier
  }
}
