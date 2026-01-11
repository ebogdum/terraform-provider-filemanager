# Create a file with sensitive content (masked in logs/state)
resource "filemanager_sensitive_file" "credentials" {
  path    = "/etc/app/credentials.json"
  content = jsonencode({
    api_key    = var.api_key
    api_secret = var.api_secret
  })

  file_permission    = "0600"
  create_parent_dirs = true
}

# Create SSH private key file
resource "filemanager_sensitive_file" "ssh_key" {
  path    = "/home/deploy/.ssh/id_rsa"
  content = var.ssh_private_key

  file_permission    = "0600"
  create_parent_dirs = true
}

# Create TLS certificate key
resource "filemanager_sensitive_file" "tls_key" {
  path           = "/etc/ssl/private/server.key"
  content_base64 = var.tls_key_base64

  file_permission    = "0600"
  create_parent_dirs = true
}
