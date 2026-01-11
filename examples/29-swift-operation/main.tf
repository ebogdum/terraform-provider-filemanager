# =============================================================================
# SWIFT OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_swift_operation resource which
# performs operations on OpenStack Swift objects such as head, copy, and
# set metadata.
#
# IMPORTANT: Swift operations require a configured Swift service. First define
# a filemanager_swift_service resource, then reference its ID in the service field.
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
# SWIFT SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The Swift service resource configures access to OpenStack Swift object storage.
# Supports Keystone authentication (v2 and v3) and application credentials.
# -----------------------------------------------------------------------------

resource "filemanager_swift_service" "storage" {
  auth_url  = var.swift_auth_url
  container = var.swift_container
  username  = var.swift_username
  password  = var.swift_password
  tenant    = var.swift_tenant
  region    = var.swift_region
}

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get Object Metadata
# -----------------------------------------------------------------------------
# Retrieves metadata about a Swift object without downloading it.
# Returns size, last modified, content type, etag, and custom metadata.
# -----------------------------------------------------------------------------

resource "filemanager_swift_operation" "head" {
  service     = filemanager_swift_service.storage.id
  object_path = "/data/config.json"
  operation   = "head"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy Object Within Container
# -----------------------------------------------------------------------------
# Server-side copy of an object within the Swift container.
# -----------------------------------------------------------------------------

resource "filemanager_swift_operation" "backup" {
  service          = filemanager_swift_service.storage.id
  object_path      = "/data/config.json"
  operation        = "copy"
  destination_path = "/backups/config-backup.json"
}

# -----------------------------------------------------------------------------
# SET METADATA OPERATION - Set Custom Metadata
# -----------------------------------------------------------------------------
# Sets or updates custom metadata (X-Object-Meta-*) on a Swift object.
# -----------------------------------------------------------------------------

resource "filemanager_swift_operation" "metadata" {
  service     = filemanager_swift_service.storage.id
  object_path = "/data/config.json"
  operation   = "set_metadata"
  metadata = {
    "Version"     = "1.0"
    "Author"      = "terraform"
    "Environment" = "production"
  }
}
