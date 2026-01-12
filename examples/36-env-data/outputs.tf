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

# Environment data source outputs
output "local_env_count" {
  description = "Number of local environment variables"
  value       = data.filemanager_environment.local.var_count
}

output "local_path" {
  description = "PATH environment variable"
  value       = data.filemanager_environment.local.vars["PATH"]
  sensitive   = true
}

output "local_home" {
  description = "HOME environment variable"
  value       = data.filemanager_environment.local.vars["HOME"]
}

output "path_vars_count" {
  description = "Number of PATH-related environment variables"
  value       = data.filemanager_environment.path_vars.var_count
}

output "path_vars" {
  description = "All PATH-related environment variables"
  value       = data.filemanager_environment.path_vars.vars
  sensitive   = true
}
