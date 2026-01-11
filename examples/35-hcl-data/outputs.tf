output "app_name" {
  description = "Application name from HCL config"
  value       = data.filemanager_hcl.config.data.app_name
}

output "server_port" {
  description = "Server port from HCL config"
  value       = data.filemanager_hcl.config.data.server.port
}

output "file_size" {
  description = "Size of the HCL file"
  value       = data.filemanager_hcl.config.size
}

output "file_md5" {
  description = "MD5 checksum of the HCL file"
  value       = data.filemanager_hcl.config.md5
}
