# Read and parse TOML file
data "filemanager_toml" "config" {
  path = "/etc/app/config.toml"
}

# Access parsed data
output "database_server" {
  value = data.filemanager_toml.config.data.database.server
}

output "database_ports" {
  value = data.filemanager_toml.config.data.database.ports
}

# Read TOML from remote server
data "filemanager_toml" "remote" {
  path    = "/etc/app/config.toml"
  service = filemanager_ssh_service.server.name
}

# Read Cargo.toml
data "filemanager_toml" "cargo" {
  path = "${var.project_path}/Cargo.toml"
}

output "package_version" {
  value = data.filemanager_toml.cargo.data.package.version
}

output "dependencies" {
  value = data.filemanager_toml.cargo.data.dependencies
}
