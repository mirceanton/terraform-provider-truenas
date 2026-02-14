package resources

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

// -- Test helpers --

func getVMResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewVMResource()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", resp.Diagnostics)
	}
	return *resp
}

// Block type helpers for tftypes.

func vmDiskBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":           tftypes.Number,
		"path":                tftypes.String,
		"type":                tftypes.String,
		"logical_sectorsize":  tftypes.Number,
		"physical_sectorsize": tftypes.Number,
		"iotype":              tftypes.String,
		"serial":              tftypes.String,
		"order":               tftypes.Number,
	}}
}

func vmRawBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":           tftypes.Number,
		"path":                tftypes.String,
		"type":                tftypes.String,
		"boot":                tftypes.Bool,
		"size":                tftypes.Number,
		"logical_sectorsize":  tftypes.Number,
		"physical_sectorsize": tftypes.Number,
		"iotype":              tftypes.String,
		"serial":              tftypes.String,
		"order":               tftypes.Number,
	}}
}

func vmCDROMBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id": tftypes.Number,
		"path":      tftypes.String,
		"order":     tftypes.Number,
	}}
}

func vmNICBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":              tftypes.Number,
		"type":                   tftypes.String,
		"nic_attach":             tftypes.String,
		"mac":                    tftypes.String,
		"trust_guest_rx_filters": tftypes.Bool,
		"order":                  tftypes.Number,
	}}
}

func vmDisplayBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":  tftypes.Number,
		"type":       tftypes.String,
		"resolution": tftypes.String,
		"port":       tftypes.Number,
		"web_port":   tftypes.Number,
		"bind":       tftypes.String,
		"wait":       tftypes.Bool,
		"password":   tftypes.String,
		"web":        tftypes.Bool,
		"order":      tftypes.Number,
	}}
}

func vmPCIBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id": tftypes.Number,
		"pptdev":    tftypes.String,
		"order":     tftypes.Number,
	}}
}

func vmUSBBlockType() tftypes.Object {
	return tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"device_id":       tftypes.Number,
		"controller_type": tftypes.String,
		"device":          tftypes.String,
		"order":           tftypes.Number,
	}}
}

// vmObjectType returns the full tftypes.Object type for the VM resource model.
func vmObjectType() tftypes.Object {
	return tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":                tftypes.String,
			"name":              tftypes.String,
			"description":       tftypes.String,
			"vcpus":             tftypes.Number,
			"cores":             tftypes.Number,
			"threads":           tftypes.Number,
			"memory":            tftypes.Number,
			"min_memory":        tftypes.Number,
			"autostart":         tftypes.Bool,
			"time":              tftypes.String,
			"bootloader":        tftypes.String,
			"bootloader_ovmf":   tftypes.String,
			"cpu_mode":          tftypes.String,
			"cpu_model":         tftypes.String,
			"shutdown_timeout":  tftypes.Number,
			"state":             tftypes.String,
			"display_available": tftypes.Bool,
			"disk":              tftypes.List{ElementType: vmDiskBlockType()},
			"raw":               tftypes.List{ElementType: vmRawBlockType()},
			"cdrom":             tftypes.List{ElementType: vmCDROMBlockType()},
			"nic":               tftypes.List{ElementType: vmNICBlockType()},
			"display":           tftypes.List{ElementType: vmDisplayBlockType()},
			"pci":               tftypes.List{ElementType: vmPCIBlockType()},
			"usb":               tftypes.List{ElementType: vmUSBBlockType()},
		},
	}
}

type vmModelParams struct {
	ID               interface{}
	Name             interface{}
	Description      interface{}
	VCPUs            interface{}
	Cores            interface{}
	Threads          interface{}
	Memory           interface{}
	MinMemory        interface{}
	Autostart        interface{}
	Time             interface{}
	Bootloader       interface{}
	BootloaderOVMF   interface{}
	CPUMode          interface{}
	CPUModel         interface{}
	ShutdownTimeout  interface{}
	State            interface{}
	DisplayAvailable interface{}
	Disks            []vmDiskParams
	NICs             []vmNICParams
	CDROMs           []vmCDROMParams
	Displays         []vmDisplayParams
}

type vmDiskParams struct {
	DeviceID           interface{}
	Path               interface{}
	Type               interface{}
	LogicalSectorSize  interface{}
	PhysicalSectorSize interface{}
	IOType             interface{}
	Serial             interface{}
	Order              interface{}
}

type vmNICParams struct {
	DeviceID            interface{}
	Type                interface{}
	NICAttach           interface{}
	MAC                 interface{}
	TrustGuestRXFilters interface{}
	Order               interface{}
}

type vmCDROMParams struct {
	DeviceID interface{}
	Path     interface{}
	Order    interface{}
}

