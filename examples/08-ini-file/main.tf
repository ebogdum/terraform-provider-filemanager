# =============================================================================
# INI FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/08-ini-file"
}

# -----------------------------------------------------------------------------
# BASIC INI FILES
# -----------------------------------------------------------------------------

# Case 1: Simple single section
resource "filemanager_ini_file" "simple" {
  path = "${local.output_dir}/basic/simple.ini"
  sections = {
    "General" = jsonencode({
      name    = "MyApp"
      version = ">= 1.0.0"
    })
  }
  create_parent_dirs = true
}

# Case 2: Multiple sections
resource "filemanager_ini_file" "multi_section" {
  path = "${local.output_dir}/basic/multi_section.ini"
  sections = {
    "Database" = jsonencode({
      host     = "localhost"
      port     = "5432"
      name     = "mydb"
      username = "admin"
    })
    "Cache" = jsonencode({
      enabled = "true"
      ttl     = "3600"
      type    = "redis"
    })
    "Logging" = jsonencode({
      level  = "info"
      format = "json"
      output = "stdout"
    })
  }
  create_parent_dirs = true
}

# Case 3: With sort_keys
resource "filemanager_ini_file" "sorted" {
  path = "${local.output_dir}/basic/sorted.ini"
  sections = {
    "Settings" = jsonencode({
      zebra  = "last"
      alpha  = "first"
      middle = "mid"
    })
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PHP.INI CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 4: PHP development config
resource "filemanager_ini_file" "php_dev" {
  path = "${local.output_dir}/php/development.ini"
  sections = {
    "PHP" = jsonencode({
      engine                 = "On"
      short_open_tag         = "Off"
      precision              = "14"
      output_buffering       = "4096"
      max_execution_time     = "30"
      max_input_time         = "60"
      memory_limit           = "256M"
      error_reporting        = "E_ALL"
      display_errors         = "On"
      display_startup_errors = "On"
      log_errors             = "On"
      error_log              = "/var/log/php/error.log"
    })
    "Date" = jsonencode({
      "date.timezone" = "UTC"
    })
    "opcache" = jsonencode({
      "opcache.enable"          = "0"
      "opcache.revalidate_freq" = "0"
    })
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 5: PHP production config
resource "filemanager_ini_file" "php_prod" {
  path = "${local.output_dir}/php/production.ini"
  sections = {
    "PHP" = jsonencode({
      engine                 = "On"
      short_open_tag         = "Off"
      precision              = "14"
      output_buffering       = "4096"
      max_execution_time     = "30"
      max_input_time         = "60"
      memory_limit           = "128M"
      error_reporting        = "E_ALL & ~E_DEPRECATED & ~E_STRICT"
      display_errors         = "Off"
      display_startup_errors = "Off"
      log_errors             = "On"
      error_log              = "/var/log/php/error.log"
    })
    "opcache" = jsonencode({
      "opcache.enable"                  = "1"
      "opcache.memory_consumption"      = "256"
      "opcache.interned_strings_buffer" = "8"
      "opcache.max_accelerated_files"   = "10000"
      "opcache.revalidate_freq"         = "60"
      "opcache.fast_shutdown"           = "1"
    })
    "Session" = jsonencode({
      "session.gc_maxlifetime" = "1440"
      "session.save_handler"   = "files"
    })
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# MYSQL/MARIADB CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 6: MySQL my.cnf
resource "filemanager_ini_file" "mysql" {
  path = "${local.output_dir}/mysql/my.cnf"
  sections = {
    "client" = jsonencode({
      port   = "3306"
      socket = "/var/run/mysqld/mysqld.sock"
    })
    "mysqld" = jsonencode({
      user                    = "mysql"
      pid-file                = "/var/run/mysqld/mysqld.pid"
      socket                  = "/var/run/mysqld/mysqld.sock"
      port                    = "3306"
      datadir                 = "/var/lib/mysql"
      max_connections         = "151"
      max_allowed_packet      = "64M"
      innodb_buffer_pool_size = "256M"
      innodb_log_file_size    = "64M"
      slow_query_log          = "1"
      long_query_time         = "2"
    })
    "mysqldump" = jsonencode({
      quick              = "1"
      max_allowed_packet = "64M"
    })
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# GIT CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 7: Git config
resource "filemanager_ini_file" "gitconfig" {
  path = "${local.output_dir}/git/gitconfig"
  sections = {
    "user" = jsonencode({
      name  = "Developer"
      email = "dev@example.com"
    })
    "core" = jsonencode({
      editor       = "vim"
      autocrlf     = "input"
      excludesfile = "~/.gitignore_global"
    })
    "alias" = jsonencode({
      co   = "checkout"
      br   = "branch"
      ci   = "commit"
      st   = "status"
      last = "log -1 HEAD"
    })
    "color" = jsonencode({
      ui = "auto"
    })
    "pull" = jsonencode({
      rebase = "true"
    })
    "push" = jsonencode({
      default = "current"
    })
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DESKTOP ENTRY FILES (.desktop)
# -----------------------------------------------------------------------------

# Case 8: Linux desktop entry
resource "filemanager_ini_file" "desktop_entry" {
  path = "${local.output_dir}/desktop/myapp.desktop"
  sections = {
    "Desktop Entry" = jsonencode({
      Version       = "1.0"
      Type          = "Application"
      Name          = "My Application"
      Comment       = "An example application"
      Exec          = "/usr/bin/myapp %U"
      Icon          = "myapp"
      Terminal      = "false"
      Categories    = "Utility;Application;"
      MimeType      = "text/plain;"
      StartupNotify = "true"
    })
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# AWS CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 9: AWS config
resource "filemanager_ini_file" "aws_config" {
  path = "${local.output_dir}/aws/config"
  sections = {
    "default" = jsonencode({
      region    = "us-east-1"
      output    = "json"
      cli_pager = ""
    })
    "profile dev" = jsonencode({
      region = "us-west-2"
      output = "json"
    })
    "profile prod" = jsonencode({
      region         = "eu-west-1"
      output         = "json"
      role_arn       = "arn:aws:iam::123456789:role/ProductionRole"
      source_profile = "default"
    })
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# SYSTEMD UNIT FILES
# -----------------------------------------------------------------------------

# Case 10: Systemd service
resource "filemanager_ini_file" "systemd_service" {
  path = "${local.output_dir}/systemd/myapp.service"
  sections = {
    "Unit" = jsonencode({
      Description   = "My Application Service"
      Documentation = "https://example.com/docs"
      After         = "network-online.target"
      Wants         = "network-online.target"
    })
    "Service" = jsonencode({
      Type             = "simple"
      User             = "myapp"
      Group            = "myapp"
      WorkingDirectory = "/opt/myapp"
      ExecStart        = "/opt/myapp/bin/myapp"
      ExecReload       = "/bin/kill -HUP $MAINPID"
      Restart          = "on-failure"
      RestartSec       = "5s"
      StandardOutput   = "journal"
      StandardError    = "journal"
    })
    "Install" = jsonencode({
      WantedBy = "multi-user.target"
    })
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 11: Empty section
resource "filemanager_ini_file" "empty_section" {
  path = "${local.output_dir}/edge/empty_section.ini"
  sections = {
    "HasValues" = jsonencode({
      key = "value"
    })
    "Empty" = jsonencode({})
  }
  create_parent_dirs = true
}

# Case 12: Special characters in values
resource "filemanager_ini_file" "special_chars" {
  path = "${local.output_dir}/edge/special_chars.ini"
  sections = {
    "Paths" = jsonencode({
      home = "/home/user"
      data = "C:\\Users\\Data"
      url  = "https://example.com/path?query=value"
    })
    "Strings" = jsonencode({
      quoted  = "value with spaces"
      unicode = "日本語"
      equals  = "key=value"
    })
  }
  create_parent_dirs = true
}

# Case 13: Many sections
resource "filemanager_ini_file" "many_sections" {
  path = "${local.output_dir}/edge/many_sections.ini"
  sections = {
    for i in range(10) : "Section${i}" => jsonencode({
      key1 = "value1_${i}"
      key2 = "value2_${i}"
    })
  }
  create_parent_dirs = true
}

# Case 14: Long values
resource "filemanager_ini_file" "long_values" {
  path = "${local.output_dir}/edge/long_values.ini"
  sections = {
    "Long" = jsonencode({
      short = "x"
      long  = join("", [for i in range(100) : "x"])
    })
  }
  create_parent_dirs = true
}
