output "title" {
  description = "Title from TOML config"
  value       = data.filemanager_toml.config.data.title
}

output "owner_name" {
  description = "Owner name from TOML config"
  value       = data.filemanager_toml.config.data.owner.name
}

output "database_server" {
  description = "Database server from TOML config"
  value       = data.filemanager_toml.config.data.database.server
}

output "file_size" {
  description = "Size of the TOML file"
  value       = data.filemanager_toml.config.size
}
