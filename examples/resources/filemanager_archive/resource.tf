# Create a ZIP archive from a directory
resource "filemanager_archive" "app_bundle" {
  path       = "/var/releases/app-${var.version}.zip"
  source_dir = "/var/www/app"
  type       = "zip"

  excludes = [
    "*.log",
    "*.tmp",
  ]

  create_parent_dirs = true
}

# Create a tar.gz archive
resource "filemanager_archive" "backup" {
  path       = "/var/backups/config.tar.gz"
  source_dir = "/etc/app"
  type       = "tar.gz"

  create_parent_dirs = true
}

# Create archive from specific files
resource "filemanager_archive" "logs" {
  path = "/var/archives/logs.zip"
  type = "zip"

  source_files = [
    "/var/log/app/app.log",
    "/var/log/app/error.log",
    "/var/log/nginx/access.log",
  ]

  create_parent_dirs = true
}
