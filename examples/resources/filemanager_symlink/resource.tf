# Create a symbolic link
resource "filemanager_symlink" "config" {
  path   = "/etc/app/current-config"
  target = "/etc/app/config-v2.json"
}

# Create symlink for versioned deployment
resource "filemanager_symlink" "current_release" {
  path   = "/var/www/current"
  target = "/var/www/releases/${var.release_version}"

  create_parent_dirs = true
}

# Create symlink on remote server
resource "filemanager_symlink" "remote_link" {
  path    = "/opt/app/current"
  target  = "/opt/app/releases/latest"
  service = filemanager_ssh_service.server.id
}
