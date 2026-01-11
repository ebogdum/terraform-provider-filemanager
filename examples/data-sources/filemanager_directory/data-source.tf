# Read directory information
data "filemanager_directory" "app" {
  path = "/var/app"
}

output "directory_info" {
  value = {
    exists      = data.filemanager_directory.app.exists
    permissions = data.filemanager_directory.app.permission
    files       = data.filemanager_directory.app.files
    directories = data.filemanager_directory.app.directories
  }
}

# Read remote directory
data "filemanager_directory" "remote" {
  path    = "/opt/app"
  service = filemanager_ssh_service.server.name
}

# Use in conditionals
resource "filemanager_directory" "app" {
  count          = data.filemanager_directory.app.exists ? 0 : 1
  path           = "/var/app"
  permission     = "0755"
  create_parents = true
}
