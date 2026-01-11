# Create a file from Go template
resource "filemanager_template_file" "nginx_config" {
  path     = "/etc/nginx/sites-available/app.conf"
  template = <<-TEMPLATE
    server {
        listen {{ .port }};
        server_name {{ .domain }};

        {{ if .ssl_enabled }}
        listen 443 ssl;
        ssl_certificate /etc/ssl/certs/{{ .domain }}.crt;
        ssl_certificate_key /etc/ssl/private/{{ .domain }}.key;
        {{ end }}

        location / {
            proxy_pass http://{{ .upstream_host }}:{{ .upstream_port }};
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }

        {{ range .locations }}
        location {{ .path }} {
            {{ .directive }};
        }
        {{ end }}
    }
  TEMPLATE

  vars = {
    port          = 80
    domain        = var.domain
    ssl_enabled   = var.ssl_enabled
    upstream_host = "localhost"
    upstream_port = 8080
    locations     = [
      { path = "/static", directive = "alias /var/www/static" },
      { path = "/api", directive = "proxy_pass http://api:3000" }
    ]
  }

  create_parent_dirs = true
}

# Create systemd unit file from template
resource "filemanager_template_file" "systemd_service" {
  path     = "/etc/systemd/system/${var.service_name}.service"
  template = <<-TEMPLATE
    [Unit]
    Description={{ .description }}
    After=network.target

    [Service]
    Type=simple
    User={{ .user }}
    WorkingDirectory={{ .working_dir }}
    ExecStart={{ .exec_start }}
    Restart=always
    RestartSec=5
    {{ range $key, $value := .environment }}
    Environment="{{ $key }}={{ $value }}"
    {{ end }}

    [Install]
    WantedBy=multi-user.target
  TEMPLATE

  vars = {
    description = var.service_description
    user        = var.service_user
    working_dir = var.working_directory
    exec_start  = var.exec_command
    environment = var.environment_vars
  }

  create_parent_dirs = true
}
