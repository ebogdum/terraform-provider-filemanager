# Configure FTP service
resource "filemanager_ftp_service" "main" {
  host     = var.ftp_host
  port     = 21
  username = var.ftp_user
  password = var.ftp_password
}

# FTP service with passive mode (default)
resource "filemanager_ftp_service" "passive" {
  host         = var.ftp_host
  port         = 21
  username     = var.ftp_user
  password     = var.ftp_password
  passive_mode = true
}

# FTPS with explicit TLS (AUTH TLS)
resource "filemanager_ftp_service" "ftps_explicit" {
  host         = var.ftps_host
  port         = 21
  username     = var.ftp_user
  password     = var.ftp_password
  tls_enabled  = true
  explicit_tls = true
}

# FTPS with implicit TLS (port 990)
resource "filemanager_ftp_service" "ftps_implicit" {
  host         = var.ftps_host
  port         = 990
  username     = var.ftp_user
  password     = var.ftp_password
  tls_enabled  = true
  explicit_tls = false
}
