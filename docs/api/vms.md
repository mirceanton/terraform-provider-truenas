# Virtual Machines API

Virtual machine management operations via the `vm.*` middleware namespace.

All VM methods use integer IDs (not names). Methods marked **(job)** return a job ID
and should be awaited with `CallAndWait`.

## VM Operations

### vm.query
Query virtual machines.
```bash
midclt call vm.query
midclt call vm.query '[[["name", "=", "testvm"]]]'
midclt call vm.query '[[["status.state", "=", "RUNNING"]]]'
```

Returns:
- `id` - VM ID (integer)
- `name` - VM name
- `description` - VM description
- `vcpus` - Virtual CPU sockets (min 1, max 16)
- `cores` - CPU cores per socket (min 1)
- `threads` - Threads per core (min 1)
- `memory` - Memory in MB (min 20)
- `min_memory` - Minimum memory for ballooning (int|null, min 20)
- `autostart` - Start on boot (default true)
- `time` - Clock type: `LOCAL`, `UTC` (default LOCAL)
- `bootloader` - `UEFI` or `UEFI_CSM` (default UEFI)
- `bootloader_ovmf` - OVMF firmware file (default `OVMF_CODE.fd`)
- `cpu_mode` - `CUSTOM`, `HOST-MODEL`, `HOST-PASSTHROUGH` (default CUSTOM)
- `cpu_model` - CPU model name (string|null)
- `cpuset` - CPU pinning (string|null)
- `nodeset` - NUMA node pinning (string|null)
- `enable_cpu_topology_extension` - Enable CPU topology extension (default false)
- `pin_vcpus` - Pin vCPUs to physical CPUs (default false)
- `hyperv_enlightenments` - Hyper-V enlightenments for Windows guests (default false)
- `shutdown_timeout` - Shutdown timeout in seconds (5-300, default 90)
- `hide_from_msr` - Hide KVM hypervisor from MSR discovery (default false)
- `ensure_display_device` - Ensure guest has a display device (default true)
- `suspend_on_snapshot` - Suspend VM during periodic snapshots (default false)
- `trusted_platform_module` - Enable TPM (default false)
- `enable_secure_boot` - Enable secure boot (default false)
- `arch_type` - Architecture type (string|null, e.g. `x86_64`)
- `machine_type` - Machine type (string|null, e.g. `q35`)
- `uuid` - VM UUID (string|null)
- `command_line_args` - Extra QEMU command line arguments
- `status` - Current status object: `{state, pid, domain_state}`
- `devices` - Attached devices array
- `display_available` - Whether a display device is available (bool)

### vm.create
Create a virtual machine.
```bash
midclt call vm.create '{
  "name": "testvm",
  "description": "Test virtual machine",
  "vcpus": 2,
  "cores": 1,
  "threads": 1,
  "memory": 2048,
  "bootloader": "UEFI",
  "autostart": false,
  "time": "UTC",
  "shutdown_timeout": 90
}'
```

Required fields: `name`, `memory`.

### vm.update
Update a virtual machine. Accepts partial updates.
```bash
midclt call vm.update <vm_id> '{
  "vcpus": 4,
  "memory": 4096,
  "description": "Updated description"
}'
```

Device management via `vm.update`: if `devices` is included, devices are reconciled:
1. Devices previously attached but missing from the list are **removed**.
2. Devices with a valid `id` are **updated**.
3. Devices without an `id` are **created** and attached.

If `devices` is omitted, no changes are made to devices.

### vm.get_instance
Get a single VM by ID. Raises a validation error if not found.
```bash
midclt call vm.get_instance <vm_id>
```

Returns the same object as `vm.query` entries.

### vm.delete
Delete a virtual machine.
```bash
midclt call vm.delete <vm_id>
midclt call vm.delete <vm_id> '{"zvols": true, "force": false}'
```

Options (all default false):
- `zvols` - Also delete associated zvols
- `force` - Force deletion

### vm.start
Start a virtual machine.
```bash
midclt call vm.start <vm_id>
midclt call vm.start <vm_id> '{"overcommit": false}'
```

Options:
- `overcommit` (default false) - Allow starting even if insufficient memory for all configured VMs. Error `ENOMEM(12)` if false and not enough memory.

### vm.stop **(job)**
Stop a virtual machine (graceful shutdown).
```bash
midclt call vm.stop <vm_id>
midclt call vm.stop <vm_id> '{"force": false, "force_after_timeout": true}'
```

Options (all default false):
- `force` - Force stop immediately
- `force_after_timeout` - Initiate poweroff if VM hasn't stopped within `shutdown_timeout`

### vm.poweroff
Force power off a virtual machine (immediate, no graceful shutdown).
```bash
midclt call vm.poweroff <vm_id>
```

