# Read current virtualization configuration
data "truenas_virt_config" "current" {}

# Use the configured pool for a new container
resource "truenas_virt_instance" "example" {
  name         = "my-container"
  image_name   = "ubuntu"
  image_version = "24.04"
  storage_pool = data.truenas_virt_config.current.pool
}
