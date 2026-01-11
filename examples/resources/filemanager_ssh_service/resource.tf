# Configure SSH service for remote file operations
resource "filemanager_ssh_service" "server" {
  name = "production-server"
  host = "server.example.com"
  port = 22
  user = "deploy"

  # Authentication via private key
  private_key = file("~/.ssh/deploy_key")
}

# SSH service with key from variable
resource "filemanager_ssh_service" "app_server" {
  name = "app-server"
  host = var.app_server_host
  port = var.ssh_port
  user = var.ssh_user

  private_key = var.ssh_private_key
}

# SSH service with password authentication
resource "filemanager_ssh_service" "legacy" {
  name     = "legacy-server"
  host     = "legacy.example.com"
  user     = "admin"
  password = var.legacy_password
}

# SSH service with jump host
resource "filemanager_ssh_service" "internal" {
  name = "internal-server"
  host = "10.0.0.50"
  user = "deploy"

  private_key = var.ssh_private_key

  proxy_jump = {
    host        = "bastion.example.com"
    user        = "jump"
    private_key = var.bastion_key
  }
}
