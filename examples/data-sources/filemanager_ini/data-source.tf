# Read and parse INI file
data "filemanager_ini" "config" {
  path = "/etc/app/config.ini"
}

# Access parsed data by section
output "database_host" {
  value = data.filemanager_ini.config.data.database.host
}

output "logging_level" {
  value = data.filemanager_ini.config.data.logging.level
}

# Read INI from remote server
data "filemanager_ini" "remote" {
  path    = "/etc/app/config.ini"
  service = filemanager_ssh_service.server.name
}

# Read PHP configuration
data "filemanager_ini" "php" {
  path = "/etc/php/8.2/fpm/php.ini"
}

output "memory_limit" {
  value = data.filemanager_ini.php.data.PHP.memory_limit
}

# Read MySQL configuration
data "filemanager_ini" "mysql" {
  path = "/etc/mysql/mysql.conf.d/mysqld.cnf"
}
