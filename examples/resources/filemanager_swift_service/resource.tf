# Configure OpenStack Swift service
resource "filemanager_swift_service" "main" {
  name       = "swift-main"
  auth_url   = var.swift_auth_url
  container  = var.swift_container
  username   = var.swift_username
  password   = var.swift_password
  tenant     = var.swift_tenant
  region     = var.swift_region
}

# Swift service with API key
resource "filemanager_swift_service" "api_key" {
  name       = "swift-apikey"
  auth_url   = var.swift_auth_url
  container  = var.swift_container
  username   = var.swift_username
  api_key    = var.swift_api_key
  tenant     = var.swift_tenant
}

# Swift service for object storage
resource "filemanager_swift_service" "objects" {
  name       = "swift-objects"
  auth_url   = var.swift_auth_url
  container  = "objects"
  username   = var.swift_username
  password   = var.swift_password
  tenant     = var.swift_tenant
  region     = var.swift_region
}
