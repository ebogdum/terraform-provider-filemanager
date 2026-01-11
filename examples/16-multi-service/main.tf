# =============================================================================
# MULTI-SERVICE DEPLOYMENT - MICROSERVICES ARCHITECTURE
# =============================================================================
#
# This module demonstrates a realistic microservices deployment where:
# - Multiple services share common configuration
# - Services reference each other's endpoints
# - Shared configs are generated and used across services
# - Each service has its own config derived from shared values
# - Service discovery configuration is auto-generated
#
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {}

locals {
  base_dir = "${path.module}/../../test/output/16-multi-service"

  # Shared configuration values
  environment = "production"
  domain      = "example.com"

  # Service definitions
  services = {
    api_gateway = {
      name     = "api-gateway"
      port     = 8080
      replicas = 2
    }
    user_service = {
      name     = "user-service"
      port     = 8081
      replicas = 3
    }
    order_service = {
      name     = "order-service"
      port     = 8082
      replicas = 3
    }
    notification_service = {
      name     = "notification-service"
      port     = 8083
      replicas = 2
    }
    analytics_service = {
      name     = "analytics-service"
      port     = 8084
      replicas = 1
    }
  }
}

# =============================================================================
# INFRASTRUCTURE DIRECTORIES
# =============================================================================

resource "filemanager_directory" "infrastructure" {
  path           = "${local.base_dir}/infrastructure"
  create_parents = true
}

resource "filemanager_directory" "shared" {
  path           = "${filemanager_directory.infrastructure.path}/shared"
  create_parents = true
}

resource "filemanager_directory" "services" {
  path           = "${filemanager_directory.infrastructure.path}/services"
  create_parents = true
}

resource "filemanager_directory" "kubernetes" {
  path           = "${filemanager_directory.infrastructure.path}/kubernetes"
  create_parents = true
}

resource "filemanager_directory" "monitoring" {
  path           = "${filemanager_directory.infrastructure.path}/monitoring"
  create_parents = true
}

# Per-service directories
resource "filemanager_directory" "service_api_gateway" {
  path = "${filemanager_directory.services.path}/api-gateway"
}

resource "filemanager_directory" "service_user" {
  path = "${filemanager_directory.services.path}/user-service"
}

resource "filemanager_directory" "service_order" {
  path = "${filemanager_directory.services.path}/order-service"
}

resource "filemanager_directory" "service_notification" {
  path = "${filemanager_directory.services.path}/notification-service"
}

resource "filemanager_directory" "service_analytics" {
  path = "${filemanager_directory.services.path}/analytics-service"
}

# =============================================================================
# SHARED CONFIGURATION
# =============================================================================

# Shared environment config
resource "filemanager_yaml_file" "shared_config" {
  path = "${filemanager_directory.shared.path}/common.yaml"

  content = yamlencode({
    environment = local.environment
    domain      = local.domain

    logging = {
      level  = "INFO"
      format = "json"
    }

    tracing = {
      enabled  = true
      endpoint = "http://jaeger:14268/api/traces"
    }

    metrics = {
      enabled = true
      port    = 9090
    }
  })
}

# Database connection pool shared config
resource "filemanager_yaml_file" "database_config" {
  path = "${filemanager_directory.shared.path}/database.yaml"

  content = yamlencode({
    postgres = {
      host            = "postgres.${local.domain}"
      port            = 5432
      max_connections = 100
      ssl_mode        = "require"
    }

    redis = {
      host         = "redis.${local.domain}"
      port         = 6379
      cluster_mode = true
    }

    mongodb = {
      hosts       = ["mongo1.${local.domain}", "mongo2.${local.domain}", "mongo3.${local.domain}"]
      replica_set = "rs0"
    }
  })
}

# Message queue config
resource "filemanager_yaml_file" "messaging_config" {
  path = "${filemanager_directory.shared.path}/messaging.yaml"

  content = yamlencode({
    rabbitmq = {
      host     = "rabbitmq.${local.domain}"
      port     = 5672
      vhost    = "/"
      exchange = "events"
    }

    kafka = {
      brokers = ["kafka1.${local.domain}:9092", "kafka2.${local.domain}:9092"]
      topics = {
        user_events         = "user-events"
        order_events        = "order-events"
        notification_events = "notification-events"
      }
    }
  })
}

