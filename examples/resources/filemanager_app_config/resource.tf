# Create multi-format application configuration
resource "filemanager_app_config" "main" {
  path   = "/etc/app/config"
  format = "auto"  # Automatically detect from filename extension

  data = {
    app = {
      name        = "MyApplication"
      version     = "2.0.0"
      environment = var.environment
    }
    server = {
      host    = "0.0.0.0"
      port    = 8080
      workers = 4
    }
    database = {
      primary = {
        host     = var.db_host
        port     = 5432
        name     = var.db_name
        pool_size = 10
      }
      replica = {
        host     = var.db_replica_host
        port     = 5432
        name     = var.db_name
        pool_size = 5
      }
    }
    cache = {
      enabled = true
      ttl     = 3600
      backend = "redis"
    }
    features = var.feature_flags
  }

  create_parent_dirs = true
}

# Create configuration in specific format
resource "filemanager_app_config" "json_config" {
  path   = "/etc/app/settings.json"
  format = "json"

  data = {
    settings = var.app_settings
  }

  pretty_print       = true
  create_parent_dirs = true
}

# Create configuration with merge capability
resource "filemanager_app_config" "merged" {
  path   = "/etc/app/merged-config.yaml"
  format = "yaml"

  data = {
    new_section = {
      key = "value"
    }
  }

  merge              = true
  deep_merge         = true
  create_parent_dirs = true
}
