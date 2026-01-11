# =============================================================================
# INTEGRATION TESTS - RESOURCE DEPENDENCIES & OUTPUT CHAINING
# =============================================================================
#
# This module demonstrates realistic workflows where:
# - Resources depend on each other
# - Outputs from one resource are used in another
# - Templates generate content used elsewhere
# - Data sources read from created resources
# - Archives package dynamically created content
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

locals {
  base_dir = "${path.module}/../../test/output/15-integration"
}

# =============================================================================
# PHASE 1: BASE INFRASTRUCTURE
# =============================================================================
# Create the directory structure first, all other resources depend on these

resource "filemanager_directory" "app_root" {
  path           = "${local.base_dir}/app"
  create_parents = true
}

resource "filemanager_directory" "app_config" {
  path           = "${filemanager_directory.app_root.path}/config"
  create_parents = true
}

resource "filemanager_directory" "app_data" {
  path           = "${filemanager_directory.app_root.path}/data"
  create_parents = true
}

resource "filemanager_directory" "app_logs" {
  path           = "${filemanager_directory.app_root.path}/logs"
  create_parents = true
}

resource "filemanager_directory" "app_scripts" {
  path           = "${filemanager_directory.app_root.path}/scripts"
  create_parents = true
}

resource "filemanager_directory" "backups" {
  path           = "${local.base_dir}/backups"
  create_parents = true
}

resource "filemanager_directory" "deploy" {
  path           = "${local.base_dir}/deploy"
  create_parents = true
}

# =============================================================================
# PHASE 2: CORE CONFIGURATION FILES
# =============================================================================

# App settings - JSON config (uses jsonencode)
resource "filemanager_json_file" "app_settings" {
  path = "${filemanager_directory.app_config.path}/settings.json"

  content = jsonencode({
    app_name    = "MyApplication"
    version     = "1.0.0"
    environment = "production"
    debug       = false

    server = {
      host = "0.0.0.0"
      port = 8080
    }

    paths = {
      data_dir = filemanager_directory.app_data.path
      log_dir  = filemanager_directory.app_logs.path
    }
  })

  sort_keys          = true
  indent             = 2
  create_parent_dirs = true
}

# Database config - YAML format (uses yamlencode)
resource "filemanager_yaml_file" "database" {
  path = "${filemanager_directory.app_config.path}/database.yaml"

  content = yamlencode({
    database = {
      driver    = "postgres"
      host      = "localhost"
      port      = 5432
      name      = "myapp_db"
      user      = "app_user"
      pool_size = 10
    }

    redis = {
      host = "localhost"
      port = 6379
      db   = 0
    }
  })

  create_parent_dirs = true
}

# Logging config - uses log directory path
resource "filemanager_yaml_file" "logging" {
  path = "${filemanager_directory.app_config.path}/logging.yaml"

  content = yamlencode({
    logging = {
      level  = "INFO"
      format = "%(asctime)s - %(name)s - %(levelname)s - %(message)s"

      handlers = {
        file = {
          path         = "${filemanager_directory.app_logs.path}/app.log"
          max_size     = "10MB"
          backup_count = 5
        }
        error = {
          path  = "${filemanager_directory.app_logs.path}/error.log"
          level = "ERROR"
        }
      }
    }
  })

  create_parent_dirs = true
}

# =============================================================================
# PHASE 3: ENVIRONMENT-SPECIFIC CONFIGURATION
# =============================================================================

resource "filemanager_env_file" "production" {
  path = "${filemanager_directory.app_config.path}/.env.production"

  variables = {
    APP_NAME  = "MyApplication"
    APP_ENV   = "production"
    APP_DEBUG = "false"

    # Reference paths from created directories
    CONFIG_PATH = filemanager_directory.app_config.path
    DATA_PATH   = filemanager_directory.app_data.path
    LOG_PATH    = filemanager_directory.app_logs.path

    # Reference created config files
    SETTINGS_FILE   = filemanager_json_file.app_settings.path
    DATABASE_CONFIG = filemanager_yaml_file.database.path
    LOGGING_CONFIG  = filemanager_yaml_file.logging.path

    # Database settings
    DB_HOST = "localhost"
    DB_PORT = "5432"
    DB_NAME = "myapp_db"
  }

  create_parent_dirs = true
}

resource "filemanager_env_file" "development" {
  path = "${filemanager_directory.app_config.path}/.env.development"

  variables = {
    APP_NAME  = "MyApplication"
    APP_ENV   = "development"
    APP_DEBUG = "true"

    CONFIG_PATH = filemanager_directory.app_config.path
    DATA_PATH   = filemanager_directory.app_data.path
    LOG_PATH    = filemanager_directory.app_logs.path
  }

  create_parent_dirs = true
}

# =============================================================================
# PHASE 4: TEMPLATE-BASED CONFIGURATION
# =============================================================================