# =============================================================================
# SERVICE DISCOVERY CONFIGURATION
# =============================================================================
# Auto-generated from service definitions

resource "filemanager_json_file" "service_registry" {
  path = "${filemanager_directory.shared.path}/service-registry.json"

  content = jsonencode({
    services = {
      for name, svc in local.services : name => {
        name         = svc.name
        endpoint     = "http://${svc.name}.${local.domain}:${svc.port}"
        port         = svc.port
        replicas     = svc.replicas
        health_check = "http://${svc.name}.${local.domain}:${svc.port}/health"
      }
    }

    endpoints = {
      for name, svc in local.services : svc.name => "http://${svc.name}.${local.domain}:${svc.port}"
    }
  })

  sort_keys = true
  indent    = 2
}

# =============================================================================
# API GATEWAY SERVICE
# =============================================================================

resource "filemanager_yaml_file" "api_gateway_config" {
  path = "${filemanager_directory.service_api_gateway.path}/config.yaml"

  content = yamlencode({
    service = {
      name = local.services.api_gateway.name
      port = local.services.api_gateway.port
    }

    # Reference shared configs
    shared = {
      common   = filemanager_yaml_file.shared_config.path
      database = filemanager_yaml_file.database_config.path
    }

    # Route configuration referencing other services
    routes = [
      {
        path    = "/api/users/*"
        service = local.services.user_service.name
        port    = local.services.user_service.port
      },
      {
        path    = "/api/orders/*"
        service = local.services.order_service.name
        port    = local.services.order_service.port
      },
      {
        path    = "/api/notifications/*"
        service = local.services.notification_service.name
        port    = local.services.notification_service.port
      },
      {
        path    = "/api/analytics/*"
        service = local.services.analytics_service.name
        port    = local.services.analytics_service.port
      }
    ]

    rate_limiting = {
      enabled             = true
      requests_per_second = 1000
    }

    cors = {
      allowed_origins = ["https://${local.domain}", "https://admin.${local.domain}"]
    }
  })
}

resource "filemanager_env_file" "api_gateway_env" {
  path = "${filemanager_directory.service_api_gateway.path}/.env"

  variables = {
    SERVICE_NAME = local.services.api_gateway.name
    SERVICE_PORT = tostring(local.services.api_gateway.port)
    ENVIRONMENT  = local.environment

    CONFIG_FILE   = filemanager_yaml_file.api_gateway_config.path
    SHARED_CONFIG = filemanager_yaml_file.shared_config.path
    REGISTRY_FILE = filemanager_json_file.service_registry.path

    # Downstream services
    USER_SERVICE_URL  = "http://${local.services.user_service.name}:${local.services.user_service.port}"
    ORDER_SERVICE_URL = "http://${local.services.order_service.name}:${local.services.order_service.port}"
    NOTIFICATION_URL  = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
  }
}

# =============================================================================
# USER SERVICE
# =============================================================================

resource "filemanager_yaml_file" "user_service_config" {
  path = "${filemanager_directory.service_user.path}/config.yaml"

  content = yamlencode({
    service = {
      name = local.services.user_service.name
      port = local.services.user_service.port
    }

    shared = {
      common    = filemanager_yaml_file.shared_config.path
      database  = filemanager_yaml_file.database_config.path
      messaging = filemanager_yaml_file.messaging_config.path
    }

    database = {
      name      = "users"
      pool_size = 20
    }

    events = {
      publish = ["user.created", "user.updated", "user.deleted"]
    }

    dependencies = {
      notification_service = {
        endpoint = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
      }
    }
  })
}

