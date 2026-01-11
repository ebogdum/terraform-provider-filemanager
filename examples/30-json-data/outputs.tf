output "database_host" {
  description = "Database host from JSON config"
  value       = data.filemanager_json.config.data.database.host
}

output "database_port" {
  description = "Database port from JSON config"
  value       = data.filemanager_json.config.data.database.port
}

output "app_name" {
  description = "Application name from JSON config"
  value       = data.filemanager_json.config.data.app.name
}

output "file_size" {
  description = "Size of the JSON file"
  value       = data.filemanager_json.config.size
}

output "file_md5" {
  description = "MD5 checksum of the JSON file"
  value       = data.filemanager_json.config.md5
}
