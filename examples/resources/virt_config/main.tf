# Configure the virtualization subsystem
resource "truenas_virt_config" "main" {
  pool       = "tank"
  v4_network = "10.0.0.0/24"
}