resource "filemanager_env_file" "user_service_env" {
  path = "${filemanager_directory.service_user.path}/.env"

  variables = {
    SERVICE_NAME = local.services.user_service.name
    SERVICE_PORT = tostring(local.services.user_service.port)
    ENVIRONMENT  = local.environment

    CONFIG_FILE      = filemanager_yaml_file.user_service_config.path
    DATABASE_CONFIG  = filemanager_yaml_file.database_config.path
    MESSAGING_CONFIG = filemanager_yaml_file.messaging_config.path

    NOTIFICATION_SERVICE_URL = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
  }
}

# =============================================================================
# ORDER SERVICE
# =============================================================================

resource "filemanager_yaml_file" "order_service_config" {
  path = "${filemanager_directory.service_order.path}/config.yaml"

  content = yamlencode({
    service = {
      name = local.services.order_service.name
      port = local.services.order_service.port
    }

    shared = {
      common    = filemanager_yaml_file.shared_config.path
      database  = filemanager_yaml_file.database_config.path
      messaging = filemanager_yaml_file.messaging_config.path
    }

    database = {
      name      = "orders"
      pool_size = 30
    }

    events = {
      publish   = ["order.created", "order.updated", "order.completed", "order.cancelled"]
      subscribe = ["user.created", "user.deleted"]
    }

    dependencies = {
      user_service = {
        endpoint = "http://${local.services.user_service.name}:${local.services.user_service.port}"
      }
      notification_service = {
        endpoint = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
      }
      analytics_service = {
        endpoint = "http://${local.services.analytics_service.name}:${local.services.analytics_service.port}"
      }
    }
  })
}

resource "filemanager_env_file" "order_service_env" {
  path = "${filemanager_directory.service_order.path}/.env"

  variables = {
    SERVICE_NAME = local.services.order_service.name
    SERVICE_PORT = tostring(local.services.order_service.port)
    ENVIRONMENT  = local.environment

    CONFIG_FILE = filemanager_yaml_file.order_service_config.path

    USER_SERVICE_URL         = "http://${local.services.user_service.name}:${local.services.user_service.port}"
    NOTIFICATION_SERVICE_URL = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
    ANALYTICS_SERVICE_URL    = "http://${local.services.analytics_service.name}:${local.services.analytics_service.port}"
  }
}

# =============================================================================
# NOTIFICATION SERVICE
# =============================================================================

resource "filemanager_yaml_file" "notification_service_config" {
  path = "${filemanager_directory.service_notification.path}/config.yaml"

  content = yamlencode({
    service = {
      name = local.services.notification_service.name
      port = local.services.notification_service.port
    }

    shared = {
      common    = filemanager_yaml_file.shared_config.path
      messaging = filemanager_yaml_file.messaging_config.path
    }

    events = {
      subscribe = ["user.created", "order.created", "order.completed"]
    }

    channels = {
      email = {
        enabled  = true
        provider = "sendgrid"
      }
      sms = {
        enabled  = true
        provider = "twilio"
      }
      push = {
        enabled  = true
        provider = "firebase"
      }
    }
  })
}

resource "filemanager_env_file" "notification_service_env" {
  path = "${filemanager_directory.service_notification.path}/.env"

  variables = {
    SERVICE_NAME = local.services.notification_service.name
    SERVICE_PORT = tostring(local.services.notification_service.port)
    ENVIRONMENT  = local.environment

    CONFIG_FILE      = filemanager_yaml_file.notification_service_config.path
    MESSAGING_CONFIG = filemanager_yaml_file.messaging_config.path
  }
}

# =============================================================================
# ANALYTICS SERVICE
# =============================================================================

resource "filemanager_yaml_file" "analytics_service_config" {
  path = "${filemanager_directory.service_analytics.path}/config.yaml"

  content = yamlencode({
    service = {
      name = local.services.analytics_service.name
      port = local.services.analytics_service.port
    }

    shared = {
      common    = filemanager_yaml_file.shared_config.path
      database  = filemanager_yaml_file.database_config.path
      messaging = filemanager_yaml_file.messaging_config.path
    }

    events = {
      subscribe = ["user.created", "user.deleted", "order.created", "order.completed"]
    }

    storage = {
      clickhouse = {
        host = "clickhouse.${local.domain}"
        port = 8123
      }
    }
  })
}

