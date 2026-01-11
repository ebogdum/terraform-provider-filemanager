# =============================================================================
# FTP OPERATION RESOURCE - OUTPUTS
# =============================================================================

output "service_id" {
  description = "FTP service identifier"
  value       = filemanager_ftp_service.server.id
}

output "service_connected" {
  description = "Whether the FTP service is connected"
  value       = filemanager_ftp_service.server.connected
}

output "head_operation" {
  description = "HEAD operation results - file metadata"
  value = {
    size        = filemanager_ftp_operation.head.size
    mod_time    = filemanager_ftp_operation.head.mod_time
    permissions = filemanager_ftp_operation.head.permissions
    is_dir      = filemanager_ftp_operation.head.is_dir
    name        = filemanager_ftp_operation.head.name
  }
}

output "copy_operation" {
  description = "COPY operation results"
  value = {
    size     = filemanager_ftp_operation.backup.size
    mod_time = filemanager_ftp_operation.backup.mod_time
    name     = filemanager_ftp_operation.backup.name
  }
}

output "rename_operation" {
  description = "RENAME operation results"
  value = {
    size     = filemanager_ftp_operation.archive.size
    mod_time = filemanager_ftp_operation.archive.mod_time
    name     = filemanager_ftp_operation.archive.name
  }
}
