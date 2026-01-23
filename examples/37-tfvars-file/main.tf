# =============================================================================
# TFVARS FILE RESOURCE - ALL USE CASES
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.2.0"
    }
  }
}

provider "filemanager" {}

locals {
  output_dir = "${path.module}/../../test/output/37-tfvars-file"
}

# -----------------------------------------------------------------------------
# BASIC TFVARS FILES
# -----------------------------------------------------------------------------

# Case 1: Simple tfvars with native types
resource "filemanager_tfvars_file" "basic" {
  path = "${local.output_dir}/basic/dev.tfvars"

  vars = {
    project_name      = "myapp"
    environment       = "development"
    region            = "us-east-1"
    instance_type     = "t3.micro"
    min_count         = 1
    max_count         = 3
    enable_monitoring = false
  }

  sort_keys          = true
  create_parent_dirs = true
}

# Case 2: Complex types (maps, lists)
resource "filemanager_tfvars_file" "complex" {
  path = "${local.output_dir}/basic/complex.tfvars"

  vars = {
    project = "myapp"
    tags = {
      team        = "platform"
      cost_center = "engineering"
      managed_by  = "terraform"
    }
    availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]
    ports              = [80, 443, 8080]
    database = {
      host     = "db.example.com"
      port     = 5432
      name     = "myapp_prod"
      ssl_mode = "require"
    }
  }

  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# INTERPOLATION
# -----------------------------------------------------------------------------

# Case 3: Internal variable interpolation
resource "filemanager_tfvars_file" "interpolation" {
  path = "${local.output_dir}/interpolation/self-ref.tfvars"

  vars = {
    project     = "myapp"
    environment = "production"
    bucket_name = "{{ .project }}-{{ .environment }}-assets"
    log_group   = "/aws/{{ .project }}/{{ .environment }}"
    domain      = "{{ .project }}.example.com"
  }

  sort_keys          = true
  create_parent_dirs = true
}

# Case 4: Interpolation with template functions
resource "filemanager_tfvars_file" "template_funcs" {
  path = "${local.output_dir}/interpolation/functions.tfvars"

  vars = {
    project     = "MyApp"
    environment = "  production  "
    bucket_name = "{{ lower .project }}-{{ trim .environment }}-data"
    upper_name  = "{{ upper .project }}"
  }

  sort_keys          = true
  create_parent_dirs = true
}

# Case 5: Interpolation with template_vars
resource "filemanager_tfvars_file" "template_vars" {
  path = "${local.output_dir}/interpolation/template-vars.tfvars"

  vars = {
    api_url = "https://{{ .domain }}/api/v1"
    cdn_url = "https://cdn.{{ .domain }}"
  }

  template_vars = {
    domain = "example.com"
  }

  create_parent_dirs = true
}

# Case 6: Deep interpolation (nested maps/lists)
resource "filemanager_tfvars_file" "deep_interpolation" {
  path = "${local.output_dir}/interpolation/deep.tfvars"

  vars = {
    project = "myapp"
    tags = {
      name = "{{ .project }}-service"
      env  = "production"
    }
    endpoints = ["{{ .project }}.example.com", "api.{{ .project }}.example.com"]
  }

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# JSON FORMAT
# -----------------------------------------------------------------------------

# Case 7: JSON format output
resource "filemanager_tfvars_file" "json_format" {
  path = "${local.output_dir}/json/config.tfvars.json"

  vars = {
    region        = "us-west-2"
    instance_type = "t3.large"
    min_count     = 2
    max_count     = 10
    enable_ha     = true
  }

  json_format        = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# MERGE WITH EXISTING
# -----------------------------------------------------------------------------

# Case 8: Create a base file first, then merge with it
resource "filemanager_tfvars_file" "merge_base" {
  path = "${local.output_dir}/merge/existing.tfvars"

  vars = {
    existing_var  = "original_value"
    keep_this     = "preserved"
    override_this = "old_value"
  }

  create_parent_dirs = true
}

# Case 9: Merge with existing (replace strategy)
resource "filemanager_tfvars_file" "merge_replace" {
  path = "${local.output_dir}/merge/merged-replace.tfvars"

  vars = {
    new_setting     = "added"
    override_this   = "new_value"
  }

  merge_with_existing = true
  merge_strategy      = "replace"
  create_parent_dirs  = true

  depends_on = [filemanager_tfvars_file.merge_base]
}

# -----------------------------------------------------------------------------
# DELETE VARS
# -----------------------------------------------------------------------------

# Case 10: Delete specific variables
resource "filemanager_tfvars_file" "delete_vars" {
  path = "${local.output_dir}/delete/cleaned.tfvars"

  vars = {
    keep_var_1    = "value1"
    keep_var_2    = "value2"
    deprecated    = "will be removed"
    old_setting   = "will be removed"
  }

  delete_vars = ["deprecated", "old_setting"]

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CUSTOM DELIMITERS
# -----------------------------------------------------------------------------

# Case 11: Custom template delimiters
resource "filemanager_tfvars_file" "custom_delims" {
  path = "${local.output_dir}/delimiters/custom.tfvars"

  vars = {
    project     = "myapp"
    bucket_name = "<< .project >>-assets"
  }

  left_delim  = "<<"
  right_delim = ">>"

  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# ENVIRONMENT-SPECIFIC CONFIGS
# -----------------------------------------------------------------------------

# Case 12: Production environment
resource "filemanager_tfvars_file" "prod" {
  path = "${local.output_dir}/environments/prod.tfvars"

  vars = {
    environment    = "production"
    instance_type  = "m5.xlarge"
    instance_count = 5
    multi_az       = true
    backup_enabled = true
    backup_retention_days = 30
    monitoring = {
      enabled       = true
      alarm_email   = "ops@example.com"
      log_retention = 90
    }
    allowed_cidrs = ["10.0.0.0/8", "172.16.0.0/12"]
  }

  sort_keys          = true
  create_parent_dirs = true
}

# Case 13: Staging environment
resource "filemanager_tfvars_file" "staging" {
  path = "${local.output_dir}/environments/staging.tfvars"

  vars = {
    environment    = "staging"
    instance_type  = "t3.medium"
    instance_count = 2
    multi_az       = false
    backup_enabled = true
    backup_retention_days = 7
    monitoring = {
      enabled       = true
      alarm_email   = "dev@example.com"
      log_retention = 30
    }
    allowed_cidrs = ["10.0.0.0/8"]
  }

  sort_keys          = true
  create_parent_dirs = true
}