### vm.restart **(job)**
Restart a virtual machine.
```bash
midclt call vm.restart <vm_id>
```

### vm.resume
Resume a suspended virtual machine.
```bash
midclt call vm.resume <vm_id>
```

### vm.status
Get the status of a VM.
```bash
midclt call vm.status <vm_id>
```

Returns:
- `state` - `RUNNING`, `STOPPED`, or `SUSPENDED`
- `pid` - Process ID if running (int|null)
- `domain_state` - Libvirt domain state string

### vm.clone
Clone a virtual machine. Name is optional (auto-generated if omitted).
```bash
midclt call vm.clone <vm_id>
midclt call vm.clone <vm_id> "cloned-vm"
```

### vm.get_available_memory
Get available memory for VMs (returns bytes as integer).
```bash
midclt call vm.get_available_memory
midclt call vm.get_available_memory true
```

Options:
- `overcommit` (default false) - When true: only counts actually consumed memory (not full allocation) and treats shrinkable ZFS ARC as free. When false: counts full requested memory and excludes shrinkable ARC.

### vm.get_display_devices
Get display devices for a VM.
```bash
midclt call vm.get_display_devices <vm_id>
```

### vm.get_display_web_uri
Get web display URI (SPICE). Requires websocket connection.
```bash
midclt call vm.get_display_web_uri <vm_id>
midclt call vm.get_display_web_uri <vm_id> "192.168.1.10"
midclt call vm.get_display_web_uri <vm_id> "192.168.1.10" '{"protocol": "HTTPS"}'
```

Arguments:
- `id` (required) - VM ID
- `host` (optional, default `""`) - Host address for URI
- `options` (optional) - `{protocol: "HTTP"|"HTTPS"}` (default HTTP)

Returns `{error: string|null, uri: string|null}`.

### vm.log_file_download **(job)**
Download VM log file contents. Returns empty file if log doesn't exist.
```bash
midclt call vm.log_file_download <vm_id>
```

### vm.random_mac
Generate a random MAC address.
```bash
midclt call vm.random_mac
```

### vm.port_wizard
Get next available display server port and web port.
```bash
midclt call vm.port_wizard
```

Returns `{port: int, web: int}`.

### vm.maximum_supported_vcpus
Get maximum supported vCPUs.
```bash
midclt call vm.maximum_supported_vcpus
```

### vm.virtualization_details
Check if virtualization is supported on the system.
```bash
midclt call vm.virtualization_details
```

Returns `{supported: bool, error: string|null}`.

### Choice Methods
```bash
midclt call vm.bootloader_options
midclt call vm.cpu_model_choices
midclt call vm.resolution_choices
```

## VM Devices

Devices are managed via `vm.device.*` methods or inline via `vm.update`.

Each device has:
- `id` - Device ID (integer, assigned by server)
- `attributes` - Device-specific attributes (discriminated by `attributes.dtype`)
- `vm` - Parent VM ID (integer)
- `order` - Boot/device order (integer)

### vm.device.query
Query VM devices.
```bash
midclt call vm.device.query
midclt call vm.device.query '[[["vm", "=", <vm_id>]]]'
```

### vm.device.create
Create a VM device.

Top-level fields:
- `vm` (required) - Parent VM ID
- `attributes` (required) - Device attributes (must include `dtype`)
- `order` (optional, default null) - Device order

Disk device (zvol):
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "DISK",
    "path": "/dev/zvol/tank/vms/testvm-disk0",
    "type": "VIRTIO",
    "logical_sectorsize": null,
    "physical_sectorsize": null
  },
  "order": 1000
}'
```

NIC device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "NIC",
    "type": "VIRTIO",
    "mac": "00:a0:98:xx:xx:xx",
    "nic_attach": "br0"
  },
  "order": 1001
}'
```

CD-ROM device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "CDROM",
    "path": "/mnt/tank/iso/ubuntu.iso"
  },
  "order": 1002
}'
```

Display device (SPICE):
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "DISPLAY",
    "type": "SPICE",
    "bind": "0.0.0.0",
    "port": 5900,
    "resolution": "1024x768",
    "web": true,
    "password": ""
  },
  "order": 1003
}'
```

RAW file device:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "RAW",
    "path": "/mnt/tank/vms/testvm-disk1.img",
    "type": "VIRTIO",
    "size": 10737418240,
    "boot": false
  },
  "order": 1004
}'
```

PCI passthrough:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "PCI",
    "pptdev": "0000:01:00.0"
  },
  "order": 1005
}'
```

USB passthrough:
```bash
midclt call vm.device.create '{
  "vm": <vm_id>,
  "attributes": {
    "dtype": "USB",
    "controller_type": "qemu-xhci",
    "device": "usb_8086_1234_..."
  },
  "order": 1006
}'
```

