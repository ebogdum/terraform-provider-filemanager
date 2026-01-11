# Read and parse HCL file
data "filemanager_hcl" "config" {
  path = "/etc/app/config.hcl"
}

# Access parsed data
output "server_host" {
  value = data.filemanager_hcl.config.data.server.host
}

output "features" {
  value = data.filemanager_hcl.config.data.features
}

# Read HCL from remote server
data "filemanager_hcl" "remote" {
  path    = "/etc/consul.d/config.hcl"
  service = filemanager_ssh_service.server.name
}

# Read Vault configuration
data "filemanager_hcl" "vault" {
  path = "/etc/vault.d/config.hcl"
}

output "vault_storage" {
  value = data.filemanager_hcl.vault.data.storage
}

# Read Nomad configuration
data "filemanager_hcl" "nomad" {
  path = "/etc/nomad.d/config.hcl"
}

output "datacenter" {
  value = data.filemanager_hcl.nomad.data.datacenter
}
