---
page_title: "Structured Files"
subcategory: "Guides"
description: |-
  Working with structured configuration files - JSON, YAML, TOML, HCL, XML, INI, and ENV formats.
---

# Working with Structured Files

The FileManager provider offers native support for structured configuration files, allowing you to create, read, modify, and merge data in various formats.

## Supported Formats

| Format | Resource | Data Source | Use Case |
|--------|----------|-------------|----------|
| JSON | `filemanager_json_file` | `filemanager_json` | API configs, package.json, tsconfig.json |
| YAML | `filemanager_yaml_file` | `filemanager_yaml` | Kubernetes, Docker Compose, CI/CD |
| TOML | `filemanager_toml_file` | `filemanager_toml` | Rust configs, pyproject.toml |
| HCL | `filemanager_hcl_file` | `filemanager_hcl` | Terraform, Consul, Vault |
| XML | `filemanager_xml_file` | `filemanager_xml` | Maven, Spring, legacy configs |
| INI | `filemanager_ini_file` | `filemanager_ini` | PHP, MySQL, git configs |
| ENV | `filemanager_env_file` | `filemanager_env` | Environment variables, .env files |

## Writing Structured Files

### JSON Files

```hcl
resource "filemanager_json_file" "app_config" {
  path = "/etc/app/config.json"

  data = {
    app = {
      name    = "myapp"
      version = "1.0.0"
      debug   = var.debug_mode
    }
    database = {
      host = var.db_host
      port = 5432
      name = "production"
    }
    features = ["auth", "logging", "metrics"]
  }

  # Optional: customize formatting
  indent = 2
}
```

### YAML Files

```hcl
resource "filemanager_yaml_file" "k8s_config" {
  path = "/etc/kubernetes/config.yaml"

  data = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name      = "app-config"
      namespace = var.namespace
    }
    data = {
      "config.json" = jsonencode(local.app_config)
    }
  }
}
```

### TOML Files

```hcl
resource "filemanager_toml_file" "cargo" {
  path = "/app/Cargo.toml"

  data = {
    package = {
      name    = "myapp"
      version = "0.1.0"
      edition = "2021"
    }
    dependencies = {
      serde = { version = "1.0", features = ["derive"] }
      tokio = { version = "1", features = ["full"] }
    }
  }
}
```

### INI Files

```hcl
resource "filemanager_ini_file" "mysql" {
  path = "/etc/mysql/my.cnf"

  sections = {
    mysqld = {
      bind-address    = "0.0.0.0"
      port            = "3306"
      max_connections = "500"
    }
    client = {
      port = "3306"
    }
  }
}
```

### ENV Files

```hcl
resource "filemanager_env_file" "dotenv" {
  path = "/app/.env"

  data = {
    DATABASE_URL = "postgresql://${var.db_user}:${var.db_pass}@${var.db_host}:5432/mydb"
    API_KEY      = var.api_key
    LOG_LEVEL    = var.debug ? "debug" : "info"
    NODE_ENV     = "production"
  }
}
```

## Reading Structured Files

### Reading JSON

```hcl
data "filemanager_json" "existing_config" {
  path = "/etc/app/config.json"
}

# Access values using dot notation
output "app_name" {
  value = data.filemanager_json.existing_config.data.app.name
}

# Access array elements
output "first_feature" {
  value = data.filemanager_json.existing_config.data.features[0]
}

# Access keys with special characters
output "api_key" {
  value     = data.filemanager_json.existing_config.data["api-key"]
  sensitive = true
}
```

### Reading YAML

```hcl
data "filemanager_yaml" "docker_compose" {
  path = "/app/docker-compose.yaml"
}

output "services" {
  value = keys(data.filemanager_yaml.docker_compose.data.services)
}
```

### Reading INI

```hcl
data "filemanager_ini" "config" {
  path = "/etc/app/config.ini"
}

# Access by section and key
output "db_host" {
  value = data.filemanager_ini.config.data.database.host
}
```

### Reading Remote Files

All data sources support the `service` parameter for reading from remote backends:

```hcl
resource "filemanager_ssh_service" "server" {
  name = "webserver"
  host = "server.example.com"
  user = "deploy"
  private_key = file("~/.ssh/id_rsa")
}

data "filemanager_json" "remote_config" {
  path    = "/etc/app/config.json"
  service = filemanager_ssh_service.server.name
}
```

## Deep Merge

The structured file resources support deep merging for complex configurations:

