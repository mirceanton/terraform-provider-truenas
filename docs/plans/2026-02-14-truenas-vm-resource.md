# `truenas_vm` Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `truenas_vm` resource to manage TrueNAS QEMU/KVM virtual machines with all 7 device types as inline nested blocks.

**Architecture:** Single resource `truenas_vm` using `vm.create`/`vm.update`/`vm.delete` for the VM lifecycle, `vm.device.*` CRUD for device reconciliation, and `vm.start`/`vm.stop` for power state management. Follows the existing `virt_instance` resource pattern.

**Tech Stack:** Go, terraform-plugin-framework, TrueNAS middleware `vm.*` and `vm.device.*` APIs

**Coverage baseline:**
- `internal/resources`: 88.9%
- `internal/provider`: 89.4%

---

## Task 1: Scaffold resource with model types and registration

**Files:**
- Create: `internal/resources/vm.go`
- Modify: `internal/provider/provider.go:388-401`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests for constructor, metadata, and schema**

```go
// vm_test.go
package resources

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
)

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

	// Required
	for _, name := range []string{"name", "memory"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsRequired() {
			t.Errorf("expected %q to be required", name)
		}
	}

	// Computed
	for _, name := range []string{"id", "display_available"} {
		attr, ok := attrs[name]
		if !ok {
			t.Fatalf("expected %q attribute", name)
		}
		if !attr.IsComputed() {
			t.Errorf("expected %q to be computed", name)
		}
	}

	// Optional
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

	// Device blocks
	blocks := schemaResp.Schema.Blocks
	for _, name := range []string{"disk", "raw", "cdrom", "nic", "display", "pci", "usb"} {
		if _, ok := blocks[name]; !ok {
			t.Errorf("expected %q block", name)
		}
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources/ -run TestNewVMResource -v 2>&1 | head -5`
Expected: compilation error, `NewVMResource` not defined

**Step 3: Implement scaffold**

Create `internal/resources/vm.go` with:
- All model structs (`VMResourceModel`, per-device models)
- `NewVMResource()` constructor
- `Metadata()`, `Schema()`, `Configure()` methods
- Stub CRUD methods (return not-implemented errors)
- Full schema definition with all attributes and 7 device blocks

Key model types:

```go
type VMResourceModel struct {
	ID                         types.String `tfsdk:"id"`
	Name                       types.String `tfsdk:"name"`
	Description                types.String `tfsdk:"description"`
	// ... all 25+ top-level attributes
	State                      types.String `tfsdk:"state"`
	DisplayAvailable           types.Bool   `tfsdk:"display_available"`
	Status                     types.Object `tfsdk:"status"`
	// Device blocks
	Disks    []VMDiskModel    `tfsdk:"disk"`
	Raws     []VMRawModel     `tfsdk:"raw"`
	CDROMs   []VMCDROMModel   `tfsdk:"cdrom"`
	NICs     []VMNICModel     `tfsdk:"nic"`
	Displays []VMDisplayModel `tfsdk:"display"`
	PCIs     []VMPCIModel     `tfsdk:"pci"`
	USBs     []VMUSBModel     `tfsdk:"usb"`
}
```

Reference: `internal/resources/virt_instance.go:40-90` for model struct pattern

**Step 4: Register in provider**

Add `resources.NewVMResource` to `internal/provider/provider.go:388-401`

**Step 5: Run tests to verify they pass**

Run: `go test ./internal/resources/ -run "TestNewVMResource|TestVMResource_Metadata|TestVMResource_Schema" -v`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go internal/provider/provider.go
git commit -m "feat(vm): scaffold truenas_vm resource with schema and model types"
```

---

## Task 2: Implement Create and Read

**Files:**
- Modify: `internal/resources/vm.go`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests for Create**

Test `buildCreateParams` converts model to API params correctly. Test the full Create flow using a mock client.

```go
func TestVMResource_buildCreateParams(t *testing.T) {
	// Test that model fields map to correct API params
	// Verify required fields (name, memory)
	// Verify optional fields are omitted when null/default
}