# Nginx config template using app settings
resource "filemanager_template_file" "nginx_config" {
  path = "${filemanager_directory.app_config.path}/nginx.conf"

  template = <<-EOF
    # Generated Nginx Configuration
    # App: {{.app_name}}
    # Config: {{.settings_path}}

    upstream app_backend {
        server {{.host}}:{{.port}};
    }

    server {
        listen 80;
        server_name {{.server_name}};

        access_log {{.log_dir}}/nginx_access.log;
        error_log {{.log_dir}}/nginx_error.log;

        location / {
            proxy_pass http://app_backend;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        location /static {
            alias {{.data_dir}}/static;
        }
    }
  EOF

  vars = {
    app_name      = "MyApplication"
    settings_path = filemanager_json_file.app_settings.path
    host          = "127.0.0.1"
    port          = "8080"
    server_name   = "app.example.com"
    log_dir       = filemanager_directory.app_logs.path
    data_dir      = filemanager_directory.app_data.path
  }

  engine             = "go"
  create_parent_dirs = true
}

# Systemd service file template
resource "filemanager_template_file" "systemd_service" {
  path = "${filemanager_directory.app_config.path}/myapp.service"

  template = <<-EOF
    [Unit]
    Description={{.description}}
    After=network.target

    [Service]
    Type=simple
    User={{.user}}
    Group={{.group}}
    WorkingDirectory={{.app_dir}}
    Environment="CONFIG_FILE={{.config_file}}"
    Environment="ENV_FILE={{.env_file}}"
    ExecStart={{.scripts_dir}}/start.sh
    ExecStop={{.scripts_dir}}/stop.sh
    Restart=always
    RestartSec=5

    [Install]
    WantedBy=multi-user.target
  EOF

  vars = {
    description = "MyApplication Service"
    user        = "app"
    group       = "app"
    app_dir     = filemanager_directory.app_root.path
    config_file = filemanager_json_file.app_settings.path
    env_file    = filemanager_env_file.production.path
    scripts_dir = filemanager_directory.app_scripts.path
  }

  engine             = "go"
  create_parent_dirs = true
}

# =============================================================================
# PHASE 5: SCRIPTS USING TEMPLATE OUTPUT
# =============================================================================

resource "filemanager_template_file" "start_script" {
  path = "${filemanager_directory.app_scripts.path}/start.sh"

  template = <<-EOF
    #!/bin/bash
    # Start script for {{.app_name}}

    set -e

    APP_ROOT="{{.app_root}}"
    CONFIG_FILE="{{.config_file}}"
    ENV_FILE="{{.env_file}}"
    LOG_DIR="{{.log_dir}}"
    PID_FILE="{{.data_dir}}/app.pid"

    echo "Starting {{.app_name}}..."
    echo "Using config: $CONFIG_FILE"
    echo "Using env: $ENV_FILE"

    # Source environment
    if [ -f "$ENV_FILE" ]; then
        set -a
        source "$ENV_FILE"
        set +a
    fi

    # Start application
    cd "$APP_ROOT"
    echo "Started" > "$PID_FILE"
  EOF

  vars = {
    app_name    = "MyApplication"
    app_root    = filemanager_directory.app_root.path
    config_file = filemanager_json_file.app_settings.path
    env_file    = filemanager_env_file.production.path
    log_dir     = filemanager_directory.app_logs.path
    data_dir    = filemanager_directory.app_data.path
  }

  engine             = "go"
  file_permission    = "0755"
  create_parent_dirs = true
}

resource "filemanager_template_file" "stop_script" {
  path = "${filemanager_directory.app_scripts.path}/stop.sh"

  template = <<-EOF
    #!/bin/bash
    # Stop script for {{.app_name}}

    PID_FILE="{{.data_dir}}/app.pid"

    if [ -f "$PID_FILE" ]; then
        echo "Stopping {{.app_name}}..."
        rm -f "$PID_FILE"
    else
        echo "PID file not found"
    fi
  EOF

  vars = {
    app_name = "MyApplication"
    data_dir = filemanager_directory.app_data.path
  }

  engine             = "go"
  file_permission    = "0755"
  create_parent_dirs = true
}

resource "filemanager_template_file" "backup_script" {
  path = "${filemanager_directory.app_scripts.path}/backup.sh"

  template = <<-EOF
    #!/bin/bash
    # Backup script for {{.app_name}}

    BACKUP_DIR="{{.backup_dir}}"
    DATA_DIR="{{.data_dir}}"
    CONFIG_DIR="{{.config_dir}}"
    TIMESTAMP=$(date +%Y%m%d_%H%M%S)
    BACKUP_FILE="$BACKUP_DIR/backup_$TIMESTAMP.tar.gz"

    echo "Creating backup: $BACKUP_FILE"
    echo "Backup complete"
  EOF

  vars = {
    app_name   = "MyApplication"
    backup_dir = filemanager_directory.backups.path
    data_dir   = filemanager_directory.app_data.path
    config_dir = filemanager_directory.app_config.path
  }

  engine             = "go"
  file_permission    = "0755"
  create_parent_dirs = true
}