```hcl
locals {
  base_config = {
    app = {
      name = "myapp"
      logging = {
        level  = "info"
        format = "json"
      }
    }
  }

  environment_overrides = {
    app = {
      logging = {
        level = var.debug ? "debug" : "info"
      }
    }
    database = {
      host = var.db_host
    }
  }
}

resource "filemanager_json_file" "merged" {
  path = "/etc/app/config.json"

  # Start with base config
  data = local.base_config

  # Merge environment-specific settings
  merge_data = local.environment_overrides

  # Result: base_config with environment_overrides deeply merged
}
```

## Format Conversion

Convert between formats by reading one and writing another:

```hcl
# Read YAML
data "filemanager_yaml" "source" {
  path = "/etc/app/config.yaml"
}

# Write as JSON
resource "filemanager_json_file" "converted" {
  path = "/etc/app/config.json"
  data = data.filemanager_yaml.source.data
}
```

## Validation

The `filemanager_validate` data source validates file formats:

```hcl
data "filemanager_validate" "json_check" {
  path   = "/etc/app/config.json"
  format = "json"
}

output "is_valid_json" {
  value = data.filemanager_validate.json_check.valid
}

output "validation_error" {
  value = data.filemanager_validate.json_check.error
}
```

## Common Patterns

### Kubernetes ConfigMap from Local Files

```hcl
data "filemanager_json" "app_config" {
  path = "${path.module}/configs/app.json"
}

data "filemanager_yaml" "db_config" {
  path = "${path.module}/configs/database.yaml"
}

resource "filemanager_yaml_file" "configmap" {
  path = "${path.module}/output/configmap.yaml"

  data = {
    apiVersion = "v1"
    kind       = "ConfigMap"
    metadata = {
      name = "app-config"
    }
    data = {
      "app.json"      = jsonencode(data.filemanager_json.app_config.data)
      "database.yaml" = yamlencode(data.filemanager_yaml.db_config.data)
    }
  }
}
```

### Environment-Specific Configuration

```hcl
locals {
  environments = {
    dev = {
      debug    = true
      replicas = 1
    }
    staging = {
      debug    = true
      replicas = 2
    }
    production = {
      debug    = false
      replicas = 3
    }
  }
}

resource "filemanager_yaml_file" "config" {
  for_each = local.environments

  path = "/etc/app/config.${each.key}.yaml"

  data = {
    environment = each.key
    debug       = each.value.debug
    replicas    = each.value.replicas
  }
}
```

### Merging Multiple Config Sources

```hcl
# Base configuration
data "filemanager_yaml" "base" {
  path = "${path.module}/configs/base.yaml"
}

# Environment-specific overrides
data "filemanager_yaml" "env" {
  path = "${path.module}/configs/${var.environment}.yaml"
}

# Secret configuration
data "filemanager_yaml" "secrets" {
  path    = "/etc/secrets/config.yaml"
  service = filemanager_ssh_service.vault.name
}

resource "filemanager_yaml_file" "final" {
  path = "/etc/app/config.yaml"

  data       = data.filemanager_yaml.base.data
  merge_data = merge(
    data.filemanager_yaml.env.data,
    data.filemanager_yaml.secrets.data
  )
}
```

## Best Practices

### 1. Use Locals for Complex Data

```hcl
locals {
  app_config = {
    name    = var.app_name
    version = var.app_version
    settings = {
      timeout = 30
      retries = 3
    }
  }
}

resource "filemanager_json_file" "config" {
  path = "/etc/app/config.json"
  data = local.app_config
}
```

### 2. Validate Before Deploy

```hcl
data "filemanager_validate" "check" {
  path   = filemanager_json_file.config.path
  format = "json"

  depends_on = [filemanager_json_file.config]
}
```

### 3. Use Sensitive Values Properly

```hcl
resource "filemanager_env_file" "secrets" {
  path = "/app/.env"

  data = {
    API_KEY      = var.api_key      # Mark as sensitive in variable
    DATABASE_URL = var.database_url # Mark as sensitive in variable
  }

  file_permission = "0600"  # Restrict access
}
```

### 4. Checksum Verification

```hcl
resource "filemanager_json_file" "verified" {
  path            = "/etc/app/config.json"
  data            = local.config
  verify_checksum = true
}

output "config_sha256" {
  value = filemanager_json_file.verified.sha256
}
```

## Troubleshooting

### Invalid Format Errors

If you see parsing errors, check:
1. File encoding (should be UTF-8)
2. Syntax errors in the source data
3. Special characters that need escaping

### Type Conversion Issues

The `data` attribute is dynamic. Use explicit type conversions when needed:

```hcl
# Convert to number
output "port" {
  value = tonumber(data.filemanager_json.config.data.port)
}

# Convert to string
output "name" {
  value = tostring(data.filemanager_json.config.data.name)
}
```

### Merge Conflicts

When using `merge_data`, nested objects are merged, but:
- Arrays are replaced, not merged
- Null values remove keys from the result
