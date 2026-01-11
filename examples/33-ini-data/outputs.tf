output "database_host" {
  description = "Database host from INI config"
  value       = data.filemanager_ini.config.data.database.host
}

output "database_port" {
  description = "Database port from INI config"
  value       = data.filemanager_ini.config.data.database.port
}

output "server_listen" {
  description = "Server listen address from INI config"
  value       = data.filemanager_ini.config.data.server.listen
}

output "file_size" {
  description = "Size of the INI file"
  value       = data.filemanager_ini.config.size
}
