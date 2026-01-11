# Perform Google Cloud Storage operations
resource "filemanager_gcs_operation" "upload_config" {
  operation = "upload"
  service   = filemanager_gcs_service.main.name

  local_path  = "${path.module}/config.json"
  remote_path = "configs/app/config.json"

  content_type = "application/json"
  metadata = {
    environment = var.environment
    version     = var.config_version
  }
}

# Download from GCS
resource "filemanager_gcs_operation" "download_artifact" {
  operation = "download"
  service   = filemanager_gcs_service.releases.name

  remote_path = "releases/app-${var.version}.tar.gz"
  local_path  = "/var/releases/app-${var.version}.tar.gz"

  create_parent_dirs = true
}

# Copy within GCS bucket
resource "filemanager_gcs_operation" "archive_backup" {
  operation = "copy"
  service   = filemanager_gcs_service.backups.name

  remote_path      = "daily/database.sql.gz"
  copy_destination = "archive/${formatdate("YYYY/MM/DD", timestamp())}/database.sql.gz"
}

# Delete from GCS
resource "filemanager_gcs_operation" "cleanup" {
  operation = "delete"
  service   = filemanager_gcs_service.temp.name

  remote_path = "temp/${var.cleanup_prefix}"
}
