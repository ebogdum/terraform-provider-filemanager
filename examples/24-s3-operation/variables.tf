# =============================================================================
# S3 OPERATION - VARIABLES
# =============================================================================

variable "s3_bucket" {
  description = "S3 bucket name"
  type        = string
}

variable "s3_region" {
  description = "AWS region for the S3 bucket"
  type        = string
  default     = "us-east-1"
}

variable "s3_access_key" {
  description = "AWS access key ID"
  type        = string
  sensitive   = true
}

variable "s3_secret_key" {
  description = "AWS secret access key"
  type        = string
  sensitive   = true
}
