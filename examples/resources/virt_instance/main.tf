# Create a basic Ubuntu container
resource "truenas_virt_instance" "example" {
  name          = "my-container"
  image_name    = "ubuntu"
  image_version = "24.04"
  storage_pool  = "tank"
  autostart     = true
}
