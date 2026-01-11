# Read and parse environment file
data "filemanager_env" "app" {
  path = "/etc/app/.env"
}

# Access parsed data
output "app_name" {
  value = data.filemanager_env.app.data.APP_NAME
}

output "database_host" {
  value = data.filemanager_env.app.data.DB_HOST
}

# Read env from remote server
data "filemanager_env" "remote" {
  path    = "/opt/app/.env"
  service = filemanager_ssh_service.server.name
}

# Read Docker .env file
data "filemanager_env" "docker" {
  path = "${var.project_path}/.env"
}

output "docker_registry" {
  value = data.filemanager_env.docker.data.DOCKER_REGISTRY
}

# Use environment variables in resources
resource "filemanager_file" "generated_config" {
  path    = "/etc/app/generated.conf"
  content = <<-EOF
    # Generated from .env
    database_host = ${data.filemanager_env.app.data.DB_HOST}
    database_port = ${data.filemanager_env.app.data.DB_PORT}
    database_name = ${data.filemanager_env.app.data.DB_DATABASE}
  EOF

  create_parent_dirs = true
}
