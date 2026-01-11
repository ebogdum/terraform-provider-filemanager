# =============================================================================
# GCS OPERATION - VARIABLES
# =============================================================================

variable "gcs_bucket" {
  description = "Google Cloud Storage bucket name"
  type        = string
}

variable "gcs_project" {
  description = "Google Cloud project ID"
  type        = string
}

variable "gcs_credentials" {
  description = "Path to GCS service account credentials JSON file"
  type        = string
  sensitive   = true
}
