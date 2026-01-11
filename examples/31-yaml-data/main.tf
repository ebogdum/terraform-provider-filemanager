terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read a YAML configuration file
data "filemanager_yaml" "config" {
  path = "${path.module}/test.yaml"
}
