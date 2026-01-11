---
page_title: "Backend Configuration"
subcategory: "Guides"
description: |-
  Configuring storage backends for the FileManager provider - local filesystem, SSH, S3, Azure, GCS, and more.
---

# Backend Configuration

The FileManager provider supports multiple storage backends, allowing you to manage files on local systems, remote servers, and cloud storage services.

## Available Backends

| Backend | Protocol | Use Case |
|---------|----------|----------|
| Local | file:// | Local filesystem operations |
| SSH | ssh:// | Remote servers via SSH/SFTP |
| S3 | s3:// | AWS S3 and S3-compatible storage |
| Azure | azure:// | Azure Blob Storage |
| GCS | gs:// | Google Cloud Storage |
| B2 | b2:// | Backblaze B2 |
| FTP | ftp:// | FTP/FTPS servers |
| Swift | swift:// | OpenStack Swift |

## Local Filesystem (Default)

The local filesystem is the default backend. No additional configuration is required:

```hcl
provider "filemanager" {}

resource "filemanager_file" "local" {
  path    = "/etc/app/config.json"
  content = jsonencode({ key = "value" })
}
```

### Base Path

Set a base path for all relative file paths:

```hcl
provider "filemanager" {
  base_path = "/opt/myapp"
}

resource "filemanager_file" "config" {
  path    = "config/app.json"  # Resolves to /opt/myapp/config/app.json
  content = "{}"
}
```

## SSH/SFTP Backend

Manage files on remote servers via SSH:

### Basic Configuration

```hcl
resource "filemanager_ssh_service" "webserver" {
  name = "webserver"
  host = "server.example.com"
  port = 22
  user = "deploy"

  # Authentication (choose one)
  private_key = file("~/.ssh/id_rsa")
  # password  = var.ssh_password
}
```

### Using the SSH Backend

```hcl
resource "filemanager_file" "remote_config" {
  path    = "/etc/nginx/nginx.conf"
  content = file("${path.module}/nginx.conf")
  service = filemanager_ssh_service.webserver.name
}

data "filemanager_json" "remote_data" {
  path    = "/etc/app/config.json"
  service = filemanager_ssh_service.webserver.name
}
```

### SSH Key Authentication

```hcl
resource "filemanager_ssh_service" "secure" {
  name        = "secure-server"
  host        = "secure.example.com"
  user        = "admin"
  private_key = file("~/.ssh/id_ed25519")

  # Optional: specify key passphrase
  # passphrase = var.key_passphrase

  # Optional: verify host key
  host_key = "ssh-ed25519 AAAAC3..."
}
```

### SSH Agent Authentication

```hcl
resource "filemanager_ssh_service" "agent" {
  name      = "agent-auth"
  host      = "server.example.com"
  user      = "deploy"
  use_agent = true
}
```

## S3-Compatible Storage

Manage files in S3 buckets or S3-compatible storage (MinIO, DigitalOcean Spaces, etc.):

### AWS S3

```hcl
resource "filemanager_s3_service" "aws" {
  name       = "aws-bucket"
  bucket     = "my-terraform-files"
  region     = "us-east-1"
  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}

resource "filemanager_upload" "config" {
  source      = "local/config.json"
  destination = "configs/app/config.json"
  service     = filemanager_s3_service.aws.name
}
```

### MinIO (Self-Hosted S3)

```hcl
resource "filemanager_s3_service" "minio" {
  name       = "minio"
  bucket     = "configs"
  endpoint   = "https://minio.example.com"
  access_key = var.minio_access_key
  secret_key = var.minio_secret_key

  # For self-signed certificates
  insecure = true
}
```

### DigitalOcean Spaces

```hcl
resource "filemanager_s3_service" "spaces" {
  name       = "do-spaces"
  bucket     = "my-space"
  endpoint   = "https://nyc3.digitaloceanspaces.com"
  region     = "nyc3"
  access_key = var.spaces_access_key
  secret_key = var.spaces_secret_key
}
```

## Azure Blob Storage

