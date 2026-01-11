# Validate JSON file
data "filemanager_validate" "json_config" {
  path   = "/etc/app/config.json"
  format = "json"
}

output "json_valid" {
  value = data.filemanager_validate.json_config.valid
}

# Validate YAML file
data "filemanager_validate" "yaml_config" {
  path   = "/etc/app/config.yaml"
  format = "yaml"
}

# Validate with schema
data "filemanager_validate" "config_schema" {
  path        = "/etc/app/config.json"
  format      = "json"
  schema_path = "${path.module}/schemas/config.schema.json"
}

# Use in precondition
resource "filemanager_file" "processed" {
  path    = "/etc/app/processed.json"
  content = data.filemanager_file.config.content

  lifecycle {
    precondition {
      condition     = data.filemanager_validate.json_config.valid
      error_message = "Source config file is not valid JSON"
    }
  }
}
