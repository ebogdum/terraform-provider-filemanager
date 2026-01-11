# ENV FILE - OUTPUTS

output "basic" {
  value = {
    simple = { path = filemanager_env_file.simple.path, rendered = filemanager_env_file.simple.rendered }
    sorted = { path = filemanager_env_file.sorted.path }
    full   = { path = filemanager_env_file.many_vars.path }
  }
}

output "environments" {
  value = {
    development = filemanager_env_file.development.path
    production  = filemanager_env_file.production.path
    testing     = filemanager_env_file.testing.path
    staging     = filemanager_env_file.staging.path
  }
}

output "database" {
  value = {
    postgres = filemanager_env_file.postgres.path
    mysql    = filemanager_env_file.mysql.path
    redis    = filemanager_env_file.redis.path
  }
}

output "services" {
  value = {
    services = filemanager_env_file.services.path
    oauth    = filemanager_env_file.oauth.path
    docker   = filemanager_env_file.docker.path
  }
}

output "frameworks" {
  value = {
    nextjs  = filemanager_env_file.nextjs.path
    laravel = filemanager_env_file.laravel.path
  }
}

output "summary" {
  value = { total = 19, categories = ["basic", "environments", "database", "services", "frameworks", "edge_cases"] }
}
