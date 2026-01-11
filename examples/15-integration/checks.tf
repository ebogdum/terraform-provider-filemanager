# =============================================================================
# VERIFICATION CHECKS - Integration test validation
# =============================================================================

# =============================================================================
# PHASE 1: DIRECTORY STRUCTURE
# =============================================================================

check "verify_app_root" {
  data "filemanager_stat" "app_root_check" {
    path = filemanager_directory.app_root.path
  }

  assert {
    condition     = data.filemanager_stat.app_root_check.exists == true
    error_message = "App root directory should exist"
  }

  assert {
    condition     = data.filemanager_stat.app_root_check.is_dir == true
    error_message = "App root should be a directory"
  }
}

check "verify_app_config_dir" {
  data "filemanager_stat" "app_config_check" {
    path = filemanager_directory.app_config.path
  }

  assert {
    condition     = data.filemanager_stat.app_config_check.is_dir == true
    error_message = "App config should be a directory"
  }
}

check "verify_app_data_dir" {
  data "filemanager_stat" "app_data_check" {
    path = filemanager_directory.app_data.path
  }

  assert {
    condition     = data.filemanager_stat.app_data_check.is_dir == true
    error_message = "App data should be a directory"
  }
}

check "verify_app_logs_dir" {
  data "filemanager_stat" "app_logs_check" {
    path = filemanager_directory.app_logs.path
  }

  assert {
    condition     = data.filemanager_stat.app_logs_check.is_dir == true
    error_message = "App logs should be a directory"
  }
}

check "verify_app_scripts_dir" {
  data "filemanager_stat" "app_scripts_check" {
    path = filemanager_directory.app_scripts.path
  }

  assert {
    condition     = data.filemanager_stat.app_scripts_check.is_dir == true
    error_message = "App scripts should be a directory"
  }
}

# =============================================================================
# PHASE 2: CORE CONFIG FILES
# =============================================================================

check "verify_app_settings" {
  data "filemanager_file" "settings_check" {
    path = filemanager_json_file.app_settings.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.settings_check.content, "MyApplication")
    error_message = "Settings should contain app name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.settings_check.content, "production")
    error_message = "Settings should contain environment"
  }
}

check "verify_database_yaml" {
  data "filemanager_file" "database_check" {
    path = filemanager_yaml_file.database.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.database_check.content, "postgres")
    error_message = "Database config should contain postgres"
  }

  assert {
    condition     = strcontains(data.filemanager_file.database_check.content, "redis")
    error_message = "Database config should contain redis"
  }
}

check "verify_logging_yaml" {
  data "filemanager_file" "logging_check" {
    path = filemanager_yaml_file.logging.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.logging_check.content, "INFO")
    error_message = "Logging config should contain INFO level"
  }
}

# =============================================================================
# PHASE 3: ENV FILES
# =============================================================================

check "verify_production_env" {
  data "filemanager_file" "prod_env_check" {
    path = filemanager_env_file.production.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.prod_env_check.content, "APP_ENV=production")
    error_message = "Production env should have APP_ENV=production"
  }

  assert {
    condition     = strcontains(data.filemanager_file.prod_env_check.content, "APP_DEBUG=false")
    error_message = "Production env should have APP_DEBUG=false"
  }
}

check "verify_development_env" {
  data "filemanager_file" "dev_env_check" {
    path = filemanager_env_file.development.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.dev_env_check.content, "APP_ENV=development")
    error_message = "Development env should have APP_ENV=development"
  }

  assert {
    condition     = strcontains(data.filemanager_file.dev_env_check.content, "APP_DEBUG=true")
    error_message = "Development env should have APP_DEBUG=true"
  }
}

# =============================================================================
# PHASE 4: TEMPLATE FILES
# =============================================================================

check "verify_nginx_config" {
  data "filemanager_file" "nginx_check" {
    path = filemanager_template_file.nginx_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.nginx_check.content, "upstream app_backend")
    error_message = "Nginx config should contain upstream"
  }

  assert {
    condition     = strcontains(data.filemanager_file.nginx_check.content, "proxy_pass")
    error_message = "Nginx config should contain proxy_pass"
  }
}

check "verify_systemd_service" {
  data "filemanager_file" "systemd_check" {
    path = filemanager_template_file.systemd_service.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.systemd_check.content, "[Unit]")
    error_message = "Systemd service should contain [Unit]"
  }

  assert {
    condition     = strcontains(data.filemanager_file.systemd_check.content, "[Service]")
    error_message = "Systemd service should contain [Service]"
  }
}

# =============================================================================
# PHASE 5: SCRIPTS
# =============================================================================

