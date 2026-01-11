# =============================================================================
# B2 OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_b2_operation resource which
# performs operations on Backblaze B2 files such as head, get_file_info, copy,
# update_file_info, hide, and update_legal_hold.
#
# IMPORTANT: B2 operations require a configured B2 service. First define
# a filemanager_b2_service resource, then reference its ID in the service field.
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
# B2 SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The B2 service resource configures access to Backblaze B2 Cloud Storage.
# -----------------------------------------------------------------------------

resource "filemanager_b2_service" "prod" {
  bucket             = var.b2_bucket
  application_key_id = var.b2_application_key_id
  application_key    = var.b2_application_key
}

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get File Metadata
# -----------------------------------------------------------------------------
# Retrieves metadata about a B2 file without downloading it.
# -----------------------------------------------------------------------------

resource "filemanager_b2_operation" "head" {
  service   = filemanager_b2_service.prod.id
  file_path = "documents/report.pdf"
  operation = "head"
}

# -----------------------------------------------------------------------------
# GET FILE INFO OPERATION - Detailed File Information
# -----------------------------------------------------------------------------
# Retrieves detailed information about a specific file including file info map.
# -----------------------------------------------------------------------------

resource "filemanager_b2_operation" "info" {
  service   = filemanager_b2_service.prod.id
  file_path = "documents/report.pdf"
  operation = "get_file_info"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy File Within B2
# -----------------------------------------------------------------------------
# Server-side copy of a file within B2.
# -----------------------------------------------------------------------------

resource "filemanager_b2_operation" "backup" {
  service          = filemanager_b2_service.prod.id
  file_path        = "documents/report.pdf"
  operation        = "copy"
  destination_path = "backups/report-backup.pdf"
}

# -----------------------------------------------------------------------------
# UPDATE FILE INFO OPERATION - Update Custom Metadata
# -----------------------------------------------------------------------------
# Updates the custom file info (metadata) on a B2 file.
# -----------------------------------------------------------------------------

resource "filemanager_b2_operation" "metadata" {
  service   = filemanager_b2_service.prod.id
  file_path = "documents/report.pdf"
  operation = "update_file_info"
  file_info = {
    "Author"     = "John Doe"
    "Department" = "Engineering"
    "Version"    = "1.0"
  }
}

# -----------------------------------------------------------------------------
# HIDE OPERATION - Soft Delete (Hide) File
# -----------------------------------------------------------------------------
# Hides a file version (soft delete). The file can be restored.
# -----------------------------------------------------------------------------

# resource "filemanager_b2_operation" "hide" {
#   service   = filemanager_b2_service.prod.id
#   file_path = "documents/old-report.pdf"
#   operation = "hide"
# }

# -----------------------------------------------------------------------------
# UPDATE LEGAL HOLD OPERATION
# -----------------------------------------------------------------------------
# Sets or releases a legal hold on a file.
# Requires B2 bucket to have file lock enabled.
# -----------------------------------------------------------------------------

# resource "filemanager_b2_operation" "legal_hold" {
#   service    = filemanager_b2_service.prod.id
#   file_path  = "legal/contract.pdf"
#   operation  = "update_legal_hold"
#   legal_hold = true
# }
