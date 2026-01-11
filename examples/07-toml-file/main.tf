# =============================================================================
# TOML FILE RESOURCE - ALL USE CASES
# =============================================================================

terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = ">= 1.0.0"
    }
  }
}

provider "filemanager" {}

locals {
  output_dir = "${path.module}/../../test/output/07-toml-file"
}

# -----------------------------------------------------------------------------
# BASIC TOML FILES
# -----------------------------------------------------------------------------

# Case 1: Simple key-value
resource "filemanager_toml_file" "simple" {
  path = "${local.output_dir}/basic/simple.toml"
  content = {
    title = "My Config"
    count = "42"
  }
  create_parent_dirs = true
}

# Case 2: Dotted keys (nested structure)
resource "filemanager_toml_file" "dotted" {
  path = "${local.output_dir}/basic/dotted.toml"
  content = {
    "server.host"   = "localhost"
    "server.port"   = "8080"
    "database.host" = "db.example.com"
    "database.port" = "5432"
    "database.name" = "mydb"
  }
  create_parent_dirs = true
}

# Case 3: With sort_keys
resource "filemanager_toml_file" "sorted" {
  path = "${local.output_dir}/basic/sorted.toml"
  content = {
    "z_last"   = "last"
    "a_first"  = "first"
    "m_middle" = "middle"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CARGO.TOML EXAMPLES
# -----------------------------------------------------------------------------

# Case 4: Simple Cargo.toml
resource "filemanager_toml_file" "cargo_simple" {
  path = "${local.output_dir}/rust/simple/Cargo.toml"
  content = {
    "package.name"    = "my-app"
    "package.version" = "0.1.0"
    "package.edition" = "2021"
  }
  create_parent_dirs = true
}

# Case 5: Cargo.toml with dependencies
resource "filemanager_toml_file" "cargo_deps" {
  path = "${local.output_dir}/rust/with-deps/Cargo.toml"
  content = {
    "package.name"               = "my-web-app"
    "package.version"            = "1.0.0"
    "package.edition"            = "2021"
    "package.authors"            = "Developer <dev@example.com>"
    "package.description"        = "A sample web application"
    "dependencies.tokio"         = "1.0"
    "dependencies.axum"          = "0.7"
    "dependencies.serde"         = "1.0"
    "dependencies.tracing"       = "0.1"
    "dev-dependencies.criterion" = "0.5"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 6: Cargo.toml workspace
resource "filemanager_toml_file" "cargo_workspace" {
  path = "${local.output_dir}/rust/workspace/Cargo.toml"
  content = {
    "workspace.members"         = "crate1, crate2, crate3"
    "workspace.resolver"        = "2"
    "workspace.package.version" = "1.0.0"
    "workspace.package.edition" = "2021"
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# PYPROJECT.TOML EXAMPLES
# -----------------------------------------------------------------------------

# Case 7: Simple pyproject.toml
resource "filemanager_toml_file" "pyproject_simple" {
  path = "${local.output_dir}/python/simple/pyproject.toml"
  content = {
    "project.name"               = "my-python-app"
    "project.version"            = "0.1.0"
    "project.requires-python"    = ">=3.9"
    "build-system.requires"      = "setuptools>=61.0"
    "build-system.build-backend" = "setuptools.build_meta"
  }
  create_parent_dirs = true
}

# Case 8: Poetry pyproject.toml
resource "filemanager_toml_file" "pyproject_poetry" {
  path = "${local.output_dir}/python/poetry/pyproject.toml"
  content = {
    "tool.poetry.name"                          = "my-poetry-app"
    "tool.poetry.version"                       = "0.1.0"
    "tool.poetry.description"                   = "A Python project using Poetry"
    "tool.poetry.authors"                       = "Dev <dev@example.com>"
    "tool.poetry.dependencies.python"           = "^3.11"
    "tool.poetry.dependencies.requests"         = "^2.31"
    "tool.poetry.group.dev.dependencies.pytest" = "^7.4"
    "build-system.requires"                     = "poetry-core"
    "build-system.build-backend"                = "poetry.core.masonry.api"
  }
  create_parent_dirs = true
}

# Case 9: Black/Ruff configuration
resource "filemanager_toml_file" "python_tools" {
  path = "${local.output_dir}/python/tools/pyproject.toml"
  content = {
    "tool.black.line-length"            = "88"
    "tool.black.target-version"         = "py311"
    "tool.ruff.line-length"             = "88"
    "tool.ruff.select"                  = "E, F, W"
    "tool.pytest.ini_options.testpaths" = "tests"
    "tool.pytest.ini_options.addopts"   = "-v --tb=short"
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CONFIGURATION FILES
# -----------------------------------------------------------------------------

# Case 10: Application config
resource "filemanager_toml_file" "app_config" {
  path = "${local.output_dir}/configs/app.toml"
  content = {
    "app.name"           = "MyApp"
    "app.version"        = "1.0.0"
    "app.debug"          = "false"
    "server.host"        = "0.0.0.0"
    "server.port"        = "8080"
    "server.workers"     = "4"
    "database.url"       = "postgres://localhost/mydb"
    "database.pool_size" = "10"
    "logging.level"      = "info"
    "logging.format"     = "json"
  }
  sort_keys          = true
  create_parent_dirs = true
}

# Case 11: Hugo config
resource "filemanager_toml_file" "hugo_config" {
  path = "${local.output_dir}/configs/hugo.toml"
  content = {
    "baseURL"            = "https://example.com/"
    "languageCode"       = "en-us"
    "title"              = "My Hugo Site"
    "theme"              = "ananke"
    "params.author"      = "Developer"
    "params.description" = "A Hugo site"
    "menu.main.name"     = "Home"
    "menu.main.weight"   = "1"
  }
  create_parent_dirs = true
}

# Case 12: Starship prompt config
resource "filemanager_toml_file" "starship" {
  path = "${local.output_dir}/configs/starship.toml"
  content = {
    "add_newline"              = "true"
    "character.success_symbol" = "[➜](bold green)"
    "character.error_symbol"   = "[➜](bold red)"
    "git_branch.symbol"        = "🌱 "
    "golang.symbol"            = "🐹 "
    "rust.symbol"              = "🦀 "
    "python.symbol"            = "🐍 "
  }
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 13: Special characters in values
resource "filemanager_toml_file" "special" {
  path = "${local.output_dir}/edge/special.toml"
  content = {
    "path"    = "/path/to/file"
    "url"     = "https://example.com/api?key=value"
    "regex"   = "^[a-z]+$"
    "unicode" = "日本語"
  }
  create_parent_dirs = true
}

# Case 14: Empty values
resource "filemanager_toml_file" "empty_values" {
  path = "${local.output_dir}/edge/empty.toml"
  content = {
    "empty_string" = ""
    "has_value"    = "value"
  }
  create_parent_dirs = true
}

# Case 15: Boolean and numeric strings
resource "filemanager_toml_file" "types" {
  path = "${local.output_dir}/edge/types.toml"
  content = {
    "bool_true"  = "true"
    "bool_false" = "false"
    "int"        = "42"
    "float"      = "3.14"
    "negative"   = "-100"
  }
  create_parent_dirs = true
}
