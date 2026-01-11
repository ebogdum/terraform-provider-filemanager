# Download a file from a remote server
resource "filemanager_download" "config" {
  source      = "/etc/app/config.json"
  destination = "${path.module}/downloaded/config.json"
  service     = filemanager_ssh_service.server.name

  create_parent_dirs = true
}

# Download database backup
resource "filemanager_download" "backup" {
  source      = "/var/backups/database.sql.gz"
  destination = "${path.module}/backups/database-${formatdate("YYYY-MM-DD", timestamp())}.sql.gz"
  service     = filemanager_ssh_service.server.name

  verify_checksum    = true
  create_parent_dirs = true
}

# Download from S3
resource "filemanager_download" "s3_artifact" {
  source      = "releases/app-${var.version}.tar.gz"
  destination = "/var/releases/app-${var.version}.tar.gz"
  service     = filemanager_s3_service.releases.name

  verify_checksum    = true
  create_parent_dirs = true
}

# Download logs from remote server
resource "filemanager_download" "logs" {
  source      = "/var/log/app/app.log"
  destination = "${path.module}/logs/app-${formatdate("YYYY-MM-DD-hhmm", timestamp())}.log"
  service     = filemanager_ssh_service.server.name

  create_parent_dirs = true
}