type vmDisplayParams struct {
	DeviceID   interface{}
	Type       interface{}
	Resolution interface{}
	Port       interface{}
	WebPort    interface{}
	Bind       interface{}
	Wait       interface{}
	Password   interface{}
	Web        interface{}
	Order      interface{}
}

func emptyBlockList(elemType tftypes.Object) tftypes.Value {
	return tftypes.NewValue(tftypes.List{ElementType: elemType}, []tftypes.Value{})
}

func createVMModelValue(p vmModelParams) tftypes.Value {
	// Build disk block values
	var diskValues []tftypes.Value
	for _, d := range p.Disks {
		diskValues = append(diskValues, tftypes.NewValue(vmDiskBlockType(), map[string]tftypes.Value{
			"device_id":           tftypes.NewValue(tftypes.Number, d.DeviceID),
			"path":                tftypes.NewValue(tftypes.String, d.Path),
			"type":                tftypes.NewValue(tftypes.String, d.Type),
			"logical_sectorsize":  tftypes.NewValue(tftypes.Number, d.LogicalSectorSize),
			"physical_sectorsize": tftypes.NewValue(tftypes.Number, d.PhysicalSectorSize),
			"iotype":              tftypes.NewValue(tftypes.String, d.IOType),
			"serial":              tftypes.NewValue(tftypes.String, d.Serial),
			"order":               tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	diskList := emptyBlockList(vmDiskBlockType())
	if len(diskValues) > 0 {
		diskList = tftypes.NewValue(tftypes.List{ElementType: vmDiskBlockType()}, diskValues)
	}

	// Build NIC block values
	var nicValues []tftypes.Value
	for _, n := range p.NICs {
		nicValues = append(nicValues, tftypes.NewValue(vmNICBlockType(), map[string]tftypes.Value{
			"device_id":              tftypes.NewValue(tftypes.Number, n.DeviceID),
			"type":                   tftypes.NewValue(tftypes.String, n.Type),
			"nic_attach":             tftypes.NewValue(tftypes.String, n.NICAttach),
			"mac":                    tftypes.NewValue(tftypes.String, n.MAC),
			"trust_guest_rx_filters": tftypes.NewValue(tftypes.Bool, n.TrustGuestRXFilters),
			"order":                  tftypes.NewValue(tftypes.Number, n.Order),
		}))
	}
	nicList := emptyBlockList(vmNICBlockType())
	if len(nicValues) > 0 {
		nicList = tftypes.NewValue(tftypes.List{ElementType: vmNICBlockType()}, nicValues)
	}

	// Build CDROM block values
	var cdromValues []tftypes.Value
	for _, c := range p.CDROMs {
		cdromValues = append(cdromValues, tftypes.NewValue(vmCDROMBlockType(), map[string]tftypes.Value{
			"device_id": tftypes.NewValue(tftypes.Number, c.DeviceID),
			"path":      tftypes.NewValue(tftypes.String, c.Path),
			"order":     tftypes.NewValue(tftypes.Number, c.Order),
		}))
	}
	cdromList := emptyBlockList(vmCDROMBlockType())
	if len(cdromValues) > 0 {
		cdromList = tftypes.NewValue(tftypes.List{ElementType: vmCDROMBlockType()}, cdromValues)
	}

	// Build Display block values
	var displayValues []tftypes.Value
	for _, d := range p.Displays {
		displayValues = append(displayValues, tftypes.NewValue(vmDisplayBlockType(), map[string]tftypes.Value{
			"device_id":  tftypes.NewValue(tftypes.Number, d.DeviceID),
			"type":       tftypes.NewValue(tftypes.String, d.Type),
			"resolution": tftypes.NewValue(tftypes.String, d.Resolution),
			"port":       tftypes.NewValue(tftypes.Number, d.Port),
			"web_port":   tftypes.NewValue(tftypes.Number, d.WebPort),
			"bind":       tftypes.NewValue(tftypes.String, d.Bind),
			"wait":       tftypes.NewValue(tftypes.Bool, d.Wait),
			"password":   tftypes.NewValue(tftypes.String, d.Password),
			"web":        tftypes.NewValue(tftypes.Bool, d.Web),
			"order":      tftypes.NewValue(tftypes.Number, d.Order),
		}))
	}
	displayList := emptyBlockList(vmDisplayBlockType())
	if len(displayValues) > 0 {
		displayList = tftypes.NewValue(tftypes.List{ElementType: vmDisplayBlockType()}, displayValues)
	}

	values := map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, p.ID),
		"name":              tftypes.NewValue(tftypes.String, p.Name),
		"description":       tftypes.NewValue(tftypes.String, p.Description),
		"vcpus":             tftypes.NewValue(tftypes.Number, p.VCPUs),
		"cores":             tftypes.NewValue(tftypes.Number, p.Cores),
		"threads":           tftypes.NewValue(tftypes.Number, p.Threads),
		"memory":            tftypes.NewValue(tftypes.Number, p.Memory),
		"min_memory":        tftypes.NewValue(tftypes.Number, p.MinMemory),
		"autostart":         tftypes.NewValue(tftypes.Bool, p.Autostart),
		"time":              tftypes.NewValue(tftypes.String, p.Time),
		"bootloader":        tftypes.NewValue(tftypes.String, p.Bootloader),
		"bootloader_ovmf":   tftypes.NewValue(tftypes.String, p.BootloaderOVMF),
		"cpu_mode":          tftypes.NewValue(tftypes.String, p.CPUMode),
		"cpu_model":         tftypes.NewValue(tftypes.String, p.CPUModel),
		"shutdown_timeout":  tftypes.NewValue(tftypes.Number, p.ShutdownTimeout),
		"state":             tftypes.NewValue(tftypes.String, p.State),
		"display_available": tftypes.NewValue(tftypes.Bool, p.DisplayAvailable),
		"disk":              diskList,
		"raw":               emptyBlockList(vmRawBlockType()),
		"cdrom":             cdromList,
		"nic":               nicList,
		"display":           displayList,
		"pci":               emptyBlockList(vmPCIBlockType()),
		"usb":               emptyBlockList(vmUSBBlockType()),
	}

	return tftypes.NewValue(vmObjectType(), values)
}

// mockVMResponse generates a valid vm.get_instance JSON response.
func mockVMResponse(id int64, name string, memory int64, state string) json.RawMessage {
	return json.RawMessage(fmt.Sprintf(`{
		"id": %d,
		"name": %q,
		"description": "",
		"vcpus": 1,
		"cores": 1,
		"threads": 1,
		"memory": %d,
		"min_memory": null,
		"autostart": true,
		"time": "LOCAL",
		"bootloader": "UEFI",
		"bootloader_ovmf": "OVMF_CODE.fd",
		"cpu_mode": "CUSTOM",
		"cpu_model": null,
		"shutdown_timeout": 90,
		"status": {"state": %q, "pid": null, "domain_state": "SHUTOFF"},
		"display_available": false,
		"devices": []
	}`, id, name, memory, state))
}

// mockVMDevicesResponse generates a vm.device.query response.
func mockVMDevicesResponse(devices ...map[string]any) json.RawMessage {
	data, _ := json.Marshal(devices)
	return json.RawMessage(data)
}

// defaultVMPlanParams returns params for a minimal VM plan.
func defaultVMPlanParams() vmModelParams {
	return vmModelParams{
		Name:            "test-vm",
		Description:     "",
		VCPUs:           float64(1),
		Cores:           float64(1),
		Threads:         float64(1),
		Memory:          float64(2048),
		MinMemory:       nil,
		Autostart:       true,
		Time:            "LOCAL",
		Bootloader:      "UEFI",
		BootloaderOVMF:  "OVMF_CODE.fd",
		CPUMode:         "CUSTOM",
		CPUModel:        nil,
		ShutdownTimeout: float64(90),
		State:           "STOPPED",
		DisplayAvailable: nil,
	}
}

// -- Scaffold tests --

func TestNewVMResource(t *testing.T) {
	r := NewVMResource()
	if r == nil {
		t.Fatal("NewVMResource returned nil")
	}

	vmResource, ok := r.(*VMResource)
	if !ok {
		t.Fatalf("expected *VMResource, got %T", r)
	}

	_ = resource.Resource(r)
	_ = resource.ResourceWithConfigure(vmResource)
	_ = resource.ResourceWithImportState(vmResource)
}

func TestVMResource_Metadata(t *testing.T) {
	r := NewVMResource()
	req := resource.MetadataRequest{ProviderTypeName: "truenas"}
	resp := &resource.MetadataResponse{}
	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_vm" {
		t.Errorf("expected TypeName 'truenas_vm', got %q", resp.TypeName)
	}
}

func TestVMResource_Schema(t *testing.T) {
	r := NewVMResource()
	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	attrs := schemaResp.Schema.Attributes

	for _, name := range []string{"name", "memory"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsRequired() {
			t.Errorf("expected %q to be required", name)
		}
	}

	for _, name := range []string{"id", "display_available"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed", name)
		}
	}

	for _, name := range []string{
		"description", "vcpus", "cores", "threads", "autostart", "time",
		"bootloader", "bootloader_ovmf", "cpu_mode", "cpu_model",
		"shutdown_timeout", "state",
	} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsOptional() {
			t.Errorf("expected %q to be optional", name)
		}
	}

	blocks := schemaResp.Schema.Blocks
	for _, name := range []string{"disk", "raw", "cdrom", "nic", "display", "pci", "usb"} {
		if _, ok := blocks[name]; !ok {
			t.Errorf("expected %q block", name)
		}
	}
}

