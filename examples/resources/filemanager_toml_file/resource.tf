# Create a TOML configuration file
resource "filemanager_toml_file" "config" {
  path = "/etc/app/config.toml"
  data = {
    title = "Application Config"
    owner = {
      name  = "DevOps Team"
      email = "devops@example.com"
    }
    database = {
      server         = var.db_server
      ports          = [8001, 8002, 8003]
      connection_max = 5000
      enabled        = true
    }
  }

  create_parent_dirs = true
}

# Create Cargo.toml for Rust project
resource "filemanager_toml_file" "cargo" {
  path = "${var.project_path}/Cargo.toml"
  data = {
    package = {
      name    = var.package_name
      version = var.package_version
      edition = "2021"
    }
    dependencies = {
      serde = { version = "1.0", features = ["derive"] }
      tokio = { version = "1", features = ["full"] }
    }
  }

  create_parent_dirs = true
}
