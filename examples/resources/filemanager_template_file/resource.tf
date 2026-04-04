# Create a file from Go template
resource "filemanager_template_file" "nginx_config" {
  path     = "/etc/nginx/sites-available/app.conf"
  template = <<-TEMPLATE
    server {
        listen {{ .port }};
        server_name {{ .domain }};

        location / {
            proxy_pass http://{{ .upstream_host }}:{{ .upstream_port }};
            proxy_set_header Host $host;
            proxy_set_header X-Real-IP $remote_addr;
        }
    }
  TEMPLATE

  vars = {
    port          = "80"
    domain        = "example.com"
    upstream_host = "localhost"
    upstream_port = "8080"
  }

  create_parent_dirs = true
}

# Create systemd unit file from template
resource "filemanager_template_file" "systemd_service" {
  path     = "/etc/systemd/system/myapp.service"
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

    [Install]
    WantedBy=multi-user.target
  TEMPLATE

  vars = {
    description = "My Application Service"
    user        = "appuser"
    working_dir = "/opt/myapp"
    exec_start  = "/opt/myapp/bin/server"
  }

  create_parent_dirs = true
}
