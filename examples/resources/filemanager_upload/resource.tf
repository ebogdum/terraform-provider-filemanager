# Upload a file to a remote server
resource "filemanager_upload" "config" {
  source      = "${path.module}/config.json"
  destination = "/etc/app/config.json"
  service     = filemanager_ssh_service.server.name

  file_permission    = "0644"
  create_parent_dirs = true
}

# Upload application artifact
resource "filemanager_upload" "artifact" {
  source      = "${path.module}/dist/app.tar.gz"
  destination = "/opt/releases/app-${var.version}.tar.gz"
  service     = filemanager_ssh_service.server.name

  verify_checksum    = true
  create_parent_dirs = true
}

# Upload with specific permissions
resource "filemanager_upload" "script" {
  source      = "${path.module}/scripts/deploy.sh"
  destination = "/opt/scripts/deploy.sh"
  service     = filemanager_ssh_service.server.name

  file_permission    = "0755"
  create_parent_dirs = true
}

# Upload to S3-compatible storage
resource "filemanager_upload" "s3_backup" {
  source      = "/var/backups/database.sql.gz"
  destination = "backups/database-${formatdate("YYYY-MM-DD", timestamp())}.sql.gz"
  service     = filemanager_s3_service.backup.name

  verify_checksum = true
}
