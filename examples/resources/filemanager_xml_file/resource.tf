# Create an XML configuration file
resource "filemanager_xml_file" "config" {
  path      = "/etc/app/config.xml"
  root_name = "configuration"
  data = {
    application = {
      name    = "MyApplication"
      version = "1.0.0"
    }
    database = {
      host     = var.db_host
      port     = "5432"
      username = var.db_user
    }
    logging = {
      level = "INFO"
      file  = "/var/log/app.log"
    }
  }

  pretty_print       = true
  indent             = "  "
  create_parent_dirs = true
}

# Create Maven pom.xml
resource "filemanager_xml_file" "pom" {
  path      = "${var.project_path}/pom.xml"
  root_name = "project"
  data = {
    modelVersion = "4.0.0"
    groupId      = var.group_id
    artifactId   = var.artifact_id
    version      = var.project_version
    packaging    = "jar"
    dependencies = {
      dependency = [
        {
          groupId    = "org.springframework.boot"
          artifactId = "spring-boot-starter-web"
          version    = "3.0.0"
        }
      ]
    }
  }

  pretty_print       = true
  create_parent_dirs = true
}
