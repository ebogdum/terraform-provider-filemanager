# Create an HCL configuration file
resource "filemanager_hcl_file" "config" {
  path = "/etc/app/config.hcl"
  data = {
    server = {
      host = var.server_host
      port = 8080
    }
    database = {
      connection_string = var.db_connection_string
      max_connections   = 100
    }
    features = {
      enable_caching = true
      cache_ttl      = 3600
    }
  }

  create_parent_dirs = true
}

# Create Consul configuration
resource "filemanager_hcl_file" "consul" {
  path = "/etc/consul.d/config.hcl"
  data = {
    datacenter = var.datacenter
    data_dir   = "/opt/consul/data"
    log_level  = "INFO"
    server     = true
    bootstrap_expect = 3
    ui_config = {
      enabled = true
    }
    connect = {
      enabled = true
    }
  }

  create_parent_dirs = true
}

# Create Vault configuration
resource "filemanager_hcl_file" "vault" {
  path = "/etc/vault.d/config.hcl"
  data = {
    storage = {
      consul = {
        address = "127.0.0.1:8500"
        path    = "vault/"
      }
    }
    listener = {
      tcp = {
        address     = "0.0.0.0:8200"
        tls_disable = false
      }
    }
  }

  create_parent_dirs = true
}
