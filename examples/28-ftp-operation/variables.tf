# =============================================================================
# FTP OPERATION - VARIABLES
# =============================================================================

variable "ftp_host" {
  description = "FTP server hostname or IP address"
  type        = string
}

variable "ftp_port" {
  description = "FTP server port"
  type        = number
  default     = 21
}

variable "ftp_username" {
  description = "FTP username"
  type        = string
}

variable "ftp_password" {
  description = "FTP password"
  type        = string
  sensitive   = true
}

variable "ftp_tls" {
  description = "Enable FTPS (FTP over TLS)"
  type        = bool
  default     = false
}
