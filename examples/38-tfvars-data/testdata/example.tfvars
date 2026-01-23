region         = "us-west-2"
environment    = "production"
instance_type  = "t3.large"
instance_count = 3
enable_ha      = true

tags = {
  team       = "platform"
  managed_by = "terraform"
}

allowed_cidrs = ["10.0.0.0/8", "172.16.0.0/12"]
