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

# Read local environment variables
data "filemanager_environment" "local" {}

# Read local environment variables filtered by pattern
data "filemanager_environment" "path_vars" {
  filter = "PATH*"
}
