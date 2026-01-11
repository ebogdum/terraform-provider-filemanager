# =============================================================================
# VERIFICATION CHECKS - Multi-service deployment validation
# =============================================================================

# =============================================================================
# INFRASTRUCTURE DIRECTORIES
# =============================================================================

check "verify_infrastructure_dir" {
  data "filemanager_stat" "infrastructure_check" {
    path = filemanager_directory.infrastructure.path
  }

  assert {
    condition     = data.filemanager_stat.infrastructure_check.exists == true
    error_message = "Infrastructure directory should exist"
  }

  assert {
    condition     = data.filemanager_stat.infrastructure_check.is_dir == true
    error_message = "Infrastructure should be a directory"
  }
}

check "verify_shared_dir" {
  data "filemanager_stat" "shared_check" {
    path = filemanager_directory.shared.path
  }

  assert {
    condition     = data.filemanager_stat.shared_check.is_dir == true
    error_message = "Shared directory should exist"
  }
}

check "verify_services_dir" {
  data "filemanager_stat" "services_check" {
    path = filemanager_directory.services.path
  }

  assert {
    condition     = data.filemanager_stat.services_check.is_dir == true
    error_message = "Services directory should exist"
  }
}

check "verify_kubernetes_dir" {
  data "filemanager_stat" "kubernetes_check" {
    path = filemanager_directory.kubernetes.path
  }

  assert {
    condition     = data.filemanager_stat.kubernetes_check.is_dir == true
    error_message = "Kubernetes directory should exist"
  }
}

check "verify_monitoring_dir" {
  data "filemanager_stat" "monitoring_check" {
    path = filemanager_directory.monitoring.path
  }

  assert {
    condition     = data.filemanager_stat.monitoring_check.is_dir == true
    error_message = "Monitoring directory should exist"
  }
}

# =============================================================================
# SHARED CONFIGURATION
# =============================================================================

check "verify_shared_config" {
  data "filemanager_file" "shared_config_check" {
    path = filemanager_yaml_file.shared_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.shared_config_check.content, "production")
    error_message = "Shared config should contain environment"
  }

  assert {
    condition     = strcontains(data.filemanager_file.shared_config_check.content, "tracing")
    error_message = "Shared config should contain tracing section"
  }
}

check "verify_database_config" {
  data "filemanager_file" "database_config_check" {
    path = filemanager_yaml_file.database_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.database_config_check.content, "postgres")
    error_message = "Database config should contain postgres"
  }

  assert {
    condition     = strcontains(data.filemanager_file.database_config_check.content, "redis")
    error_message = "Database config should contain redis"
  }

  assert {
    condition     = strcontains(data.filemanager_file.database_config_check.content, "mongodb")
    error_message = "Database config should contain mongodb"
  }
}

check "verify_messaging_config" {
  data "filemanager_file" "messaging_config_check" {
    path = filemanager_yaml_file.messaging_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.messaging_config_check.content, "rabbitmq")
    error_message = "Messaging config should contain rabbitmq"
  }

  assert {
    condition     = strcontains(data.filemanager_file.messaging_config_check.content, "kafka")
    error_message = "Messaging config should contain kafka"
  }
}

check "verify_service_registry" {
  data "filemanager_file" "service_registry_check" {
    path = filemanager_json_file.service_registry.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.service_registry_check.content, "api_gateway")
    error_message = "Service registry should contain api_gateway"
  }

  assert {
    condition     = strcontains(data.filemanager_file.service_registry_check.content, "user_service")
    error_message = "Service registry should contain user_service"
  }

  assert {
    condition     = strcontains(data.filemanager_file.service_registry_check.content, "order_service")
    error_message = "Service registry should contain order_service"
  }
}

# =============================================================================
# API GATEWAY SERVICE
# =============================================================================

check "verify_api_gateway_config" {
  data "filemanager_file" "api_gateway_config_check" {
    path = filemanager_yaml_file.api_gateway_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.api_gateway_config_check.content, "api-gateway")
    error_message = "API Gateway config should contain service name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.api_gateway_config_check.content, "routes")
    error_message = "API Gateway config should contain routes"
  }
}

check "verify_api_gateway_env" {
  data "filemanager_file" "api_gateway_env_check" {
    path = filemanager_env_file.api_gateway_env.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.api_gateway_env_check.content, "SERVICE_NAME=api-gateway")
    error_message = "API Gateway env should have correct SERVICE_NAME"
  }
}

# =============================================================================
# USER SERVICE
# =============================================================================

check "verify_user_service_config" {
  data "filemanager_file" "user_service_config_check" {
    path = filemanager_yaml_file.user_service_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.user_service_config_check.content, "user-service")
    error_message = "User service config should contain service name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.user_service_config_check.content, "events")
    error_message = "User service config should contain events"
  }
}

check "verify_user_service_env" {
  data "filemanager_file" "user_service_env_check" {
    path = filemanager_env_file.user_service_env.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.user_service_env_check.content, "SERVICE_NAME=user-service")
    error_message = "User service env should have correct SERVICE_NAME"
  }
}

# =============================================================================
# ORDER SERVICE
# =============================================================================

