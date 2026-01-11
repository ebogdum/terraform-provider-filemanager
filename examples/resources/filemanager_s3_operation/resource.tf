# Perform S3 operations (upload, download, copy)
resource "filemanager_s3_operation" "upload_config" {
  operation = "upload"
  service   = filemanager_s3_service.main.name

  local_path  = "${path.module}/config.json"
  remote_path = "configs/app/config.json"

  content_type = "application/json"
  metadata = {
    environment = var.environment
    version     = var.config_version
  }
}

# Download from S3
resource "filemanager_s3_operation" "download_artifact" {
  operation = "download"
  service   = filemanager_s3_service.releases.name

  remote_path = "releases/app-${var.version}.tar.gz"
  local_path  = "/var/releases/app-${var.version}.tar.gz"

  create_parent_dirs = true
}

# Copy within S3
resource "filemanager_s3_operation" "archive_backup" {
  operation = "copy"
  service   = filemanager_s3_service.backups.name

  remote_path = "daily/database.sql.gz"
  copy_destination = "archive/${formatdate("YYYY/MM/DD", timestamp())}/database.sql.gz"
}

# Delete old files
resource "filemanager_s3_operation" "cleanup" {
  operation = "delete"
  service   = filemanager_s3_service.temp.name

  remote_path = "temp/${var.cleanup_prefix}*"
}
