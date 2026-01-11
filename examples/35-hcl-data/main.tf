terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read an HCL configuration file
data "filemanager_hcl" "config" {
  path = "${path.module}/test.hcl"
}
