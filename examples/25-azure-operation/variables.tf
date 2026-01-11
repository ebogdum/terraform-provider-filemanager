# =============================================================================
# AZURE OPERATION - VARIABLES
# =============================================================================

variable "azure_storage_account" {
  description = "Azure Storage account name"
  type        = string
}

variable "azure_container" {
  description = "Azure Blob Storage container name"
  type        = string
}

variable "azure_access_key" {
  description = "Azure Storage account access key"
  type        = string
  sensitive   = true
}