```hcl
resource "filemanager_azure_service" "blob" {
  name           = "azure-storage"
  account_name   = "mystorageaccount"
  container_name = "configs"

  # Authentication options
  account_key = var.azure_account_key
  # Or use SAS token
  # sas_token = var.azure_sas_token
}

resource "filemanager_upload" "azure_config" {
  source      = "configs/app.json"
  destination = "production/app.json"
  service     = filemanager_azure_service.blob.name
}
```

## Google Cloud Storage

```hcl
resource "filemanager_gcs_service" "bucket" {
  name        = "gcs-configs"
  bucket      = "my-gcs-bucket"
  credentials = file("${path.module}/gcp-credentials.json")
}

resource "filemanager_upload" "gcs_config" {
  source      = "local/config.yaml"
  destination = "configs/app.yaml"
  service     = filemanager_gcs_service.bucket.name
}
```

## Backblaze B2

```hcl
resource "filemanager_b2_service" "bucket" {
  name           = "b2-storage"
  bucket         = "my-b2-bucket"
  application_id = var.b2_app_id
  application_key = var.b2_app_key
}
```

## FTP/FTPS

```hcl
resource "filemanager_ftp_service" "ftp" {
  name     = "ftp-server"
  host     = "ftp.example.com"
  port     = 21
  user     = "ftpuser"
  password = var.ftp_password

  # For FTPS (FTP over TLS)
  tls = true
}
```

## OpenStack Swift

```hcl
resource "filemanager_swift_service" "swift" {
  name       = "swift-storage"
  container  = "configs"
  auth_url   = "https://identity.example.com/v3"
  username   = var.swift_user
  password   = var.swift_password
  tenant     = "my-project"
  region     = "RegionOne"
}
```

## Cross-Backend Transfers

Transfer files between different backends:

```hcl
# Download from S3 to local
resource "filemanager_download" "from_s3" {
  source      = "configs/app.json"
  destination = "/etc/app/config.json"
  service     = filemanager_s3_service.aws.name
}

# Transfer between backends
resource "filemanager_transfer" "s3_to_server" {
  source             = "configs/app.json"
  source_service     = filemanager_s3_service.aws.name
  destination        = "/etc/app/config.json"
  destination_service = filemanager_ssh_service.webserver.name
}
```

## Directory Synchronization

Sync directories between backends:

```hcl
resource "filemanager_sync" "configs" {
  source             = "/local/configs"
  destination        = "/remote/configs"
  destination_service = filemanager_ssh_service.webserver.name

  delete = true  # Delete files in destination not in source
}
```

## Best Practices

### 1. Use Variables for Credentials

Never hardcode credentials in your configuration:

```hcl
variable "ssh_private_key" {
  type      = string
  sensitive = true
}

resource "filemanager_ssh_service" "server" {
  name        = "server"
  host        = "server.example.com"
  user        = "deploy"
  private_key = var.ssh_private_key
}
```

### 2. Reuse Service Resources

Define services once and reference them across resources:

```hcl
resource "filemanager_ssh_service" "prod" {
  name = "production"
  # ... configuration
}

resource "filemanager_file" "config1" {
  path    = "/etc/app1/config.json"
  content = jsonencode(local.app1_config)
  service = filemanager_ssh_service.prod.name
}

resource "filemanager_file" "config2" {
  path    = "/etc/app2/config.json"
  content = jsonencode(local.app2_config)
  service = filemanager_ssh_service.prod.name
}
```

### 3. Test Connectivity

Use data sources to verify backend connectivity:

```hcl
data "filemanager_stat" "verify_access" {
  path    = "/tmp"
  service = filemanager_ssh_service.server.name
}
```

## Troubleshooting

### SSH Connection Issues

1. **Permission denied**: Verify the private key has correct permissions (`chmod 600`)
2. **Host key verification failed**: Add the server's host key or set `host_key_check = false` (not recommended for production)
3. **Connection timeout**: Check firewall rules and network connectivity

### S3 Access Issues

1. **Access denied**: Verify IAM permissions include the required S3 actions
2. **Bucket not found**: Check bucket name and region
3. **SSL errors with MinIO**: Set `insecure = true` for self-signed certificates

### Azure Connection Issues

1. **Authentication failed**: Verify account key or SAS token is valid
2. **Container not found**: Ensure the container exists before uploading
