output "basic_rendered" {
  value     = filemanager_tfvars_file.basic.rendered
  sensitive = true
}

output "basic_var_count" {
  value = filemanager_tfvars_file.basic.var_count
}

output "interpolation_rendered" {
  value     = filemanager_tfvars_file.interpolation.rendered
  sensitive = true
}

output "complex_md5" {
  value = filemanager_tfvars_file.complex.md5
}

output "prod_size" {
  value = filemanager_tfvars_file.prod.size
}

output "prod_absolute_path" {
  value = filemanager_tfvars_file.prod.absolute_path
}
