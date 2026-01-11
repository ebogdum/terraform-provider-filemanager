# Configure FTP service
resource "filemanager_ftp_service" "main" {
  name     = "ftp-main"
  host     = var.ftp_host
  port     = 21
  user     = var.ftp_user
  password = var.ftp_password
}

# FTP service with passive mode
resource "filemanager_ftp_service" "passive" {
  name         = "ftp-passive"
  host         = var.ftp_host
  port         = 21
  user         = var.ftp_user
  password     = var.ftp_password
  passive_mode = true
}

# FTPS (FTP over TLS)
resource "filemanager_ftp_service" "ftps" {
  name         = "ftps-secure"
  host         = var.ftps_host
  port         = 990
  user         = var.ftp_user
  password     = var.ftp_password
  use_tls      = true
  implicit_tls = true
}

# SFTP via FTP service
resource "filemanager_ftp_service" "sftp" {
  name        = "sftp-server"
  host        = var.sftp_host
  port        = 22
  user        = var.sftp_user
  private_key = var.sftp_private_key
  protocol    = "sftp"
}
