# =============================================================================
# APP CONFIG RESOURCE - OUTPUTS
# =============================================================================

output "web_server_configs" {
  description = "Web server configuration tests"
  value = {
    nginx = {
      path     = filemanager_app_config.nginx_basic.path
      md5      = filemanager_app_config.nginx_basic.md5
      rendered = filemanager_app_config.nginx_basic.rendered
    }
    haproxy = {
      path = filemanager_app_config.haproxy.path
      md5  = filemanager_app_config.haproxy.md5
    }
  }
}

output "database_configs" {
  description = "Database configuration tests"
  value = {
    redis = {
      path = filemanager_app_config.redis.path
      md5  = filemanager_app_config.redis.md5
    }
    mysql = {
      path = filemanager_app_config.mysql.path
      md5  = filemanager_app_config.mysql.md5
    }
    postgresql = {
      path = filemanager_app_config.postgresql.path
      md5  = filemanager_app_config.postgresql.md5
    }
  }
}

output "monitoring_configs" {
  description = "Monitoring configuration tests"
  value = {
    prometheus = {
      path = filemanager_app_config.prometheus.path
      md5  = filemanager_app_config.prometheus.md5
    }
  }
}

output "infrastructure_configs" {
  description = "Infrastructure configuration tests"
  value = {
    consul = {
      path = filemanager_app_config.consul.path
      md5  = filemanager_app_config.consul.md5
    }
    docker = {
      path = filemanager_app_config.docker.path
      md5  = filemanager_app_config.docker.md5
    }
    systemd = {
      path = filemanager_app_config.systemd_service.path
      md5  = filemanager_app_config.systemd_service.md5
    }
  }
}

output "kubernetes_configs" {
  description = "Kubernetes configuration tests"
  value = {
    deployment = {
      path = filemanager_app_config.k8s_deployment.path
      md5  = filemanager_app_config.k8s_deployment.md5
    }
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_app_configs = 10
    categories = [
      "web_servers",
      "databases",
      "monitoring",
      "infrastructure",
      "kubernetes"
    ]
  }
}
