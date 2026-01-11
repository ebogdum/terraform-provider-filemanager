# INTEGRATION TESTS - OUTPUTS

# =============================================================================
# PHASE 1: DIRECTORY STRUCTURE
# =============================================================================

output "directories" {
  value = {
    app_root = {
      path = filemanager_directory.app_root.path
    }
    config = {
      path = filemanager_directory.app_config.path
    }
    data = {
      path = filemanager_directory.app_data.path
    }
    logs = {
      path = filemanager_directory.app_logs.path
    }
    scripts = {
      path = filemanager_directory.app_scripts.path
    }
    backups = {
      path = filemanager_directory.backups.path
    }
    deploy = {
      path = filemanager_directory.deploy.path
    }
  }
  description = "Created directory structure"
}

# =============================================================================
# PHASE 2-3: CONFIGURATION FILES
# =============================================================================

output "config_files" {
  value = {
    json_settings = {
      path = filemanager_json_file.app_settings.path
      md5  = filemanager_json_file.app_settings.md5
    }
    yaml_database = {
      path = filemanager_yaml_file.database.path
      md5  = filemanager_yaml_file.database.md5
    }
    yaml_logging = {
      path = filemanager_yaml_file.logging.path
      md5  = filemanager_yaml_file.logging.md5
    }
    env_production = {
      path = filemanager_env_file.production.path
      md5  = filemanager_env_file.production.md5
    }
    env_development = {
      path = filemanager_env_file.development.path
      md5  = filemanager_env_file.development.md5
    }
  }
  description = "Created configuration files"
}

# =============================================================================
# PHASE 4: TEMPLATE FILES
# =============================================================================

output "template_files" {
  value = {
    nginx = {
      path             = filemanager_template_file.nginx_config.path
      rendered_content = filemanager_template_file.nginx_config.rendered_content
    }
    systemd = {
      path = filemanager_template_file.systemd_service.path
    }
  }
  description = "Created template files"
}

# =============================================================================
# PHASE 5: SCRIPTS
# =============================================================================

output "scripts" {
  value = {
    start = {
      path = filemanager_template_file.start_script.path
    }
    stop = {
      path = filemanager_template_file.stop_script.path
    }
    backup = {
      path = filemanager_template_file.backup_script.path
    }
  }
  description = "Created script files"
}

# =============================================================================
# PHASE 6-7: DATA AND MANIFEST FILES
# =============================================================================

output "data_files" {
  value = {
    sample_data = {
      path = filemanager_json_file.sample_data.path
    }
    initial_log = {
      path = filemanager_file.initial_log.path
    }
    manifest = {
      path = filemanager_json_file.manifest.path
      md5  = filemanager_json_file.manifest.md5
    }
    registry = {
      path = filemanager_ini_file.registry.path
    }
  }
  description = "Created data and manifest files"
}

# =============================================================================
# PHASE 8: DATA SOURCE RESULTS
# =============================================================================

output "data_source_results" {
  value = {
    read_settings = {
      path    = data.filemanager_file.read_settings.path
      content = data.filemanager_file.read_settings.content
      size    = data.filemanager_file.read_settings.size
      md5     = data.filemanager_file.read_settings.md5
    }
    settings_checksum = {
      path      = data.filemanager_checksum.settings_checksum.path
      algorithm = "sha256"
      checksum  = data.filemanager_checksum.settings_checksum.checksum
      size      = data.filemanager_checksum.settings_checksum.size
    }
    database_checksum = {
      path     = data.filemanager_checksum.database_checksum.path
      checksum = data.filemanager_checksum.database_checksum.checksum
    }
    config_dir_stat = {
      path   = data.filemanager_stat.config_dir_stat.path
      exists = data.filemanager_stat.config_dir_stat.exists
      is_dir = data.filemanager_stat.config_dir_stat.is_dir
      mode   = data.filemanager_stat.config_dir_stat.mode
    }
    list_configs = {
      path       = data.filemanager_directory.list_configs.path
      file_count = data.filemanager_directory.list_configs.file_count
      total_size = data.filemanager_directory.list_configs.total_size
    }
    list_scripts = {
      path       = data.filemanager_directory.list_scripts.path
      file_count = data.filemanager_directory.list_scripts.file_count
    }
  }
  description = "Results from data sources reading created content"
}