func TestVMResource_Configure_Success(t *testing.T) {
	r := NewVMResource().(*VMResource)
	mockClient := &client.MockClient{}
	req := resource.ConfigureRequest{ProviderData: mockClient}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestVMResource_Configure_NilProviderData(t *testing.T) {
	r := NewVMResource().(*VMResource)
	req := resource.ConfigureRequest{ProviderData: nil}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestVMResource_Configure_WrongType(t *testing.T) {
	r := NewVMResource().(*VMResource)
	req := resource.ConfigureRequest{ProviderData: "not a client"}
	resp := &resource.ConfigureResponse{}
	r.Configure(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// -- buildCreateParams tests --

func TestVMResource_buildCreateParams(t *testing.T) {
	r := &VMResource{}

	t.Run("minimal", func(t *testing.T) {
		data := &VMResourceModel{
			Name:   types.StringValue("test-vm"),
			Memory: types.Int64Value(2048),
		}
		params := r.buildCreateParams(data)
		if params["name"] != "test-vm" {
			t.Errorf("expected name 'test-vm', got %v", params["name"])
		}
		if params["memory"] != int64(2048) {
			t.Errorf("expected memory 2048, got %v", params["memory"])
		}
	})

	t.Run("with optional fields", func(t *testing.T) {
		data := &VMResourceModel{
			Name:        types.StringValue("test-vm"),
			Memory:      types.Int64Value(4096),
			Description: types.StringValue("A test VM"),
			VCPUs:       types.Int64Value(2),
			Cores:       types.Int64Value(2),
			Threads:     types.Int64Value(2),
			Autostart:   types.BoolValue(false),
			Time:        types.StringValue("UTC"),
			Bootloader:  types.StringValue("UEFI"),
			CPUMode:     types.StringValue("HOST-PASSTHROUGH"),
		}
		params := r.buildCreateParams(data)
		if params["description"] != "A test VM" {
			t.Errorf("expected description, got %v", params["description"])
		}
		if params["vcpus"] != int64(2) {
			t.Errorf("expected vcpus 2, got %v", params["vcpus"])
		}
		if params["autostart"] != false {
			t.Errorf("expected autostart false, got %v", params["autostart"])
		}
		if params["time"] != "UTC" {
			t.Errorf("expected time UTC, got %v", params["time"])
		}
		if params["cpu_mode"] != "HOST-PASSTHROUGH" {
			t.Errorf("expected cpu_mode HOST-PASSTHROUGH, got %v", params["cpu_mode"])
		}
	})

	t.Run("null optional fields omitted", func(t *testing.T) {
		data := &VMResourceModel{
			Name:      types.StringValue("test-vm"),
			Memory:    types.Int64Value(2048),
			CPUModel:  types.StringNull(),
			MinMemory: types.Int64Null(),
		}
		params := r.buildCreateParams(data)
		if _, ok := params["cpu_model"]; ok {
			t.Error("null cpu_model should not be in params")
		}
		if _, ok := params["min_memory"]; ok {
			t.Error("null min_memory should not be in params")
		}
	})
}

// -- Create tests --

func TestVMResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				switch method {
				case "vm.create":
					capturedMethod = method
					capturedParams = params
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.device.query":
					return mockVMDevicesResponse(), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "vm.create" {
		t.Errorf("expected method 'vm.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}
	if params["name"] != "test-vm" {
		t.Errorf("expected name 'test-vm', got %v", params["name"])
	}
	if params["memory"] != int64(2048) {
		t.Errorf("expected memory 2048, got %v", params["memory"])
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.State.ValueString() != "STOPPED" {
		t.Errorf("expected state STOPPED, got %q", model.State.ValueString())
	}
}

func TestVMResource_Create_WithStateRunning(t *testing.T) {
	var methods []string

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				switch method {
				case "vm.create":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "RUNNING"), nil
				case "vm.start":
					return json.RawMessage(`true`), nil
				case "vm.device.query":
					return mockVMDevicesResponse(), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.State = "RUNNING"
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify vm.start was called
	foundStart := false
	for _, m := range methods {
		if m == "vm.start" {
			foundStart = true
			break
		}
	}
	if !foundStart {
		t.Errorf("expected vm.start to be called, got methods: %v", methods)
	}
}

func TestVMResource_Create_APIError(t *testing.T) {
	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("vm already exists")
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	planValue := createVMModelValue(defaultVMPlanParams())
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestVMResource_Create_WithDevices(t *testing.T) {
	var deviceCreateCalls []map[string]any

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				switch method {
				case "vm.create":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.device.create":
					p, _ := params.(map[string]any)
					deviceCreateCalls = append(deviceCreateCalls, p)
					return json.RawMessage(fmt.Sprintf(`{"id": %d}`, len(deviceCreateCalls)+100)), nil
				case "vm.device.query":
					return mockVMDevicesResponse(
						map[string]any{"id": float64(101), "vm": float64(1), "order": float64(1000),
							"attributes": map[string]any{"dtype": "DISK", "path": "/dev/zvol/tank/vms/disk0", "type": "VIRTIO"}},
						map[string]any{"id": float64(102), "vm": float64(1), "order": float64(1001),
							"attributes": map[string]any{"dtype": "NIC", "type": "VIRTIO", "nic_attach": "br0", "mac": "00:11:22:33:44:55"}},
					), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.Disks = []vmDiskParams{{
		DeviceID: nil, Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: nil,
	}}
	p.NICs = []vmNICParams{{
		DeviceID: nil, Type: "VIRTIO", NICAttach: "br0", MAC: nil,
		TrustGuestRXFilters: false, Order: nil,
	}}
	planValue := createVMModelValue(p)
	req := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if len(deviceCreateCalls) != 2 {
		t.Fatalf("expected 2 device create calls, got %d", len(deviceCreateCalls))
	}
}

// -- Read tests --

func TestVMResource_Read_Success(t *testing.T) {
	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				switch method {
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.device.query":
					return mockVMDevicesResponse(), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "1" {
		t.Errorf("expected ID '1', got %q", model.ID.ValueString())
	}
	if model.Name.ValueString() != "test-vm" {
		t.Errorf("expected name 'test-vm', got %q", model.Name.ValueString())
	}
}

func TestVMResource_Read_NotFound(t *testing.T) {
	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("does not exist")
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "999"
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be removed (resource deleted externally)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed")
	}
}

func TestVMResource_Read_WithDevices(t *testing.T) {
	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				switch method {
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "RUNNING"), nil
				case "vm.device.query":
					return mockVMDevicesResponse(
						map[string]any{"id": float64(10), "vm": float64(1), "order": float64(1000),
							"attributes": map[string]any{"dtype": "DISK", "path": "/dev/zvol/tank/vms/disk0", "type": "VIRTIO"}},
						map[string]any{"id": float64(11), "vm": float64(1), "order": float64(1001),
							"attributes": map[string]any{"dtype": "NIC", "type": "VIRTIO", "nic_attach": "br0", "mac": "00:aa:bb:cc:dd:ee"}},
					), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	p.State = "RUNNING"
	p.Disks = []vmDiskParams{{
		DeviceID: float64(10), Path: "/dev/zvol/tank/vms/disk0", Type: "VIRTIO",
		LogicalSectorSize: nil, PhysicalSectorSize: nil, IOType: "THREADS",
		Serial: nil, Order: float64(1000),
	}}
	p.NICs = []vmNICParams{{
		DeviceID: float64(11), Type: "VIRTIO", NICAttach: "br0", MAC: "00:aa:bb:cc:dd:ee",
		TrustGuestRXFilters: false, Order: float64(1001),
	}}
	stateValue := createVMModelValue(p)
	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if len(model.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(model.Disks))
	}
	if model.Disks[0].Path.ValueString() != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("expected disk path '/dev/zvol/tank/vms/disk0', got %q", model.Disks[0].Path.ValueString())
	}
	if len(model.NICs) != 1 {
		t.Fatalf("expected 1 NIC, got %d", len(model.NICs))
	}
}

// -- Update tests --

func TestVMResource_Update_TopLevel(t *testing.T) {
	var capturedUpdateParams any

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				switch method {
				case "vm.update":
					capturedUpdateParams = params
					return mockVMResponse(1, "test-vm", 4096, "STOPPED"), nil
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 4096, "STOPPED"), nil
				case "vm.device.query":
					return mockVMDevicesResponse(), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)

	stateParams := defaultVMPlanParams()
	stateParams.ID = "1"
	stateParams.Memory = float64(2048)
	stateValue := createVMModelValue(stateParams)

	planParams := defaultVMPlanParams()
	planParams.ID = "1"
	planParams.Memory = float64(4096)
	planParams.Description = "Updated"
	planValue := createVMModelValue(planParams)

	req := resource.UpdateRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planValue},
	}
	resp := &resource.UpdateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedUpdateParams == nil {
		t.Fatal("expected vm.update to be called")
	}

	// Params is []any{vmID, updateMap}
	paramSlice, ok := capturedUpdateParams.([]any)
	if !ok {
		t.Fatalf("expected []any, got %T", capturedUpdateParams)
	}
	updateMap, ok := paramSlice[1].(map[string]any)
	if !ok {
		t.Fatalf("expected map[string]any, got %T", paramSlice[1])
	}
	if updateMap["memory"] != int64(4096) {
		t.Errorf("expected memory 4096, got %v", updateMap["memory"])
	}
}

// -- Delete tests --

func TestVMResource_Delete_Stopped(t *testing.T) {
	var methods []string

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				switch method {
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "STOPPED"), nil
				case "vm.delete":
					return json.RawMessage(`true`), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Should not call vm.stop
	for _, m := range methods {
		if m == "vm.stop" {
			t.Error("should not call vm.stop for stopped VM")
		}
	}
}

func TestVMResource_Delete_Running(t *testing.T) {
	var methods []string

	r := &VMResource{
		client: &client.MockClient{
			VersionVal: api.Version{Major: 24, Minor: 10},
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				switch method {
				case "vm.get_instance":
					return mockVMResponse(1, "test-vm", 2048, "RUNNING"), nil
				case "vm.delete":
					return json.RawMessage(`true`), nil
				}
				return nil, fmt.Errorf("unexpected method: %s", method)
			},
			CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`true`), nil
			},
		},
	}

	schemaResp := getVMResourceSchema(t)
	p := defaultVMPlanParams()
	p.ID = "1"
	p.State = "RUNNING"
	stateValue := createVMModelValue(p)
	req := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateValue},
	}
	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Should call vm.stop then vm.delete
	foundStop := false
	for _, m := range methods {
		if m == "vm.stop" {
			foundStop = true
		}
	}
	if !foundStop {
		t.Errorf("expected vm.stop to be called for running VM, got: %v", methods)
	}
}

