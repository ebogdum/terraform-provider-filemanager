terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read a TOML configuration file
data "filemanager_toml" "config" {
  path = "${path.module}/test.toml"
}
