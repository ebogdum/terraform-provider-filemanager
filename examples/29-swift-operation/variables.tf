# =============================================================================
# SWIFT OPERATION - VARIABLES
# =============================================================================

variable "swift_auth_url" {
  description = "OpenStack Keystone authentication URL"
  type        = string
}

variable "swift_container" {
  description = "Swift container name"
  type        = string
}

variable "swift_username" {
  description = "OpenStack username"
  type        = string
}

variable "swift_password" {
  description = "OpenStack password"
  type        = string
  sensitive   = true
}

variable "swift_tenant" {
  description = "OpenStack tenant/project name"
  type        = string
}

variable "swift_region" {
  description = "OpenStack region"
  type        = string
  default     = ""
}