# =============================================================================
# PHASE 6: DATA FILES USING CONFIG VALUES
# =============================================================================

resource "filemanager_json_file" "sample_data" {
  path = "${filemanager_directory.app_data.path}/sample_data.json"

  content = jsonencode({
    metadata = {
      created_by  = "terraform"
      config_file = filemanager_json_file.app_settings.path
      app_version = ">= 1.0.0"
    }

    records = [
      { id = 1, name = "Record 1", active = true },
      { id = 2, name = "Record 2", active = true },
      { id = 3, name = "Record 3", active = false }
    ]
  })

  sort_keys          = true
  indent             = 2
  create_parent_dirs = true
}

# Initial log file
resource "filemanager_file" "initial_log" {
  path               = "${filemanager_directory.app_logs.path}/app.log"
  content            = "[INFO] Application initialized\n[INFO] Config loaded from: ${filemanager_json_file.app_settings.path}\n"
  create_parent_dirs = true
}

# =============================================================================
# PHASE 7: MANIFEST FILES - AGGREGATING INFORMATION
# =============================================================================

resource "filemanager_json_file" "manifest" {
  path = "${filemanager_directory.app_root.path}/manifest.json"

  content = jsonencode({
    application = {
      name    = "MyApplication"
      version = ">= 1.0.0"
    }

    paths = {
      root    = filemanager_directory.app_root.path
      config  = filemanager_directory.app_config.path
      data    = filemanager_directory.app_data.path
      logs    = filemanager_directory.app_logs.path
      scripts = filemanager_directory.app_scripts.path
      backups = filemanager_directory.backups.path
    }

    config_files = {
      settings = filemanager_json_file.app_settings.path
      database = filemanager_yaml_file.database.path
      logging  = filemanager_yaml_file.logging.path
      nginx    = filemanager_template_file.nginx_config.path
      systemd  = filemanager_template_file.systemd_service.path
    }

    env_files = {
      production  = filemanager_env_file.production.path
      development = filemanager_env_file.development.path
    }

    scripts = {
      start  = filemanager_template_file.start_script.path
      stop   = filemanager_template_file.stop_script.path
      backup = filemanager_template_file.backup_script.path
    }

    data_files = {
      sample = filemanager_json_file.sample_data.path
    }
  })

  sort_keys          = true
  indent             = 2
  create_parent_dirs = true
}

# INI-style registry of all components
resource "filemanager_ini_file" "registry" {
  path = "${filemanager_directory.app_root.path}/registry.ini"

  sections = {
    "application" = jsonencode({
      name    = "MyApplication"
      version = ">= 1.0.0"
    })

    "directories" = jsonencode({
      root    = filemanager_directory.app_root.path
      config  = filemanager_directory.app_config.path
      data    = filemanager_directory.app_data.path
      logs    = filemanager_directory.app_logs.path
      scripts = filemanager_directory.app_scripts.path
    })

    "configs" = jsonencode({
      settings = filemanager_json_file.app_settings.path
      database = filemanager_yaml_file.database.path
      logging  = filemanager_yaml_file.logging.path
    })

    "scripts" = jsonencode({
      start  = filemanager_template_file.start_script.path
      stop   = filemanager_template_file.stop_script.path
      backup = filemanager_template_file.backup_script.path
    })
  }

  create_parent_dirs = true
}

# =============================================================================
# PHASE 8: DATA SOURCES READING CREATED CONTENT
# =============================================================================

data "filemanager_file" "read_settings" {
  path = filemanager_json_file.app_settings.path

  depends_on = [filemanager_json_file.app_settings]
}

data "filemanager_file" "read_manifest" {
  path = filemanager_json_file.manifest.path

  depends_on = [filemanager_json_file.manifest]
}

data "filemanager_checksum" "settings_checksum" {
  path      = filemanager_json_file.app_settings.path
  algorithm = "sha256"

  depends_on = [filemanager_json_file.app_settings]
}

data "filemanager_checksum" "database_checksum" {
  path      = filemanager_yaml_file.database.path
  algorithm = "sha256"

  depends_on = [filemanager_yaml_file.database]
}

data "filemanager_stat" "config_dir_stat" {
  path = filemanager_directory.app_config.path

  depends_on = [filemanager_directory.app_config]
}

data "filemanager_directory" "list_configs" {
  path      = filemanager_directory.app_config.path
  recursive = true

  depends_on = [
    filemanager_json_file.app_settings,
    filemanager_yaml_file.database,
    filemanager_yaml_file.logging,
    filemanager_template_file.nginx_config,
    filemanager_env_file.production,
    filemanager_env_file.development,
  ]
}

