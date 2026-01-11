# Transfer files between different backends
resource "filemanager_transfer" "s3_to_server" {
  source          = "backups/latest.sql.gz"
  source_service  = filemanager_s3_service.backups.name
  destination     = "/var/backups/database.sql.gz"
  dest_service    = filemanager_ssh_service.server.name

  verify_checksum = true
}

# Transfer from Azure to S3
resource "filemanager_transfer" "azure_to_s3" {
  source          = "exports/data.csv"
  source_service  = filemanager_azure_service.source.name
  destination     = "imports/data.csv"
  dest_service    = filemanager_s3_service.destination.name

  verify_checksum = true
}

# Transfer between servers
resource "filemanager_transfer" "server_to_server" {
  source          = "/var/data/important.db"
  source_service  = filemanager_ssh_service.primary.name
  destination     = "/var/data/replica.db"
  dest_service    = filemanager_ssh_service.secondary.name

  verify_checksum = true
}

# Transfer from GCS to local
resource "filemanager_transfer" "gcs_to_local" {
  source         = "artifacts/build-${var.build_number}.tar.gz"
  source_service = filemanager_gcs_service.artifacts.name
  destination    = "/var/releases/build-${var.build_number}.tar.gz"

  verify_checksum    = true
  create_parent_dirs = true
}
