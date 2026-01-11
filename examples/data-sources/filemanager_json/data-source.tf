# Read and parse JSON file
data "filemanager_json" "config" {
  path = "/etc/app/config.json"
}

# Access parsed data
output "database_host" {
  value = data.filemanager_json.config.data.database.host
}

output "feature_flags" {
  value = data.filemanager_json.config.data.features
}

# Read JSON from remote server
data "filemanager_json" "remote_config" {
  path    = "/etc/app/config.json"
  service = filemanager_ssh_service.server.name
}

# Access nested and array values
output "servers" {
  value = data.filemanager_json.config.data.servers[*].host
}

# Access keys with special characters
output "api_endpoint" {
  value = data.filemanager_json.config.data["api-endpoint"]
}
