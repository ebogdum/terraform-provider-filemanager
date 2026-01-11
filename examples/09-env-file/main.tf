# =============================================================================
# ENV FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/09-env-file"
}

# -----------------------------------------------------------------------------
# BASIC ENV FILES
# -----------------------------------------------------------------------------

# Case 1: Simple .env
resource "filemanager_env_file" "simple" {
  path = "${local.output_dir}/basic/.env"
  variables = {
    APP_NAME = "MyApp"
    APP_ENV  = "development"
  }
  create_parent_dirs = true
}

# Case 2: With sort_keys
resource "filemanager_env_file" "sorted" {
  path = "${local.output_dir}/basic/.env.sorted"
  variables = {
    ZEBRA  = "last"
    ALPHA  = "first"
    MIDDLE = "mid"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 3: Many variables
resource "filemanager_env_file" "many_vars" {
  path = "${local.output_dir}/basic/.env.full"
  variables = {
    APP_NAME         = "MyApplication"
    APP_ENV          = "development"
    APP_DEBUG        = "true"
    APP_URL          = "http://localhost"
    APP_PORT         = "8080"
    APP_KEY          = "base64:supersecretkey=="
    LOG_LEVEL        = "debug"
    LOG_CHANNEL      = "stack"
    CACHE_DRIVER     = "redis"
    SESSION_DRIVER   = "redis"
    QUEUE_CONNECTION = "redis"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# ENVIRONMENT-SPECIFIC FILES
# -----------------------------------------------------------------------------

# Case 4: Development environment
resource "filemanager_env_file" "development" {
  path = "${local.output_dir}/environments/.env.development"
  variables = {
    NODE_ENV     = "development"
    DEBUG        = "true"
    LOG_LEVEL    = "debug"
    API_URL      = "http://localhost:3000"
    DATABASE_URL = "postgres://dev:dev@localhost:5432/dev_db"
    REDIS_URL    = "redis://localhost:6379"
    HOT_RELOAD   = "true"
    SOURCE_MAPS  = "true"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 5: Production environment
resource "filemanager_env_file" "production" {
  path = "${local.output_dir}/environments/.env.production"
  variables = {
    NODE_ENV     = "production"
    DEBUG        = "false"
    LOG_LEVEL    = "error"
    API_URL      = "https://api.example.com"
    DATABASE_URL = "postgres://prod:secret@db.example.com:5432/prod_db"
    REDIS_URL    = "redis://cache.example.com:6379"
    HOT_RELOAD   = "false"
    SOURCE_MAPS  = "false"
    SENTRY_DSN   = "https://xxx@sentry.io/123"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 6: Testing environment
resource "filemanager_env_file" "testing" {
  path = "${local.output_dir}/environments/.env.test"
  variables = {
    NODE_ENV     = "test"
    DEBUG        = "false"
    LOG_LEVEL    = "silent"
    API_URL      = "http://localhost:3001"
    DATABASE_URL = "postgres://test:test@localhost:5432/test_db"
    REDIS_URL    = "redis://localhost:6380"
    CI           = "true"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 7: Staging environment
resource "filemanager_env_file" "staging" {
  path = "${local.output_dir}/environments/.env.staging"
  variables = {
    NODE_ENV     = "staging"
    DEBUG        = "true"
    LOG_LEVEL    = "info"
    API_URL      = "https://staging-api.example.com"
    DATABASE_URL = "postgres://staging:secret@staging-db.example.com:5432/staging_db"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DATABASE CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 8: PostgreSQL connection
resource "filemanager_env_file" "postgres" {
  path = "${local.output_dir}/database/.env.postgres"
  variables = {
    DB_CONNECTION = "pgsql"
    DB_HOST       = "localhost"
    DB_PORT       = "5432"
    DB_DATABASE   = "myapp"
    DB_USERNAME   = "postgres"
    DB_PASSWORD   = "secret"
    DB_CHARSET    = "utf8"
    DB_COLLATION  = "utf8_unicode_ci"
    DB_POOL_SIZE  = "10"
    DB_SSL        = "false"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 9: MySQL connection
resource "filemanager_env_file" "mysql" {
  path = "${local.output_dir}/database/.env.mysql"
  variables = {
    DB_CONNECTION = "mysql"
    DB_HOST       = "localhost"
    DB_PORT       = "3306"
    DB_DATABASE   = "myapp"
    DB_USERNAME   = "root"
    DB_PASSWORD   = "secret"
    DB_CHARSET    = "utf8mb4"
    DB_COLLATION  = "utf8mb4_unicode_ci"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 10: Redis configuration
resource "filemanager_env_file" "redis" {
  path = "${local.output_dir}/database/.env.redis"
  variables = {
    REDIS_HOST     = "localhost"
    REDIS_PORT     = "6379"
    REDIS_PASSWORD = ""
    REDIS_DATABASE = "0"
    REDIS_PREFIX   = "myapp_"
    REDIS_CLUSTER  = "false"
    REDIS_SENTINEL = ""
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# API AND SERVICE KEYS
# -----------------------------------------------------------------------------

# Case 11: Third-party services
resource "filemanager_env_file" "services" {
  path = "${local.output_dir}/services/.env.services"
  variables = {
    # AWS
    AWS_ACCESS_KEY_ID     = "AKIAIOSFODNN7EXAMPLE"
    AWS_SECRET_ACCESS_KEY = "wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY"
    AWS_DEFAULT_REGION    = "us-east-1"
    AWS_BUCKET            = "my-bucket"
    # Stripe
    STRIPE_KEY            = "pk_test_xxxxx"
    STRIPE_SECRET         = "sk_test_xxxxx"
    STRIPE_WEBHOOK_SECRET = "whsec_xxxxx"
    # SendGrid
    SENDGRID_API_KEY  = "SG.xxxxx"
    MAIL_FROM_ADDRESS = "noreply@example.com"
    MAIL_FROM_NAME    = "MyApp"
    # Sentry
    SENTRY_DSN = "https://xxx@o1.ingest.sentry.io/123"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 12: OAuth providers
resource "filemanager_env_file" "oauth" {
  path = "${local.output_dir}/services/.env.oauth"
  variables = {
    GOOGLE_CLIENT_ID        = "123456789.apps.googleusercontent.com"
    GOOGLE_CLIENT_SECRET    = "google_secret"
    GOOGLE_CALLBACK_URL     = "http://localhost:3000/auth/google/callback"
    GITHUB_CLIENT_ID        = "github_client_id"
    GITHUB_CLIENT_SECRET    = "github_secret"
    GITHUB_CALLBACK_URL     = "http://localhost:3000/auth/github/callback"
    FACEBOOK_APP_ID         = "facebook_app_id"
    FACEBOOK_APP_SECRET     = "facebook_secret"
    TWITTER_CONSUMER_KEY    = "twitter_key"
    TWITTER_CONSUMER_SECRET = "twitter_secret"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DOCKER CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 13: Docker env file
resource "filemanager_env_file" "docker" {
  path = "${local.output_dir}/docker/.env"
  variables = {
    COMPOSE_PROJECT_NAME     = "myproject"
    COMPOSE_FILE             = "docker-compose.yml:docker-compose.override.yml"
    DOCKER_BUILDKIT          = "1"
    COMPOSE_DOCKER_CLI_BUILD = "1"
    APP_IMAGE_TAG            = "latest"
    APP_PORT                 = "8080"
    DB_PORT                  = "5432"
    REDIS_PORT               = "6379"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# FRAMEWORK-SPECIFIC
# -----------------------------------------------------------------------------

# Case 14: Next.js environment
resource "filemanager_env_file" "nextjs" {
  path = "${local.output_dir}/frameworks/.env.nextjs"
  variables = {
    NEXT_PUBLIC_API_URL     = "https://api.example.com"
    NEXT_PUBLIC_APP_NAME    = "My Next.js App"
    NEXT_PUBLIC_GA_ID       = "G-XXXXXXXXXX"
    NEXT_TELEMETRY_DISABLED = "1"
    NEXTAUTH_URL            = "http://localhost:3000"
    NEXTAUTH_SECRET         = "supersecret"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 15: Laravel environment
resource "filemanager_env_file" "laravel" {
  path = "${local.output_dir}/frameworks/.env.laravel"
  variables = {
    APP_NAME          = "Laravel"
    APP_ENV           = "local"
    APP_KEY           = "base64:xxxxxxxxxxxxxxxxxxxxx="
    APP_DEBUG         = "true"
    APP_URL           = "http://localhost"
    LOG_CHANNEL       = "stack"
    LOG_LEVEL         = "debug"
    BROADCAST_DRIVER  = "log"
    CACHE_DRIVER      = "file"
    FILESYSTEM_DRIVER = "local"
    QUEUE_CONNECTION  = "sync"
    SESSION_DRIVER    = "file"
    SESSION_LIFETIME  = "120"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 16: Empty value
resource "filemanager_env_file" "empty_value" {
  path = "${local.output_dir}/edge/.env.empty"
  variables = {
    EMPTY_VAR = ""
    HAS_VALUE = "value"
  }
  create_parent_dirs = true
}

# Case 17: Special characters
resource "filemanager_env_file" "special" {
  path = "${local.output_dir}/edge/.env.special"
  variables = {
    URL_WITH_QUERY = "https://example.com/path?key=value&other=123"
    PATH_UNIX      = "/home/user/data"
    QUOTED_VALUE   = "value with spaces"
    EQUALS_SIGN    = "key=value=another"
  }
  create_parent_dirs = true
}

# Case 18: Long values
resource "filemanager_env_file" "long" {
  path = "${local.output_dir}/edge/.env.long"
  variables = {
    SHORT_KEY = "x"
    LONG_KEY  = join("", [for i in range(100) : "x"])
    JWT_TOKEN = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.eyJzdWIiOiIxMjM0NTY3ODkwIiwibmFtZSI6IkpvaG4gRG9lIiwiaWF0IjoxNTE2MjM5MDIyfQ.SflKxwRJSMeKKF2QT4fwpMeJf36POk6yJV_adQssw5c"
  }
  create_parent_dirs = true
}

# Case 19: Many variables
resource "filemanager_env_file" "many" {
  path = "${local.output_dir}/edge/.env.many"
  variables = {
    for i in range(50) : "VAR_${format("%02d", i)}" => "value_${i}"
  }
  sort_keys          = true
  create_parent_dirs = true
}
