# =============================================================================
# VERIFICATION CHECKS - ENV file resource validation
# =============================================================================

check "verify_simple_env" {
  data "filemanager_file" "simple_check" {
    path = filemanager_env_file.simple.path
  }

  assert {
    condition     = data.filemanager_file.simple_check.size > 0
    error_message = "Simple env file is empty"
  }

  assert {
    condition     = strcontains(data.filemanager_file.simple_check.content, "APP_NAME=")
    error_message = "Simple env should contain 'APP_NAME='"
  }
}

check "verify_sorted_env" {
  data "filemanager_file" "sorted_check" {
    path = filemanager_env_file.sorted.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.sorted_check.content, "ALPHA=")
    error_message = "Sorted env should contain 'ALPHA='"
  }

  assert {
    condition     = strcontains(data.filemanager_file.sorted_check.content, "ZEBRA=")
    error_message = "Sorted env should contain 'ZEBRA='"
  }
}

check "verify_many_vars_env" {
  data "filemanager_file" "many_check" {
    path = filemanager_env_file.many_vars.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.many_check.content, "APP_DEBUG=")
    error_message = "Full env should contain 'APP_DEBUG='"
  }

  assert {
    condition     = strcontains(data.filemanager_file.many_check.content, "CACHE_DRIVER=")
    error_message = "Full env should contain 'CACHE_DRIVER='"
  }
}

check "verify_development_env" {
  data "filemanager_file" "dev_check" {
    path = filemanager_env_file.development.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.dev_check.content, "NODE_ENV=development")
    error_message = "Dev env should contain 'NODE_ENV=development'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.dev_check.content, "DEBUG=true")
    error_message = "Dev env should have 'DEBUG=true'"
  }
}

check "verify_production_env" {
  data "filemanager_file" "prod_check" {
    path = filemanager_env_file.production.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.prod_check.content, "NODE_ENV=production")
    error_message = "Prod env should contain 'NODE_ENV=production'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.prod_check.content, "DEBUG=false")
    error_message = "Prod env should have 'DEBUG=false'"
  }
}

check "verify_docker_env" {
  data "filemanager_file" "docker_check" {
    path = filemanager_env_file.docker.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_check.content, "COMPOSE_PROJECT_NAME=")
    error_message = "Docker env should contain 'COMPOSE_PROJECT_NAME='"
  }
}

check "verify_postgres_env" {
  data "filemanager_file" "db_check" {
    path = filemanager_env_file.postgres.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.db_check.content, "DB_HOST=")
    error_message = "Postgres env should contain 'DB_HOST='"
  }
}

# =============================================================================
# FORMAT VALIDATION CHECKS (filemanager_validate)
# =============================================================================

check "validate_simple_env_format" {
  data "filemanager_validate" "simple_env" {
    path   = filemanager_env_file.simple.path
    format = "env"
  }

  assert {
    condition     = data.filemanager_validate.simple_env.is_valid == true
    error_message = "Simple ENV file should be valid"
  }
}

check "validate_development_env_format" {
  data "filemanager_validate" "dev_env" {
    path = filemanager_env_file.development.path
  }

  assert {
    condition     = data.filemanager_validate.dev_env.is_valid == true
    error_message = "Development ENV file should be valid"
  }
}

check "validate_production_env_format" {
  data "filemanager_validate" "prod_env" {
    path = filemanager_env_file.production.path
  }

  assert {
    condition     = data.filemanager_validate.prod_env.is_valid == true
    error_message = "Production ENV file should be valid"
  }
}

# =============================================================================
# FILE COMPARISON CHECKS (filemanager_compare)
# =============================================================================

check "compare_env_same_file" {
  data "filemanager_compare" "env_same" {
    source = filemanager_env_file.simple.path
    target = filemanager_env_file.simple.path
  }

  assert {
    condition     = data.filemanager_compare.env_same.identical == true
    error_message = "Same ENV file compared to itself should be identical"
  }
}

check "compare_dev_vs_prod_env" {
  data "filemanager_compare" "env_compare" {
    source = filemanager_env_file.development.path
    target = filemanager_env_file.production.path
  }

  # Dev and prod should differ
  assert {
    condition     = data.filemanager_compare.env_compare.source_exists == true
    error_message = "Dev ENV file should exist"
  }

  assert {
    condition     = data.filemanager_compare.env_compare.target_exists == true
    error_message = "Prod ENV file should exist"
  }

  # They should have same size structure but different values
  assert {
    condition     = data.filemanager_compare.env_compare.identical == false
    error_message = "Dev and Prod ENV files should differ"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_env_time_check" {
  data "filemanager_stat" "env_time_check" {
    path            = filemanager_env_file.simple.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.env_time_check.is_modified_within == true
    error_message = "Newly created ENV file should be modified within last hour"
  }

  assert {
    condition     = data.filemanager_stat.env_time_check.owner_name != null
    error_message = "ENV file owner name should be resolved"
  }
}
