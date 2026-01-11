# Create a JSON configuration file
resource "filemanager_json_file" "config" {
  path = "/etc/app/config.json"
  data = {
    app_name    = "MyApplication"
    version     = "1.0.0"
    environment = var.environment
    database = {
      host = var.db_host
      port = 5432
      name = "myapp"
    }
    features = {
      logging   = true
      analytics = var.enable_analytics
    }
  }

  pretty_print       = true
  create_parent_dirs = true
}

# Merge with existing JSON file
resource "filemanager_json_file" "merged" {
  path = "/etc/app/settings.json"
  data = {
    new_setting = "value"
    nested = {
      updated_field = "new_value"
    }
  }

  merge      = true
  deep_merge = true
}

# Create JSON file on remote server
resource "filemanager_json_file" "remote_config" {
  path    = "/opt/app/config.json"
  service = filemanager_ssh_service.server.name
  data = {
    server_id = "prod-01"
  }

  pretty_print       = true
  create_parent_dirs = true
}