resource "filemanager_env_file" "analytics_service_env" {
  path = "${filemanager_directory.service_analytics.path}/.env"

  variables = {
    SERVICE_NAME = local.services.analytics_service.name
    SERVICE_PORT = tostring(local.services.analytics_service.port)
    ENVIRONMENT  = local.environment

    CONFIG_FILE = filemanager_yaml_file.analytics_service_config.path
  }
}

# =============================================================================
# KUBERNETES MANIFESTS
# =============================================================================

# Generate deployment for each service
resource "filemanager_template_file" "k8s_api_gateway" {
  path = "${filemanager_directory.kubernetes.path}/api-gateway-deployment.yaml"

  template = <<-EOF
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: {{.name}}
      labels:
        app: {{.name}}
        environment: {{.environment}}
    spec:
      replicas: {{.replicas}}
      selector:
        matchLabels:
          app: {{.name}}
      template:
        metadata:
          labels:
            app: {{.name}}
        spec:
          containers:
          - name: {{.name}}
            image: {{.name}}:latest
            ports:
            - containerPort: {{.port}}
            envFrom:
            - configMapRef:
                name: {{.name}}-config
            volumeMounts:
            - name: config
              mountPath: /app/config
              readOnly: true
          volumes:
          - name: config
            configMap:
              name: {{.name}}-files
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: {{.name}}
    spec:
      selector:
        app: {{.name}}
      ports:
      - port: {{.port}}
        targetPort: {{.port}}
  EOF

  vars = {
    name        = local.services.api_gateway.name
    port        = tostring(local.services.api_gateway.port)
    replicas    = tostring(local.services.api_gateway.replicas)
    environment = local.environment
  }

  engine = "go"
}

resource "filemanager_template_file" "k8s_user_service" {
  path = "${filemanager_directory.kubernetes.path}/user-service-deployment.yaml"

  template = <<-EOF
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: {{.name}}
      labels:
        app: {{.name}}
    spec:
      replicas: {{.replicas}}
      selector:
        matchLabels:
          app: {{.name}}
      template:
        metadata:
          labels:
            app: {{.name}}
        spec:
          containers:
          - name: {{.name}}
            image: {{.name}}:latest
            ports:
            - containerPort: {{.port}}
            env:
            - name: CONFIG_FILE
              value: {{.config_file}}
            - name: NOTIFICATION_SERVICE_URL
              value: {{.notification_url}}
    ---
    apiVersion: v1
    kind: Service
    metadata:
      name: {{.name}}
    spec:
      selector:
        app: {{.name}}
      ports:
      - port: {{.port}}
  EOF

  vars = {
    name             = local.services.user_service.name
    port             = tostring(local.services.user_service.port)
    replicas         = tostring(local.services.user_service.replicas)
    config_file      = filemanager_yaml_file.user_service_config.path
    notification_url = "http://${local.services.notification_service.name}:${local.services.notification_service.port}"
  }

  engine = "go"
}

# =============================================================================
# MONITORING CONFIGURATION
# =============================================================================

# Prometheus targets from service registry
resource "filemanager_yaml_file" "prometheus_targets" {
  path = "${filemanager_directory.monitoring.path}/prometheus-targets.yaml"

  content = yamlencode({
    scrape_configs = [
      for name, svc in local.services : {
        job_name = svc.name
        static_configs = [{
          targets = ["${svc.name}.${local.domain}:9090"]
          labels = {
            service     = svc.name
            environment = local.environment
          }
        }]
      }
    ]
  })
}

# Grafana datasources
resource "filemanager_yaml_file" "grafana_datasources" {
  path = "${filemanager_directory.monitoring.path}/grafana-datasources.yaml"

  content = yamlencode({
    apiVersion = 1
    datasources = [
      {
        name      = "Prometheus"
        type      = "prometheus"
        url       = "http://prometheus:9090"
        isDefault = true
      },
      {
        name = "Jaeger"
        type = "jaeger"
        url  = "http://jaeger:16686"
      }
    ]
  })
}