// -- ImportState tests --

func TestVMResource_ImportState(t *testing.T) {
	r := NewVMResource().(*VMResource)

	schemaResp := getVMResourceSchema(t)
	emptyState := createVMModelValue(defaultVMPlanParams())

	req := resource.ImportStateRequest{ID: "42"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyState},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var model VMResourceModel
	resp.State.Get(context.Background(), &model)
	if model.ID.ValueString() != "42" {
		t.Errorf("expected ID '42', got %q", model.ID.ValueString())
	}
}

// -- Device mapping tests --

func TestVMResource_mapDevicesToModel(t *testing.T) {
	r := &VMResource{}

	devices := []vmDeviceAPIResponse{
		{ID: 10, VM: 1, Order: 1000, Attributes: map[string]any{
			"dtype": "DISK", "path": "/dev/zvol/tank/vms/disk0", "type": "VIRTIO",
		}},
		{ID: 11, VM: 1, Order: 1001, Attributes: map[string]any{
			"dtype": "RAW", "path": "/mnt/tank/vms/raw.img", "type": "AHCI",
			"boot": true, "size": float64(10737418240),
		}},
		{ID: 12, VM: 1, Order: 1002, Attributes: map[string]any{
			"dtype": "CDROM", "path": "/mnt/tank/iso/ubuntu.iso",
		}},
		{ID: 13, VM: 1, Order: 1003, Attributes: map[string]any{
			"dtype": "NIC", "type": "VIRTIO", "nic_attach": "br0", "mac": "00:aa:bb:cc:dd:ee",
		}},
		{ID: 14, VM: 1, Order: 1004, Attributes: map[string]any{
			"dtype": "DISPLAY", "type": "SPICE", "resolution": "1920x1080",
			"port": float64(5900), "bind": "0.0.0.0", "web": true,
		}},
		{ID: 15, VM: 1, Order: 1005, Attributes: map[string]any{
			"dtype": "PCI", "pptdev": "0000:01:00.0",
		}},
		{ID: 16, VM: 1, Order: 1006, Attributes: map[string]any{
			"dtype": "USB", "controller_type": "qemu-xhci", "device": "usb_0001",
		}},
	}

	data := &VMResourceModel{}
	r.mapDevicesToModel(devices, data)

	if len(data.Disks) != 1 {
		t.Fatalf("expected 1 disk, got %d", len(data.Disks))
	}
	if data.Disks[0].Path.ValueString() != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("disk path: got %q", data.Disks[0].Path.ValueString())
	}
	if data.Disks[0].DeviceID.ValueInt64() != 10 {
		t.Errorf("disk device_id: got %d", data.Disks[0].DeviceID.ValueInt64())
	}

	if len(data.Raws) != 1 {
		t.Fatalf("expected 1 raw, got %d", len(data.Raws))
	}
	if data.Raws[0].Path.ValueString() != "/mnt/tank/vms/raw.img" {
		t.Errorf("raw path: got %q", data.Raws[0].Path.ValueString())
	}

	if len(data.CDROMs) != 1 {
		t.Fatalf("expected 1 cdrom, got %d", len(data.CDROMs))
	}
	if data.CDROMs[0].Path.ValueString() != "/mnt/tank/iso/ubuntu.iso" {
		t.Errorf("cdrom path: got %q", data.CDROMs[0].Path.ValueString())
	}

	if len(data.NICs) != 1 {
		t.Fatalf("expected 1 nic, got %d", len(data.NICs))
	}
	if data.NICs[0].MAC.ValueString() != "00:aa:bb:cc:dd:ee" {
		t.Errorf("nic mac: got %q", data.NICs[0].MAC.ValueString())
	}

	if len(data.Displays) != 1 {
		t.Fatalf("expected 1 display, got %d", len(data.Displays))
	}
	if data.Displays[0].Resolution.ValueString() != "1920x1080" {
		t.Errorf("display resolution: got %q", data.Displays[0].Resolution.ValueString())
	}

	if len(data.PCIs) != 1 {
		t.Fatalf("expected 1 pci, got %d", len(data.PCIs))
	}
	if data.PCIs[0].PPTDev.ValueString() != "0000:01:00.0" {
		t.Errorf("pci pptdev: got %q", data.PCIs[0].PPTDev.ValueString())
	}

	if len(data.USBs) != 1 {
		t.Fatalf("expected 1 usb, got %d", len(data.USBs))
	}
	if data.USBs[0].ControllerType.ValueString() != "qemu-xhci" {
		t.Errorf("usb controller_type: got %q", data.USBs[0].ControllerType.ValueString())
	}
}

