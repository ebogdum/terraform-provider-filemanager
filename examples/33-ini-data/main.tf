terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read an INI configuration file
data "filemanager_ini" "config" {
  path = "${path.module}/test.ini"
}