# Alert rules referencing services
resource "filemanager_yaml_file" "alert_rules" {
  path = "${filemanager_directory.monitoring.path}/alert-rules.yaml"

  content = yamlencode({
    groups = [
      {
        name = "service-alerts"
        rules = [
          for name, svc in local.services : {
            alert = "${svc.name}_down"
            expr  = "up{job=\"${svc.name}\"} == 0"
            for   = "5m"
            labels = {
              severity = "critical"
              service  = svc.name
            }
            annotations = {
              summary     = "${svc.name} is down"
              description = "Service ${svc.name} has been down for more than 5 minutes"
            }
          }
        ]
      }
    ]
  })
}

# =============================================================================
# DATA SOURCES - READ CREATED CONFIGS
# =============================================================================

data "filemanager_directory" "all_services" {
  path      = filemanager_directory.services.path
  recursive = true

  depends_on = [
    filemanager_yaml_file.api_gateway_config,
    filemanager_yaml_file.user_service_config,
    filemanager_yaml_file.order_service_config,
    filemanager_yaml_file.notification_service_config,
    filemanager_yaml_file.analytics_service_config,
    filemanager_env_file.api_gateway_env,
    filemanager_env_file.user_service_env,
    filemanager_env_file.order_service_env,
    filemanager_env_file.notification_service_env,
    filemanager_env_file.analytics_service_env,
  ]
}

data "filemanager_checksum" "service_registry" {
  path      = filemanager_json_file.service_registry.path
  algorithm = "sha256"

  depends_on = [filemanager_json_file.service_registry]
}

# =============================================================================
# DEPLOYMENT PACKAGE
# =============================================================================

resource "filemanager_archive" "infrastructure_package" {
  path       = "${local.base_dir}/deploy/infrastructure.tar.gz"
  type       = "tar.gz"
  source_dir = filemanager_directory.infrastructure.path

  create_parent_dirs = true

  depends_on = [
    filemanager_yaml_file.prometheus_targets,
    filemanager_yaml_file.alert_rules,
    filemanager_template_file.k8s_api_gateway,
    filemanager_template_file.k8s_user_service,
    data.filemanager_directory.all_services,
  ]
}

# =============================================================================
# DEPLOYMENT MANIFEST
# =============================================================================

resource "filemanager_json_file" "deployment_manifest" {
  path = "${local.base_dir}/deploy/manifest.json"

  content = jsonencode({
    generated_at = timestamp()
    environment  = local.environment
    domain       = local.domain

    services = {
      for name, svc in local.services : name => {
        name        = svc.name
        port        = svc.port
        replicas    = svc.replicas
        config_file = "${filemanager_directory.services.path}/${svc.name}/config.yaml"
        env_file    = "${filemanager_directory.services.path}/${svc.name}/.env"
      }
    }

    shared_configs = {
      common    = filemanager_yaml_file.shared_config.path
      database  = filemanager_yaml_file.database_config.path
      messaging = filemanager_yaml_file.messaging_config.path
      registry  = filemanager_json_file.service_registry.path
    }

    monitoring = {
      prometheus_targets  = filemanager_yaml_file.prometheus_targets.path
      grafana_datasources = filemanager_yaml_file.grafana_datasources.path
      alert_rules         = filemanager_yaml_file.alert_rules.path
    }

    kubernetes = {
      api_gateway  = filemanager_template_file.k8s_api_gateway.path
      user_service = filemanager_template_file.k8s_user_service.path
    }

    package = {
      path     = filemanager_archive.infrastructure_package.path
      size     = filemanager_archive.infrastructure_package.size
      checksum = data.filemanager_checksum.service_registry.checksum
    }

    statistics = {
      total_files      = data.filemanager_directory.all_services.file_count
      total_size_bytes = data.filemanager_directory.all_services.total_size
    }
  })

  create_parent_dirs = true
  sort_keys          = true
  indent             = 2

  depends_on = [filemanager_archive.infrastructure_package]
}
