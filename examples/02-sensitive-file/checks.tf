# =============================================================================
# VERIFICATION CHECKS - Using data sources to validate sensitive files
# =============================================================================

# -----------------------------------------------------------------------------
# BASIC SENSITIVE FILE CHECKS
# -----------------------------------------------------------------------------

check "verify_basic_secrets" {
  data "filemanager_stat" "basic_secrets_check" {
    path = filemanager_sensitive_file.basic_secrets.path
  }

  assert {
    condition     = data.filemanager_stat.basic_secrets_check.exists == true
    error_message = "Basic secrets file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.basic_secrets_check.size > 0
    error_message = "Basic secrets file is empty"
  }
}

check "verify_multi_secrets" {
  data "filemanager_stat" "multi_secrets_check" {
    path = filemanager_sensitive_file.multi_secrets.path
  }

  assert {
    condition     = data.filemanager_stat.multi_secrets_check.exists == true
    error_message = "Multi secrets file does not exist"
  }
}

check "verify_json_secrets_pretty_printed" {
  data "filemanager_file" "json_secrets_check" {
    path = filemanager_sensitive_file.json_secrets.path
  }

  assert {
    condition     = data.filemanager_file.json_secrets_check.size > 0
    error_message = "JSON secrets file is empty"
  }

  # Verify it's pretty-printed (contains newlines and indentation)
  assert {
    condition     = strcontains(data.filemanager_file.json_secrets_check.content, "\n  ")
    error_message = "JSON secrets file should be pretty-printed with indentation"
  }
}

# -----------------------------------------------------------------------------
# PERMISSION CHECKS
# -----------------------------------------------------------------------------

check "verify_owner_only_permissions" {
  data "filemanager_stat" "owner_only_check" {
    path = filemanager_sensitive_file.owner_only.path
  }

  assert {
    condition     = data.filemanager_stat.owner_only_check.mode == "0600"
    error_message = "Owner-only file should have mode 0600, got ${data.filemanager_stat.owner_only_check.mode}"
  }
}

check "verify_readonly_permissions" {
  data "filemanager_stat" "readonly_check" {
    path = filemanager_sensitive_file.readonly.path
  }

  assert {
    condition     = data.filemanager_stat.readonly_check.mode == "0400"
    error_message = "Readonly file should have mode 0400, got ${data.filemanager_stat.readonly_check.mode}"
  }
}

# -----------------------------------------------------------------------------
# KEY FILE CHECKS
# -----------------------------------------------------------------------------

check "verify_private_key_content" {
  data "filemanager_file" "private_key_check" {
    path = filemanager_sensitive_file.private_key.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.private_key_check.content, "BEGIN RSA PRIVATE KEY")
    error_message = "Private key file should contain PEM header"
  }
}

check "verify_private_key_permissions" {
  data "filemanager_stat" "private_key_stat" {
    path = filemanager_sensitive_file.private_key.path
  }

  assert {
    condition     = data.filemanager_stat.private_key_stat.mode == "0600"
    error_message = "Private key should have mode 0600"
  }
}

check "verify_ssh_key" {
  data "filemanager_file" "ssh_key_check" {
    path = filemanager_sensitive_file.ssh_key.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.ssh_key_check.content, "BEGIN OPENSSH PRIVATE KEY")
    error_message = "SSH key file should contain OpenSSH header"
  }
}

# -----------------------------------------------------------------------------
# CLOUD CREDENTIALS CHECKS
# -----------------------------------------------------------------------------

check "verify_aws_credentials" {
  data "filemanager_file" "aws_creds_check" {
    path = filemanager_sensitive_file.aws_credentials.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.aws_creds_check.content, "[default]")
    error_message = "AWS credentials should contain [default] section"
  }

  assert {
    condition     = strcontains(data.filemanager_file.aws_creds_check.content, "aws_access_key_id")
    error_message = "AWS credentials should contain aws_access_key_id"
  }
}

check "verify_gcp_service_account" {
  data "filemanager_file" "gcp_sa_check" {
    path = filemanager_sensitive_file.gcp_sa.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.gcp_sa_check.content, "service_account")
    error_message = "GCP service account should contain 'service_account' type"
  }

  # Verify it's pretty-printed
  assert {
    condition     = strcontains(data.filemanager_file.gcp_sa_check.content, "\n  ")
    error_message = "GCP service account JSON should be pretty-printed"
  }
}

