# =============================================================================
# APP CONFIG RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/19-app-config"
}

# -----------------------------------------------------------------------------
# NGINX CONFIGURATION
# -----------------------------------------------------------------------------

# Case 1: Basic nginx config
resource "filemanager_app_config" "nginx_basic" {
  path = "${local.output_dir}/nginx/nginx.conf"
  app  = "nginx"

  config = jsonencode({
    worker_processes = "auto"
    error_log        = "/var/log/nginx/error.log"

    events = {
      worker_connections = 1024
    }

    http = {
      include           = "/etc/nginx/mime.types"
      default_type      = "application/octet-stream"
      sendfile          = true
      keepalive_timeout = 65

      server = [{
        listen      = 80
        server_name = "localhost"
        root        = "/var/www/html"

        location = [{
          path  = "/"
          index = "index.html"
        }]
      }]
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# REDIS CONFIGURATION
# -----------------------------------------------------------------------------

# Case 2: Redis config
resource "filemanager_app_config" "redis" {
  path = "${local.output_dir}/redis/redis.conf"
  app  = "redis"

  config = jsonencode({
    bind             = ["127.0.0.1"]
    port             = 6379
    daemonize        = false
    loglevel         = "notice"
    databases        = 16
    maxmemory        = "256mb"
    maxmemory-policy = "allkeys-lru"
    appendonly       = true
    appendfsync      = "everysec"
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PROMETHEUS CONFIGURATION
# -----------------------------------------------------------------------------

# Case 3: Prometheus config
resource "filemanager_app_config" "prometheus" {
  path = "${local.output_dir}/prometheus/prometheus.yml"
  app  = "prometheus"

  config = jsonencode({
    global = {
      scrape_interval     = "15s"
      evaluation_interval = "15s"
    }

    scrape_configs = [
      {
        job_name = "prometheus"
        static_configs = [{
          targets = ["localhost:9090"]
        }]
      },
      {
        job_name = "node"
        static_configs = [{
          targets = ["localhost:9100"]
        }]
      }
    ]
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CONSUL CONFIGURATION
# -----------------------------------------------------------------------------

# Case 4: Consul config
resource "filemanager_app_config" "consul" {
  path = "${local.output_dir}/consul/consul.hcl"
  app  = "consul"

  config = jsonencode({
    datacenter       = "dc1"
    data_dir         = "/opt/consul/data"
    server           = true
    bootstrap_expect = 1

    bind_addr   = "127.0.0.1"
    client_addr = "127.0.0.1"

    ui_config = {
      enabled = true
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DOCKER DAEMON CONFIGURATION
# -----------------------------------------------------------------------------

# Case 5: Docker daemon config
resource "filemanager_app_config" "docker" {
  path = "${local.output_dir}/docker/daemon.json"
  app  = "docker"

  config = jsonencode({
    storage-driver = "overlay2"
    log-driver     = "json-file"
    log-opts = {
      max-size = "10m"
      max-file = "3"
    }
    dns = ["8.8.8.8", "8.8.4.4"]
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# SYSTEMD UNIT CONFIGURATION
# -----------------------------------------------------------------------------

# Case 6: Systemd service unit
resource "filemanager_app_config" "systemd_service" {
  path = "${local.output_dir}/systemd/myapp.service"
  app  = "systemd"

  config = jsonencode({
    Unit = {
      Description = "My Application Service"
      After       = "network.target"
    }
    Service = {
      Type                  = "simple"
      User                  = "myapp"
      ExecStart             = "/usr/bin/myapp --config /etc/myapp/config.yaml"
      Restart               = "always"
      RestartSec            = "5"
      StartLimitIntervalSec = "60"
      StartLimitBurst       = "3"
    }
    Install = {
      WantedBy = "multi-user.target"
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# MYSQL CONFIGURATION
# -----------------------------------------------------------------------------

# Case 7: MySQL config
resource "filemanager_app_config" "mysql" {
  path = "${local.output_dir}/mysql/my.cnf"
  app  = "mysql"

  config = jsonencode({
    mysqld = {
      port                    = 3306
      bind-address            = "127.0.0.1"
      datadir                 = "/var/lib/mysql"
      socket                  = "/var/run/mysqld/mysqld.sock"
      max_connections         = 150
      innodb_buffer_pool_size = "256M"
    }
    client = {
      port   = 3306
      socket = "/var/run/mysqld/mysqld.sock"
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# POSTGRESQL CONFIGURATION
# -----------------------------------------------------------------------------

# Case 8: PostgreSQL config
resource "filemanager_app_config" "postgresql" {
  path = "${local.output_dir}/postgresql/postgresql.conf"
  app  = "postgresql"

  config = jsonencode({
    listen_addresses     = "localhost"
    port                 = 5432
    max_connections      = 100
    shared_buffers       = "128MB"
    effective_cache_size = "512MB"
    work_mem             = "4MB"
    wal_level            = "replica"
    logging_collector    = true
    log_directory        = "log"
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# HAPROXY CONFIGURATION
# -----------------------------------------------------------------------------

# Case 9: HAProxy config
resource "filemanager_app_config" "haproxy" {
  path = "${local.output_dir}/haproxy/haproxy.cfg"
  app  = "haproxy"

  config = jsonencode({
    global = {
      daemon  = true
      maxconn = 4096
    }
    defaults = {
      mode            = "http"
      timeout_connect = "5000ms"
      timeout_client  = "50000ms"
      timeout_server  = "50000ms"
    }
    frontend = {
      http_front = {
        bind            = "*:80"
        default_backend = "http_back"
      }
    }
    backend = {
      http_back = {
        balance = "roundrobin"
        servers = [
          { name = "server1", address = "192.168.1.1:8080" },
          { name = "server2", address = "192.168.1.2:8080" }
        ]
      }
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# KUBERNETES MANIFEST
# -----------------------------------------------------------------------------

# Case 10: Kubernetes deployment
resource "filemanager_app_config" "k8s_deployment" {
  path = "${local.output_dir}/kubernetes/deployment.yaml"
  app  = "kubernetes"

  config = jsonencode({
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name = "my-app"
      labels = {
        app = "my-app"
      }
    }
    spec = {
      replicas = 3
      selector = {
        matchLabels = {
          app = "my-app"
        }
      }
      template = {
        metadata = {
          labels = {
            app = "my-app"
          }
        }
        spec = {
          containers = [{
            name  = "my-app"
            image = "my-app:v1.0.0"
            ports = [{
              containerPort = 8080
            }]
            resources = {
              requests = {
                memory = "128Mi"
                cpu    = "100m"
              }
              limits = {
                memory = "256Mi"
                cpu    = "200m"
              }
            }
          }]
        }
      }
    }
  })

  create_parent_dirs = true
}

