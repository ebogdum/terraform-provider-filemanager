# Create an INI configuration file
resource "filemanager_ini_file" "config" {
  path = "/etc/app/config.ini"
  data = {
    general = {
      app_name = "MyApplication"
      version  = "1.0.0"
      debug    = "false"
    }
    database = {
      host     = var.db_host
      port     = "5432"
      username = var.db_user
      password = var.db_password
    }
    logging = {
      level  = "INFO"
      file   = "/var/log/app.log"
      format = "json"
    }
  }

  create_parent_dirs = true
}

# Create PHP configuration
resource "filemanager_ini_file" "php" {
  path = "/etc/php/8.2/fpm/conf.d/custom.ini"
  data = {
    PHP = {
      memory_limit        = "256M"
      upload_max_filesize = "64M"
      post_max_size       = "64M"
      max_execution_time  = "300"
    }
  }

  create_parent_dirs = true
}

# Merge with existing INI file
resource "filemanager_ini_file" "merged" {
  path = "/etc/app/settings.ini"
  data = {
    new_section = {
      key = "value"
    }
  }

  merge = true
}
