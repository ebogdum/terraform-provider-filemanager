# List files matching a pattern
data "filemanager_files" "configs" {
  path    = "/etc/app"
  pattern = "*.json"
}

output "config_files" {
  value = data.filemanager_files.configs.files
}

# List files recursively
data "filemanager_files" "all_logs" {
  path      = "/var/log"
  pattern   = "*.log"
  recursive = true
}

# List files on remote server
data "filemanager_files" "remote_configs" {
  path    = "/etc/app"
  pattern = "*.yaml"
  service = filemanager_ssh_service.server.name
}

# Use file list in for_each
data "filemanager_checksum" "config_checksums" {
  for_each = toset(data.filemanager_files.configs.files)
  path     = each.value
}
