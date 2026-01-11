# =============================================================================
# FTP OPERATION RESOURCE - ALL USE CASES
# =============================================================================
#
# This example demonstrates the filemanager_ftp_operation resource which
# performs operations on FTP files such as head, copy, and rename.
#
# IMPORTANT: FTP operations require a configured FTP service. First define
# a filemanager_ftp_service resource, then reference its ID in the service field.
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
# FTP SERVICE CONFIGURATION
# -----------------------------------------------------------------------------
# The FTP service resource configures access to an FTP server.
# Supports both FTP and FTPS (FTP over TLS).
# -----------------------------------------------------------------------------

resource "filemanager_ftp_service" "server" {
  host     = var.ftp_host
  port     = var.ftp_port
  username = var.ftp_username
  password = var.ftp_password
  tls      = var.ftp_tls
}

# -----------------------------------------------------------------------------
# HEAD OPERATION - Get File Metadata
# -----------------------------------------------------------------------------
# Retrieves metadata about an FTP file without downloading it.
# Returns size, modification time, permissions, and file type.
# -----------------------------------------------------------------------------

resource "filemanager_ftp_operation" "head" {
  service   = filemanager_ftp_service.server.id
  path      = "/data/config.txt"
  operation = "head"
}

# -----------------------------------------------------------------------------
# COPY OPERATION - Copy File Within FTP Server
# -----------------------------------------------------------------------------
# Server-side copy of a file within the FTP server.
# Note: FTP doesn't have native copy, so this downloads and re-uploads.
# -----------------------------------------------------------------------------

resource "filemanager_ftp_operation" "backup" {
  service          = filemanager_ftp_service.server.id
  path             = "/data/config.txt"
  operation        = "copy"
  destination_path = "/backups/config-backup.txt"
}

# -----------------------------------------------------------------------------
# RENAME OPERATION - Rename/Move File
# -----------------------------------------------------------------------------
# Renames or moves a file on the FTP server using the RENAME command.
# This is a true server-side operation.
# -----------------------------------------------------------------------------

resource "filemanager_ftp_operation" "archive" {
  service          = filemanager_ftp_service.server.id
  path             = "/data/old-file.txt"
  operation        = "rename"
  destination_path = "/archive/old-file.txt"
}