// -- buildDeviceParams tests --

func TestVMResource_buildDiskDeviceParams(t *testing.T) {
	disk := &VMDiskModel{
		Path:   types.StringValue("/dev/zvol/tank/vms/disk0"),
		Type:   types.StringValue("VIRTIO"),
		IOType: types.StringValue("THREADS"),
	}
	params := buildDiskDeviceParams(disk, 1)
	if params["vm"] != int64(1) {
		t.Errorf("expected vm=1, got %v", params["vm"])
	}
	attrs, ok := params["attributes"].(map[string]any)
	if !ok {
		t.Fatal("expected attributes to be map[string]any")
	}
	if attrs["dtype"] != "DISK" {
		t.Errorf("expected dtype=DISK, got %v", attrs["dtype"])
	}
	if attrs["path"] != "/dev/zvol/tank/vms/disk0" {
		t.Errorf("expected path, got %v", attrs["path"])
	}
}

func TestVMResource_buildNICDeviceParams(t *testing.T) {
	nic := &VMNICModel{
		Type:      types.StringValue("VIRTIO"),
		NICAttach: types.StringValue("br0"),
	}
	params := buildNICDeviceParams(nic, 1)
	attrs := params["attributes"].(map[string]any)
	if attrs["dtype"] != "NIC" {
		t.Errorf("expected dtype=NIC, got %v", attrs["dtype"])
	}
	if attrs["type"] != "VIRTIO" {
		t.Errorf("expected type=VIRTIO, got %v", attrs["type"])
	}
}

