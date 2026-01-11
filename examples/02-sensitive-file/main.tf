# =============================================================================
# SENSITIVE FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/02-sensitive-file"
}

# -----------------------------------------------------------------------------
# BASIC SENSITIVE FILES
# -----------------------------------------------------------------------------

# Case 1: Simple secrets file
resource "filemanager_sensitive_file" "basic_secrets" {
  path    = "${local.output_dir}/basic/secrets.txt"
  content = "API_KEY=super-secret-key-12345"

  create_parent_dirs = true
}

# Case 2: Multiple secrets
resource "filemanager_sensitive_file" "multi_secrets" {
  path    = "${local.output_dir}/basic/multi_secrets.txt"
  content = <<-EOF
    DATABASE_PASSWORD=hunter2
    API_KEY=sk-1234567890abcdef
    JWT_SECRET=my-super-secret-jwt-key
    ENCRYPTION_KEY=aes-256-key-here
  EOF

  create_parent_dirs = true
}

# Case 3: JSON secrets
resource "filemanager_sensitive_file" "json_secrets" {
  path = "${local.output_dir}/basic/secrets.json"
  content = jsonencode({
    database = {
      username = "admin"
      password = "super-secret-password"
    }
    api = {
      key    = "api-key-12345"
      secret = "api-secret-67890"
    }
  })

  pretty_print_json  = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PERMISSION VARIATIONS (Secure defaults)
# -----------------------------------------------------------------------------

# Case 4: Owner-only access (0600) - most secure
resource "filemanager_sensitive_file" "owner_only" {
  path            = "${local.output_dir}/permissions/owner_only.txt"
  content         = "TOP_SECRET=classified-data"
  file_permission = "0600"

  create_parent_dirs = true
}

# Case 5: Read-only for owner (0400)
resource "filemanager_sensitive_file" "readonly" {
  path            = "${local.output_dir}/permissions/readonly.txt"
  content         = "IMMUTABLE_SECRET=cannot-change"
  file_permission = "0400"

  create_parent_dirs = true
}

# Case 6: Restricted directory permissions
resource "filemanager_sensitive_file" "restricted_dir" {
  path                 = "${local.output_dir}/restricted/secrets.txt"
  content              = "RESTRICTED=in-protected-directory"
  file_permission      = "0600"
  directory_permission = "0700"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# VARIOUS SECRET FORMATS
# -----------------------------------------------------------------------------

# Case 7: Private key (PEM format)
resource "filemanager_sensitive_file" "private_key" {
  path    = "${local.output_dir}/keys/private.pem"
  content = <<-EOF
    -----BEGIN RSA PRIVATE KEY-----
    MIIEpAIBAAKCAQEA0Z3VS5JJcds3xfn/ygWyF8PbnGy0AHB7MAS
    (simulated key content - not a real key)
    -----END RSA PRIVATE KEY-----
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 8: SSH key
resource "filemanager_sensitive_file" "ssh_key" {
  path    = "${local.output_dir}/keys/id_rsa"
  content = <<-EOF
    -----BEGIN OPENSSH PRIVATE KEY-----
    b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAAB
    (simulated key content - not a real key)
    -----END OPENSSH PRIVATE KEY-----
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 9: Certificate file
resource "filemanager_sensitive_file" "certificate" {
  path    = "${local.output_dir}/keys/cert.pem"
  content = <<-EOF
    -----BEGIN CERTIFICATE-----
    MIIDXTCCAkWgAwIBAgIJAJC1HiIAZAiUMA0GCSqGSIb3Qa4LGxgcG
    (simulated certificate - not real)
    -----END CERTIFICATE-----
  EOF

  file_permission    = "0644"
  create_parent_dirs = true
}

# Case 10: Password file
resource "filemanager_sensitive_file" "passwords" {
  path    = "${local.output_dir}/auth/passwords.txt"
  content = <<-EOF
    root:$6$rounds=5000$saltsalt$hashhashhash
    admin:$6$rounds=5000$different$anotherhash
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 11: Token file
resource "filemanager_sensitive_file" "tokens" {
  path = "${local.output_dir}/auth/tokens.json"
  content = jsonencode({
    access_token  = "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.simulated"
    refresh_token = "dGhpcyBpcyBhIHNpbXVsYXRlZCByZWZyZXNoIHRva2Vu"
    expires_in    = 3600
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# KUBERNETES/DOCKER SECRETS
# -----------------------------------------------------------------------------

# Case 12: Kubernetes secret data
resource "filemanager_sensitive_file" "k8s_secret" {
  path    = "${local.output_dir}/k8s/secret.yaml"
  content = <<-EOF
    apiVersion: v1
    kind: Secret
    metadata:
      name: app-secrets
    type: Opaque
    data:
      username: YWRtaW4=
      password: c3VwZXItc2VjcmV0
  EOF

  create_parent_dirs = true
}

# Case 13: Docker registry credentials
resource "filemanager_sensitive_file" "docker_config" {
  path = "${local.output_dir}/docker/config.json"
  content = jsonencode({
    auths = {
      "registry.example.com" = {
        username = "user"
        password = "registry-password"
        auth     = base64encode("user:registry-password")
      }
    }
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DATABASE CREDENTIALS
# -----------------------------------------------------------------------------

# Case 14: PostgreSQL pgpass file
resource "filemanager_sensitive_file" "pgpass" {
  path    = "${local.output_dir}/database/.pgpass"
  content = <<-EOF
    localhost:5432:mydb:admin:super-secret-password
    prod.db.example.com:5432:proddb:app:production-password
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 15: MySQL option file
resource "filemanager_sensitive_file" "mysql_cnf" {
  path    = "${local.output_dir}/database/.my.cnf"
  content = <<-EOF
    [client]
    user=root
    password=mysql-root-password
    host=localhost
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 16: Redis auth file
resource "filemanager_sensitive_file" "redis_auth" {
  path               = "${local.output_dir}/database/redis.conf"
  content            = "requirepass very-secure-redis-password"
  file_permission    = "0600"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CLOUD CREDENTIALS
# -----------------------------------------------------------------------------

# Case 17: AWS credentials
resource "filemanager_sensitive_file" "aws_credentials" {
  path    = "${local.output_dir}/cloud/aws_credentials"
  content = <<-EOF
    [default]
    aws_access_key_id = AKIAIOSFODNN7EXAMPLE
    aws_secret_access_key = wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY

    [production]
    aws_access_key_id = AKIAI44QH8DHBEXAMPLE
    aws_secret_access_key = je7MtGbClwBF/2Zp9Utk/h3yCo8nvbEXAMPLEKEY
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 18: GCP service account
resource "filemanager_sensitive_file" "gcp_sa" {
  path = "${local.output_dir}/cloud/gcp_service_account.json"
  content = jsonencode({
    type           = "service_account"
    project_id     = "my-project"
    private_key_id = "key-id-12345"
    private_key    = "-----BEGIN PRIVATE KEY-----\nMIIE...\n-----END PRIVATE KEY-----\n"
    client_email   = "sa@my-project.iam.gserviceaccount.com"
    client_id      = "123456789"
    auth_uri       = "https://accounts.google.com/o/oauth2/auth"
    token_uri      = "https://oauth2.googleapis.com/token"
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 19: Azure credentials
resource "filemanager_sensitive_file" "azure_creds" {
  path = "${local.output_dir}/cloud/azure_credentials.json"
  content = jsonencode({
    clientId       = "00000000-0000-0000-0000-000000000000"
    clientSecret   = "azure-client-secret"
    subscriptionId = "11111111-1111-1111-1111-111111111111"
    tenantId       = "22222222-2222-2222-2222-222222222222"
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# APPLICATION SECRETS
# -----------------------------------------------------------------------------

# Case 20: .env file with secrets
resource "filemanager_sensitive_file" "dotenv_secrets" {
  path    = "${local.output_dir}/app/.env.production"
  content = <<-EOF
    # Database
    DATABASE_URL=postgres://user:password@host:5432/db

    # API Keys
    STRIPE_SECRET_KEY=sk_live_xxxxxxxxxxxxxxxxxxxxxxxx
    SENDGRID_API_KEY=SG.xxxxxxxxxxxxxxxxxxxxxxxx

    # Encryption
    APP_SECRET=base64:xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
    ENCRYPTION_KEY=32-byte-encryption-key-here-xxx
  EOF

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 21: Encryption keys
resource "filemanager_sensitive_file" "encryption_keys" {
  path = "${local.output_dir}/app/encryption_keys.json"
  content = jsonencode({
    current_key = "aes-256-gcm-key-current-xxxxxx"
    previous_keys = [
      "aes-256-gcm-key-old-1-xxxxxxx",
      "aes-256-gcm-key-old-2-xxxxxxx"
    ]
    key_rotation_date = "2024-01-15"
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 22: OAuth secrets
resource "filemanager_sensitive_file" "oauth_secrets" {
  path = "${local.output_dir}/app/oauth.json"
  content = jsonencode({
    google = {
      client_id     = "123456789.apps.googleusercontent.com"
      client_secret = "google-client-secret"
    }
    github = {
      client_id     = "github-client-id"
      client_secret = "github-client-secret"
    }
    facebook = {
      app_id     = "facebook-app-id"
      app_secret = "facebook-app-secret"
    }
  })

  pretty_print_json  = true
  file_permission    = "0600"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 23: Empty sensitive file (might be populated later)
resource "filemanager_sensitive_file" "empty" {
  path               = "${local.output_dir}/edge/empty_secrets.txt"
  content            = ""
  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 24: Binary-like sensitive data via base64
resource "filemanager_sensitive_file" "binary_key" {
  path           = "${local.output_dir}/edge/binary.key"
  content_base64 = base64encode("binary-key-data-here")

  file_permission    = "0600"
  create_parent_dirs = true
}

# Case 25: Very long secret
resource "filemanager_sensitive_file" "long_secret" {
  path    = "${local.output_dir}/edge/long_secret.txt"
  content = join("", [for i in range(1000) : "x"])

  file_permission    = "0600"
  create_parent_dirs = true
}
