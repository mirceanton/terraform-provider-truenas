# Basic zvol for VM disk
resource "truenas_zvol" "vm_disk" {
  pool    = "tank"
  path    = "vms/my-vm-disk0"
  volsize = "50G"
}

# Zvol with all options
resource "truenas_zvol" "data_volume" {
  pool         = "tank"
  path         = "iscsi/target0-lun0"
  volsize      = "100G"
  volblocksize = "16K"
  sparse       = true
  compression  = "LZ4"
  comments     = "iSCSI target LUN"
}

# Using parent instead of pool
resource "truenas_zvol" "child_volume" {
  parent  = "tank/vms"
  path    = "disk1"
  volsize = "20G"
}
