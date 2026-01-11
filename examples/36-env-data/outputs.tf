output "database_url" {
  description = "Database URL from ENV config"
  value       = data.filemanager_env.config.data.DATABASE_URL
  sensitive   = true
}

output "api_key" {
  description = "API key from ENV config"
  value       = data.filemanager_env.config.data["API_KEY"]
  sensitive   = true
}

output "app_env" {
  description = "Application environment from ENV config"
  value       = data.filemanager_env.config.data.APP_ENV
}

output "file_size" {
  description = "Size of the ENV file"
  value       = data.filemanager_env.config.size
}
