# Get file statistics
data "filemanager_stat" "config" {
  path = "/etc/app/config.json"
}

output "file_info" {
  value = {
    exists       = data.filemanager_stat.config.exists
    is_file      = data.filemanager_stat.config.is_file
    is_directory = data.filemanager_stat.config.is_directory
    is_symlink   = data.filemanager_stat.config.is_symlink
    size         = data.filemanager_stat.config.size
    permissions  = data.filemanager_stat.config.permission
    modified     = data.filemanager_stat.config.modified_time
  }
}

# Check remote file stats
data "filemanager_stat" "remote" {
  path    = "/var/log/app.log"
  service = filemanager_ssh_service.server.name
}

# Use for conditional logic
locals {
  config_exists = data.filemanager_stat.config.exists
  config_size   = data.filemanager_stat.config.size
}
