# =============================================================================
# HCL FILE RESOURCE - ALL USE CASES
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
  output_dir = "${path.module}/../../test/output/18-hcl-file"
}

# -----------------------------------------------------------------------------
# BASIC HCL FILES
# -----------------------------------------------------------------------------

# Case 1: Simple HCL config
resource "filemanager_hcl_file" "simple" {
  path = "${local.output_dir}/basic/simple.hcl"

  content = jsonencode({
    name    = "example"
    version = "1.0"
    enabled = true
  })

  create_parent_dirs = true
}

# Case 2: Nested blocks
resource "filemanager_hcl_file" "nested" {
  path = "${local.output_dir}/basic/nested.hcl"

  content = jsonencode({
    server = {
      host = "localhost"
      port = 8080
      tls = {
        enabled = true
        cert    = "/path/to/cert"
        key     = "/path/to/key"
      }
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# TERRAFORM-LIKE CONFIGS
# -----------------------------------------------------------------------------

# Case 3: Terraform backend config
resource "filemanager_hcl_file" "terraform_backend" {
  path = "${local.output_dir}/terraform/backend.hcl"

  content = jsonencode({
    terraform = {
      backend = {
        s3 = {
          bucket         = "my-terraform-state"
          key            = "state/terraform.tfstate"
          region         = "us-east-1"
          encrypt        = true
          dynamodb_table = "terraform-locks"
        }
      }
    }
  })

  create_parent_dirs = true
}

# Case 4: Terraform variables
resource "filemanager_hcl_file" "terraform_vars" {
  path = "${local.output_dir}/terraform/terraform.tfvars.hcl"

  content = jsonencode({
    environment    = "production"
    instance_type  = "t3.medium"
    instance_count = 3
    tags = {
      Project = "MyApp"
      Team    = "Platform"
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CONSUL/VAULT/NOMAD CONFIGS
# -----------------------------------------------------------------------------

# Case 5: Consul config
resource "filemanager_hcl_file" "consul" {
  path = "${local.output_dir}/hashicorp/consul.hcl"

  content = jsonencode({
    datacenter       = "dc1"
    data_dir         = "/opt/consul/data"
    log_level        = "INFO"
    server           = true
    bootstrap_expect = 3

    bind_addr   = "0.0.0.0"
    client_addr = "0.0.0.0"

    ui_config = {
      enabled = true
    }

    connect = {
      enabled = true
    }
  })

  create_parent_dirs = true
}

# Case 6: Vault config
resource "filemanager_hcl_file" "vault" {
  path = "${local.output_dir}/hashicorp/vault.hcl"

  content = jsonencode({
    storage = {
      consul = {
        address = "127.0.0.1:8500"
        path    = "vault/"
      }
    }

    listener = {
      tcp = {
        address       = "0.0.0.0:8200"
        tls_disable   = false
        tls_cert_file = "/etc/vault/certs/cert.pem"
        tls_key_file  = "/etc/vault/certs/key.pem"
      }
    }

    api_addr     = "https://vault.example.com:8200"
    cluster_addr = "https://vault.example.com:8201"

    ui = true
  })

  create_parent_dirs = true
}

# Case 7: Nomad config
resource "filemanager_hcl_file" "nomad" {
  path = "${local.output_dir}/hashicorp/nomad.hcl"

  content = jsonencode({
    datacenter = "dc1"
    data_dir   = "/opt/nomad/data"

    server = {
      enabled          = true
      bootstrap_expect = 3
    }

    client = {
      enabled = true
    }

    consul = {
      address = "127.0.0.1:8500"
    }
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PACKER CONFIG
# -----------------------------------------------------------------------------

# Case 8: Packer variables
resource "filemanager_hcl_file" "packer" {
  path = "${local.output_dir}/packer/variables.pkrvars.hcl"

  content = jsonencode({
    aws_region    = "us-west-2"
    instance_type = "t3.micro"
    ami_name      = "my-custom-ami"
    ssh_username  = "ubuntu"
  })

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 9: Empty config
resource "filemanager_hcl_file" "empty" {
  path = "${local.output_dir}/edge/empty.hcl"

  content = jsonencode({})

  create_parent_dirs = true
}

# Case 10: Arrays and complex types
resource "filemanager_hcl_file" "complex" {
  path = "${local.output_dir}/edge/complex.hcl"

  content = jsonencode({
    ports = [80, 443, 8080]
    hosts = ["web1.example.com", "web2.example.com"]

    database = {
      primary = {
        host = "db-primary.example.com"
        port = 5432
      }
      replicas = [
        {
          host = "db-replica1.example.com"
          port = 5432
        },
        {
          host = "db-replica2.example.com"
          port = 5432
        }
      ]
    }
  })

  create_parent_dirs = true
}

