# Calculate checksums for a file
data "filemanager_checksum" "config" {
  path = "/etc/app/config.json"
}

output "checksums" {
  value = {
    md5    = data.filemanager_checksum.config.md5
    sha1   = data.filemanager_checksum.config.sha1
    sha256 = data.filemanager_checksum.config.sha256
    sha512 = data.filemanager_checksum.config.sha512
  }
}

# Calculate checksum for remote file
data "filemanager_checksum" "remote" {
  path    = "/etc/app/config.json"
  service = filemanager_ssh_service.server.name
}

# Use checksum for verification
resource "null_resource" "verify" {
  triggers = {
    config_hash = data.filemanager_checksum.config.sha256
  }
}
