# Create a YAML configuration file
resource "filemanager_yaml_file" "config" {
  path = "/etc/app/config.yaml"
  data = {
    app_name    = "MyApplication"
    environment = var.environment
    database = {
      host     = var.db_host
      port     = 5432
      pool_size = 10
    }
    logging = {
      level   = "info"
      format  = "json"
      outputs = ["stdout", "file"]
    }
  }

  create_parent_dirs = true
}

# Merge with existing YAML file
resource "filemanager_yaml_file" "kubernetes_config" {
  path = "/etc/kubernetes/config.yaml"
  data = {
    spec = {
      replicas = var.replicas
    }
  }

  merge      = true
  deep_merge = true
}

# YAML file with multi-line strings
resource "filemanager_yaml_file" "playbook" {
  path = "/etc/ansible/playbook.yaml"
  data = {
    hosts = "all"
    tasks = [
      {
        name   = "Install packages"
        apt    = { name = ["nginx", "python3"], state = "present" }
      }
    ]
  }

  create_parent_dirs = true
}
