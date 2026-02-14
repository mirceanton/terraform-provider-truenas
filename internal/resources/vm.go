package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// VM state constants matching TrueNAS API values.
const (
	VMStateRunning = "RUNNING"
	VMStateStopped = "STOPPED"
)

var (
	_ resource.Resource                = &VMResource{}
	_ resource.ResourceWithConfigure   = &VMResource{}
	_ resource.ResourceWithImportState = &VMResource{}
)

// VMResourceModel describes the resource data model.
type VMResourceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	VCPUs            types.Int64  `tfsdk:"vcpus"`
	Cores            types.Int64  `tfsdk:"cores"`
	Threads          types.Int64  `tfsdk:"threads"`
	Memory           types.Int64  `tfsdk:"memory"`
	MinMemory        types.Int64  `tfsdk:"min_memory"`
	Autostart        types.Bool   `tfsdk:"autostart"`
	Time             types.String `tfsdk:"time"`
	Bootloader       types.String `tfsdk:"bootloader"`
	BootloaderOVMF   types.String `tfsdk:"bootloader_ovmf"`
	CPUMode          types.String `tfsdk:"cpu_mode"`
	CPUModel         types.String `tfsdk:"cpu_model"`
	ShutdownTimeout  types.Int64  `tfsdk:"shutdown_timeout"`
	State            types.String `tfsdk:"state"`
	DisplayAvailable types.Bool   `tfsdk:"display_available"`
	// Device blocks
	Disks    []VMDiskModel    `tfsdk:"disk"`
	Raws     []VMRawModel     `tfsdk:"raw"`
	CDROMs   []VMCDROMModel   `tfsdk:"cdrom"`
	NICs     []VMNICModel     `tfsdk:"nic"`
	Displays []VMDisplayModel `tfsdk:"display"`
	PCIs     []VMPCIModel     `tfsdk:"pci"`
	USBs     []VMUSBModel     `tfsdk:"usb"`
}

// VMDiskModel represents a DISK device.
type VMDiskModel struct {
	DeviceID           types.Int64  `tfsdk:"device_id"`
	Path               types.String `tfsdk:"path"`
	Type               types.String `tfsdk:"type"`
	LogicalSectorSize  types.Int64  `tfsdk:"logical_sectorsize"`
	PhysicalSectorSize types.Int64  `tfsdk:"physical_sectorsize"`
	IOType             types.String `tfsdk:"iotype"`
	Serial             types.String `tfsdk:"serial"`
	Order              types.Int64  `tfsdk:"order"`
}

// VMRawModel represents a RAW device.
type VMRawModel struct {
	DeviceID           types.Int64  `tfsdk:"device_id"`
	Path               types.String `tfsdk:"path"`
	Type               types.String `tfsdk:"type"`
	Boot               types.Bool   `tfsdk:"boot"`
	Size               types.Int64  `tfsdk:"size"`
	LogicalSectorSize  types.Int64  `tfsdk:"logical_sectorsize"`
	PhysicalSectorSize types.Int64  `tfsdk:"physical_sectorsize"`
	IOType             types.String `tfsdk:"iotype"`
	Serial             types.String `tfsdk:"serial"`
	Order              types.Int64  `tfsdk:"order"`
}

// VMCDROMModel represents a CDROM device.
type VMCDROMModel struct {
	DeviceID types.Int64  `tfsdk:"device_id"`
	Path     types.String `tfsdk:"path"`
	Order    types.Int64  `tfsdk:"order"`
}

// VMNICModel represents a NIC device.
type VMNICModel struct {
	DeviceID            types.Int64  `tfsdk:"device_id"`
	Type                types.String `tfsdk:"type"`
	NICAttach           types.String `tfsdk:"nic_attach"`
	MAC                 types.String `tfsdk:"mac"`
	TrustGuestRXFilters types.Bool   `tfsdk:"trust_guest_rx_filters"`
	Order               types.Int64  `tfsdk:"order"`
}

// VMDisplayModel represents a DISPLAY device.
type VMDisplayModel struct {
	DeviceID   types.Int64  `tfsdk:"device_id"`
	Type       types.String `tfsdk:"type"`
	Resolution types.String `tfsdk:"resolution"`
	Port       types.Int64  `tfsdk:"port"`
	WebPort    types.Int64  `tfsdk:"web_port"`
	Bind       types.String `tfsdk:"bind"`
	Wait       types.Bool   `tfsdk:"wait"`
	Password   types.String `tfsdk:"password"`
	Web        types.Bool   `tfsdk:"web"`
	Order      types.Int64  `tfsdk:"order"`
}

// VMPCIModel represents a PCI passthrough device.
type VMPCIModel struct {
	DeviceID types.Int64  `tfsdk:"device_id"`
	PPTDev   types.String `tfsdk:"pptdev"`
	Order    types.Int64  `tfsdk:"order"`
}

// VMUSBModel represents a USB passthrough device.
type VMUSBModel struct {
	DeviceID       types.Int64  `tfsdk:"device_id"`
	ControllerType types.String `tfsdk:"controller_type"`
	Device         types.String `tfsdk:"device"`
	Order          types.Int64  `tfsdk:"order"`
}

