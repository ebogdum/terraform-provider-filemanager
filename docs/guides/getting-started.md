---
page_title: "Getting Started"
subcategory: "Guides"
description: |-
  Getting started with the FileManager provider - installation, configuration, and your first resource.
---

# Getting Started with FileManager

This guide walks you through installing the FileManager provider and creating your first managed file.

## Prerequisites

- Terraform 1.0 or later
- Access to a filesystem (local or remote via SSH)

## Installation

The FileManager provider is available from the Terraform Registry. Add it to your `required_providers` block:

```hcl
terraform {
  required_providers {
    filemanager = {
      source  = "ebogdum/filemanager"
      version = "~> 1.0"
    }
  }
}

provider "filemanager" {}
```

Run `terraform init` to download the provider.

## Your First File

Let's create a simple configuration file:

```hcl
resource "filemanager_file" "hello" {
  path    = "/tmp/hello.txt"
  content = "Hello, FileManager!"
}
```

Apply the configuration:

```bash
terraform apply
```

Verify the file was created:

```bash
cat /tmp/hello.txt
# Output: Hello, FileManager!
```

## Creating a JSON Configuration File

FileManager excels at managing structured configuration files:

```hcl
resource "filemanager_json_file" "config" {
  path = "/tmp/app-config.json"

  data = {
    app = {
      name    = "myapp"
      version = "1.0.0"
      debug   = false
    }
    database = {
      host = "localhost"
      port = 5432
    }
  }
}
```

The provider automatically formats the JSON with proper indentation.

## Reading Configuration Files

Use data sources to read existing configuration files:

```hcl
data "filemanager_json" "existing" {
  path = "/etc/app/config.json"
}

output "database_host" {
  value = data.filemanager_json.existing.data.database.host
}
```

## Managing Directories

Create directories with proper permissions:

```hcl
resource "filemanager_directory" "app" {
  path       = "/opt/myapp"
  permission = "0755"
}

resource "filemanager_directory" "logs" {
  path       = "/opt/myapp/logs"
  permission = "0755"

  depends_on = [filemanager_directory.app]
}
```

## Using Atomic Writes

FileManager uses atomic writes by default to prevent partial file writes:

```hcl
resource "filemanager_file" "important" {
  path         = "/etc/important-config"
  content      = "critical data"
  atomic_write = true  # This is the default
}
```

With atomic writes:
1. Content is written to a temporary file
2. The temporary file is renamed to the target path
3. If anything fails, the original file remains intact

## Next Steps

- Learn about [Backend Configuration](backends.md) to manage files on remote systems
- Explore [Structured Files](structured-files.md) for JSON, YAML, TOML, and more
- Understand [Atomic Operations](atomic-operations.md) for data integrity guarantees

## Common Patterns

### Configuration File with Backup

```hcl
resource "filemanager_file" "nginx" {
  path    = "/etc/nginx/nginx.conf"
  content = file("${path.module}/nginx.conf")

  backup           = true
  backup_retention = 3
}
```

### Template Rendering

```hcl
resource "filemanager_template_file" "config" {
  path     = "/etc/app/config.yaml"
  template = file("${path.module}/config.yaml.tpl")

  variables = {
    environment = var.environment
    log_level   = var.debug ? "debug" : "info"
  }
}
```

### File with Checksum Verification

```hcl
resource "filemanager_file" "verified" {
  path            = "/opt/app/binary"
  source          = "files/binary"
  verify_checksum = true
}

output "file_checksum" {
  value = filemanager_file.verified.sha256
}
```
