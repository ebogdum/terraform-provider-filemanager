# Create a ZIP archive from a directory
resource "filemanager_archive" "app_bundle" {
  path        = "/var/releases/app-${var.version}.zip"
  source_dir  = "/var/www/app"
  format      = "zip"
  compression = "deflate"

  excludes = [
    "*.log",
    "*.tmp",
    ".git/**",
    "node_modules/**"
  ]

  create_parent_dirs = true
}

# Create a tar.gz archive
resource "filemanager_archive" "backup" {
  path        = "/var/backups/config-${formatdate("YYYY-MM-DD", timestamp())}.tar.gz"
  source_dir  = "/etc/app"
  format      = "tar.gz"
  compression = "gzip"

  create_parent_dirs = true
}

# Create archive from specific files
resource "filemanager_archive" "logs" {
  path   = "/var/archives/logs.zip"
  format = "zip"

  files = {
    "app.log"    = "/var/log/app/app.log"
    "error.log"  = "/var/log/app/error.log"
    "access.log" = "/var/log/nginx/access.log"
  }

  create_parent_dirs = true
}
