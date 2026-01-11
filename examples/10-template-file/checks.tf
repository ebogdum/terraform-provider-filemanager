# =============================================================================
# VERIFICATION CHECKS - Template file resource validation
# =============================================================================

check "verify_go_simple" {
  data "filemanager_file" "go_simple_check" {
    path = filemanager_template_file.go_simple.path
  }

  assert {
    condition     = data.filemanager_file.go_simple_check.content == "Hello, World!"
    error_message = "Go simple template should render to 'Hello, World!'"
  }
}

check "verify_go_multiple" {
  data "filemanager_file" "go_multiple_check" {
    path = filemanager_template_file.go_multiple.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.go_multiple_check.content, "Application: MyApp")
    error_message = "Go template should contain 'Application: MyApp'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.go_multiple_check.content, "Version: 1.0.0")
    error_message = "Go template should contain 'Version: 1.0.0'"
  }
}

check "verify_go_config" {
  data "filemanager_file" "go_config_check" {
    path = filemanager_template_file.go_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.go_config_check.content, "host: 0.0.0.0")
    error_message = "Config template should contain 'host: 0.0.0.0'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.go_config_check.content, "port: 8080")
    error_message = "Config template should contain 'port: 8080'"
  }
}

check "verify_mustache_simple" {
  data "filemanager_file" "mustache_check" {
    path = filemanager_template_file.mustache_simple.path
  }

  assert {
    condition     = data.filemanager_file.mustache_check.content == "Hello, Mustache!"
    error_message = "Mustache simple template should render to 'Hello, Mustache!'"
  }
}

check "verify_mustache_html" {
  data "filemanager_file" "html_check" {
    path = filemanager_template_file.mustache_html.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.html_check.content, "<title>Welcome to My Site</title>")
    error_message = "HTML template should contain rendered title"
  }

  assert {
    condition     = strcontains(data.filemanager_file.html_check.content, "<h1>My Website</h1>")
    error_message = "HTML template should contain rendered heading"
  }
}

check "verify_go_nginx" {
  data "filemanager_file" "nginx_check" {
    path = filemanager_template_file.go_nginx.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.nginx_check.content, "listen 80")
    error_message = "Nginx template should contain 'listen 80'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.nginx_check.content, "server_name example.com")
    error_message = "Nginx template should contain 'server_name example.com'"
  }
}

check "verify_docker_compose" {
  data "filemanager_file" "docker_check" {
    path = filemanager_template_file.docker_compose.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_check.content, "version:")
    error_message = "Docker Compose template should contain 'version:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.docker_check.content, "services:")
    error_message = "Docker Compose template should contain 'services:'"
  }
}

check "verify_k8s_deployment" {
  data "filemanager_file" "k8s_check" {
    path = filemanager_template_file.k8s_deployment.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_check.content, "apiVersion:")
    error_message = "K8s template should contain 'apiVersion:'"
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_check.content, "kind: Deployment")
    error_message = "K8s template should contain 'kind: Deployment'"
  }
}

# =============================================================================
# ENHANCED STAT CHECKS (time-based)
# =============================================================================

check "stat_template_time_check" {
  data "filemanager_stat" "template_time_check" {
    path            = filemanager_template_file.go_simple.path
    modified_within = "1h"
  }

  assert {
    condition     = data.filemanager_stat.template_time_check.is_modified_within == true
    error_message = "Newly created template file should be modified within last hour"
  }
}
