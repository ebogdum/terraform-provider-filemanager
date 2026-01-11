output "server_host" {
  description = "Server host from YAML config"
  value       = data.filemanager_yaml.config.data.server.host
}

output "server_port" {
  description = "Server port from YAML config"
  value       = data.filemanager_yaml.config.data.server.port
}

output "log_level" {
  description = "Log level from YAML config"
  value       = data.filemanager_yaml.config.data.logging.level
}

output "file_size" {
  description = "Size of the YAML file"
  value       = data.filemanager_yaml.config.size
}

output "file_sha256" {
  description = "SHA-256 checksum of the YAML file"
  value       = data.filemanager_yaml.config.sha256
}