# =============================================================================
# PHASE 9: ARCHIVES
# =============================================================================

output "archives" {
  value = {
    config_archive = {
      path = filemanager_archive.config_archive.path
      type = filemanager_archive.config_archive.type
      size = filemanager_archive.config_archive.size
    }
    scripts_archive = {
      path = filemanager_archive.scripts_archive.path
      type = filemanager_archive.scripts_archive.type
      size = filemanager_archive.scripts_archive.size
    }
    full_app_archive = {
      path = filemanager_archive.full_app_archive.path
      type = filemanager_archive.full_app_archive.type
      size = filemanager_archive.full_app_archive.size
    }
  }
  description = "Created archives"
}

# =============================================================================
# PHASE 10: DEPLOYMENT COPIES
# =============================================================================

output "deployment" {
  value = {
    deploy_configs = {
      source       = filemanager_copy.deploy_configs.source
      destination  = filemanager_copy.deploy_configs.destination
      files_copied = filemanager_copy.deploy_configs.files_copied
      bytes_copied = filemanager_copy.deploy_configs.bytes_copied
    }
    deploy_scripts = {
      source       = filemanager_copy.deploy_scripts.source
      destination  = filemanager_copy.deploy_scripts.destination
      files_copied = filemanager_copy.deploy_scripts.files_copied
    }
    deploy_manifest = {
      source      = filemanager_copy.deploy_manifest.source
      destination = filemanager_copy.deploy_manifest.destination
      md5         = filemanager_copy.deploy_manifest.md5
    }
  }
  description = "Deployment copy operations"
}

# =============================================================================
# PHASE 11: CHECKSUM MANIFEST
# =============================================================================

output "checksums_manifest" {
  value = {
    path = filemanager_json_file.checksums.path
    md5  = filemanager_json_file.checksums.md5
  }
  description = "Final checksums manifest file"
}

# =============================================================================
# DEPENDENCY CHAIN VISUALIZATION
# =============================================================================

output "dependency_chain" {
  value = {
    description = "Demonstrates resource dependency flow"

    chain = [
      "1. Directories created first (app_root -> app_config, app_data, etc.)",
      "2. Config files use directory paths (app_settings uses app_config.path)",
      "3. ENV files reference config file paths",
      "4. Templates use directory and config file paths",
      "5. Scripts reference directories, configs, and env files",
      "6. Data files reference config paths",
      "7. Manifest aggregates all resource paths",
      "8. Data sources read created content",
      "9. Archives package created directories",
      "10. Copy operations deploy created content",
      "11. Checksum manifest uses data source outputs and archive sizes"
    ]

    resource_count = {
      directories    = 7
      json_files     = 4
      yaml_files     = 2
      env_files      = 2
      template_files = 5
      ini_files      = 1
      regular_files  = 1
      archives       = 3
      copy_ops       = 3
      data_sources   = 7
    }
  }
  description = "Visualization of the dependency chain"
}

# =============================================================================
# SUMMARY
# =============================================================================

output "summary" {
  value = {
    total_resources = 35
    phases          = 11

    demonstrates = [
      "Directory outputs used in file paths",
      "Config file paths used in env files",
      "Template variables from other resource outputs",
      "Scripts referencing multiple config files",
      "Data sources reading created resources",
      "Archives packaging dynamically created content",
      "Copy operations using resource paths",
      "Manifest files aggregating all resource info",
      "Checksum computation on created files",
      "Multi-phase deployment workflow"
    ]
  }
}
