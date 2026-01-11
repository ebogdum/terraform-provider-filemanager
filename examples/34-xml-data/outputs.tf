output "config_version" {
  description = "Config version from XML attribute"
  value       = data.filemanager_xml.config.data.config["@version"]
}

output "database_host" {
  description = "Database host from XML config"
  value       = data.filemanager_xml.config.data.config.database.host
}

output "database_port" {
  description = "Database port from XML config"
  value       = data.filemanager_xml.config.data.config.database.port
}

output "file_size" {
  description = "Size of the XML file"
  value       = data.filemanager_xml.config.size
}