// vmAPIResponse represents the JSON response from vm.get_instance.
type vmAPIResponse struct {
	ID               int64           `json:"id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	VCPUs            int64           `json:"vcpus"`
	Cores            int64           `json:"cores"`
	Threads          int64           `json:"threads"`
	Memory           int64           `json:"memory"`
	MinMemory        *int64          `json:"min_memory"`
	Autostart        bool            `json:"autostart"`
	Time             string          `json:"time"`
	Bootloader       string          `json:"bootloader"`
	BootloaderOVMF   string          `json:"bootloader_ovmf"`
	CPUMode          string          `json:"cpu_mode"`
	CPUModel         *string         `json:"cpu_model"`
	ShutdownTimeout  int64           `json:"shutdown_timeout"`
	Status           vmStatusResponse `json:"status"`
	DisplayAvailable bool            `json:"display_available"`
}

type vmStatusResponse struct {
	State       string `json:"state"`
	PID         *int64 `json:"pid"`
	DomainState string `json:"domain_state"`
}

// vmDeviceAPIResponse represents a device from vm.device.query.
type vmDeviceAPIResponse struct {
	ID         int64          `json:"id"`
	VM         int64          `json:"vm"`
	Order      int64          `json:"order"`
	Attributes map[string]any `json:"attributes"`
}

// VMResource defines the resource implementation.
type VMResource struct {
	client client.Client
}

// NewVMResource creates a new VMResource.
func NewVMResource() resource.Resource {
	return &VMResource{}
}

func (r *VMResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vm"
}

func (r *VMResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a QEMU/KVM virtual machine on TrueNAS.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "VM ID (numeric, stored as string for Terraform compatibility).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				Description: "VM name.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "VM description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"vcpus": schema.Int64Attribute{
				Description: "Number of virtual CPU sockets. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.Between(1, 16),
				},
			},
			"cores": schema.Int64Attribute{
				Description: "CPU cores per socket. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"threads": schema.Int64Attribute{
				Description: "Threads per core. Defaults to 1.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(1),
				Validators: []validator.Int64{
					int64validator.AtLeast(1),
				},
			},
			"memory": schema.Int64Attribute{
				Description: "Memory in MB (minimum 20).",
				Required:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(20),
				},
			},
			"min_memory": schema.Int64Attribute{
				Description: "Minimum memory for ballooning in MB. Null to disable.",
				Optional:    true,
				Validators: []validator.Int64{
					int64validator.AtLeast(20),
				},
			},
			"autostart": schema.BoolAttribute{
				Description: "Start VM on boot. Defaults to true.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"time": schema.StringAttribute{
				Description: "Clock type: LOCAL or UTC. Defaults to LOCAL.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("LOCAL"),
				Validators: []validator.String{
					stringvalidator.OneOf("LOCAL", "UTC"),
				},
			},
			"bootloader": schema.StringAttribute{
				Description: "Bootloader type: UEFI or UEFI_CSM. Defaults to UEFI.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("UEFI"),
				Validators: []validator.String{
					stringvalidator.OneOf("UEFI", "UEFI_CSM"),
				},
			},
			"bootloader_ovmf": schema.StringAttribute{
				Description: "OVMF firmware file. Defaults to OVMF_CODE.fd.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("OVMF_CODE.fd"),
			},
			"cpu_mode": schema.StringAttribute{
				Description: "CPU mode: CUSTOM, HOST-MODEL, or HOST-PASSTHROUGH. Defaults to CUSTOM.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString("CUSTOM"),
				Validators: []validator.String{
					stringvalidator.OneOf("CUSTOM", "HOST-MODEL", "HOST-PASSTHROUGH"),
				},
			},
			"cpu_model": schema.StringAttribute{
				Description: "CPU model name (when cpu_mode is CUSTOM).",
				Optional:    true,
			},
			"shutdown_timeout": schema.Int64Attribute{
				Description: "Shutdown timeout in seconds (5-300). Defaults to 90.",
				Optional:    true,
				Computed:    true,
				Default:     int64default.StaticInt64(90),
				Validators: []validator.Int64{
					int64validator.Between(5, 300),
				},
			},
			"state": schema.StringAttribute{
				Description: "Desired VM power state: RUNNING or STOPPED. Defaults to STOPPED.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(VMStateStopped),
				Validators: []validator.String{
					stringvalidator.OneOf(VMStateRunning, VMStateStopped),
				},
			},
			"display_available": schema.BoolAttribute{
				Description: "Whether a display device is available.",
				Computed:    true,
			},
		},
		Blocks: map[string]schema.Block{
			"disk": schema.ListNestedBlock{
				Description: "DISK devices (zvol block devices).",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{
							Description: "Device ID assigned by TrueNAS.",
							Computed:    true,
						},
						"path": schema.StringAttribute{
							Description: "Path to zvol device (e.g., /dev/zvol/tank/vms/disk0).",
							Required:    true,
						},
						"type": schema.StringAttribute{
							Description: "Disk bus type: AHCI or VIRTIO. Defaults to AHCI.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("AHCI"),
							Validators: []validator.String{
								stringvalidator.OneOf("AHCI", "VIRTIO"),
							},
						},
						"logical_sectorsize": schema.Int64Attribute{
							Description: "Logical sector size: 512 or 4096.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.OneOf(512, 4096),
							},
						},
						"physical_sectorsize": schema.Int64Attribute{
							Description: "Physical sector size: 512 or 4096.",
							Optional:    true,
							Validators: []validator.Int64{
								int64validator.OneOf(512, 4096),
							},
						},
						"iotype": schema.StringAttribute{
							Description: "I/O type: NATIVE, THREADS, or IO_URING. Defaults to THREADS.",
							Optional:    true,
							Computed:    true,
							Default:     stringdefault.StaticString("THREADS"),
							Validators: []validator.String{
								stringvalidator.OneOf("NATIVE", "THREADS", "IO_URING"),
							},
						},
						"serial": schema.StringAttribute{
							Description: "Disk serial number.",
							Optional:    true,
						},
						"order": schema.Int64Attribute{
							Description: "Device boot/load order.",
							Optional:    true,
							Computed:    true,
						},
					},
				},
			},
			"raw": schema.ListNestedBlock{
				Description: "RAW file devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"path":      schema.StringAttribute{Required: true, Description: "Path to raw file."},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("AHCI"),
							Description: "Disk bus type: AHCI or VIRTIO. Defaults to AHCI.",
							Validators:  []validator.String{stringvalidator.OneOf("AHCI", "VIRTIO")},
						},
						"boot": schema.BoolAttribute{
							Optional: true, Computed: true, Default: booldefault.StaticBool(false),
							Description: "Bootable device. Defaults to false.",
						},
						"size":                schema.Int64Attribute{Optional: true, Description: "File size in bytes (for creation)."},
						"logical_sectorsize":  schema.Int64Attribute{Optional: true, Description: "Logical sector size: 512 or 4096.", Validators: []validator.Int64{int64validator.OneOf(512, 4096)}},
						"physical_sectorsize": schema.Int64Attribute{Optional: true, Description: "Physical sector size: 512 or 4096.", Validators: []validator.Int64{int64validator.OneOf(512, 4096)}},
						"iotype": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("THREADS"),
							Description: "I/O type: NATIVE, THREADS, or IO_URING. Defaults to THREADS.",
							Validators:  []validator.String{stringvalidator.OneOf("NATIVE", "THREADS", "IO_URING")},
						},
						"serial": schema.StringAttribute{Optional: true, Description: "Disk serial number."},
						"order":  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
			"cdrom": schema.ListNestedBlock{
				Description: "CD-ROM/ISO devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"path":      schema.StringAttribute{Required: true, Description: "Path to ISO file (must start with /mnt/)."},
						"order":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
			"nic": schema.ListNestedBlock{
				Description: "Network interface devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("E1000"),
							Description: "NIC emulation type: E1000 or VIRTIO. Defaults to E1000.",
							Validators:  []validator.String{stringvalidator.OneOf("E1000", "VIRTIO")},
						},
						"nic_attach":             schema.StringAttribute{Optional: true, Description: "Host interface to attach to."},
						"mac":                    schema.StringAttribute{Optional: true, Computed: true, Description: "MAC address (auto-generated if not set)."},
						"trust_guest_rx_filters": schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Trust guest RX filters. Defaults to false."},
						"order":                  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
			"display": schema.ListNestedBlock{
				Description: "SPICE display devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("SPICE"),
							Description: "Display protocol. Currently only SPICE.",
							Validators:  []validator.String{stringvalidator.OneOf("SPICE")},
						},
						"resolution": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("1024x768"),
							Description: "Screen resolution. Defaults to 1024x768.",
							Validators: []validator.String{stringvalidator.OneOf(
								"1920x1200", "1920x1080", "1600x1200", "1600x900",
								"1400x1050", "1280x1024", "1280x720", "1024x768",
								"800x600", "640x480",
							)},
						},
						"port":     schema.Int64Attribute{Optional: true, Computed: true, Description: "SPICE port (auto-assigned if not set). Range 5900-65535.", Validators: []validator.Int64{int64validator.Between(5900, 65535)}},
						"web_port": schema.Int64Attribute{Optional: true, Computed: true, Description: "Web client port (auto-assigned if not set)."},
						"bind": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("127.0.0.1"),
							Description: "Bind address. Defaults to 127.0.0.1.",
						},
						"wait":     schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(false), Description: "Wait for client before booting. Defaults to false."},
						"password": schema.StringAttribute{Optional: true, Sensitive: true, Description: "Connection password."},
						"web":      schema.BoolAttribute{Optional: true, Computed: true, Default: booldefault.StaticBool(true), Description: "Enable web client. Defaults to true."},
						"order":    schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
			"pci": schema.ListNestedBlock{
				Description: "PCI passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"pptdev":    schema.StringAttribute{Required: true, Description: "PCI device address (e.g., 0000:01:00.0)."},
						"order":     schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
			"usb": schema.ListNestedBlock{
				Description: "USB passthrough devices.",
				NestedObject: schema.NestedBlockObject{
					Attributes: map[string]schema.Attribute{
						"device_id": schema.Int64Attribute{Computed: true, Description: "Device ID assigned by TrueNAS."},
						"controller_type": schema.StringAttribute{
							Optional: true, Computed: true, Default: stringdefault.StaticString("nec-xhci"),
							Description: "USB controller type. Defaults to nec-xhci.",
							Validators: []validator.String{stringvalidator.OneOf(
								"piix3-uhci", "piix4-uhci", "ehci", "ich9-ehci1",
								"vt82c686b-uhci", "pci-ohci", "nec-xhci", "qemu-xhci",
							)},
						},
						"device": schema.StringAttribute{Optional: true, Description: "USB device identifier."},
						"order":  schema.Int64Attribute{Optional: true, Computed: true, Description: "Device boot/load order."},
					},
				},
			},
		},
	}
}

func (r *VMResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Resource Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	r.client = c
}

// -- CRUD --

func (r *VMResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	params := r.buildCreateParams(&data)
	result, err := r.client.Call(ctx, "vm.create", params)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Create VM", fmt.Sprintf("Unable to create VM %q: %s", data.Name.ValueString(), err.Error()))
		return
	}

	// vm.create returns the full VM object
	var vm vmAPIResponse
	if err := json.Unmarshal(result, &vm); err != nil {
		resp.Diagnostics.AddError("Unable to Parse VM Response", err.Error())
		return
	}
	vmID := vm.ID

	// Create devices
	for i := range data.Disks {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildDiskDeviceParams(&data.Disks[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Disk Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.Disks[i].DeviceID)
	}
	for i := range data.Raws {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildRawDeviceParams(&data.Raws[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Raw Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.Raws[i].DeviceID)
	}
	for i := range data.CDROMs {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildCDROMDeviceParams(&data.CDROMs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create CDROM Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.CDROMs[i].DeviceID)
	}
	for i := range data.NICs {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildNICDeviceParams(&data.NICs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create NIC Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.NICs[i].DeviceID)
	}
	for i := range data.Displays {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildDisplayDeviceParams(&data.Displays[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create Display Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.Displays[i].DeviceID)
	}
	for i := range data.PCIs {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildPCIDeviceParams(&data.PCIs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create PCI Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.PCIs[i].DeviceID)
	}
	for i := range data.USBs {
		devResult, err := r.client.Call(ctx, "vm.device.create", buildUSBDeviceParams(&data.USBs[i], vmID))
		if err != nil {
			resp.Diagnostics.AddError("Unable to Create USB Device", err.Error())
			return
		}
		r.setDeviceIDFromResult(devResult, &data.USBs[i].DeviceID)
	}

	// Handle desired state
	desiredState := data.State.ValueString()
	if desiredState == VMStateRunning {
		if err := r.reconcileState(ctx, vmID, VMStateStopped, VMStateRunning); err != nil {
			resp.Diagnostics.AddError("Unable to Start VM", err.Error())
			return
		}
	}

	// Read back to get final state
	freshVM, err := r.getVM(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read VM After Create", err.Error())
		return
	}
	r.mapVMToModel(freshVM, &data)

	// Read devices
	devices, err := r.queryVMDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	r.mapDevicesToModel(devices, &data)

	// Restore desired state (mapVMToModel sets state from API status)
	data.State = types.StringValue(desiredState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", fmt.Sprintf("Cannot parse VM ID %q: %s", data.ID.ValueString(), err.Error()))
		return
	}

	// Preserve user-specified desired state
	priorState := data.State

	vm, err := r.getVM(ctx, vmID)
	if err != nil {
		if isNotFoundError(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Unable to Read VM", err.Error())
		return
	}

	r.mapVMToModel(vm, &data)

	// Read devices
	devices, err := r.queryVMDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	r.mapDevicesToModel(devices, &data)

	// Restore desired state from prior state (user-specified)
	if !priorState.IsNull() && !priorState.IsUnknown() {
		data.State = priorState
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var data VMResourceModel
	var stateData VMResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}
	resp.Diagnostics.Append(req.State.Get(ctx, &stateData)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", err.Error())
		return
	}

	// Build update params (only changed fields)
	updateParams := r.buildUpdateParams(&data, &stateData)
	if len(updateParams) > 0 {
		_, err := r.client.Call(ctx, "vm.update", []any{vmID, updateParams})
		if err != nil {
			resp.Diagnostics.AddError("Unable to Update VM", err.Error())
			return
		}
	}

	// Reconcile devices
	if err := r.reconcileDevices(ctx, vmID, &data, &stateData); err != nil {
		resp.Diagnostics.AddError("Unable to Update VM Devices", err.Error())
		return
	}

	// Handle state transitions
	currentState := stateData.State.ValueString()
	desiredState := data.State.ValueString()
	if currentState != desiredState {
		// Get actual current state from API
		vm, err := r.getVM(ctx, vmID)
		if err != nil {
			resp.Diagnostics.AddError("Unable to Query VM State", err.Error())
			return
		}
		if err := r.reconcileState(ctx, vmID, vm.Status.State, desiredState); err != nil {
			resp.Diagnostics.AddError("Unable to Reconcile VM State", err.Error())
			return
		}
	}

	// Read back fresh state
	freshVM, err := r.getVM(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Read VM After Update", err.Error())
		return
	}
	r.mapVMToModel(freshVM, &data)

	devices, err := r.queryVMDevices(ctx, vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Query VM Devices", err.Error())
		return
	}
	r.mapDevicesToModel(devices, &data)

	// Restore desired state
	data.State = types.StringValue(desiredState)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}

func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data VMResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	vmID, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError("Invalid VM ID", err.Error())
		return
	}

	// Check current state - if running, stop first
	vm, err := r.getVM(ctx, vmID)
	if err != nil {
		if isNotFoundError(err) {
			return // Already deleted
		}
		resp.Diagnostics.AddError("Unable to Query VM State", err.Error())
		return
	}

	if vm.Status.State == VMStateRunning {
		stopOpts := map[string]any{"force": true, "force_after_timeout": true}
		_, err := r.client.CallAndWait(ctx, "vm.stop", []any{vmID, stopOpts})
		if err != nil {
			resp.Diagnostics.AddError("Unable to Stop VM", fmt.Sprintf("Unable to stop VM before delete: %s", err.Error()))
			return
		}
	}

	// Delete the VM
	_, err = r.client.Call(ctx, "vm.delete", vmID)
	if err != nil {
		resp.Diagnostics.AddError("Unable to Delete VM", err.Error())
		return
	}
}

func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}

// -- Helpers --

func (r *VMResource) buildCreateParams(data *VMResourceModel) map[string]any {
	params := map[string]any{
		"name":   data.Name.ValueString(),
		"memory": data.Memory.ValueInt64(),
	}

	if !data.Description.IsNull() && !data.Description.IsUnknown() {
		params["description"] = data.Description.ValueString()
	}
	if !data.VCPUs.IsNull() && !data.VCPUs.IsUnknown() {
		params["vcpus"] = data.VCPUs.ValueInt64()
	}
	if !data.Cores.IsNull() && !data.Cores.IsUnknown() {
		params["cores"] = data.Cores.ValueInt64()
	}
	if !data.Threads.IsNull() && !data.Threads.IsUnknown() {
		params["threads"] = data.Threads.ValueInt64()
	}
	if !data.MinMemory.IsNull() && !data.MinMemory.IsUnknown() {
		params["min_memory"] = data.MinMemory.ValueInt64()
	}
	if !data.Autostart.IsNull() && !data.Autostart.IsUnknown() {
		params["autostart"] = data.Autostart.ValueBool()
	}
	if !data.Time.IsNull() && !data.Time.IsUnknown() {
		params["time"] = data.Time.ValueString()
	}
	if !data.Bootloader.IsNull() && !data.Bootloader.IsUnknown() {
		params["bootloader"] = data.Bootloader.ValueString()
	}
	if !data.BootloaderOVMF.IsNull() && !data.BootloaderOVMF.IsUnknown() {
		params["bootloader_ovmf"] = data.BootloaderOVMF.ValueString()
	}
	if !data.CPUMode.IsNull() && !data.CPUMode.IsUnknown() {
		params["cpu_mode"] = data.CPUMode.ValueString()
	}
	if !data.CPUModel.IsNull() && !data.CPUModel.IsUnknown() {
		params["cpu_model"] = data.CPUModel.ValueString()
	}
	if !data.ShutdownTimeout.IsNull() && !data.ShutdownTimeout.IsUnknown() {
		params["shutdown_timeout"] = data.ShutdownTimeout.ValueInt64()
	}

	return params
}

func (r *VMResource) buildUpdateParams(plan, state *VMResourceModel) map[string]any {
	params := map[string]any{}

	if !plan.Name.Equal(state.Name) {
		params["name"] = plan.Name.ValueString()
	}
	if !plan.Description.Equal(state.Description) {
		params["description"] = plan.Description.ValueString()
	}
	if !plan.VCPUs.Equal(state.VCPUs) {
		params["vcpus"] = plan.VCPUs.ValueInt64()
	}
	if !plan.Cores.Equal(state.Cores) {
		params["cores"] = plan.Cores.ValueInt64()
	}
	if !plan.Threads.Equal(state.Threads) {
		params["threads"] = plan.Threads.ValueInt64()
	}
	if !plan.Memory.Equal(state.Memory) {
		params["memory"] = plan.Memory.ValueInt64()
	}
	if !plan.MinMemory.Equal(state.MinMemory) {
		if plan.MinMemory.IsNull() {
			params["min_memory"] = nil
		} else {
			params["min_memory"] = plan.MinMemory.ValueInt64()
		}
	}
	if !plan.Autostart.Equal(state.Autostart) {
		params["autostart"] = plan.Autostart.ValueBool()
	}
	if !plan.Time.Equal(state.Time) {
		params["time"] = plan.Time.ValueString()
	}
	if !plan.Bootloader.Equal(state.Bootloader) {
		params["bootloader"] = plan.Bootloader.ValueString()
	}
	if !plan.BootloaderOVMF.Equal(state.BootloaderOVMF) {
		params["bootloader_ovmf"] = plan.BootloaderOVMF.ValueString()
	}
	if !plan.CPUMode.Equal(state.CPUMode) {
		params["cpu_mode"] = plan.CPUMode.ValueString()
	}
	if !plan.CPUModel.Equal(state.CPUModel) {
		if plan.CPUModel.IsNull() {
			params["cpu_model"] = nil
		} else {
			params["cpu_model"] = plan.CPUModel.ValueString()
		}
	}
	if !plan.ShutdownTimeout.Equal(state.ShutdownTimeout) {
		params["shutdown_timeout"] = plan.ShutdownTimeout.ValueInt64()
	}

	return params
}

// getVM retrieves a VM by ID using vm.get_instance.
func (r *VMResource) getVM(ctx context.Context, id int64) (*vmAPIResponse, error) {
	result, err := r.client.Call(ctx, "vm.get_instance", id)
	if err != nil {
		return nil, err
	}

	var vm vmAPIResponse
	if err := json.Unmarshal(result, &vm); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	return &vm, nil
}

// mapVMToModel maps an API response to the resource model.
func (r *VMResource) mapVMToModel(vm *vmAPIResponse, data *VMResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(vm.ID, 10))
	data.Name = types.StringValue(vm.Name)
	data.Description = types.StringValue(vm.Description)
	data.VCPUs = types.Int64Value(vm.VCPUs)
	data.Cores = types.Int64Value(vm.Cores)
	data.Threads = types.Int64Value(vm.Threads)
	data.Memory = types.Int64Value(vm.Memory)
	if vm.MinMemory != nil {
		data.MinMemory = types.Int64Value(*vm.MinMemory)
	} else {
		data.MinMemory = types.Int64Null()
	}
	data.Autostart = types.BoolValue(vm.Autostart)
	data.Time = types.StringValue(vm.Time)
	data.Bootloader = types.StringValue(vm.Bootloader)
	data.BootloaderOVMF = types.StringValue(vm.BootloaderOVMF)
	data.CPUMode = types.StringValue(vm.CPUMode)
	if vm.CPUModel != nil {
		data.CPUModel = types.StringValue(*vm.CPUModel)
	} else {
		data.CPUModel = types.StringNull()
	}
	data.ShutdownTimeout = types.Int64Value(vm.ShutdownTimeout)
	data.State = types.StringValue(vm.Status.State)
	data.DisplayAvailable = types.BoolValue(vm.DisplayAvailable)
}

// queryVMDevices queries devices for a VM using vm.device.query.
func (r *VMResource) queryVMDevices(ctx context.Context, vmID int64) ([]vmDeviceAPIResponse, error) {
	filter := []any{[]any{[]any{"vm", "=", vmID}}}
	result, err := r.client.Call(ctx, "vm.device.query", filter)
	if err != nil {
		return nil, err
	}

	var devices []vmDeviceAPIResponse
	if err := json.Unmarshal(result, &devices); err != nil {
		return nil, fmt.Errorf("parse devices: %w", err)
	}

	return devices, nil
}

// mapDevicesToModel maps API device responses to the resource model.
func (r *VMResource) mapDevicesToModel(devices []vmDeviceAPIResponse, data *VMResourceModel) {
	data.Disks = nil
	data.Raws = nil
	data.CDROMs = nil
	data.NICs = nil
	data.Displays = nil
	data.PCIs = nil
	data.USBs = nil

	for _, dev := range devices {
		dtype, _ := dev.Attributes["dtype"].(string)
		switch dtype {
		case "DISK":
			data.Disks = append(data.Disks, mapDiskDevice(dev))
		case "RAW":
			data.Raws = append(data.Raws, mapRawDevice(dev))
		case "CDROM":
			data.CDROMs = append(data.CDROMs, mapCDROMDevice(dev))
		case "NIC":
			data.NICs = append(data.NICs, mapNICDevice(dev))
		case "DISPLAY":
			data.Displays = append(data.Displays, mapDisplayDevice(dev))
		case "PCI":
			data.PCIs = append(data.PCIs, mapPCIDevice(dev))
		case "USB":
			data.USBs = append(data.USBs, mapUSBDevice(dev))
		}
	}
}

func mapDiskDevice(dev vmDeviceAPIResponse) VMDiskModel {
	m := VMDiskModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Path = stringAttrFromMap(dev.Attributes, "path")
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.IOType = stringAttrFromMap(dev.Attributes, "iotype")
	m.Serial = stringAttrFromMap(dev.Attributes, "serial")
	m.LogicalSectorSize = intAttrFromMap(dev.Attributes, "logical_sectorsize")
	m.PhysicalSectorSize = intAttrFromMap(dev.Attributes, "physical_sectorsize")
	return m
}

func mapRawDevice(dev vmDeviceAPIResponse) VMRawModel {
	m := VMRawModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Path = stringAttrFromMap(dev.Attributes, "path")
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.IOType = stringAttrFromMap(dev.Attributes, "iotype")
	m.Serial = stringAttrFromMap(dev.Attributes, "serial")
	m.Boot = boolAttrFromMap(dev.Attributes, "boot")
	m.Size = intAttrFromMap(dev.Attributes, "size")
	m.LogicalSectorSize = intAttrFromMap(dev.Attributes, "logical_sectorsize")
	m.PhysicalSectorSize = intAttrFromMap(dev.Attributes, "physical_sectorsize")
	return m
}

func mapCDROMDevice(dev vmDeviceAPIResponse) VMCDROMModel {
	return VMCDROMModel{
		DeviceID: types.Int64Value(dev.ID),
		Path:     stringAttrFromMap(dev.Attributes, "path"),
		Order:    types.Int64Value(dev.Order),
	}
}

func mapNICDevice(dev vmDeviceAPIResponse) VMNICModel {
	m := VMNICModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.NICAttach = stringAttrFromMap(dev.Attributes, "nic_attach")
	m.MAC = stringAttrFromMap(dev.Attributes, "mac")
	m.TrustGuestRXFilters = boolAttrFromMap(dev.Attributes, "trust_guest_rx_filters")
	return m
}

func mapDisplayDevice(dev vmDeviceAPIResponse) VMDisplayModel {
	m := VMDisplayModel{
		DeviceID: types.Int64Value(dev.ID),
		Order:    types.Int64Value(dev.Order),
	}
	m.Type = stringAttrFromMap(dev.Attributes, "type")
	m.Resolution = stringAttrFromMap(dev.Attributes, "resolution")
	m.Port = intAttrFromMap(dev.Attributes, "port")
	m.WebPort = intAttrFromMap(dev.Attributes, "web_port")
	m.Bind = stringAttrFromMap(dev.Attributes, "bind")
	m.Wait = boolAttrFromMap(dev.Attributes, "wait")
	m.Password = stringAttrFromMap(dev.Attributes, "password")
	m.Web = boolAttrFromMap(dev.Attributes, "web")
	return m
}

func mapPCIDevice(dev vmDeviceAPIResponse) VMPCIModel {
	return VMPCIModel{
		DeviceID: types.Int64Value(dev.ID),
		PPTDev:   stringAttrFromMap(dev.Attributes, "pptdev"),
		Order:    types.Int64Value(dev.Order),
	}
}

func mapUSBDevice(dev vmDeviceAPIResponse) VMUSBModel {
	return VMUSBModel{
		DeviceID:       types.Int64Value(dev.ID),
		ControllerType: stringAttrFromMap(dev.Attributes, "controller_type"),
		Device:         stringAttrFromMap(dev.Attributes, "device"),
		Order:          types.Int64Value(dev.Order),
	}
}

// Attribute extraction helpers

func stringAttrFromMap(m map[string]any, key string) types.String {
	v, ok := m[key]
	if !ok || v == nil {
		return types.StringNull()
	}
	s, ok := v.(string)
	if !ok {
		return types.StringNull()
	}
	return types.StringValue(s)
}

func intAttrFromMap(m map[string]any, key string) types.Int64 {
	v, ok := m[key]
	if !ok || v == nil {
		return types.Int64Null()
	}
	switch n := v.(type) {
	case float64:
		return types.Int64Value(int64(n))
	case int64:
		return types.Int64Value(n)
	case json.Number:
		i, _ := n.Int64()
		return types.Int64Value(i)
	}
	return types.Int64Null()
}

func boolAttrFromMap(m map[string]any, key string) types.Bool {
	v, ok := m[key]
	if !ok || v == nil {
		return types.BoolNull()
	}
	b, ok := v.(bool)
	if !ok {
		return types.BoolNull()
	}
	return types.BoolValue(b)
}

// -- Device param builders --

func buildDiskDeviceParams(disk *VMDiskModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "DISK"}
	if !disk.Path.IsNull() {
		attrs["path"] = disk.Path.ValueString()
	}
	if !disk.Type.IsNull() && !disk.Type.IsUnknown() {
		attrs["type"] = disk.Type.ValueString()
	}
	if !disk.LogicalSectorSize.IsNull() {
		attrs["logical_sectorsize"] = disk.LogicalSectorSize.ValueInt64()
	}
	if !disk.PhysicalSectorSize.IsNull() {
		attrs["physical_sectorsize"] = disk.PhysicalSectorSize.ValueInt64()
	}
	if !disk.IOType.IsNull() && !disk.IOType.IsUnknown() {
		attrs["iotype"] = disk.IOType.ValueString()
	}
	if !disk.Serial.IsNull() {
		attrs["serial"] = disk.Serial.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !disk.Order.IsNull() && !disk.Order.IsUnknown() {
		params["order"] = disk.Order.ValueInt64()
	}
	return params
}

func buildRawDeviceParams(raw *VMRawModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "RAW"}
	if !raw.Path.IsNull() {
		attrs["path"] = raw.Path.ValueString()
	}
	if !raw.Type.IsNull() && !raw.Type.IsUnknown() {
		attrs["type"] = raw.Type.ValueString()
	}
	if !raw.Boot.IsNull() && !raw.Boot.IsUnknown() {
		attrs["boot"] = raw.Boot.ValueBool()
	}
	if !raw.Size.IsNull() {
		attrs["size"] = raw.Size.ValueInt64()
	}
	if !raw.LogicalSectorSize.IsNull() {
		attrs["logical_sectorsize"] = raw.LogicalSectorSize.ValueInt64()
	}
	if !raw.PhysicalSectorSize.IsNull() {
		attrs["physical_sectorsize"] = raw.PhysicalSectorSize.ValueInt64()
	}
	if !raw.IOType.IsNull() && !raw.IOType.IsUnknown() {
		attrs["iotype"] = raw.IOType.ValueString()
	}
	if !raw.Serial.IsNull() {
		attrs["serial"] = raw.Serial.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !raw.Order.IsNull() && !raw.Order.IsUnknown() {
		params["order"] = raw.Order.ValueInt64()
	}
	return params
}

func buildCDROMDeviceParams(cdrom *VMCDROMModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "CDROM"}
	if !cdrom.Path.IsNull() {
		attrs["path"] = cdrom.Path.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !cdrom.Order.IsNull() && !cdrom.Order.IsUnknown() {
		params["order"] = cdrom.Order.ValueInt64()
	}
	return params
}

func buildNICDeviceParams(nic *VMNICModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "NIC"}
	if !nic.Type.IsNull() && !nic.Type.IsUnknown() {
		attrs["type"] = nic.Type.ValueString()
	}
	if !nic.NICAttach.IsNull() {
		attrs["nic_attach"] = nic.NICAttach.ValueString()
	}
	if !nic.MAC.IsNull() && !nic.MAC.IsUnknown() {
		attrs["mac"] = nic.MAC.ValueString()
	}
	if !nic.TrustGuestRXFilters.IsNull() && !nic.TrustGuestRXFilters.IsUnknown() {
		attrs["trust_guest_rx_filters"] = nic.TrustGuestRXFilters.ValueBool()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !nic.Order.IsNull() && !nic.Order.IsUnknown() {
		params["order"] = nic.Order.ValueInt64()
	}
	return params
}

func buildDisplayDeviceParams(display *VMDisplayModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "DISPLAY"}
	if !display.Type.IsNull() && !display.Type.IsUnknown() {
		attrs["type"] = display.Type.ValueString()
	}
	if !display.Resolution.IsNull() && !display.Resolution.IsUnknown() {
		attrs["resolution"] = display.Resolution.ValueString()
	}
	if !display.Port.IsNull() && !display.Port.IsUnknown() {
		attrs["port"] = display.Port.ValueInt64()
	}
	if !display.WebPort.IsNull() && !display.WebPort.IsUnknown() {
		attrs["web_port"] = display.WebPort.ValueInt64()
	}
	if !display.Bind.IsNull() && !display.Bind.IsUnknown() {
		attrs["bind"] = display.Bind.ValueString()
	}
	if !display.Wait.IsNull() && !display.Wait.IsUnknown() {
		attrs["wait"] = display.Wait.ValueBool()
	}
	if !display.Password.IsNull() && !display.Password.IsUnknown() {
		attrs["password"] = display.Password.ValueString()
	}
	if !display.Web.IsNull() && !display.Web.IsUnknown() {
		attrs["web"] = display.Web.ValueBool()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !display.Order.IsNull() && !display.Order.IsUnknown() {
		params["order"] = display.Order.ValueInt64()
	}
	return params
}

func buildPCIDeviceParams(pci *VMPCIModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "PCI"}
	if !pci.PPTDev.IsNull() {
		attrs["pptdev"] = pci.PPTDev.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !pci.Order.IsNull() && !pci.Order.IsUnknown() {
		params["order"] = pci.Order.ValueInt64()
	}
	return params
}

func buildUSBDeviceParams(usb *VMUSBModel, vmID int64) map[string]any {
	attrs := map[string]any{"dtype": "USB"}
	if !usb.ControllerType.IsNull() && !usb.ControllerType.IsUnknown() {
		attrs["controller_type"] = usb.ControllerType.ValueString()
	}
	if !usb.Device.IsNull() {
		attrs["device"] = usb.Device.ValueString()
	}

	params := map[string]any{"vm": vmID, "attributes": attrs}
	if !usb.Order.IsNull() && !usb.Order.IsUnknown() {
		params["order"] = usb.Order.ValueInt64()
	}
	return params
}

// setDeviceIDFromResult extracts the device ID from a vm.device.create response.
func (r *VMResource) setDeviceIDFromResult(result json.RawMessage, target *types.Int64) {
	var devResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &devResp); err == nil && devResp.ID != 0 {
		*target = types.Int64Value(devResp.ID)
	}
}

// -- Device reconciliation --

// reconcileDevices compares plan vs state devices and creates/updates/deletes as needed.
func (r *VMResource) reconcileDevices(ctx context.Context, vmID int64, plan, state *VMResourceModel) error {
	// Build maps of state device IDs to detect what exists
	stateDeviceIDs := make(map[int64]bool)
	collectDeviceIDs(stateDeviceIDs, state)

	planDeviceIDs := make(map[int64]bool)
	collectDeviceIDs(planDeviceIDs, plan)

	// Delete devices in state but not in plan
	if err := r.deleteRemovedDevices(ctx, stateDeviceIDs, planDeviceIDs); err != nil {
		return err
	}

	// Create/update devices
	if err := r.reconcileDiskDevices(ctx, vmID, plan.Disks, state.Disks); err != nil {
		return err
	}
	if err := r.reconcileRawDevices(ctx, vmID, plan.Raws, state.Raws); err != nil {
		return err
	}
	if err := r.reconcileCDROMDevices(ctx, vmID, plan.CDROMs, state.CDROMs); err != nil {
		return err
	}
	if err := r.reconcileNICDevices(ctx, vmID, plan.NICs, state.NICs); err != nil {
		return err
	}
	if err := r.reconcileDisplayDevices(ctx, vmID, plan.Displays, state.Displays); err != nil {
		return err
	}
	if err := r.reconcilePCIDevices(ctx, vmID, plan.PCIs, state.PCIs); err != nil {
		return err
	}
	if err := r.reconcileUSBDevices(ctx, vmID, plan.USBs, state.USBs); err != nil {
		return err
	}

	return nil
}

func collectDeviceIDs(ids map[int64]bool, data *VMResourceModel) {
	for _, d := range data.Disks {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.Raws {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.CDROMs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.NICs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.Displays {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.PCIs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
	for _, d := range data.USBs {
		if !d.DeviceID.IsNull() && !d.DeviceID.IsUnknown() {
			ids[d.DeviceID.ValueInt64()] = true
		}
	}
}

func (r *VMResource) deleteRemovedDevices(ctx context.Context, stateIDs, planIDs map[int64]bool) error {
	for id := range stateIDs {
		if !planIDs[id] {
			_, err := r.client.Call(ctx, "vm.device.delete", id)
			if err != nil {
				return fmt.Errorf("failed to delete device %d: %w", id, err)
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileDiskDevices(ctx context.Context, vmID int64, plan, state []VMDiskModel) error {
	stateByID := make(map[int64]VMDiskModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}

	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			// New device - create
			_, err := r.client.Call(ctx, "vm.device.create", buildDiskDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create disk device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			// Existing device - update if changed
			if !diskEqual(p, s) {
				params := buildDiskDeviceParams(&p, vmID)
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), params})
				if err != nil {
					return fmt.Errorf("failed to update disk device %d: %w", p.DeviceID.ValueInt64(), err)
				}
			}
		}
	}
	return nil
}

func diskEqual(a, b VMDiskModel) bool {
	return a.Path.Equal(b.Path) && a.Type.Equal(b.Type) &&
		a.LogicalSectorSize.Equal(b.LogicalSectorSize) &&
		a.PhysicalSectorSize.Equal(b.PhysicalSectorSize) &&
		a.IOType.Equal(b.IOType) && a.Serial.Equal(b.Serial)
}

func (r *VMResource) reconcileRawDevices(ctx context.Context, vmID int64, plan, state []VMRawModel) error {
	stateByID := make(map[int64]VMRawModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildRawDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create raw device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !rawEqual(p, s) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildRawDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update raw device: %w", err)
				}
			}
		}
	}
	return nil
}

func rawEqual(a, b VMRawModel) bool {
	return a.Path.Equal(b.Path) && a.Type.Equal(b.Type) && a.Boot.Equal(b.Boot) && a.Size.Equal(b.Size)
}

func (r *VMResource) reconcileCDROMDevices(ctx context.Context, vmID int64, plan, state []VMCDROMModel) error {
	stateByID := make(map[int64]VMCDROMModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildCDROMDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create cdrom device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !p.Path.Equal(s.Path) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildCDROMDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update cdrom device: %w", err)
				}
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileNICDevices(ctx context.Context, vmID int64, plan, state []VMNICModel) error {
	stateByID := make(map[int64]VMNICModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildNICDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create nic device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !nicEqual(p, s) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildNICDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update nic device: %w", err)
				}
			}
		}
	}
	return nil
}

func nicEqual(a, b VMNICModel) bool {
	return a.Type.Equal(b.Type) && a.NICAttach.Equal(b.NICAttach) && a.MAC.Equal(b.MAC) && a.TrustGuestRXFilters.Equal(b.TrustGuestRXFilters)
}

func (r *VMResource) reconcileDisplayDevices(ctx context.Context, vmID int64, plan, state []VMDisplayModel) error {
	stateByID := make(map[int64]VMDisplayModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildDisplayDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create display device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !displayEqual(p, s) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildDisplayDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update display device: %w", err)
				}
			}
		}
	}
	return nil
}

func displayEqual(a, b VMDisplayModel) bool {
	return a.Type.Equal(b.Type) && a.Resolution.Equal(b.Resolution) && a.Bind.Equal(b.Bind) &&
		a.Web.Equal(b.Web) && a.Wait.Equal(b.Wait) && a.Port.Equal(b.Port) && a.WebPort.Equal(b.WebPort)
}

func (r *VMResource) reconcilePCIDevices(ctx context.Context, vmID int64, plan, state []VMPCIModel) error {
	stateByID := make(map[int64]VMPCIModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildPCIDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create pci device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !p.PPTDev.Equal(s.PPTDev) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildPCIDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update pci device: %w", err)
				}
			}
		}
	}
	return nil
}

func (r *VMResource) reconcileUSBDevices(ctx context.Context, vmID int64, plan, state []VMUSBModel) error {
	stateByID := make(map[int64]VMUSBModel)
	for _, s := range state {
		if !s.DeviceID.IsNull() && !s.DeviceID.IsUnknown() {
			stateByID[s.DeviceID.ValueInt64()] = s
		}
	}
	for _, p := range plan {
		if p.DeviceID.IsNull() || p.DeviceID.IsUnknown() {
			_, err := r.client.Call(ctx, "vm.device.create", buildUSBDeviceParams(&p, vmID))
			if err != nil {
				return fmt.Errorf("failed to create usb device: %w", err)
			}
		} else if s, ok := stateByID[p.DeviceID.ValueInt64()]; ok {
			if !usbEqual(p, s) {
				_, err := r.client.Call(ctx, "vm.device.update", []any{p.DeviceID.ValueInt64(), buildUSBDeviceParams(&p, vmID)})
				if err != nil {
					return fmt.Errorf("failed to update usb device: %w", err)
				}
			}
		}
	}
	return nil
}

func usbEqual(a, b VMUSBModel) bool {
	return a.ControllerType.Equal(b.ControllerType) && a.Device.Equal(b.Device)
}

// -- State management --

// reconcileState starts or stops the VM to match the desired state.
// vm.start is NOT a job. vm.stop IS a job (use CallAndWait).
func (r *VMResource) reconcileState(ctx context.Context, vmID int64, currentState, desiredState string) error {
	if currentState == desiredState {
		return nil
	}

	if desiredState == VMStateRunning {
		_, err := r.client.Call(ctx, "vm.start", vmID)
		return err
	}

	// vm.stop is a job
	stopOpts := map[string]any{"force": false, "force_after_timeout": true}
	_, err := r.client.CallAndWait(ctx, "vm.stop", []any{vmID, stopOpts})
	return err
}
