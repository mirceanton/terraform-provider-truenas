# Deploy a configuration file using host_path reference
resource "truenas_host_path" "app_config" {
  path = "/mnt/tank/apps/myapp/config"
  mode = "755"
  uid  = 1000
  gid  = 1000
}

resource "truenas_file" "app_config" {
  host_path     = truenas_host_path.app_config.id
  relative_path = "settings.json"
  content       = jsonencode({
    app_name = "myapp"
    debug    = false
  })
  mode = "0644"
}

# Deploy a file using absolute path
resource "truenas_file" "nginx_config" {
  path    = "/mnt/tank/apps/nginx/nginx.conf"
  content = file("${path.module}/nginx.conf")
  mode    = "0644"
  uid     = 0
  gid     = 0
}

# Deploy a file from template
resource "truenas_file" "app_env" {
  host_path     = truenas_host_path.app_config.id
  relative_path = ".env"
  content       = templatefile("${path.module}/env.tpl", {
    database_url = "postgres://localhost:5432/mydb"
    api_key      = var.api_key
  })
}
