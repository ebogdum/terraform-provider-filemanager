terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read a JSON configuration file
data "filemanager_json" "config" {
  path = "${path.module}/test.json"
}