check "verify_order_service_config" {
  data "filemanager_file" "order_service_config_check" {
    path = filemanager_yaml_file.order_service_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.order_service_config_check.content, "order-service")
    error_message = "Order service config should contain service name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.order_service_config_check.content, "dependencies")
    error_message = "Order service config should contain dependencies"
  }
}

check "verify_order_service_env" {
  data "filemanager_file" "order_service_env_check" {
    path = filemanager_env_file.order_service_env.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.order_service_env_check.content, "SERVICE_NAME=order-service")
    error_message = "Order service env should have correct SERVICE_NAME"
  }
}

# =============================================================================
# NOTIFICATION SERVICE
# =============================================================================

check "verify_notification_service_config" {
  data "filemanager_file" "notification_service_config_check" {
    path = filemanager_yaml_file.notification_service_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.notification_service_config_check.content, "notification-service")
    error_message = "Notification service config should contain service name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.notification_service_config_check.content, "channels")
    error_message = "Notification service config should contain channels"
  }
}

check "verify_notification_service_env" {
  data "filemanager_file" "notification_service_env_check" {
    path = filemanager_env_file.notification_service_env.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.notification_service_env_check.content, "SERVICE_NAME=notification-service")
    error_message = "Notification service env should have correct SERVICE_NAME"
  }
}

# =============================================================================
# ANALYTICS SERVICE
# =============================================================================

check "verify_analytics_service_config" {
  data "filemanager_file" "analytics_service_config_check" {
    path = filemanager_yaml_file.analytics_service_config.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.analytics_service_config_check.content, "analytics-service")
    error_message = "Analytics service config should contain service name"
  }

  assert {
    condition     = strcontains(data.filemanager_file.analytics_service_config_check.content, "clickhouse")
    error_message = "Analytics service config should contain clickhouse storage"
  }
}

check "verify_analytics_service_env" {
  data "filemanager_file" "analytics_service_env_check" {
    path = filemanager_env_file.analytics_service_env.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.analytics_service_env_check.content, "SERVICE_NAME=analytics-service")
    error_message = "Analytics service env should have correct SERVICE_NAME"
  }
}

# =============================================================================
# KUBERNETES MANIFESTS
# =============================================================================

check "verify_k8s_api_gateway" {
  data "filemanager_file" "k8s_api_gateway_check" {
    path = filemanager_template_file.k8s_api_gateway.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_api_gateway_check.content, "kind: Deployment")
    error_message = "K8s API Gateway should contain Deployment"
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_api_gateway_check.content, "kind: Service")
    error_message = "K8s API Gateway should contain Service"
  }
}

check "verify_k8s_user_service" {
  data "filemanager_file" "k8s_user_service_check" {
    path = filemanager_template_file.k8s_user_service.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.k8s_user_service_check.content, "kind: Deployment")
    error_message = "K8s User Service should contain Deployment"
  }
}

# =============================================================================
# MONITORING CONFIGURATION
# =============================================================================

check "verify_prometheus_targets" {
  data "filemanager_file" "prometheus_targets_check" {
    path = filemanager_yaml_file.prometheus_targets.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.prometheus_targets_check.content, "scrape_configs")
    error_message = "Prometheus targets should contain scrape_configs"
  }
}

check "verify_grafana_datasources" {
  data "filemanager_file" "grafana_datasources_check" {
    path = filemanager_yaml_file.grafana_datasources.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.grafana_datasources_check.content, "Prometheus")
    error_message = "Grafana datasources should contain Prometheus"
  }

  assert {
    condition     = strcontains(data.filemanager_file.grafana_datasources_check.content, "Jaeger")
    error_message = "Grafana datasources should contain Jaeger"
  }
}

check "verify_alert_rules" {
  data "filemanager_file" "alert_rules_check" {
    path = filemanager_yaml_file.alert_rules.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.alert_rules_check.content, "service-alerts")
    error_message = "Alert rules should contain service-alerts group"
  }
}

# =============================================================================
# DEPLOYMENT PACKAGE
# =============================================================================

check "verify_infrastructure_package" {
  data "filemanager_stat" "infrastructure_package_check" {
    path = filemanager_archive.infrastructure_package.path
  }

  assert {
    condition     = data.filemanager_stat.infrastructure_package_check.exists == true
    error_message = "Infrastructure package should exist"
  }

  assert {
    condition     = data.filemanager_stat.infrastructure_package_check.size > 0
    error_message = "Infrastructure package should not be empty"
  }
}

check "verify_deployment_manifest" {
  data "filemanager_file" "deployment_manifest_check" {
    path = filemanager_json_file.deployment_manifest.path
  }

  assert {
    condition     = strcontains(data.filemanager_file.deployment_manifest_check.content, "services")
    error_message = "Deployment manifest should contain services"
  }

  assert {
    condition     = strcontains(data.filemanager_file.deployment_manifest_check.content, "shared_configs")
    error_message = "Deployment manifest should contain shared_configs"
  }

  assert {
    condition     = strcontains(data.filemanager_file.deployment_manifest_check.content, "monitoring")
    error_message = "Deployment manifest should contain monitoring"
  }

  assert {
    condition     = strcontains(data.filemanager_file.deployment_manifest_check.content, "kubernetes")
    error_message = "Deployment manifest should contain kubernetes"
  }
}
