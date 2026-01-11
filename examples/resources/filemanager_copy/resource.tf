# Copy a single file
resource "filemanager_copy" "config_backup" {
  source      = "/etc/app/config.json"
  destination = "/var/backups/config.json.bak"

  create_parent_dirs = true
}

# Copy a directory recursively
resource "filemanager_copy" "app_release" {
  source      = "/var/releases/latest"
  destination = "/var/www/app"
  recursive   = true

  excludes = [
    "*.log",
    ".git/**"
  ]
}

# Copy with permission preservation
resource "filemanager_copy" "scripts" {
  source      = "/opt/scripts"
  destination = "/usr/local/bin/app-scripts"
  recursive   = true

  preserve_permissions = true
  preserve_timestamps  = true
}

# Copy from local to remote server
resource "filemanager_copy" "deploy" {
  source          = "/var/releases/app-${var.version}.tar.gz"
  destination     = "/opt/releases/"
  dest_service    = filemanager_ssh_service.server.name

  create_parent_dirs = true
}

# Copy between two remote servers
resource "filemanager_copy" "replicate" {
  source          = "/var/data/important.db"
  source_service  = filemanager_ssh_service.primary.name
  destination     = "/var/data/important.db"
  dest_service    = filemanager_ssh_service.secondary.name
}
