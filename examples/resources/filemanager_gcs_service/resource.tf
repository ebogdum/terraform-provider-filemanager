# Configure Google Cloud Storage service
resource "filemanager_gcs_service" "main" {
  name    = "gcs-main"
  bucket  = var.gcs_bucket
  project = var.gcp_project

  # Uses Application Default Credentials
}

# GCS service with service account key
resource "filemanager_gcs_service" "service_account" {
  name        = "gcs-sa"
  bucket      = var.gcs_bucket
  project     = var.gcp_project
  credentials = file(var.service_account_key_path)
}

# GCS service with credentials JSON
resource "filemanager_gcs_service" "explicit" {
  name        = "gcs-explicit"
  bucket      = var.gcs_bucket
  project     = var.gcp_project
  credentials = var.gcp_credentials_json
}

# GCS service for artifacts bucket
resource "filemanager_gcs_service" "artifacts" {
  name    = "gcs-artifacts"
  bucket  = "${var.gcp_project}-artifacts"
  project = var.gcp_project
}