func TestVMResource_buildCDROMDeviceParams(t *testing.T) {
	cdrom := &VMCDROMModel{
		Path: types.StringValue("/mnt/tank/iso/ubuntu.iso"),
	}
	params := buildCDROMDeviceParams(cdrom, 1)
	attrs := params["attributes"].(map[string]any)
	if attrs["dtype"] != "CDROM" {
		t.Errorf("expected dtype=CDROM, got %v", attrs["dtype"])
	}
}

func TestVMResource_buildDisplayDeviceParams(t *testing.T) {
	display := &VMDisplayModel{
		Type:       types.StringValue("SPICE"),
		Resolution: types.StringValue("1920x1080"),
		Bind:       types.StringValue("0.0.0.0"),
		Web:        types.BoolValue(true),
	}
	params := buildDisplayDeviceParams(display, 1)
	attrs := params["attributes"].(map[string]any)
	if attrs["dtype"] != "DISPLAY" {
		t.Errorf("expected dtype=DISPLAY, got %v", attrs["dtype"])
	}
	if attrs["resolution"] != "1920x1080" {
		t.Errorf("expected resolution=1920x1080, got %v", attrs["resolution"])
	}
}

func TestVMResource_buildPCIDeviceParams(t *testing.T) {
	pci := &VMPCIModel{
		PPTDev: types.StringValue("0000:01:00.0"),
	}
	params := buildPCIDeviceParams(pci, 1)
	attrs := params["attributes"].(map[string]any)
	if attrs["dtype"] != "PCI" {
		t.Errorf("expected dtype=PCI, got %v", attrs["dtype"])
	}
}

