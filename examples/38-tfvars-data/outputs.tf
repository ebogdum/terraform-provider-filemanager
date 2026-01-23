output "hcl_data" {
  value = data.filemanager_tfvars.hcl_config.data
}

output "hcl_var_count" {
  value = data.filemanager_tfvars.hcl_config.var_count
}

output "hcl_var_names" {
  value = data.filemanager_tfvars.hcl_config.var_names
}

output "queried_region" {
  value = data.filemanager_tfvars.query.query_result
}

output "json_data" {
  value = data.filemanager_tfvars.json_config.data
}

output "json_tags" {
  value = data.filemanager_tfvars.json_query.query_result
}
