# =============================================================================
# B2 OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "B2 service identifier"
  value       = filemanager_b2_service.prod.id
}

output "service_connected" {
  description = "Whether the B2 service is connected"
  value       = filemanager_b2_service.prod.connected
}

output "head_operation" {
  description = "HEAD operation results - file metadata"
  value = {
    file_id          = filemanager_b2_operation.head.file_id
    file_name        = filemanager_b2_operation.head.file_name
    content_length   = filemanager_b2_operation.head.content_length
    content_type     = filemanager_b2_operation.head.content_type
    content_sha1     = filemanager_b2_operation.head.content_sha1
    upload_timestamp = filemanager_b2_operation.head.upload_timestamp
    action           = filemanager_b2_operation.head.action
  }
}

output "info_operation" {
  description = "GET_FILE_INFO operation results"
  value = {
    file_id           = filemanager_b2_operation.info.file_id
    current_file_info = filemanager_b2_operation.info.current_file_info
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    file_id   = filemanager_b2_operation.backup.file_id
    file_name = filemanager_b2_operation.backup.file_name
  }
}

output "metadata_operation" {
  description = "UPDATE_FILE_INFO operation results"
  value = {
    current_file_info = filemanager_b2_operation.metadata.current_file_info
  }
}