check "verify_azure_credentials" {
  data "filemanager_file" "azure_creds_check" {
    path = filemanager_sensitive_file.azure_creds.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.azure_creds_check.content, "clientId")
    error_message = "Azure credentials should contain clientId"
  }

  # Verify it's pretty-printed
  assert {
    condition     = strcontains(data.filemanager_file.azure_creds_check.content, "\n  ")
    error_message = "Azure credentials JSON should be pretty-printed"
  }
}

# -----------------------------------------------------------------------------
# APP SECRETS CHECKS
# -----------------------------------------------------------------------------

check "verify_encryption_keys" {
  data "filemanager_file" "encryption_keys_check" {
    path = filemanager_sensitive_file.encryption_keys.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.encryption_keys_check.content, "current_key")
    error_message = "Encryption keys should contain current_key"
  }

  assert {
    condition     = strcontains(data.filemanager_file.encryption_keys_check.content, "previous_keys")
    error_message = "Encryption keys should contain previous_keys array"
  }

  # Verify it's pretty-printed
  assert {
    condition     = strcontains(data.filemanager_file.encryption_keys_check.content, "\n  ")
    error_message = "Encryption keys JSON should be pretty-printed"
  }
}

check "verify_oauth_secrets" {
  data "filemanager_file" "oauth_check" {
    path = filemanager_sensitive_file.oauth_secrets.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.oauth_check.content, "google")
    error_message = "OAuth secrets should contain google provider"
  }

  assert {
    condition     = strcontains(data.filemanager_file.oauth_check.content, "github")
    error_message = "OAuth secrets should contain github provider"
  }
}

check "verify_dotenv" {
  data "filemanager_file" "dotenv_check" {
    path = filemanager_sensitive_file.dotenv_secrets.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.dotenv_check.content, "DATABASE_URL")
    error_message = "Dotenv should contain DATABASE_URL"
  }

  assert {
    condition     = strcontains(data.filemanager_file.dotenv_check.content, "STRIPE_SECRET_KEY")
    error_message = "Dotenv should contain STRIPE_SECRET_KEY"
  }
}

# -----------------------------------------------------------------------------
# DATABASE CREDENTIALS CHECKS
# -----------------------------------------------------------------------------

check "verify_pgpass" {
  data "filemanager_stat" "pgpass_check" {
    path = filemanager_sensitive_file.pgpass.path
  }

  assert {
    condition     = data.filemanager_stat.pgpass_check.exists == true
    error_message = "pgpass file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.pgpass_check.mode == "0600"
    error_message = "pgpass should have mode 0600"
  }
}

check "verify_mysql_cnf" {
  data "filemanager_file" "mysql_cnf_check" {
    path = filemanager_sensitive_file.mysql_cnf.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.mysql_cnf_check.content, "[client]")
    error_message = "MySQL config should contain [client] section"
  }
}

# -----------------------------------------------------------------------------
# EDGE CASE CHECKS
# -----------------------------------------------------------------------------

check "verify_empty_sensitive" {
  data "filemanager_stat" "empty_check" {
    path = filemanager_sensitive_file.empty.path
  }

  assert {
    condition     = data.filemanager_stat.empty_check.exists == true
    error_message = "Empty sensitive file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.empty_check.size == 0
    error_message = "Empty sensitive file should have size 0"
  }
}

check "verify_binary_key" {
  data "filemanager_stat" "binary_key_check" {
    path = filemanager_sensitive_file.binary_key.path
  }

  assert {
    condition     = data.filemanager_stat.binary_key_check.exists == true
    error_message = "Binary key file does not exist"
  }

  assert {
    condition     = data.filemanager_stat.binary_key_check.size > 0
    error_message = "Binary key file should not be empty"
  }
}

check "verify_long_secret" {
  data "filemanager_stat" "long_secret_check" {
    path = filemanager_sensitive_file.long_secret.path
  }

  assert {
    condition     = data.filemanager_stat.long_secret_check.size == 1000
    error_message = "Long secret should be exactly 1000 bytes"
  }
}
