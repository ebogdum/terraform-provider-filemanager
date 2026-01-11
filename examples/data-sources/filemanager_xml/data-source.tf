# Read and parse XML file
data "filemanager_xml" "config" {
  path = "/etc/app/config.xml"
}

# Access parsed data
output "database_host" {
  value = data.filemanager_xml.config.data.configuration.database.host
}

output "app_name" {
  value = data.filemanager_xml.config.data.configuration.application.name
}

# Read XML from remote server
data "filemanager_xml" "remote" {
  path    = "/etc/app/config.xml"
  service = filemanager_ssh_service.server.name
}

# Read Maven pom.xml
data "filemanager_xml" "pom" {
  path = "${var.project_path}/pom.xml"
}

output "artifact_id" {
  value = data.filemanager_xml.pom.data.project.artifactId
}

output "version" {
  value = data.filemanager_xml.pom.data.project.version
}

# Read web.xml
data "filemanager_xml" "webapp" {
  path = "${var.project_path}/src/main/webapp/WEB-INF/web.xml"
}
