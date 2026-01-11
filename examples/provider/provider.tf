# Basic provider configuration
provider "filemanager" {}

# Provider with ACID guarantees enabled
provider "filemanager" {
  alias = "safe"

  atomic_writes   = true
  verify_checksum = true
  enable_locking  = true
  lock_timeout    = "30s"
}

# Provider with backup configuration
provider "filemanager" {
  alias = "with_backups"

  backup_enabled   = true
  backup_retention = 5
  backup_dir       = "/var/backups/terraform"
}