check "verify_start_script" {
  data "filemanager_stat" "start_script_check" {
    path = filemanager_template_file.start_script.path
  }

  assert {
    condition     = data.filemanager_stat.start_script_check.exists == true
    error_message = "Start script should exist"
  }

  assert {
    condition     = data.filemanager_stat.start_script_check.mode == "0755"
    error_message = "Start script should be executable (0755)"
  }
}

check "verify_stop_script" {
  data "filemanager_stat" "stop_script_check" {
    path = filemanager_template_file.stop_script.path
  }

  assert {
    condition     = data.filemanager_stat.stop_script_check.mode == "0755"
    error_message = "Stop script should be executable (0755)"
  }
}

check "verify_backup_script" {
  data "filemanager_stat" "backup_script_check" {
    path = filemanager_template_file.backup_script.path
  }

  assert {
    condition     = data.filemanager_stat.backup_script_check.mode == "0755"
    error_message = "Backup script should be executable (0755)"
  }
}

# =============================================================================
# PHASE 6: DATA FILES
# =============================================================================

check "verify_sample_data" {
  data "filemanager_file" "sample_data_check" {
    path = filemanager_json_file.sample_data.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.sample_data_check.content, "records")
    error_message = "Sample data should contain records"
  }
}

check "verify_initial_log" {
  data "filemanager_file" "initial_log_check" {
    path = filemanager_file.initial_log.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.initial_log_check.content, "[INFO]")
    error_message = "Initial log should contain [INFO] entries"
  }
}

# =============================================================================
# PHASE 7: MANIFEST FILES
# =============================================================================

check "verify_manifest" {
  data "filemanager_file" "manifest_check" {
    path = filemanager_json_file.manifest.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.manifest_check.content, "application")
    error_message = "Manifest should contain application section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.manifest_check.content, "paths")
    error_message = "Manifest should contain paths section"
  }
}

check "verify_registry_ini" {
  data "filemanager_file" "registry_check" {
    path = filemanager_ini_file.registry.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.registry_check.content, "[application]")
    error_message = "Registry should contain [application] section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.registry_check.content, "[directories]")
    error_message = "Registry should contain [directories] section"
  }
}

# =============================================================================
# PHASE 9: ARCHIVES
# =============================================================================

check "verify_config_archive" {
  data "filemanager_stat" "config_archive_check" {
    path = filemanager_archive.config_archive.path
  }

  assert {
    condition     = data.filemanager_stat.config_archive_check.exists == true
    error_message = "Config archive should exist"
  }

  assert {
    condition     = data.filemanager_stat.config_archive_check.size > 0
    error_message = "Config archive should not be empty"
  }
}

check "verify_scripts_archive" {
  data "filemanager_stat" "scripts_archive_check" {
    path = filemanager_archive.scripts_archive.path
  }

  assert {
    condition     = data.filemanager_stat.scripts_archive_check.exists == true
    error_message = "Scripts archive should exist"
  }
}

check "verify_full_app_archive" {
  data "filemanager_stat" "full_archive_check" {
    path = filemanager_archive.full_app_archive.path
  }

  assert {
    condition     = data.filemanager_stat.full_archive_check.exists == true
    error_message = "Full app archive should exist"
  }
}

# =============================================================================
# PHASE 10: COPY OPERATIONS
# =============================================================================

check "verify_deploy_configs" {
  data "filemanager_stat" "deploy_configs_check" {
    path = filemanager_copy.deploy_configs.destination
  }

  assert {
    condition     = data.filemanager_stat.deploy_configs_check.exists == true
    error_message = "Deployed configs directory should exist"
  }

  assert {
    condition     = data.filemanager_stat.deploy_configs_check.is_dir == true
    error_message = "Deployed configs should be a directory"
  }
}

check "verify_deploy_scripts" {
  data "filemanager_stat" "deploy_scripts_check" {
    path = filemanager_copy.deploy_scripts.destination
  }

  assert {
    condition     = data.filemanager_stat.deploy_scripts_check.exists == true
    error_message = "Deployed scripts directory should exist"
  }
}

check "verify_deploy_manifest" {
  data "filemanager_stat" "deploy_manifest_check" {
    path = filemanager_copy.deploy_manifest.destination
  }

  assert {
    condition     = data.filemanager_stat.deploy_manifest_check.exists == true
    error_message = "Deployed manifest should exist"
  }
}

# =============================================================================
# PHASE 11: CHECKSUMS
# =============================================================================

check "verify_checksums_file" {
  data "filemanager_file" "checksums_check" {
    path = filemanager_json_file.checksums.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.checksums_check.content, "config_files")
    error_message = "Checksums file should contain config_files section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.checksums_check.content, "archives")
    error_message = "Checksums file should contain archives section"
  }
}
