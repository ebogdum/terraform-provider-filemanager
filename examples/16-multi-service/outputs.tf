# MULTI-SERVICE DEPLOYMENT - OUTPUTS

# =============================================================================
# INFRASTRUCTURE
# =============================================================================

output "infrastructure" {
  value = {
    base_dir   = filemanager_directory.infrastructure.path
    shared     = filemanager_directory.shared.path
    services   = filemanager_directory.services.path
    kubernetes = filemanager_directory.kubernetes.path
    monitoring = filemanager_directory.monitoring.path
  }
}

# =============================================================================
# SHARED CONFIGURATION
# =============================================================================

output "shared_config" {
  value = {
    common = {
      path = filemanager_yaml_file.shared_config.path
      md5  = filemanager_yaml_file.shared_config.md5
    }
    database = {
      path = filemanager_yaml_file.database_config.path
      md5  = filemanager_yaml_file.database_config.md5
    }
    messaging = {
      path = filemanager_yaml_file.messaging_config.path
      md5  = filemanager_yaml_file.messaging_config.md5
    }
    service_registry = {
      path     = filemanager_json_file.service_registry.path
      md5      = filemanager_json_file.service_registry.md5
      checksum = data.filemanager_checksum.service_registry.checksum
    }
  }
}

# =============================================================================
# SERVICE CONFIGURATIONS
# =============================================================================

output "api_gateway" {
  value = {
    directory = filemanager_directory.service_api_gateway.path
    config    = filemanager_yaml_file.api_gateway_config.path
    env       = filemanager_env_file.api_gateway_env.path
    k8s       = filemanager_template_file.k8s_api_gateway.path
  }
}

output "user_service" {
  value = {
    directory = filemanager_directory.service_user.path
    config    = filemanager_yaml_file.user_service_config.path
    env       = filemanager_env_file.user_service_env.path
    k8s       = filemanager_template_file.k8s_user_service.path
  }
}

output "order_service" {
  value = {
    directory = filemanager_directory.service_order.path
    config    = filemanager_yaml_file.order_service_config.path
    env       = filemanager_env_file.order_service_env.path
  }
}

output "notification_service" {
  value = {
    directory = filemanager_directory.service_notification.path
    config    = filemanager_yaml_file.notification_service_config.path
    env       = filemanager_env_file.notification_service_env.path
  }
}

output "analytics_service" {
  value = {
    directory = filemanager_directory.service_analytics.path
    config    = filemanager_yaml_file.analytics_service_config.path
    env       = filemanager_env_file.analytics_service_env.path
  }
}

# =============================================================================
# MONITORING
# =============================================================================

output "monitoring" {
  value = {
    prometheus_targets  = filemanager_yaml_file.prometheus_targets.path
    grafana_datasources = filemanager_yaml_file.grafana_datasources.path
    alert_rules         = filemanager_yaml_file.alert_rules.path
  }
}

# =============================================================================
# DEPLOYMENT
# =============================================================================

output "deployment" {
  value = {
    package = {
      path = filemanager_archive.infrastructure_package.path
      size = filemanager_archive.infrastructure_package.size
    }
    manifest = {
      path = filemanager_json_file.deployment_manifest.path
      md5  = filemanager_json_file.deployment_manifest.md5
    }
  }
}

# =============================================================================
# STATISTICS
# =============================================================================

output "statistics" {
  value = {
    services_directory = {
      file_count = data.filemanager_directory.all_services.file_count
      total_size = data.filemanager_directory.all_services.total_size
    }
  }
}

# =============================================================================
# SUMMARY
# =============================================================================

output "summary" {
  value = {
    total_services  = 5
    total_resources = 32

    demonstrates = [
      "Services referencing each other's endpoints",
      "Shared configs used across all services",
      "Service discovery auto-generated from definitions",
      "Environment files referencing config paths",
      "Kubernetes manifests using template variables",
      "Monitoring config derived from service list",
      "Deployment package with all infrastructure",
      "Manifest aggregating all service information"
    ]

    dependency_flow = [
      "1. Infrastructure directories created",
      "2. Shared configs (database, messaging) created",
      "3. Service registry generated from service definitions",
      "4. Per-service configs reference shared configs",
      "5. ENV files reference config paths and other services",
      "6. K8s manifests use service configs and inter-service URLs",
      "7. Monitoring configs iterate over all services",
      "8. Data sources read created content",
      "9. Archive packages all infrastructure",
      "10. Manifest aggregates paths, checksums, and stats"
    ]
  }
}