func TestVMResource_buildUSBDeviceParams(t *testing.T) {
	usb := &VMUSBModel{
		ControllerType: types.StringValue("qemu-xhci"),
		Device:         types.StringValue("usb_0001"),
	}
	params := buildUSBDeviceParams(usb, 1)
	attrs := params["attributes"].(map[string]any)
	if attrs["dtype"] != "USB" {
		t.Errorf("expected dtype=USB, got %v", attrs["dtype"])
	}
	if attrs["controller_type"] != "qemu-xhci" {
		t.Errorf("expected controller_type=qemu-xhci, got %v", attrs["controller_type"])
	}
}

// -- Device reconciliation tests --

func TestVMResource_reconcileDevices(t *testing.T) {
	t.Run("create new device", func(t *testing.T) {
		var createdDevices []map[string]any
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method == "vm.device.create" {
						p, _ := params.(map[string]any)
						createdDevices = append(createdDevices, p)
						return json.RawMessage(`{"id": 100}`), nil
					}
					return nil, fmt.Errorf("unexpected method: %s", method)
				},
			},
		}

		plan := &VMResourceModel{
			Disks: []VMDiskModel{{
				Path: types.StringValue("/dev/zvol/tank/vms/new-disk"),
				Type: types.StringValue("VIRTIO"),
			}},
		}
		state := &VMResourceModel{}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(createdDevices) != 1 {
			t.Fatalf("expected 1 device create, got %d", len(createdDevices))
		}
	})

	t.Run("delete removed device", func(t *testing.T) {
		var deletedIDs []int64
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method == "vm.device.delete" {
						// params is device ID
						id, ok := params.(int64)
						if ok {
							deletedIDs = append(deletedIDs, id)
						}
						return json.RawMessage(`true`), nil
					}
					return nil, fmt.Errorf("unexpected method: %s", method)
				},
			},
		}

		plan := &VMResourceModel{}
		state := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/old-disk"),
				Type:     types.StringValue("VIRTIO"),
			}},
		}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(deletedIDs) != 1 || deletedIDs[0] != 50 {
			t.Fatalf("expected device 50 to be deleted, got %v", deletedIDs)
		}
	})

	t.Run("update existing device", func(t *testing.T) {
		var updatedParams []any
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					if method == "vm.device.update" {
						updatedParams = append(updatedParams, params)
						return json.RawMessage(`{"id": 50}`), nil
					}
					return nil, fmt.Errorf("unexpected method: %s", method)
				},
			},
		}

		plan := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
				Type:     types.StringValue("VIRTIO"), // changed from AHCI
			}},
		}
		state := &VMResourceModel{
			Disks: []VMDiskModel{{
				DeviceID: types.Int64Value(50),
				Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
				Type:     types.StringValue("AHCI"),
			}},
		}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if len(updatedParams) != 1 {
			t.Fatalf("expected 1 device update, got %d", len(updatedParams))
		}
	})

	t.Run("no changes no calls", func(t *testing.T) {
		callCount := 0
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					callCount++
					return nil, fmt.Errorf("unexpected call: %s", method)
				},
			},
		}

		disk := VMDiskModel{
			DeviceID: types.Int64Value(50),
			Path:     types.StringValue("/dev/zvol/tank/vms/disk0"),
			Type:     types.StringValue("VIRTIO"),
		}
		plan := &VMResourceModel{Disks: []VMDiskModel{disk}}
		state := &VMResourceModel{Disks: []VMDiskModel{disk}}

		err := r.reconcileDevices(context.Background(), 1, plan, state)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 0 {
			t.Errorf("expected no API calls, got %d", callCount)
		}
	})
}

