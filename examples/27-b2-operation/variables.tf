# =============================================================================
# B2 OPERATION - VARIABLES
# =============================================================================

variable "b2_bucket" {
  description = "Backblaze B2 bucket name"
  type        = string
}

variable "b2_application_key_id" {
  description = "Backblaze B2 application key ID"
  type        = string
  sensitive   = true
}

variable "b2_application_key" {
  description = "Backblaze B2 application key"
  type        = string
  sensitive   = true
}
