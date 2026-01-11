# =============================================================================
# TEMPLATE FILE RESOURCE - ALL USE CASES
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
  output_dir   = "${path.module}/../../test/output/10-template-file"
  template_dir = "${path.module}/templates"
}

# -----------------------------------------------------------------------------
# GO TEMPLATES (Inline)
# -----------------------------------------------------------------------------

# Case 1: Simple variable substitution
resource "filemanager_template_file" "go_simple" {
  path     = "${local.output_dir}/go/simple.txt"
  template = "Hello, {{.name}}!"
  vars = {
    name = "World"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 2: Multiple variables
resource "filemanager_template_file" "go_multiple" {
  path     = "${local.output_dir}/go/multiple.txt"
  template = <<-EOF
    Application: {{.app_name}}
    Version: {{.version}}
    Environment: {{.environment}}
    Debug: {{.debug}}
  EOF
  vars = {
    app_name    = "MyApp"
    version     = "1.0.0"
    environment = "production"
    debug       = "false"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 3: Nested template with conditionals concept (using vars)
resource "filemanager_template_file" "go_config" {
  path     = "${local.output_dir}/go/config.yaml"
  template = <<-EOF
    server:
      host: {{.host}}
      port: {{.port}}
      workers: {{.workers}}
    database:
      url: {{.db_url}}
      pool_size: {{.pool_size}}
    features:
      cache_enabled: {{.cache_enabled}}
      rate_limit: {{.rate_limit}}
  EOF
  vars = {
    host          = "0.0.0.0"
    port          = "8080"
    workers       = "4"
    db_url        = "postgres://localhost/mydb"
    pool_size     = "10"
    cache_enabled = "true"
    rate_limit    = "1000"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 4: Nginx configuration template
resource "filemanager_template_file" "go_nginx" {
  path     = "${local.output_dir}/go/nginx.conf"
  template = <<-EOF
    upstream backend {
        server {{.upstream_host}}:{{.upstream_port}};
    }

    server {
        listen {{.listen_port}};
        server_name {{.server_name}};

        root {{.root_path}};
        index index.html index.htm;

        location / {
            try_files $uri $uri/ =404;
        }

        location /api {
            proxy_pass http://backend;
            proxy_http_version 1.1;
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
    }
  EOF
  vars = {
    upstream_host = "127.0.0.1"
    upstream_port = "3000"
    listen_port   = "80"
    server_name   = "example.com"
    root_path     = "/var/www/html"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 5: Script template
resource "filemanager_template_file" "go_script" {
  path     = "${local.output_dir}/go/deploy.sh"
  template = <<-EOF
    #!/bin/bash
    # Deploy script for {{.app_name}}

    APP_NAME="{{.app_name}}"
    VERSION="{{.version}}"
    DEPLOY_DIR="{{.deploy_dir}}"

    echo "Deploying $APP_NAME version $VERSION..."
    cd "$DEPLOY_DIR"
    git pull origin {{.branch}}
    {{.build_command}}
    {{.restart_command}}
    echo "Deployment complete!"
  EOF
  vars = {
    app_name        = "my-application"
    version         = "1.2.3"
    deploy_dir      = "/opt/app"
    branch          = "main"
    build_command   = "npm run build"
    restart_command = "systemctl restart myapp"
  }
  engine             = "go"
  file_permission    = "0755"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# MUSTACHE TEMPLATES (Inline)
# -----------------------------------------------------------------------------

# Case 6: Simple mustache
resource "filemanager_template_file" "mustache_simple" {
  path     = "${local.output_dir}/mustache/simple.txt"
  template = "Hello, {{name}}!"
  vars = {
    name = "Mustache"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# Case 7: Multiple variables mustache
resource "filemanager_template_file" "mustache_readme" {
  path     = "${local.output_dir}/mustache/README.md"
  template = <<-EOF
    # {{project_name}}

    Version: {{version}}
    Author: {{author}}

    ## Description

    {{description}}

    ## Installation

    ```bash
    {{install_command}}
    ```

    ## Usage

    ```bash
    {{usage_command}}
    ```

    ## License

    {{license}}
  EOF
  vars = {
    project_name    = "My Awesome Project"
    version         = "2.0.0"
    author          = "Developer"
    description     = "An awesome project that does amazing things."
    install_command = "npm install my-awesome-project"
    usage_command   = "npx my-awesome-project --help"
    license         = "MIT"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# Case 8: HTML template with mustache
resource "filemanager_template_file" "mustache_html" {
  path     = "${local.output_dir}/mustache/index.html"
  template = <<-EOF
    <!DOCTYPE html>
    <html lang="en">
    <head>
        <meta charset="UTF-8">
        <meta name="viewport" content="width=device-width, initial-scale=1.0">
        <title>{{title}}</title>
        <link rel="stylesheet" href="{{css_path}}">
    </head>
    <body>
        <header>
            <h1>{{heading}}</h1>
            <nav>{{nav_links}}</nav>
        </header>
        <main>
            <p>{{content}}</p>
        </main>
        <footer>
            <p>&copy; {{year}} {{company}}</p>
        </footer>
        <script src="{{js_path}}"></script>
    </body>
    </html>
  EOF
  vars = {
    title     = "Welcome to My Site"
    css_path  = "/css/style.css"
    heading   = "My Website"
    nav_links = "<a href='/'>Home</a> | <a href='/about'>About</a>"
    content   = "Welcome to our website. We're glad you're here!"
    year      = "2024"
    company   = "My Company"
    js_path   = "/js/main.js"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CUSTOM DELIMITERS
# -----------------------------------------------------------------------------

# Case 9: Custom delimiters for Go templates
resource "filemanager_template_file" "custom_delims" {
  path     = "${local.output_dir}/custom/delims.txt"
  template = <<-EOF
    Normal braces: {example}
    Custom delimited: [[.name]]
    More custom: [[.value]]
  EOF
  vars = {
    name  = "CustomName"
    value = "CustomValue"
  }
  engine             = "go"
  left_delim         = "[["
  right_delim        = "]]"
  create_parent_dirs = true
}

# Case 10: Angle bracket delimiters
resource "filemanager_template_file" "angle_delims" {
  path     = "${local.output_dir}/custom/angle.txt"
  template = <<-EOF
    Value: <%.name%>
    Other: <%.other%>
  EOF
  vars = {
    name  = "AngleName"
    other = "AngleOther"
  }
  engine             = "go"
  left_delim         = "<%"
  right_delim        = "%>"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# CONFIGURATION FILE TEMPLATES
# -----------------------------------------------------------------------------

# Case 11: Docker Compose template
resource "filemanager_template_file" "docker_compose" {
  path     = "${local.output_dir}/configs/docker-compose.yml"
  template = <<-EOF
    version: '3.8'
    services:
      app:
        image: {{.app_image}}:{{.app_tag}}
        ports:
          - "{{.app_port}}:{{.app_port}}"
        environment:
          - NODE_ENV={{.node_env}}
          - DATABASE_URL={{.db_url}}
        depends_on:
          - db
      db:
        image: {{.db_image}}:{{.db_tag}}
        environment:
          - POSTGRES_USER={{.db_user}}
          - POSTGRES_PASSWORD={{.db_pass}}
          - POSTGRES_DB={{.db_name}}
        volumes:
          - pgdata:/var/lib/postgresql/data
    volumes:
      pgdata:
  EOF
  vars = {
    app_image = "myapp"
    app_tag   = "latest"
    app_port  = "3000"
    node_env  = "production"
    db_url    = "postgres://user:pass@db:5432/mydb"
    db_image  = "postgres"
    db_tag    = "15-alpine"
    db_user   = "user"
    db_pass   = "pass"
    db_name   = "mydb"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# Case 12: Kubernetes deployment template
resource "filemanager_template_file" "k8s_deployment" {
  path     = "${local.output_dir}/configs/deployment.yaml"
  template = <<-EOF
    apiVersion: apps/v1
    kind: Deployment
    metadata:
      name: {{.name}}
      labels:
        app: {{.app}}
    spec:
      replicas: {{.replicas}}
      selector:
        matchLabels:
          app: {{.app}}
      template:
        metadata:
          labels:
            app: {{.app}}
        spec:
          containers:
          - name: {{.container_name}}
            image: {{.image}}:{{.tag}}
            ports:
            - containerPort: {{.port}}
            resources:
              limits:
                cpu: {{.cpu_limit}}
                memory: {{.memory_limit}}
              requests:
                cpu: {{.cpu_request}}
                memory: {{.memory_request}}
  EOF
  vars = {
    name           = "my-deployment"
    app            = "my-app"
    replicas       = "3"
    container_name = "app"
    image          = "my-registry/my-app"
    tag            = "v1.0.0"
    port           = "8080"
    cpu_limit      = "500m"
    memory_limit   = "256Mi"
    cpu_request    = "100m"
    memory_request = "128Mi"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# Case 13: Terraform backend config template
resource "filemanager_template_file" "tf_backend" {
  path     = "${local.output_dir}/configs/backend.tf"
  template = <<-EOF
    terraform {
      backend "s3" {
        bucket         = "{{.bucket}}"
        key            = "{{.key}}"
        region         = "{{.region}}"
        encrypt        = {{.encrypt}}
        dynamodb_table = "{{.dynamodb_table}}"
      }
    }
  EOF
  vars = {
    bucket         = "my-terraform-state"
    key            = "prod/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = "true"
    dynamodb_table = "terraform-locks"
  }
  engine             = "mustache"
  create_parent_dirs = true
}

# -----------------------------------------------------------------------------
# EDGE CASES
# -----------------------------------------------------------------------------

# Case 14: Minimal template (single space)
resource "filemanager_template_file" "empty" {
  path               = "${local.output_dir}/edge/empty.txt"
  template           = " "
  vars               = {}
  engine             = "go"
  create_parent_dirs = true
}

# Case 15: No variables
resource "filemanager_template_file" "no_vars" {
  path               = "${local.output_dir}/edge/no_vars.txt"
  template           = "This template has no variables."
  vars               = {}
  engine             = "go"
  create_parent_dirs = true
}

# Case 16: Special characters in values
resource "filemanager_template_file" "special_chars" {
  path     = "${local.output_dir}/edge/special.txt"
  template = <<-EOF
    Path: {{.path}}
    URL: {{.url}}
    Unicode: {{.unicode}}
    Newlines: {{.newlines}}
  EOF
  vars = {
    path     = "/home/user/data"
    url      = "https://example.com/api?key=value"
    unicode  = "日本語 中文 한국어"
    newlines = "line1\\nline2"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 17: Large template
resource "filemanager_template_file" "large" {
  path = "${local.output_dir}/edge/large.txt"
  template = join("\n", [
    for i in range(100) : "Line {{.prefix}}${i}: {{.value}}"
  ])
  vars = {
    prefix = "NUM_"
    value  = "some-value"
  }
  engine             = "go"
  create_parent_dirs = true
}

# Case 18: Many variables
resource "filemanager_template_file" "many_vars" {
  path = "${local.output_dir}/edge/many_vars.txt"
  template = join("\n", [
    for i in range(20) : "Var${i}: {{.var${i}}}"
  ])
  vars = {
    for i in range(20) : "var${i}" => "value_${i}"
  }
  engine             = "go"
  create_parent_dirs = true
}