// -- State management tests --

func TestVMResource_reconcileState(t *testing.T) {
	t.Run("stopped to running calls vm.start", func(t *testing.T) {
		var calledMethod string
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					calledMethod = method
					return json.RawMessage(`true`), nil
				},
			},
		}

		err := r.reconcileState(context.Background(), 1, "STOPPED", "RUNNING")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledMethod != "vm.start" {
			t.Errorf("expected vm.start, got %q", calledMethod)
		}
	})

	t.Run("running to stopped calls vm.stop via CallAndWait", func(t *testing.T) {
		var calledMethod string
		r := &VMResource{
			client: &client.MockClient{
				CallAndWaitFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					calledMethod = method
					return json.RawMessage(`true`), nil
				},
			},
		}

		err := r.reconcileState(context.Background(), 1, "RUNNING", "STOPPED")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if calledMethod != "vm.stop" {
			t.Errorf("expected vm.stop, got %q", calledMethod)
		}
	})

	t.Run("same state is no-op", func(t *testing.T) {
		callCount := 0
		r := &VMResource{
			client: &client.MockClient{
				CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
					callCount++
					return nil, nil
				},
			},
		}

		err := r.reconcileState(context.Background(), 1, "RUNNING", "RUNNING")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if callCount != 0 {
			t.Errorf("expected no calls for same state, got %d", callCount)
		}
	})
}
