# Sync a directory to a remote server
resource "filemanager_sync" "deploy" {
  source       = "${path.module}/dist/"
  destination  = "/var/www/app/"
  dest_service = filemanager_ssh_service.server.name

  delete = true  # Remove files at destination not in source

  excludes = [
    ".git/**",
    "*.log",
    "*.tmp"
  ]
}

# Sync configuration directory
resource "filemanager_sync" "config" {
  source       = "${path.module}/config/"
  destination  = "/etc/app/"
  dest_service = filemanager_ssh_service.server.name

  delete = false  # Keep existing files at destination
}

# Sync to S3 bucket
resource "filemanager_sync" "s3_assets" {
  source       = "${path.module}/public/assets/"
  destination  = "assets/"
  dest_service = filemanager_s3_service.cdn.name

  delete = true
}

# Bidirectional sync between servers
resource "filemanager_sync" "replicate" {
  source          = "/var/data/"
  source_service  = filemanager_ssh_service.primary.name
  destination     = "/var/data/"
  dest_service    = filemanager_ssh_service.replica.name

  delete = false
}
