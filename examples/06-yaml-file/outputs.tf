# YAML FILE - OUTPUTS

output "basic_yaml" {
  value = {
    simple = filemanager_yaml_file.simple.path
    nested = filemanager_yaml_file.nested.path
    list   = filemanager_yaml_file.list.path
    mixed  = filemanager_yaml_file.mixed.path
  }
}

output "docker_compose" {
  value = {
    simple   = filemanager_yaml_file.docker_simple.path
    complex  = filemanager_yaml_file.docker_complex.path
    networks = filemanager_yaml_file.docker_networks.path
  }
}

output "kubernetes" {
  value = {
    deployment = filemanager_yaml_file.k8s_deployment.path
    service    = filemanager_yaml_file.k8s_service.path
    configmap  = filemanager_yaml_file.k8s_configmap.path
  }
}

output "ci_cd" {
  value = {
    github = filemanager_yaml_file.github_actions.path
    gitlab = filemanager_yaml_file.gitlab_ci.path
  }
}

output "summary" {
  value = {
    total      = 17
    categories = ["basic", "docker", "kubernetes", "ci_cd", "configs", "edge_cases"]
  }
}
