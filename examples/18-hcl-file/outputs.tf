# =============================================================================
# HCL FILE RESOURCE - OUTPUTS
# =============================================================================

output "basic_hcl_files" {
  description = "Basic HCL file tests"
  value = {
    simple = {
      path     = filemanager_hcl_file.simple.path
      md5      = filemanager_hcl_file.simple.md5
      rendered = filemanager_hcl_file.simple.rendered
    }
    nested = {
      path = filemanager_hcl_file.nested.path
      md5  = filemanager_hcl_file.nested.md5
    }
  }
}

output "terraform_configs" {
  description = "Terraform-like configuration tests"
  value = {
    backend = {
      path = filemanager_hcl_file.terraform_backend.path
      md5  = filemanager_hcl_file.terraform_backend.md5
    }
    vars = {
      path = filemanager_hcl_file.terraform_vars.path
      md5  = filemanager_hcl_file.terraform_vars.md5
    }
  }
}

output "hashicorp_configs" {
  description = "HashiCorp tool configuration tests"
  value = {
    consul = {
      path = filemanager_hcl_file.consul.path
      md5  = filemanager_hcl_file.consul.md5
    }
    vault = {
      path = filemanager_hcl_file.vault.path
      md5  = filemanager_hcl_file.vault.md5
    }
    nomad = {
      path = filemanager_hcl_file.nomad.path
      md5  = filemanager_hcl_file.nomad.md5
    }
  }
}

output "packer_configs" {
  description = "Packer configuration tests"
  value = {
    packer = {
      path = filemanager_hcl_file.packer.path
      md5  = filemanager_hcl_file.packer.md5
    }
  }
}

output "edge_cases" {
  description = "Edge case tests"
  value = {
    empty = {
      path = filemanager_hcl_file.empty.path
      md5  = filemanager_hcl_file.empty.md5
    }
    complex = {
      path = filemanager_hcl_file.complex.path
      md5  = filemanager_hcl_file.complex.md5
    }
  }
}

output "summary" {
  description = "Test summary"
  value = {
    total_hcl_files = 10
    categories = [
      "basic_structures",
      "terraform_configs",
      "hashicorp_tools",
      "packer_configs",
      "edge_cases"
    ]
  }
}
