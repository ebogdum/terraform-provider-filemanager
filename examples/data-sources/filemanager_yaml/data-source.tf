# Read and parse YAML file
data "filemanager_yaml" "config" {
  path = "/etc/app/config.yaml"
}

# Access parsed data
output "app_name" {
  value = data.filemanager_yaml.config.data.app.name
}

output "database_config" {
  value = data.filemanager_yaml.config.data.database
}

# Read YAML from remote server
data "filemanager_yaml" "remote" {
  path    = "/etc/app/config.yaml"
  service = filemanager_ssh_service.server.name
}

# Read Kubernetes manifest
data "filemanager_yaml" "deployment" {
  path = "${path.module}/manifests/deployment.yaml"
}

output "replicas" {
  value = data.filemanager_yaml.deployment.data.spec.replicas
}

# Access multi-document YAML
data "filemanager_yaml" "multi_doc" {
  path = "/etc/app/multi.yaml"
}
