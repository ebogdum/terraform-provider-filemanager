# INI FILE - OUTPUTS

output "basic" {
  value = {
    simple = { path = filemanager_ini_file.simple.path, rendered = filemanager_ini_file.simple.rendered }
    multi  = { path = filemanager_ini_file.multi_section.path }
    sorted = { path = filemanager_ini_file.sorted.path }
  }
}

output "php" {
  value = {
    development = { path = filemanager_ini_file.php_dev.path, rendered = filemanager_ini_file.php_dev.rendered }
    production  = { path = filemanager_ini_file.php_prod.path }
  }
}

output "configs" {
  value = {
    mysql     = filemanager_ini_file.mysql.path
    gitconfig = filemanager_ini_file.gitconfig.path
    desktop   = filemanager_ini_file.desktop_entry.path
    aws       = filemanager_ini_file.aws_config.path
    systemd   = filemanager_ini_file.systemd_service.path
  }
}

output "summary" {
  value = { total = 14, categories = ["basic", "php", "mysql", "git", "desktop", "aws", "systemd", "edge_cases"] }
}
