# TEMPLATE FILE - OUTPUTS

output "go_templates" {
  value = {
    simple   = { path = filemanager_template_file.go_simple.path, rendered = filemanager_template_file.go_simple.rendered_content }
    multiple = { path = filemanager_template_file.go_multiple.path }
    config   = { path = filemanager_template_file.go_config.path }
    nginx    = { path = filemanager_template_file.go_nginx.path }
    script   = { path = filemanager_template_file.go_script.path }
  }
}

output "mustache_templates" {
  value = {
    simple = { path = filemanager_template_file.mustache_simple.path, rendered = filemanager_template_file.mustache_simple.rendered_content }
    readme = { path = filemanager_template_file.mustache_readme.path }
    html   = { path = filemanager_template_file.mustache_html.path }
  }
}

output "custom_delimiters" {
  value = {
    brackets = { path = filemanager_template_file.custom_delims.path, rendered = filemanager_template_file.custom_delims.rendered_content }
    angle    = { path = filemanager_template_file.angle_delims.path }
  }
}

output "configs" {
  value = {
    docker_compose = filemanager_template_file.docker_compose.path
    k8s_deployment = filemanager_template_file.k8s_deployment.path
    tf_backend     = filemanager_template_file.tf_backend.path
  }
}

output "summary" {
  value = { total = 18, categories = ["go_templates", "mustache_templates", "custom_delimiters", "configs", "edge_cases"] }
}
