# Perform Backblaze B2 operations
resource "filemanager_b2_operation" "upload_backup" {
  operation = "upload"
  service   = filemanager_b2_service.backups.name

  local_path  = "/var/backups/database.sql.gz"
  remote_path = "backups/database-${formatdate("YYYY-MM-DD", timestamp())}.sql.gz"

  content_type = "application/gzip"
}

# Download from B2
resource "filemanager_b2_operation" "download_archive" {
  operation = "download"
  service   = filemanager_b2_service.archives.name

  remote_path = "archives/${var.archive_name}"
  local_path  = "/var/restore/${var.archive_name}"

  create_parent_dirs = true
}

# Copy within B2
resource "filemanager_b2_operation" "archive_log" {
  operation = "copy"
  service   = filemanager_b2_service.logs.name

  remote_path      = "current/app.log"
  copy_destination = "archive/${formatdate("YYYY/MM", timestamp())}/app-${formatdate("DD", timestamp())}.log"
}

# List files in B2 bucket
resource "filemanager_b2_operation" "list_backups" {
  operation = "list"
  service   = filemanager_b2_service.backups.name

  remote_path = "backups/"
  prefix      = "database-"
}
