# Configure Azure Blob Storage service
resource "filemanager_azure_service" "main" {
  name             = "azure-main"
  storage_account  = var.storage_account
  container        = var.container_name
  connection_string = var.azure_connection_string
}

# Azure service with SAS token
resource "filemanager_azure_service" "sas" {
  name            = "azure-sas"
  storage_account = var.storage_account
  container       = var.container_name
  sas_token       = var.sas_token
}

# Azure service with account key
resource "filemanager_azure_service" "key" {
  name            = "azure-key"
  storage_account = var.storage_account
  container       = var.container_name
  account_key     = var.storage_account_key
}

# Azure service for backup container
resource "filemanager_azure_service" "backup" {
  name              = "azure-backup"
  storage_account   = var.storage_account
  container         = "backups"
  connection_string = var.azure_connection_string
}
