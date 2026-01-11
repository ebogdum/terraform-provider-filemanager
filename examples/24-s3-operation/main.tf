# =============================================================================
# S3 OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_s3_operation resource which performs
# operations on S3 objects such as head, copy, set metadata, set tags, change
# storage class, or restore from Glacier.
#
# IMPORTANT: S3 operations require a configured S3 service. First define a
# filemanager_s3_service resource, then reference its ID in the service field.
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
# S3 SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The S3 service resource configures access to an S3-compatible storage backend.
# This can be AWS S3, MinIO, DigitalOcean Spaces, or any S3-compatible service.
# -----------------------------------------------------------------------------

# AWS S3 Service
resource "filemanager_s3_service" "prod" {
  bucket     = var.s3_bucket
  region     = var.s3_region
  access_key = var.s3_access_key
  secret_key = var.s3_secret_key
}

# MinIO (S3-compatible) Service - Alternative configuration
# resource "filemanager_s3_service" "minio" {
#   bucket   = var.minio_bucket
#   endpoint = var.minio_endpoint
#   region   = "us-east-1"
#   access_key = var.minio_access_key
#   secret_key = var.minio_secret_key
# }

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get Object Metadata
# -----------------------------------------------------------------------------
# Retrieves metadata about an S3 object without downloading it.
# Useful for checking existence, size, content-type, and other attributes.
# -----------------------------------------------------------------------------

resource "filemanager_s3_operation" "head" {
  service   = filemanager_s3_service.prod.id
  key       = "data/config.json"
  operation = "head"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy Object Within S3
# -----------------------------------------------------------------------------
# Server-side copy of an object within S3 (no data transfer through client).
# -----------------------------------------------------------------------------

resource "filemanager_s3_operation" "backup" {
  service         = filemanager_s3_service.prod.id
  key             = "data/original.json"
  operation       = "copy"
  destination_key = "backups/original-backup.json"
}

# -----------------------------------------------------------------------------
# SET METADATA OPERATION
# -----------------------------------------------------------------------------
# Sets or updates custom metadata on an S3 object.
# Note: This performs a COPY operation with new metadata in S3.
# -----------------------------------------------------------------------------

resource "filemanager_s3_operation" "metadata" {
  service   = filemanager_s3_service.prod.id
  key       = "data/document.txt"
  operation = "set_metadata"
  metadata = {
    "x-amz-meta-version" = "1.0"
    "x-amz-meta-author"  = "terraform"
  }
}

# -----------------------------------------------------------------------------
# SET TAGS OPERATION
# -----------------------------------------------------------------------------
# Sets object tags for organization, billing, and lifecycle management.
# -----------------------------------------------------------------------------

resource "filemanager_s3_operation" "tags" {
  service   = filemanager_s3_service.prod.id
  key       = "data/document.txt"
  operation = "set_tags"
  tags = {
    Environment = "production"
    ManagedBy   = "terraform"
    Project     = "filemanager"
  }
}

# -----------------------------------------------------------------------------
# DELETE TAGS OPERATION
# -----------------------------------------------------------------------------
# Removes all tags from an S3 object.
# -----------------------------------------------------------------------------

# resource "filemanager_s3_operation" "delete_tags" {
#   service   = filemanager_s3_service.prod.id
#   key       = "data/document.txt"
#   operation = "delete_tags"
# }

# -----------------------------------------------------------------------------
# SET STORAGE CLASS OPERATION
# -----------------------------------------------------------------------------
# Changes the storage class of an object for cost optimization.
# Valid values: STANDARD, REDUCED_REDUNDANCY, STANDARD_IA, ONEZONE_IA,
#               INTELLIGENT_TIERING, GLACIER, DEEP_ARCHIVE, GLACIER_IR
# -----------------------------------------------------------------------------

resource "filemanager_s3_operation" "archive" {
  service       = filemanager_s3_service.prod.id
  key           = "archive/old-data.tar.gz"
  operation     = "set_storage_class"
  storage_class = "GLACIER"
}

# -----------------------------------------------------------------------------
# RESTORE OPERATION - Restore from Glacier
# -----------------------------------------------------------------------------
# Initiates restoration of an object from Glacier or Deep Archive.
# -----------------------------------------------------------------------------

# resource "filemanager_s3_operation" "restore" {
#   service      = filemanager_s3_service.prod.id
#   key          = "archive/glacier-file.zip"
#   operation    = "restore"
#   restore_days = 7
# }
