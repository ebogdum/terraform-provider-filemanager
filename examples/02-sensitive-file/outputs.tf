# =============================================================================
# SENSITIVE FILE RESOURCE - OUTPUTS
# =============================================================================

# Note: Outputs for sensitive files hide content by default

output "basic_secrets" {
  description = "Basic secrets file paths (content hidden)"
  value = {
    basic = filemanager_sensitive_file.basic_secrets.path
    multi = filemanager_sensitive_file.multi_secrets.path
    json  = filemanager_sensitive_file.json_secrets.path
  }
  sensitive = true
}

output "permission_files" {
  description = "Sensitive files with various permissions"
  value = {
    owner_only     = filemanager_sensitive_file.owner_only.path
    readonly       = filemanager_sensitive_file.readonly.path
    restricted_dir = filemanager_sensitive_file.restricted_dir.path
  }
  sensitive = true
}

output "key_files" {
  description = "Key and certificate files"
  value = {
    private_key = filemanager_sensitive_file.private_key.path
    ssh_key     = filemanager_sensitive_file.ssh_key.path
    certificate = filemanager_sensitive_file.certificate.path
  }
  sensitive = true
}

output "auth_files" {
  description = "Authentication files"
  value = {
    passwords = filemanager_sensitive_file.passwords.path
    tokens    = filemanager_sensitive_file.tokens.path
  }
  sensitive = true
}

output "container_secrets" {
  description = "Container orchestration secrets"
  value = {
    k8s_secret    = filemanager_sensitive_file.k8s_secret.path
    docker_config = filemanager_sensitive_file.docker_config.path
  }
  sensitive = true
}

output "database_credentials" {
  description = "Database credential files"
  value = {
    pgpass     = filemanager_sensitive_file.pgpass.path
    mysql_cnf  = filemanager_sensitive_file.mysql_cnf.path
    redis_auth = filemanager_sensitive_file.redis_auth.path
  }
  sensitive = true
}

output "cloud_credentials" {
  description = "Cloud provider credential files"
  value = {
    aws   = filemanager_sensitive_file.aws_credentials.path
    gcp   = filemanager_sensitive_file.gcp_sa.path
    azure = filemanager_sensitive_file.azure_creds.path
  }
  sensitive = true
}

output "app_secrets" {
  description = "Application secret files"
  value = {
    dotenv          = filemanager_sensitive_file.dotenv_secrets.path
    encryption_keys = filemanager_sensitive_file.encryption_keys.path
    oauth           = filemanager_sensitive_file.oauth_secrets.path
  }
  sensitive = true
}

output "summary" {
  description = "Test summary"
  value = {
    total_sensitive_files = 25
    categories = [
      "basic_secrets",
      "permissions",
      "keys_and_certs",
      "auth_files",
      "container_secrets",
      "database_credentials",
      "cloud_credentials",
      "app_secrets",
      "edge_cases"
    ]
  }
}
