# Basic VM with minimal configuration
resource "truenas_vm" "basic" {
  name   = "my-vm"
  memory = 2048

  disk {
    path = "/dev/zvol/tank/vms/my-vm-disk0"
    type = "VIRTIO"
  }

  nic {
    type       = "VIRTIO"
    nic_attach = "br0"
  }
}

# VM with ISO installer and display
resource "truenas_vm" "with_installer" {
  name       = "ubuntu-install"
  memory     = 4096
  vcpus      = 2
  cores      = 2
  autostart  = false
  bootloader = "UEFI"
  cpu_mode   = "HOST-PASSTHROUGH"
  state      = "RUNNING"

  disk {
    path = "/dev/zvol/tank/vms/ubuntu-disk0"
    type = "VIRTIO"
  }

  cdrom {
    path = "/mnt/tank/iso/ubuntu-24.04-server.iso"
  }

  nic {
    type       = "VIRTIO"
    nic_attach = "br0"
  }

  display {
    type       = "SPICE"
    resolution = "1920x1080"
    bind       = "0.0.0.0"
    web        = true
  }
}

# Windows VM with TPM and Hyper-V enlightenments
resource "truenas_vm" "windows" {
  name              = "windows-11"
  memory            = 8192
  vcpus             = 4
  cores             = 2
  threads           = 2
  bootloader        = "UEFI"
  time              = "LOCAL"
  shutdown_timeout  = 120
  state             = "STOPPED"

  disk {
    path = "/dev/zvol/tank/vms/windows-disk0"
    type = "VIRTIO"
  }

  nic {
    type       = "VIRTIO"
    nic_attach = "br0"
  }

  display {
    type       = "SPICE"
    resolution = "1920x1080"
    web        = true
  }
}
