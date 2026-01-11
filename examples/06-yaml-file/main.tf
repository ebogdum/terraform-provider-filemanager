# =============================================================================
# YAML FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/06-yaml-file"
}

# -----------------------------------------------------------------------------
# BASIC YAML FILES
# -----------------------------------------------------------------------------

# Case 1: Simple key-value
resource "filemanager_yaml_file" "simple" {
  path = "${local.output_dir}/basic/simple.yaml"
  content = yamlencode({
    key   = "value"
    count = 42
  })
  create_parent_dirs = true
}

# Case 2: Nested structure
resource "filemanager_yaml_file" "nested" {
  path = "${local.output_dir}/basic/nested.yaml"
  content = yamlencode({
    database = {
      host = "localhost"
      port = 5432
      name = "mydb"
      credentials = {
        username = "admin"
        password = "secret"
      }
    }
  })
  create_parent_dirs = true
}

# Case 3: Array/List
resource "filemanager_yaml_file" "list" {
  path = "${local.output_dir}/basic/list.yaml"
  content = yamlencode({
    fruits  = ["apple", "banana", "cherry"]
    numbers = [1, 2, 3, 4, 5]
  })
  create_parent_dirs = true
}

# Case 4: Mixed types
resource "filemanager_yaml_file" "mixed" {
  path = "${local.output_dir}/basic/mixed.yaml"
  content = yamlencode({
    string   = "hello"
    integer  = 42
    float    = 3.14
    boolean  = true
    null_val = null
  })
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# DOCKER COMPOSE FILES
# -----------------------------------------------------------------------------

# Case 5: Simple docker-compose
resource "filemanager_yaml_file" "docker_simple" {
  path = "${local.output_dir}/docker/simple.yaml"
  content = yamlencode({
    version = "3.8"
    services = {
      web = {
        image = "nginx:alpine"
        ports = ["80:80"]
      }
    }
  })
  create_parent_dirs = true
}

# Case 6: Complex docker-compose
resource "filemanager_yaml_file" "docker_complex" {
  path = "${local.output_dir}/docker/complex.yaml"
  content = yamlencode({
    version = "3.8"
    services = {
      frontend = {
        build = {
          context    = "./frontend"
          dockerfile = "Dockerfile"
        }
        ports = ["3000:3000"]
        environment = {
          NODE_ENV = "production"
          API_URL  = "http://backend:8080"
        }
        depends_on = ["backend"]
        volumes    = ["./frontend:/app"]
        restart    = "unless-stopped"
      }
      backend = {
        build = "./backend"
        ports = ["8080:8080"]
        environment = {
          DATABASE_URL = "postgres://user:pass@db:5432/mydb"
        }
        depends_on = ["db"]
      }
      db = {
        image = "postgres:15"
        environment = {
          POSTGRES_USER     = "user"
          POSTGRES_PASSWORD = "pass"
          POSTGRES_DB       = "mydb"
        }
        volumes = ["pgdata:/var/lib/postgresql/data"]
      }
    }
    volumes = {
      pgdata = {}
    }
    networks = {
      default = {
        driver = "bridge"
      }
    }
  })
  create_parent_dirs = true
}

# Case 7: Docker compose with multiple networks
resource "filemanager_yaml_file" "docker_networks" {
  path = "${local.output_dir}/docker/networks.yaml"
  content = yamlencode({
    version = "3.8"
    services = {
      app = {
        image    = "myapp:latest"
        networks = ["frontend", "backend"]
      }
      db = {
        image    = "mysql:8"
        networks = ["backend"]
      }
      nginx = {
        image    = "nginx:alpine"
        networks = ["frontend"]
        ports    = ["80:80", "443:443"]
      }
    }
    networks = {
      frontend = {
        driver = "bridge"
      }
      backend = {
        driver   = "bridge"
        internal = true
      }
    }
  })
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# KUBERNETES MANIFESTS
# -----------------------------------------------------------------------------

# Case 8: K8s Deployment
resource "filemanager_yaml_file" "k8s_deployment" {
  path = "${local.output_dir}/k8s/deployment.yaml"
  content = yamlencode({
    apiVersion = "apps/v1"
    kind       = "Deployment"
    metadata = {
      name = "nginx-deployment"
      labels = {
        app = "nginx"
      }
    }
    spec = {
      replicas = 3
      selector = {
        matchLabels = {
          app = "nginx"
        }
      }
      template = {
        metadata = {
          labels = {
            app = "nginx"
          }
        }
        spec = {
          containers = [{
            name  = "nginx"
            image = "nginx:1.25"
            ports = [{
              containerPort = 80
            }]
            resources = {
              limits = {
                cpu    = "100m"
                memory = "128Mi"
              }
              requests = {
                cpu    = "50m"
                memory = "64Mi"
              }
            }
          }]
        }
      }
    }
  })
  create_parent_dirs = true
}

# Case 9: K8s Service
resource "filemanager_yaml_file" "k8s_service" {
  path = "${local.output_dir}/k8s/service.yaml"
  content = yamlencode({
    apiVersion = "v1"
    kind       = "Service"
    metadata = {
      name = "nginx-service"
    }
    spec = {
      selector = {
        app = "nginx"
      }
      ports = [{
        protocol   = "TCP"
        port       = 80
        targetPort = 80
      }]
      type = "LoadBalancer"
    }
  })
  create_parent_dirs = true
}

# Case 10: K8s ConfigMap
resource "filemanager_yaml_file" "k8s_configmap" {
  path = "${local.output_dir}/k8s/configmap.yaml"
  content = yamlencode({
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "app-config"
    }
    data = {
      "config.json" = jsonencode({
        debug    = false
        logLevel = "info"
      })
      "settings.properties" = "key1=value1\nkey2=value2"
    }
  })
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CI/CD CONFIGURATIONS
# -----------------------------------------------------------------------------

# Case 11: GitHub Actions workflow
resource "filemanager_yaml_file" "github_actions" {
  path = "${local.output_dir}/ci/github-actions.yaml"
  content = yamlencode({
    name = "CI/CD Pipeline"
    on = {
      push = {
        branches = ["main", "develop"]
      }
      pull_request = {
        branches = ["main"]
      }
    }
    jobs = {
      build = {
        runs-on = "ubuntu-latest"
        steps = [
          {
            uses = "actions/checkout@v4"
          },
          {
            name = "Setup Node"
            uses = "actions/setup-node@v4"
            with = {
              node-version = "20"
            }
          },
          {
            name = "Install dependencies"
            run  = "npm ci"
          },
          {
            name = "Run tests"
            run  = "npm test"
          },
          {
            name = "Build"
            run  = "npm run build"
          }
        ]
      }
      deploy = {
        needs   = "build"
        runs-on = "ubuntu-latest"
        if      = "github.ref == 'refs/heads/main'"
        steps = [
          {
            name = "Deploy to production"
            run  = "echo 'Deploying...'"
          }
        ]
      }
    }
  })
  create_parent_dirs = true
}

# Case 12: GitLab CI
resource "filemanager_yaml_file" "gitlab_ci" {
  path = "${local.output_dir}/ci/gitlab-ci.yaml"
  content = yamlencode({
    stages = ["build", "test", "deploy"]
    variables = {
      NODE_VERSION = "20"
    }
    build = {
      stage  = "build"
      image  = "node:20"
      script = ["npm ci", "npm run build"]
      artifacts = {
        paths = ["dist/"]
      }
    }
    test = {
      stage  = "test"
      image  = "node:20"
      script = ["npm test"]
    }
    deploy = {
      stage  = "deploy"
      script = ["./deploy.sh"]
      only   = ["main"]
    }
  })
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# APPLICATION CONFIGS
# -----------------------------------------------------------------------------

# Case 13: Prometheus config
resource "filemanager_yaml_file" "prometheus" {
  path = "${local.output_dir}/configs/prometheus.yaml"
  content = yamlencode({
    global = {
      scrape_interval     = "15s"
      evaluation_interval = "15s"
    }
    alerting = {
      alertmanagers = [{
        static_configs = [{
          targets = ["localhost:9093"]
        }]
      }]
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
      },
      {
        job_name     = "app"
        metrics_path = "/metrics"
        static_configs = [{
          targets = ["app:8080"]
        }]
      }
    ]
  })
  create_parent_dirs = true
}

# Case 14: Ansible playbook
resource "filemanager_yaml_file" "ansible" {
  path = "${local.output_dir}/configs/playbook.yaml"
  content = yamlencode([
    {
      name   = "Configure web servers"
      hosts  = "webservers"
      become = true
      vars = {
        http_port   = 80
        max_clients = 200
      }
      tasks = [
        {
          name = "Install nginx"
          apt = {
            name  = "nginx"
            state = "present"
          }
        },
        {
          name = "Start nginx"
          service = {
            name    = "nginx"
            state   = "started"
            enabled = true
          }
        }
      ]
    }
  ])
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 15: Multiline strings
resource "filemanager_yaml_file" "multiline" {
  path = "${local.output_dir}/edge/multiline.yaml"
  content = yamlencode({
    description = "This is a\nmultiline\nstring"
    script      = "#!/bin/bash\necho 'hello'\nexit 0"
  })
  create_parent_dirs = true
}

# Case 16: Special characters
resource "filemanager_yaml_file" "special_chars" {
  path = "${local.output_dir}/edge/special_chars.yaml"
  content = yamlencode({
    colon_value   = "key: value"
    hash_value    = "# not a comment"
    bracket_value = "[not an array]"
    unicode       = "日本語 中文 한국어"
    emoji         = "🎉 🚀 💻"
  })
  create_parent_dirs = true
}

# Case 17: Empty structures
resource "filemanager_yaml_file" "empty" {
  path = "${local.output_dir}/edge/empty.yaml"
  content = yamlencode({
    empty_object = {}
    empty_array  = []
    null_value   = null
  })
  create_parent_dirs = true
}
