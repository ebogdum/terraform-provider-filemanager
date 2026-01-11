# TOML FILE - OUTPUTS

output "basic" {
  value = {
    simple = { path = filemanager_toml_file.simple.path, rendered = filemanager_toml_file.simple.rendered }
    dotted = { path = filemanager_toml_file.dotted.path, rendered = filemanager_toml_file.dotted.rendered }
    sorted = { path = filemanager_toml_file.sorted.path, rendered = filemanager_toml_file.sorted.rendered }
  }
}

output "rust" {
  value = {
    cargo_simple    = filemanager_toml_file.cargo_simple.path
    cargo_deps      = filemanager_toml_file.cargo_deps.path
    cargo_workspace = filemanager_toml_file.cargo_workspace.path
  }
}

output "python" {
  value = {
    simple = filemanager_toml_file.pyproject_simple.path
    poetry = filemanager_toml_file.pyproject_poetry.path
    tools  = filemanager_toml_file.python_tools.path
  }
}

output "configs" {
  value = {
    app      = filemanager_toml_file.app_config.path
    hugo     = filemanager_toml_file.hugo_config.path
    starship = filemanager_toml_file.starship.path
  }
}

output "summary" {
  value = { total = 15, categories = ["basic", "rust", "python", "configs", "edge_cases"] }
}
