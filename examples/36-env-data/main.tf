terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read an ENV configuration file
data "filemanager_env" "config" {
  path = "${path.module}/test.env"
}