### vm.device.update
Update a VM device.
```bash
midclt call vm.device.update <device_id> '{
  "attributes": {
    "dtype": "CDROM",
    "path": "/mnt/tank/iso/different.iso"
  }
}'
```

### vm.device.delete
Delete a VM device.
```bash
midclt call vm.device.delete <device_id>
midclt call vm.device.delete <device_id> '{"zvol": true, "raw_file": true, "force": false}'
```

Options (all default false):
- `zvol` - Also delete the zvol
- `raw_file` - Also delete the raw file
- `force` - Force deletion

### Choice Methods
```bash
midclt call vm.device.bind_choices
midclt call vm.device.disk_choices
midclt call vm.device.nic_attach_choices
midclt call vm.device.passthrough_device_choices
midclt call vm.device.usb_passthrough_choices
midclt call vm.device.usb_controller_choices
```

## Device Type Reference

| Type | Description |
|------|-------------|
| `DISK` | Block device (zvol) |
| `RAW` | Raw file disk |
| `CDROM` | CD-ROM/ISO image |
| `NIC` | Network interface |
| `DISPLAY` | SPICE display |
| `PCI` | PCI passthrough |
| `USB` | USB passthrough |

### CDROM Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"CDROM"` | - | Required discriminator |
| `path` | string | - | Path to ISO (must start with `/mnt/`) |

### DISK Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"DISK"` | - | Required discriminator |
| `path` | string\|null | null | Path to zvol device |
| `type` | `AHCI`\|`VIRTIO` | `AHCI` | Disk bus type |
| `create_zvol` | bool | false | Create a new zvol |
| `zvol_name` | string\|null | null | Zvol name (when creating) |
| `zvol_volsize` | int\|null | null | Zvol size in bytes (when creating) |
| `logical_sectorsize` | null\|512\|4096 | null | Logical sector size |
| `physical_sectorsize` | null\|512\|4096 | null | Physical sector size |
| `iotype` | `NATIVE`\|`THREADS`\|`IO_URING` | `THREADS` | I/O type |
| `serial` | string\|null | null | Disk serial number |

### RAW Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"RAW"` | - | Required discriminator |
| `path` | string | - | Path to raw file |
| `type` | `AHCI`\|`VIRTIO` | `AHCI` | Disk bus type |
| `exists` | bool | false | Whether the file already exists |
| `boot` | bool | false | Bootable device |
| `size` | int\|null | null | File size in bytes (for creation) |
| `logical_sectorsize` | null\|512\|4096 | null | Logical sector size |
| `physical_sectorsize` | null\|512\|4096 | null | Physical sector size |
| `iotype` | `NATIVE`\|`THREADS`\|`IO_URING` | `THREADS` | I/O type |
| `serial` | string\|null | null | Disk serial number |

### NIC Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"NIC"` | - | Required discriminator |
| `type` | `E1000`\|`VIRTIO` | `E1000` | NIC emulation type |
| `nic_attach` | string\|null | null | Host interface to attach to |
| `mac` | string\|null | null | MAC address (auto-generated if null) |
| `trust_guest_rx_filters` | bool | false | Trust guest RX filters |

### DISPLAY Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"DISPLAY"` | - | Required discriminator |
| `type` | `SPICE` | `SPICE` | Display protocol |
| `resolution` | enum | `1024x768` | Screen resolution |
| `port` | int\|null (5900-65535) | null | SPICE port (auto-assigned if null) |
| `web_port` | int\|null | null | Web client port |
| `bind` | string | `127.0.0.1` | Bind address |
| `wait` | bool | false | Wait for client before booting |
| `password` | string\|null | null | Connection password |
| `web` | bool | true | Enable web client |
| `password_configured` | bool | - | Read-only. Whether a password is set (returned by `vm.get_display_devices`). |

Resolution options: `1920x1200`, `1920x1080`, `1600x1200`, `1600x900`, `1400x1050`, `1280x1024`, `1280x720`, `1024x768`, `800x600`, `640x480`

### PCI Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"PCI"` | - | Required discriminator |
| `pptdev` | string | - | **Required.** PCI device address (e.g. `0000:01:00.0`) |

### USB Attributes
| Field | Type | Default | Description |
|-------|------|---------|-------------|
| `dtype` | `"USB"` | - | Required discriminator |
| `usb` | object\|null | null | `{vendor_id, product_id}` (0x-prefixed hex) |
| `controller_type` | enum | `nec-xhci` | USB controller type |
| `device` | string\|null | null | USB device identifier |

Controller types: `piix3-uhci`, `piix4-uhci`, `ehci`, `ich9-ehci1`, `vt82c686b-uhci`, `pci-ohci`, `nec-xhci`, `qemu-xhci`
