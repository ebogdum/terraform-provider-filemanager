# Create a simple text file
resource "filemanager_file" "example" {
  path    = "/etc/app/config.txt"
  content = "Hello, World!"

  file_permission    = "0644"
  create_parent_dirs = true
}

# Create a configuration file from template
resource "filemanager_file" "config" {
  path    = "/etc/app/app.conf"
  content = <<-EOF
    # Application Configuration
    server_host = ${var.server_host}
    server_port = ${var.server_port}
    debug_mode  = ${var.debug_enabled}
  EOF

  file_permission    = "0600"
  create_parent_dirs = true

  # Enable integrity verification
  atomic_write    = true
  verify_checksum = true
}

# Create a file on remote server via SSH
resource "filemanager_file" "remote" {
  path    = "/opt/app/config.txt"
  content = "Remote configuration"
  service = filemanager_ssh_service.server.name

  file_permission    = "0644"
  create_parent_dirs = true
}
