# Configure S3 service for AWS
resource "filemanager_s3_service" "main" {
  name   = "aws-main"
  bucket = var.s3_bucket
  region = var.aws_region

  # Credentials from environment or instance profile
}

# S3 service with explicit credentials
resource "filemanager_s3_service" "backup" {
  name   = "aws-backup"
  bucket = var.backup_bucket
  region = var.aws_region

  access_key = var.aws_access_key
  secret_key = var.aws_secret_key
}

# S3-compatible service (MinIO, DigitalOcean Spaces, etc.)
resource "filemanager_s3_service" "minio" {
  name     = "minio-storage"
  bucket   = "my-bucket"
  endpoint = "https://minio.example.com"
  region   = "us-east-1"

  access_key       = var.minio_access_key
  secret_key       = var.minio_secret_key
  force_path_style = true
}

# DigitalOcean Spaces
resource "filemanager_s3_service" "spaces" {
  name     = "do-spaces"
  bucket   = var.spaces_bucket
  endpoint = "https://${var.spaces_region}.digitaloceanspaces.com"
  region   = var.spaces_region

  access_key = var.spaces_access_key
  secret_key = var.spaces_secret_key
}
