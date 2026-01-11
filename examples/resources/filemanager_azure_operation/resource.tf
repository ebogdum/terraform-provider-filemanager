# Perform Azure Blob Storage operations
resource "filemanager_azure_operation" "upload_config" {
  operation = "upload"
  service   = filemanager_azure_service.main.name

  local_path  = "${path.module}/config.json"
  remote_path = "configs/app/config.json"

  content_type = "application/json"
  metadata = {
    environment = var.environment
    deployed_by = "terraform"
  }
}

# Download from Azure
resource "filemanager_azure_operation" "download_artifact" {
  operation = "download"
  service   = filemanager_azure_service.releases.name

  remote_path = "releases/app-${var.version}.tar.gz"
  local_path  = "/var/releases/app-${var.version}.tar.gz"

  create_parent_dirs = true
}

# Copy within Azure container
resource "filemanager_azure_operation" "archive_backup" {
  operation = "copy"
  service   = filemanager_azure_service.backups.name

  remote_path      = "daily/database.sql.gz"
  copy_destination = "archive/${formatdate("YYYY/MM/DD", timestamp())}/database.sql.gz"
}

# List files in container
resource "filemanager_azure_operation" "list_configs" {
  operation = "list"
  service   = filemanager_azure_service.main.name

  remote_path = "configs/"
  prefix      = "app-"
}
