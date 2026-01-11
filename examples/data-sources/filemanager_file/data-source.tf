# Read a local file
data "filemanager_file" "config" {
  path = "/etc/app/config.txt"
}

output "config_content" {
  value = data.filemanager_file.config.content
}

output "config_sha256" {
  value = data.filemanager_file.config.sha256
}

# Read a file from remote server
data "filemanager_file" "remote_config" {
  path    = "/etc/app/config.txt"
  service = filemanager_ssh_service.server.name
}

# Read base64-encoded content (useful for binary files)
data "filemanager_file" "certificate" {
  path = "/etc/ssl/certs/server.crt"
}

output "cert_base64" {
  value = data.filemanager_file.certificate.content_base64
}