func TestVMResource_Create(t *testing.T) {
	// Mock client that returns a vm.create response
	// Verify vm.create called with correct params
	// Verify devices created via vm.device.create
	// Verify vm.start called when state="running"
	// Verify state is populated correctly
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources/ -run TestVMResource_buildCreateParams -v`

**Step 3: Implement buildCreateParams and Create**

```go
func (r *VMResource) buildCreateParams(data *VMResourceModel) map[string]any {
	params := map[string]any{
		"name":   data.Name.ValueString(),
		"memory": data.Memory.ValueInt64(),
	}
	// Add optional fields only if set (not null/unknown)
	if !data.Description.IsNull() { params["description"] = data.Description.ValueString() }
	// ... repeat for all optional fields
	return params
}
```

Create flow:
1. `r.client.Call(ctx, "vm.create", params)` -> get VM ID
2. For each device block: `r.client.Call(ctx, "vm.device.create", deviceParams)` -> get device IDs
3. If `state == "running"`: `r.client.Call(ctx, "vm.start", vmID)`
4. Read back and populate state

Reference: `internal/resources/virt_instance.go:430-510` for Create pattern

**Step 4: Implement Read**

```go
func (r *VMResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// Get state
	// Call vm.get_instance
	// If not found: resp.State.RemoveResource()
	// Map response to model
	// Query devices via vm.device.query with filter [["vm", "=", vmID]]
	// Map devices to model blocks
}
```

Reference: `internal/resources/virt_instance.go:520-570` for Read pattern

**Step 5: Write tests for Read (including not-found handling)**

**Step 6: Run all tests**

Run: `go test ./internal/resources/ -run TestVMResource -v`

**Step 7: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go
git commit -m "feat(vm): implement Create and Read for truenas_vm"
```

---

## Task 3: Implement Update with device reconciliation

**Files:**
- Modify: `internal/resources/vm.go`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests for device reconciliation**

```go
func TestVMResource_reconcileDevices(t *testing.T) {
	// Case 1: Device in plan but not state -> create
	// Case 2: Device in state but not plan -> delete
	// Case 3: Device in both with changes -> update
	// Case 4: No changes -> no API calls
}

func TestVMResource_Update(t *testing.T) {
	// Mock client, verify:
	// - vm.update called with changed fields only
	// - Device reconciliation happens
	// - State transitions (running<->stopped) handled
}
```

**Step 2: Run tests to verify they fail**

**Step 3: Implement device reconciliation**

VM devices use integer IDs (not names like virt_instance). The reconciliation compares `device_id` between plan and state:

```go
func (r *VMResource) reconcileDevices(ctx context.Context, vmID int64, plan, state *VMResourceModel) error {
	// Build map of state device IDs -> device type+index
	// For each device type in plan:
	//   - If device has device_id matching state: call vm.device.update if changed
	//   - If device has no device_id: call vm.device.create
	// For each device in state not in plan: call vm.device.delete
}
```

Reference: `internal/resources/virt_instance.go:1155-1264` for reconcileDevices pattern (adapt from name-based to ID-based matching)

**Step 4: Implement Update**

Update flow:
1. Get plan and state data
2. Build update params (only changed fields)
3. Call `vm.update` if any top-level changes
4. Reconcile devices
5. Handle state transitions (start/stop)
6. Read back and populate state

Reference: `internal/resources/virt_instance.go:573-705` for Update pattern

**Step 5: Run tests**

Run: `go test ./internal/resources/ -run TestVMResource -v`

**Step 6: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go
git commit -m "feat(vm): implement Update with device reconciliation"
```

---

## Task 4: Implement Delete and Import

**Files:**
- Modify: `internal/resources/vm.go`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests**

```go
func TestVMResource_Delete(t *testing.T) {
	// Test: running VM -> stop first, then delete
	// Test: stopped VM -> delete directly
	// Test: delete with zvols option
}

func TestVMResource_ImportState(t *testing.T) {
	// Test: import by numeric VM ID
	// Test: verify state populated correctly after import
}
```

**Step 2: Run tests to verify they fail**

**Step 3: Implement Delete**

```go
func (r *VMResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// Get state data
	// Query VM status
	// If running: call vm.stop with force=true, force_after_timeout=true
	// Call vm.delete (optionally with zvols=true)
}
```

Reference: `internal/resources/virt_instance.go:707-754` for Delete pattern

**Step 4: Implement ImportState**

```go
func (r *VMResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	// Parse ID as integer
	// Set id attribute
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 5: Run tests**

Run: `go test ./internal/resources/ -run TestVMResource -v`

**Step 6: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go
git commit -m "feat(vm): implement Delete and ImportState"
```

---

## Task 5: State management (start/stop)

**Files:**
- Modify: `internal/resources/vm.go`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests for state transitions**

```go
func TestVMResource_reconcileState(t *testing.T) {
	// Test: stopped -> running (calls vm.start)
	// Test: running -> stopped (calls vm.stop)
	// Test: already in desired state (no-op)
	// Test: vm.stop is a job - uses CallAndWait
}
```

**Step 2: Run tests to verify they fail**

**Step 3: Implement state reconciliation**

```go
func (r *VMResource) reconcileState(ctx context.Context, vmID int64, currentState, desiredState string) error {
	if currentState == desiredState {
		return nil
	}
	if desiredState == "RUNNING" {
		_, err := r.client.Call(ctx, "vm.start", vmID)
		return err
	}
	// vm.stop is a job
	stopOpts := map[string]any{"force": false, "force_after_timeout": true}
	_, err := r.client.CallAndWait(ctx, "vm.stop", []any{vmID, stopOpts})
	return err
}
```

Note: `vm.stop` is marked as a **(job)** in the API, so use `CallAndWait`. `vm.start` is NOT a job.

Reference:
- `internal/resources/virt_instance.go:1267-1311` for reconcileDesiredState
- `internal/resources/app_state.go:44-78` for waitForStableState
- `internal/client/jobs.go` for CallAndWait

**Step 4: Run tests**

Run: `go test ./internal/resources/ -run TestVMResource -v`

**Step 5: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go
git commit -m "feat(vm): implement state management (start/stop)"
```

---

## Task 6: Device mapping helpers (API response -> model)

**Files:**
- Modify: `internal/resources/vm.go`
- Test: `internal/resources/vm_test.go`

**Step 1: Write failing tests for device mapping**

```go
func TestVMResource_mapDevicesToModel(t *testing.T) {
	// Test each device type maps correctly from API response
	// Test discriminator (dtype) routes to correct model type
	// Test nullable fields handled correctly
	// Test USB nested object flattened to vendor_id/product_id
}

func TestVMResource_buildDeviceParams(t *testing.T) {
	// Test each device type builds correct API params
	// Test dtype is included in attributes
	// Test order is set when specified
}
```

**Step 2: Run tests to verify they fail**

**Step 3: Implement mappers**

```go
// vmDeviceAPIResponse represents a device from the API
type vmDeviceAPIResponse struct {
	ID         int64                  `json:"id"`
	VM         int64                  `json:"vm"`
	Attributes map[string]any         `json:"attributes"`
	Order      int64                  `json:"order"`
}

func (r *VMResource) mapDevicesToModel(devices []vmDeviceAPIResponse, data *VMResourceModel) {
	// Clear all device slices
	// For each device, switch on attributes["dtype"]
	// Map to appropriate model type
}

func buildDiskDeviceParams(disk *VMDiskModel) map[string]any {
	// Build {vm, attributes: {dtype: "DISK", ...}, order}
}
// ... repeat for each device type
```

Reference: `internal/resources/virt_instance.go:988-1077` for mapDevicesToModel pattern

**Step 4: Run tests**

Run: `go test ./internal/resources/ -run TestVMResource -v`

**Step 5: Commit**

```bash
git add internal/resources/vm.go internal/resources/vm_test.go
git commit -m "feat(vm): implement device mapping helpers"
```

---

## Task 7: Example and documentation

**Files:**
- Create: `examples/resources/truenas_vm/resource.tf`
- Create: `docs/resources/vm.md`

**Step 1: Create example**

```hcl
resource "truenas_vm" "example" {
  name        = "my-vm"
  description = "Example virtual machine"
  memory      = 2048
  vcpus       = 2
  cores       = 1
  threads     = 1
  bootloader  = "UEFI"
  autostart   = false
  time        = "UTC"
  state       = "running"

  disk {
    path = "/dev/zvol/tank/vms/my-vm-disk0"
    type = "VIRTIO"
  }

  nic {
    type       = "VIRTIO"
    nic_attach = "br0"
  }

  cdrom {
    path = "/mnt/tank/iso/ubuntu-22.04.iso"
  }

  display {
    resolution = "1920x1080"
    web        = true
  }
}
```

**Step 2: Create docs page (basic, will be auto-generated later)**

**Step 3: Commit**

```bash
git add examples/resources/truenas_vm/ docs/resources/vm.md
git commit -m "docs(vm): add example and documentation for truenas_vm"
```

---

## Task 8: Final verification

**Step 1: Run full test suite**

Run: `mise run test`
Expected: All tests pass

**Step 2: Check coverage**

Run: `mise run coverage`
Expected: `internal/resources` coverage >= 88.9% (baseline)

**Step 3: Run linter**

Run: `mise run lint` (if available)

**Step 4: Clean up docs/plans/ and commit**

```bash
rm docs/plans/2026-02-14-truenas-vm-resource.md
git add -A && git commit -m "chore: clean up plan document"
```

---

## API Reference

| Operation | Method | Notes |
|-----------|--------|-------|
| Create VM | `vm.create` | Returns full VM object |
| Read VM | `vm.get_instance` | By ID, raises error if not found |
| Update VM | `vm.update` | Partial updates, ID as first arg |
| Delete VM | `vm.delete` | Options: `{zvols, force}` |
| Start VM | `vm.start` | Options: `{overcommit}` |
| Stop VM | `vm.stop` **(job)** | Options: `{force, force_after_timeout}` |
| Query status | `vm.status` | Returns `{state, pid, domain_state}` |
| Create device | `vm.device.create` | `{vm, attributes, order}` |
| Update device | `vm.device.update` | ID + partial attributes |
| Delete device | `vm.device.delete` | Options: `{zvol, raw_file, force}` |
| Query devices | `vm.device.query` | Filter: `[["vm", "=", <id>]]` |

## Key Files to Reference

| File | What to learn |
|------|--------------|
| `internal/resources/virt_instance.go` | Full resource pattern, device reconciliation, state management |
| `internal/resources/virt_instance_test.go` | Test patterns, mock client usage |
| `internal/resources/app_state.go` | `waitForStableState()` helper |
| `internal/client/client.go` | Client interface (`Call`, `CallAndWait`) |
| `internal/provider/provider.go:388-401` | Resource registration |
| `docs/api/vms.md` | Complete VM API documentation |
