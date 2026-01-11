# Create an environment file
resource "filemanager_env_file" "app" {
  path = "/etc/app/.env"
  data = {
    APP_NAME        = "MyApplication"
    APP_ENV         = var.environment
    APP_DEBUG       = var.debug_enabled ? "true" : "false"
    APP_URL         = var.app_url
    DB_HOST         = var.db_host
    DB_PORT         = "5432"
    DB_DATABASE     = var.db_name
    DB_USERNAME     = var.db_user
    DB_PASSWORD     = var.db_password
    REDIS_HOST      = var.redis_host
    REDIS_PORT      = "6379"
    CACHE_DRIVER    = "redis"
    SESSION_DRIVER  = "redis"
    LOG_LEVEL       = "info"
  }

  file_permission    = "0600"
  create_parent_dirs = true
}

# Docker environment file
resource "filemanager_env_file" "docker" {
  path = "${var.project_path}/.env"
  data = {
    COMPOSE_PROJECT_NAME = var.project_name
    DOCKER_REGISTRY      = var.registry_url
    IMAGE_TAG            = var.image_tag
  }

  create_parent_dirs = true
}

# Merge with existing env file
resource "filemanager_env_file" "merged" {
  path = "/etc/app/.env.local"
  data = {
    NEW_VAR = "new_value"
  }

  merge = true
}
