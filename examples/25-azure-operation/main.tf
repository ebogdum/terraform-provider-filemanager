# =============================================================================
# AZURE OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_azure_operation resource which
# performs operations on Azure Blob Storage objects such as head, copy,
# set metadata, set tags, and change access tier.
#
# IMPORTANT: Azure operations require a configured Azure service. First define
# a filemanager_azure_service resource, then reference its ID in the service field.
#
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {}

# -----------------------------------------------------------------------------
# AZURE SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The Azure service resource configures access to Azure Blob Storage.
# -----------------------------------------------------------------------------

resource "filemanager_azure_service" "prod" {
  storage_account = var.azure_storage_account
  container       = var.azure_container
  access_key      = var.azure_access_key
}

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get Blob Properties
# -----------------------------------------------------------------------------
# Retrieves metadata and properties about an Azure blob without downloading it.
# -----------------------------------------------------------------------------

resource "filemanager_azure_operation" "head" {
  service   = filemanager_azure_service.prod.id
  blob_path = "data/config.json"
  operation = "head"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy Blob
# -----------------------------------------------------------------------------
# Server-side copy of a blob within the same container.
# -----------------------------------------------------------------------------

resource "filemanager_azure_operation" "backup" {
  service          = filemanager_azure_service.prod.id
  blob_path        = "source/document.pdf"
  operation        = "copy"
  destination_path = "backup/document-backup.pdf"
}

# -----------------------------------------------------------------------------
# SET METADATA OPERATION
# -----------------------------------------------------------------------------
# Sets or updates custom metadata on an Azure blob.
# -----------------------------------------------------------------------------

resource "filemanager_azure_operation" "metadata" {
  service   = filemanager_azure_service.prod.id
  blob_path = "data/document.txt"
  operation = "set_metadata"
  metadata = {
    environment = "production"
    owner       = "team-a"
    version     = "1.0"
  }
}

# -----------------------------------------------------------------------------
# SET TAGS OPERATION
# -----------------------------------------------------------------------------
# Sets blob tags for organization and billing management.
# -----------------------------------------------------------------------------

resource "filemanager_azure_operation" "tags" {
  service   = filemanager_azure_service.prod.id
  blob_path = "data/document.txt"
  operation = "set_tags"
  tags = {
    project    = "myproject"
    costcenter = "12345"
    department = "engineering"
  }
}

# -----------------------------------------------------------------------------
# SET ACCESS TIER OPERATION
# -----------------------------------------------------------------------------
# Changes the access tier of a blob for cost optimization.
# Valid values: Hot, Cool, Archive
# -----------------------------------------------------------------------------

resource "filemanager_azure_operation" "archive" {
  service     = filemanager_azure_service.prod.id
  blob_path   = "archive/old-data.zip"
  operation   = "set_access_tier"
  access_tier = "Cool"
}

# -----------------------------------------------------------------------------
# ACQUIRE LEASE OPERATION
# -----------------------------------------------------------------------------
# Acquires a lease on a blob for exclusive write access.
# Lease duration can be 15-60 seconds or -1 for infinite.
# -----------------------------------------------------------------------------

# resource "filemanager_azure_operation" "lease" {
#   service        = filemanager_azure_service.prod.id
#   blob_path      = "locks/resource.lock"
#   operation      = "acquire_lease"
#   lease_duration = 60
# }

# -----------------------------------------------------------------------------
# RELEASE LEASE OPERATION
# -----------------------------------------------------------------------------
# Releases a lease on a blob.
# -----------------------------------------------------------------------------

# resource "filemanager_azure_operation" "release_lease" {
#   service   = filemanager_azure_service.prod.id
#   blob_path = "locks/resource.lock"
#   operation = "release_lease"
#
#   depends_on = [filemanager_azure_operation.lease]
# }
