# =============================================================================
# VERIFICATION CHECKS - INI file resource validation
# =============================================================================

check "verify_simple_ini" {
  data "filemanager_file" "simple_check" {
    path = filemanager_ini_file.simple.path
  }

  assert {
    condition     = data.filemanager_file.simple_check.size > 0
    error_message = "Simple INI file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "[General]")
    error_message = "Simple INI should contain '[General]' section"
  }
}

check "verify_multi_section_ini" {
  data "filemanager_file" "multi_check" {
    path = filemanager_ini_file.multi_section.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.multi_check.content, "[Database]")
    error_message = "Multi-section INI should contain '[Database]'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.multi_check.content, "[Cache]")
    error_message = "Multi-section INI should contain '[Cache]'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.multi_check.content, "[Logging]")
    error_message = "Multi-section INI should contain '[Logging]'"
  }
}

check "verify_php_dev" {
  data "filemanager_file" "php_dev_check" {
    path = filemanager_ini_file.php_dev.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.php_dev_check.content, "[PHP]")
    error_message = "PHP config should contain '[PHP]' section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.php_dev_check.content, "memory_limit")
    error_message = "PHP config should contain 'memory_limit'"
  }
}

check "verify_php_prod" {
  data "filemanager_file" "php_prod_check" {
    path = filemanager_ini_file.php_prod.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.php_prod_check.content, "display_errors")
    error_message = "PHP prod config should contain 'display_errors'"
  }
}

check "verify_mysql" {
  data "filemanager_file" "mysql_check" {
    path = filemanager_ini_file.mysql.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.mysql_check.content, "[mysqld]")
    error_message = "MySQL config should contain '[mysqld]' section"
  }
}

check "verify_git_config" {
  data "filemanager_file" "git_check" {
    path = filemanager_ini_file.gitconfig.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.git_check.content, "[user]")
    error_message = "Git config should contain '[user]' section"
  }
}

check "verify_systemd_service" {
  data "filemanager_file" "systemd_check" {
    path = filemanager_ini_file.systemd_service.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.systemd_check.content, "[Unit]")
    error_message = "Systemd service should contain '[Unit]' section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.systemd_check.content, "[Service]")
    error_message = "Systemd service should contain '[Service]' section"
  }
}

# =============================================================================
# FORMAT VALIDATION CHECKS (filemanager_validate)
# =============================================================================

check "validate_simple_ini_format" {
  data "filemanager_validate" "simple_ini" {
    path   = filemanager_ini_file.simple.path
    format = "ini"
  }

  assert {
    condition     = data.filemanager_validate.simple_ini.is_valid == true
    error_message = "Simple INI should be valid"
  }
}

check "validate_multi_section_ini_format" {
  data "filemanager_validate" "multi_ini" {
    path = filemanager_ini_file.multi_section.path
  }

  assert {
    condition     = data.filemanager_validate.multi_ini.is_valid == true
    error_message = "Multi-section INI should be valid"
  }
}

check "validate_php_ini_format" {
  data "filemanager_validate" "php_ini" {
    path = filemanager_ini_file.php_dev.path
  }

  assert {
    condition     = data.filemanager_validate.php_ini.is_valid == true
    error_message = "PHP INI should be valid"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# =============================================================================

check "compare_ini_same_file" {
  data "filemanager_compare" "ini_same" {
    source = filemanager_ini_file.simple.path
    target = filemanager_ini_file.simple.path
  }

  assert {
    condition     = data.filemanager_compare.ini_same.identical == true
    error_message = "Same INI file compared to itself should be identical"
  }
}

check "compare_php_dev_vs_prod" {
  data "filemanager_compare" "php_compare" {
    source = filemanager_ini_file.php_dev.path
    target = filemanager_ini_file.php_prod.path
  }

  # Dev and prod configs should differ
  assert {
    condition     = data.filemanager_compare.php_compare.source_exists == true
    error_message = "PHP dev config should exist"
  }

  assert {
    condition     = data.filemanager_compare.php_compare.target_exists == true
    error_message = "PHP prod config should exist"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_ini_time_check" {
  data "filemanager_stat" "ini_time_check" {
    path            = filemanager_ini_file.simple.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.ini_time_check.is_modified_within == true
    error_message = "Newly created INI file should be modified within last hour"
  }

  assert {
    condition     = data.filemanager_stat.ini_time_check.age != null
    error_message = "INI file age should be computed"
  }
}