data "filemanager_directory" "list_scripts" {
  path    = filemanager_directory.app_scripts.path
  pattern = "*.sh"

  depends_on = [
    filemanager_template_file.start_script,
    filemanager_template_file.stop_script,
    filemanager_template_file.backup_script,
  ]
}

# =============================================================================
# PHASE 9: ARCHIVES PACKAGING CREATED CONTENT
# =============================================================================

resource "filemanager_archive" "config_archive" {
  path       = "${filemanager_directory.backups.path}/config_backup.tar.gz"
  type       = "tar.gz"
  source_dir = filemanager_directory.app_config.path

  excludes = ["*.log", "*.tmp"]

  create_parent_dirs = true

  depends_on = [
    filemanager_json_file.app_settings,
    filemanager_yaml_file.database,
    filemanager_yaml_file.logging,
    filemanager_template_file.nginx_config,
    filemanager_template_file.systemd_service,
    filemanager_env_file.production,
    filemanager_env_file.development,
  ]
}

resource "filemanager_archive" "scripts_archive" {
  path       = "${filemanager_directory.backups.path}/scripts_backup.zip"
  type       = "zip"
  source_dir = filemanager_directory.app_scripts.path

  create_parent_dirs = true

  depends_on = [
    filemanager_template_file.start_script,
    filemanager_template_file.stop_script,
    filemanager_template_file.backup_script,
  ]
}

resource "filemanager_archive" "full_app_archive" {
  path       = "${filemanager_directory.backups.path}/full_app_backup.tar.gz"
  type       = "tar.gz"
  source_dir = filemanager_directory.app_root.path

  excludes = ["*.log", "*.tmp", "*.pid"]

  create_parent_dirs = true

  depends_on = [
    filemanager_json_file.manifest,
    filemanager_ini_file.registry,
    filemanager_archive.config_archive,
    filemanager_archive.scripts_archive,
  ]
}

# =============================================================================
# PHASE 10: COPY OPERATIONS FOR DEPLOYMENT
# =============================================================================

resource "filemanager_copy" "deploy_configs" {
  source      = filemanager_directory.app_config.path
  destination = "${filemanager_directory.deploy.path}/config"

  recursive = true
  overwrite = true
  excludes  = [".env.development"]

  depends_on = [
    filemanager_json_file.app_settings,
    filemanager_yaml_file.database,
    filemanager_template_file.nginx_config,
    filemanager_env_file.production,
  ]
}

resource "filemanager_copy" "deploy_scripts" {
  source               = filemanager_directory.app_scripts.path
  destination          = "${filemanager_directory.deploy.path}/scripts"
  recursive            = true
  preserve_permissions = true

  depends_on = [
    filemanager_template_file.start_script,
    filemanager_template_file.stop_script,
  ]
}

resource "filemanager_copy" "deploy_manifest" {
  source      = filemanager_json_file.manifest.path
  destination = "${filemanager_directory.deploy.path}/manifest.json"

  depends_on = [filemanager_json_file.manifest]
}

# =============================================================================
# PHASE 11: CHECKSUM MANIFEST
# =============================================================================

resource "filemanager_json_file" "checksums" {
  path = "${filemanager_directory.app_root.path}/checksums.json"

  content = jsonencode({
    generated_at = timestamp()

    config_files = {
      settings = {
        path     = filemanager_json_file.app_settings.path
        checksum = data.filemanager_checksum.settings_checksum.checksum
        size     = data.filemanager_checksum.settings_checksum.size
      }
      database = {
        path     = filemanager_yaml_file.database.path
        checksum = data.filemanager_checksum.database_checksum.checksum
        size     = data.filemanager_checksum.database_checksum.size
      }
    }

    archives = {
      config = {
        path = filemanager_archive.config_archive.path
        size = filemanager_archive.config_archive.size
      }
      scripts = {
        path = filemanager_archive.scripts_archive.path
        size = filemanager_archive.scripts_archive.size
      }
      full = {
        path = filemanager_archive.full_app_archive.path
        size = filemanager_archive.full_app_archive.size
      }
    }

    deployment = {
      configs_copied = filemanager_copy.deploy_configs.files_copied
      scripts_copied = filemanager_copy.deploy_scripts.files_copied
    }
  })

  sort_keys          = true
  indent             = 2
  create_parent_dirs = true

  depends_on = [
    data.filemanager_checksum.settings_checksum,
    data.filemanager_checksum.database_checksum,
    filemanager_archive.config_archive,
    filemanager_archive.scripts_archive,
    filemanager_archive.full_app_archive,
    filemanager_copy.deploy_configs,
    filemanager_copy.deploy_scripts,
  ]
}
