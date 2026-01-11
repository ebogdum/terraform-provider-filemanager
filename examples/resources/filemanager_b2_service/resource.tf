# Configure Backblaze B2 service
resource "filemanager_b2_service" "main" {
  name       = "b2-main"
  bucket     = var.b2_bucket
  key_id     = var.b2_key_id
  app_key    = var.b2_app_key
}

# B2 service for backups
resource "filemanager_b2_service" "backups" {
  name       = "b2-backups"
  bucket     = var.b2_backup_bucket
  key_id     = var.b2_key_id
  app_key    = var.b2_app_key
}

# B2 service for archives with custom endpoint
resource "filemanager_b2_service" "archives" {
  name       = "b2-archives"
  bucket     = var.b2_archive_bucket
  key_id     = var.b2_key_id
  app_key    = var.b2_app_key
  endpoint   = var.b2_custom_endpoint
}
