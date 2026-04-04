# Configure SSH service with key file
resource "filemanager_ssh_service" "server" {
  host             = "server.example.com"
  port             = 22
  username         = "deploy"
  private_key_file = "~/.ssh/deploy_key"
}

# SSH service with inline key from variable
resource "filemanager_ssh_service" "app_server" {
  host        = var.app_server_host
  port        = var.ssh_port
  username    = var.ssh_user
  private_key = var.ssh_private_key
}

# SSH service with password authentication
resource "filemanager_ssh_service" "legacy" {
  host     = "legacy.example.com"
  username = "admin"
  password = var.legacy_password
}

# SSH service with encrypted key
resource "filemanager_ssh_service" "encrypted" {
  host             = "secure.example.com"
  username         = "deploy"
  private_key_file = "~/.ssh/id_ed25519"
  passphrase       = var.key_passphrase
}
