# Create a directory with specific permissions
resource "filemanager_directory" "app" {
  path           = "/var/app/data"
  permission     = "0755"
  create_parents = true
}

# Create multiple directories with ownership
resource "filemanager_directory" "logs" {
  path           = "/var/log/myapp"
  permission     = "0750"
  create_parents = true
}

# Create directory on remote server
resource "filemanager_directory" "remote_app" {
  path           = "/opt/myapp"
  permission     = "0755"
  create_parents = true
  service        = filemanager_ssh_service.server.name
}
