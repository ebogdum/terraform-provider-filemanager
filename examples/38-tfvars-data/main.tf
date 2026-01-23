# =============================================================================
# TFVARS DATA SOURCE - ALL USE CASES
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

# -----------------------------------------------------------------------------
# READ HCL FORMAT
# -----------------------------------------------------------------------------

# Case 1: Read a .tfvars file (HCL format)
data "filemanager_tfvars" "hcl_config" {
  path = "${path.module}/testdata/example.tfvars"
}

# Case 2: Query a specific variable
data "filemanager_tfvars" "query" {
  path  = "${path.module}/testdata/example.tfvars"
  query = "region"
}

# -----------------------------------------------------------------------------
# READ JSON FORMAT
# -----------------------------------------------------------------------------

# Case 3: Read a .tfvars.json file
data "filemanager_tfvars" "json_config" {
  path = "${path.module}/testdata/example.tfvars.json"
}

# Case 4: Query from JSON format
data "filemanager_tfvars" "json_query" {
  path  = "${path.module}/testdata/example.tfvars.json"
  query = "tags"
}
