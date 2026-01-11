app_name = "myapp"
environment = "production"
debug = false

server {
  host = "0.0.0.0"
  port = 8080
  timeout = "30s"
}

database "primary" {
  host = "db.example.com"
  port = 5432
  name = "production"
}

database "replica" {
  host = "db-replica.example.com"
  port = 5432
  name = "production"
  read_only = true
}

features = ["authentication", "logging", "metrics"]
