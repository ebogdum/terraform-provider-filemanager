terraform {
  required_providers {
    filemanager = {
      source = "ebogdum/filemanager"
    }
  }
}

provider "filemanager" {}

# Read an XML configuration file
data "filemanager_xml" "config" {
  path = "${path.module}/test.xml"
}
