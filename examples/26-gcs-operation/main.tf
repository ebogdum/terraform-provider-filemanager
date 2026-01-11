# =============================================================================
# GCS OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_gcs_operation resource which
# performs operations on Google Cloud Storage objects such as head, copy,
# set metadata, change storage class, and set temporary hold.
#
# IMPORTANT: GCS operations require a configured GCS service. First define
# a filemanager_gcs_service resource, then reference its ID in the service field.
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
# GCS SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The GCS service resource configures access to Google Cloud Storage.
# -----------------------------------------------------------------------------

resource "filemanager_gcs_service" "prod" {
  bucket      = var.gcs_bucket
  project     = var.gcs_project
  credentials = var.gcs_credentials
}

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get Object Metadata
# -----------------------------------------------------------------------------
# Retrieves metadata about a GCS object without downloading it.
# -----------------------------------------------------------------------------

resource "filemanager_gcs_operation" "head" {
  service     = filemanager_gcs_service.prod.id
  object_path = "data/config.json"
  operation   = "head"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy Object
# -----------------------------------------------------------------------------
# Server-side copy of an object within GCS.
# -----------------------------------------------------------------------------

resource "filemanager_gcs_operation" "backup" {
  service          = filemanager_gcs_service.prod.id
  object_path      = "source/file.txt"
  operation        = "copy"
  destination_path = "backup/file-copy.txt"
}

# -----------------------------------------------------------------------------
# SET METADATA OPERATION
# -----------------------------------------------------------------------------
# Sets or updates custom metadata on a GCS object.
# -----------------------------------------------------------------------------

resource "filemanager_gcs_operation" "metadata" {
  service     = filemanager_gcs_service.prod.id
  object_path = "data/document.txt"
  operation   = "set_metadata"
  metadata = {
    "custom-key"  = "custom-value"
    "environment" = "production"
    "owner"       = "team-platform"
  }
}

# -----------------------------------------------------------------------------
# SET STORAGE CLASS OPERATION
# -----------------------------------------------------------------------------
# Changes the storage class of an object for cost optimization.
# Valid values: STANDARD, NEARLINE, COLDLINE, ARCHIVE
# -----------------------------------------------------------------------------

resource "filemanager_gcs_operation" "archive" {
  service       = filemanager_gcs_service.prod.id
  object_path   = "archive/old-data.zip"
  operation     = "set_storage_class"
  storage_class = "NEARLINE"
}

# -----------------------------------------------------------------------------
# SET TEMPORARY HOLD OPERATION
# -----------------------------------------------------------------------------
# Sets or releases a temporary hold on an object.
# Prevents deletion while hold is active.
# -----------------------------------------------------------------------------

# resource "filemanager_gcs_operation" "hold" {
#   service        = filemanager_gcs_service.prod.id
#   object_path    = "legal/contract.pdf"
#   operation      = "set_temporary_hold"
#   temporary_hold = true
# }
