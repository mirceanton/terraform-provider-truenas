# Snapshot Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add ZFS snapshot support with `truenas_snapshot` resource, `truenas_snapshots` data source, and `snapshot_id` clone support on `truenas_dataset`.

**Architecture:** Standalone snapshot resource following AWS patterns. Uses `pool.snapshot.*` API methods via existing `client.Call()`. Clone functionality on dataset resource uses `pool.snapshot.clone`.

**Tech Stack:** Go, Terraform Plugin Framework, TrueNAS midclt API

---

## Task 1: Snapshot Resource - Scaffold and Schema

**Files:**
- Create: `internal/resources/snapshot.go`
- Modify: `internal/provider/provider.go:165-171`

**Step 1: Create snapshot.go with scaffold**

```go
package resources

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ resource.Resource = &SnapshotResource{}
var _ resource.ResourceWithConfigure = &SnapshotResource{}
var _ resource.ResourceWithImportState = &SnapshotResource{}

// SnapshotResource defines the resource implementation.
type SnapshotResource struct {
	client client.Client
}

// SnapshotResourceModel describes the resource data model.
type SnapshotResourceModel struct {
	ID              types.String `tfsdk:"id"`
	DatasetID       types.String `tfsdk:"dataset_id"`
	Name            types.String `tfsdk:"name"`
	Hold            types.Bool   `tfsdk:"hold"`
	Recursive       types.Bool   `tfsdk:"recursive"`
	CreateTXG       types.String `tfsdk:"createtxg"`
	UsedBytes       types.Int64  `tfsdk:"used_bytes"`
	ReferencedBytes types.Int64  `tfsdk:"referenced_bytes"`
}

// NewSnapshotResource creates a new SnapshotResource.
func NewSnapshotResource() resource.Resource {
	return &SnapshotResource{}
}

func (r *SnapshotResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshot"
}

func (r *SnapshotResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages a ZFS snapshot. Use for pre-upgrade backups and point-in-time recovery.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Snapshot identifier (dataset@name).",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"dataset_id": schema.StringAttribute{
				Description: "Dataset ID to snapshot. Reference a truenas_dataset resource or data source.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"name": schema.StringAttribute{
				Description: "Snapshot name.",
				Required:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"hold": schema.BoolAttribute{
				Description: "Prevent automatic deletion. Default: false.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
			},
			"recursive": schema.BoolAttribute{
				Description: "Include child datasets. Default: false. Only used at create time.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(false),
				PlanModifiers: []planmodifier.Bool{
					boolplanmodifier.RequiresReplace(),
				},
			},
			"createtxg": schema.StringAttribute{
				Description: "Transaction group when snapshot was created.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"used_bytes": schema.Int64Attribute{
				Description: "Space consumed by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
			"referenced_bytes": schema.Int64Attribute{
				Description: "Space referenced by snapshot.",
				Computed:    true,
				PlanModifiers: []planmodifier.Int64{
					int64planmodifier.UseStateForUnknown(),
				},
			},
		},
	}
}

func (r *SnapshotResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Create not yet implemented")
}

func (r *SnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Read not yet implemented")
}

func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Update not yet implemented")
}

func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Delete not yet implemented")
}

func (r *SnapshotResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 2: Register resource in provider**

In `internal/provider/provider.go`, add to the Resources function:

```go
func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewHostPathResource,
		resources.NewAppResource,
		resources.NewFileResource,
		resources.NewSnapshotResource,  // ADD THIS LINE
	}
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Success (no errors)

**Step 4: Commit scaffold**

```bash
git add internal/resources/snapshot.go internal/provider/provider.go
git commit -m "feat(snapshot): add resource scaffold and schema"
```

---

## Task 2: Snapshot Resource - Schema Tests

**Files:**
- Create: `internal/resources/snapshot_test.go`

**Step 1: Write schema tests**

```go
package resources

import (
	"context"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotResource(t *testing.T) {
	r := NewSnapshotResource()
	if r == nil {
		t.Fatal("expected non-nil resource")
	}

	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*SnapshotResource)
	var _ resource.ResourceWithImportState = r.(*SnapshotResource)
}

func TestSnapshotResource_Metadata(t *testing.T) {
	r := NewSnapshotResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshot" {
		t.Errorf("expected TypeName 'truenas_snapshot', got %q", resp.TypeName)
	}
}

func TestSnapshotResource_Schema(t *testing.T) {
	r := NewSnapshotResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify id attribute exists and is computed
	idAttr, ok := resp.Schema.Attributes["id"]
	if !ok {
		t.Fatal("expected 'id' attribute in schema")
	}
	if !idAttr.IsComputed() {
		t.Error("expected 'id' attribute to be computed")
	}

	// Verify dataset_id attribute exists and is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify name attribute exists and is required
	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if !nameAttr.IsRequired() {
		t.Error("expected 'name' attribute to be required")
	}

	// Verify hold attribute exists and is optional
	holdAttr, ok := resp.Schema.Attributes["hold"]
	if !ok {
		t.Fatal("expected 'hold' attribute in schema")
	}
	if !holdAttr.IsOptional() {
		t.Error("expected 'hold' attribute to be optional")
	}

	// Verify recursive attribute exists and is optional
	recursiveAttr, ok := resp.Schema.Attributes["recursive"]
	if !ok {
		t.Fatal("expected 'recursive' attribute in schema")
	}
	if !recursiveAttr.IsOptional() {
		t.Error("expected 'recursive' attribute to be optional")
	}

	// Verify computed attributes
	for _, attr := range []string{"createtxg", "used_bytes", "referenced_bytes"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Fatalf("expected '%s' attribute in schema", attr)
		}
		if !a.IsComputed() {
			t.Errorf("expected '%s' attribute to be computed", attr)
		}
	}
}

func TestSnapshotResource_Configure_Success(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSnapshotResource_Configure_NilProviderData(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestSnapshotResource_Configure_WrongType(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

// Test helpers

func getSnapshotResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewSnapshotResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

// snapshotModelParams holds parameters for creating test model values.
// Using a struct instead of 8 individual parameters per the 3-param rule.
type snapshotModelParams struct {
	ID              interface{}
	DatasetID       interface{}
	Name            interface{}
	Hold            interface{}
	Recursive       interface{}
	CreateTXG       interface{}
	UsedBytes       interface{}
	ReferencedBytes interface{}
}

func createSnapshotResourceModelValue(p snapshotModelParams) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":               tftypes.String,
			"dataset_id":       tftypes.String,
			"name":             tftypes.String,
			"hold":             tftypes.Bool,
			"recursive":        tftypes.Bool,
			"createtxg":        tftypes.String,
			"used_bytes":       tftypes.Number,
			"referenced_bytes": tftypes.Number,
		},
	}, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, p.ID),
		"dataset_id":       tftypes.NewValue(tftypes.String, p.DatasetID),
		"name":             tftypes.NewValue(tftypes.String, p.Name),
		"hold":             tftypes.NewValue(tftypes.Bool, p.Hold),
		"recursive":        tftypes.NewValue(tftypes.Bool, p.Recursive),
		"createtxg":        tftypes.NewValue(tftypes.String, p.CreateTXG),
		"used_bytes":       tftypes.NewValue(tftypes.Number, p.UsedBytes),
		"referenced_bytes": tftypes.NewValue(tftypes.Number, p.ReferencedBytes),
	})
}
```

**Step 2: Run tests to verify they pass**

Run: `go test ./internal/resources/snapshot_test.go ./internal/resources/snapshot.go -v -run "TestSnapshotResource_Schema\|TestSnapshotResource_Metadata\|TestSnapshotResource_Configure\|TestNewSnapshotResource"`
Expected: All tests PASS

**Step 3: Commit schema tests**

```bash
git add internal/resources/snapshot_test.go
git commit -m "test(snapshot): add schema and configure tests"
```

---

## Task 3: Snapshot Resource - Create Implementation

**Files:**
- Modify: `internal/resources/snapshot.go`
- Modify: `internal/resources/snapshot_test.go`

**Step 1: Write failing test for Create**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": "tank/data@snap1"}`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if capturedMethod != "pool.snapshot.create" {
		t.Errorf("expected method 'pool.snapshot.create', got %q", capturedMethod)
	}

	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["dataset"] != "tank/data" {
		t.Errorf("expected dataset 'tank/data', got %v", params["dataset"])
	}
	if params["name"] != "snap1" {
		t.Errorf("expected name 'snap1', got %v", params["name"])
	}
}
```

Also add the required imports at the top of the test file:

```go
import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)
```

**Step 2: Run test to verify it fails**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Create_Success"`
Expected: FAIL with "Not Implemented"

**Step 3: Implement Create**

Update `snapshot.go` Create method:

```go
// snapshotQueryResponse represents the JSON response from pool.snapshot.query.
type snapshotQueryResponse struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Dataset    string                     `json:"dataset"`
	Holds      map[string]any             `json:"holds"`
	Properties snapshotPropertiesResponse `json:"properties"`
}

type snapshotPropertiesResponse struct {
	CreateTXG  propertyValue `json:"createtxg"`
	Used       parsedValue   `json:"used"`
	Referenced parsedValue   `json:"referenced"`
}

type propertyValue struct {
	Value string `json:"value"`
}

type parsedValue struct {
	Parsed int64 `json:"parsed"`
}

// querySnapshot queries a snapshot by ID and returns the response.
// Returns nil if the snapshot is not found.
func (r *SnapshotResource) querySnapshot(ctx context.Context, snapshotID string) (*snapshotQueryResponse, error) {
	filter := [][]any{{"id", "=", snapshotID}}
	result, err := r.client.Call(ctx, "pool.snapshot.query", filter)
	if err != nil {
		return nil, err
	}

	var snapshots []snapshotQueryResponse
	if err := json.Unmarshal(result, &snapshots); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(snapshots) == 0 {
		return nil, nil
	}

	return &snapshots[0], nil
}

// mapSnapshotToModel maps API response fields to the Terraform model.
func mapSnapshotToModel(snap *snapshotQueryResponse, data *SnapshotResourceModel) {
	data.ID = types.StringValue(snap.ID)
	data.DatasetID = types.StringValue(snap.Dataset)
	data.Name = types.StringValue(snap.Name)
	data.Hold = types.BoolValue(len(snap.Holds) > 0)
	data.CreateTXG = types.StringValue(snap.Properties.CreateTXG.Value)
	data.UsedBytes = types.Int64Value(snap.Properties.Used.Parsed)
	data.ReferencedBytes = types.Int64Value(snap.Properties.Referenced.Parsed)
}

func (r *SnapshotResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build create params
	params := map[string]any{
		"dataset": data.DatasetID.ValueString(),
		"name":    data.Name.ValueString(),
	}

	if !data.Recursive.IsNull() && data.Recursive.ValueBool() {
		params["recursive"] = true
	}

	// Create the snapshot
	_, err := r.client.Call(ctx, "pool.snapshot.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Snapshot",
			fmt.Sprintf("Unable to create snapshot: %s", err.Error()),
		)
		return
	}

	// Build snapshot ID
	snapshotID := fmt.Sprintf("%s@%s", data.DatasetID.ValueString(), data.Name.ValueString())

	// If hold is requested, apply it
	if !data.Hold.IsNull() && data.Hold.ValueBool() {
		_, err := r.client.Call(ctx, "pool.snapshot.hold", snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Snapshot created but failed to apply hold: %s", err.Error()),
			)
			return
		}
	}

	// Query the snapshot to get computed fields
	snap, err := r.querySnapshot(ctx, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Snapshot created but unable to read: %s", err.Error()),
		)
		return
	}

	if snap == nil {
		resp.Diagnostics.AddError(
			"Snapshot Not Found",
			"Snapshot was created but could not be found.",
		)
		return
	}

	mapSnapshotToModel(snap, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Create_Success"`
Expected: PASS

**Step 5: Add test for Create with hold**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_Create_WithHold(t *testing.T) {
	var methods []string

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				if method == "pool.snapshot.create" {
					return json.RawMessage(`{"id": "tank/data@snap1"}`), nil
				}
				if method == "pool.snapshot.hold" {
					return json.RawMessage(`true`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      true,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify pool.snapshot.hold was called
	holdCalled := false
	for _, m := range methods {
		if m == "pool.snapshot.hold" {
			holdCalled = true
			break
		}
	}
	if !holdCalled {
		t.Error("expected pool.snapshot.hold to be called when hold=true")
	}
}

func TestSnapshotResource_Create_APIError(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("snapshot already exists")
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}
```

Also add `"errors"` to the imports.

**Step 6: Run all Create tests**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Create"`
Expected: All PASS

**Step 7: Commit Create implementation**

```bash
git add internal/resources/snapshot.go internal/resources/snapshot_test.go
git commit -m "feat(snapshot): implement Create with hold support"
```

---

## Task 4: Snapshot Resource - Read Implementation

**Files:**
- Modify: `internal/resources/snapshot.go`
- Modify: `internal/resources/snapshot_test.go`

**Step 1: Write failing test for Read**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_Read_Success(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": "tank/data@snap1",
					"name": "snap1",
					"dataset": "tank/data",
					"properties": {
						"createtxg": {"value": "12345"},
						"used": {"parsed": 1024},
						"referenced": {"parsed": 2048}
					}
				}]`), nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "",
		UsedBytes:       float64(0),
		ReferencedBytes: float64(0),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotResourceModel
	resp.State.Get(context.Background(), &data)

	if data.CreateTXG.ValueString() != "12345" {
		t.Errorf("expected createtxg '12345', got %q", data.CreateTXG.ValueString())
	}
	if data.UsedBytes.ValueInt64() != 1024 {
		t.Errorf("expected used_bytes 1024, got %d", data.UsedBytes.ValueInt64())
	}
}

func TestSnapshotResource_Read_NotFound(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "",
		UsedBytes:       float64(0),
		ReferencedBytes: float64(0),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	// Should not error - just remove from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// State should be empty (null)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be null when snapshot not found")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Read"`
Expected: FAIL with "Not Implemented"

**Step 3: Implement Read**

Update `snapshot.go` Read method:

```go
func (r *SnapshotResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snap, err := r.querySnapshot(ctx, data.ID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Unable to read snapshot %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	if snap == nil {
		// Snapshot no longer exists - remove from state
		resp.State.RemoveResource(ctx)
		return
	}

	mapSnapshotToModel(snap, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Read"`
Expected: All PASS

**Step 5: Commit Read implementation**

```bash
git add internal/resources/snapshot.go internal/resources/snapshot_test.go
git commit -m "feat(snapshot): implement Read with not-found handling"
```

---

## Task 5: Snapshot Resource - Update Implementation

**Files:**
- Modify: `internal/resources/snapshot.go`
- Modify: `internal/resources/snapshot_test.go`

**Step 1: Write failing tests for Update**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_Update_HoldToRelease(t *testing.T) {
	var releaseCalled bool

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.release" {
					releaseCalled = true
					return json.RawMessage(`true`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)

	// State has hold=true
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	// Plan has hold=false
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !releaseCalled {
		t.Error("expected pool.snapshot.release to be called")
	}
}

func TestSnapshotResource_Update_ReleaseToHold(t *testing.T) {
	var holdCalled bool

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.hold" {
					holdCalled = true
					return json.RawMessage(`true`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`[{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"createtxg": {"value": "12345"},
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)

	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.UpdateRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.UpdateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Update(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !holdCalled {
		t.Error("expected pool.snapshot.hold to be called")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Update"`
Expected: FAIL with "Not Implemented"

**Step 3: Implement Update**

Update `snapshot.go` Update method:

```go
func (r *SnapshotResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state SnapshotResourceModel
	var plan SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotID := state.ID.ValueString()

	// Handle hold changes
	stateHold := state.Hold.ValueBool()
	planHold := plan.Hold.ValueBool()

	if stateHold && !planHold {
		// Release hold
		_, err := r.client.Call(ctx, "pool.snapshot.release", snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold on snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	} else if !stateHold && planHold {
		// Apply hold
		_, err := r.client.Call(ctx, "pool.snapshot.hold", snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Hold Snapshot",
				fmt.Sprintf("Unable to hold snapshot %q: %s", snapshotID, err.Error()),
			)
			return
		}
	}

	// Refresh state from API
	snap, err := r.querySnapshot(ctx, snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshot",
			fmt.Sprintf("Unable to read snapshot %q: %s", snapshotID, err.Error()),
		)
		return
	}

	if snap == nil {
		resp.Diagnostics.AddError(
			"Snapshot Not Found",
			fmt.Sprintf("Snapshot %q no longer exists.", snapshotID),
		)
		return
	}

	mapSnapshotToModel(snap, &plan)
	plan.Hold = types.BoolValue(planHold) // Preserve the planned hold value

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Update"`
Expected: All PASS

**Step 5: Commit Update implementation**

```bash
git add internal/resources/snapshot.go internal/resources/snapshot_test.go
git commit -m "feat(snapshot): implement Update for hold/release"
```

---

## Task 6: Snapshot Resource - Delete Implementation

**Files:**
- Modify: `internal/resources/snapshot.go`
- Modify: `internal/resources/snapshot_test.go`

**Step 1: Write failing tests for Delete**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_Delete_Success(t *testing.T) {
	var deleteCalled bool
	var deleteID string

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.delete" {
					deleteCalled = true
					deleteID = params.(string)
					return json.RawMessage(`true`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if !deleteCalled {
		t.Error("expected pool.snapshot.delete to be called")
	}

	if deleteID != "tank/data@snap1" {
		t.Errorf("expected delete ID 'tank/data@snap1', got %q", deleteID)
	}
}

func TestSnapshotResource_Delete_WithHold(t *testing.T) {
	var methods []string

	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				methods = append(methods, method)
				return json.RawMessage(`true`), nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            true,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify release was called before delete
	releaseIdx := -1
	deleteIdx := -1
	for i, m := range methods {
		if m == "pool.snapshot.release" {
			releaseIdx = i
		}
		if m == "pool.snapshot.delete" {
			deleteIdx = i
		}
	}

	if releaseIdx == -1 {
		t.Error("expected pool.snapshot.release to be called")
	}
	if deleteIdx == -1 {
		t.Error("expected pool.snapshot.delete to be called")
	}
	if releaseIdx > deleteIdx {
		t.Error("expected release to be called before delete")
	}
}
```

**Step 2: Run tests to verify they fail**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Delete"`
Expected: FAIL with "Not Implemented"

**Step 3: Implement Delete**

Update `snapshot.go` Delete method:

```go
func (r *SnapshotResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data SnapshotResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	snapshotID := data.ID.ValueString()

	// If held, release first
	if data.Hold.ValueBool() {
		_, err := r.client.Call(ctx, "pool.snapshot.release", snapshotID)
		if err != nil {
			resp.Diagnostics.AddError(
				"Unable to Release Snapshot Hold",
				fmt.Sprintf("Unable to release hold before delete: %s", err.Error()),
			)
			return
		}
	}

	// Delete the snapshot
	_, err := r.client.Call(ctx, "pool.snapshot.delete", snapshotID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Snapshot",
			fmt.Sprintf("Unable to delete snapshot %q: %s", snapshotID, err.Error()),
		)
		return
	}
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Delete"`
Expected: All PASS

**Step 5: Add additional error tests**

Add these tests for full coverage:

```go
func TestSnapshotResource_Create_InvalidJSONResponse(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.create" {
					return json.RawMessage(`{"id": "tank/data@snap1"}`), nil
				}
				if method == "pool.snapshot.query" {
					return json.RawMessage(`invalid json`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	planValue := createSnapshotResourceModelValue(snapshotModelParams{
		DatasetID: "tank/data",
		Name:      "snap1",
		Hold:      false,
		Recursive: false,
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for invalid JSON response")
	}
}

func TestSnapshotResource_Read_APIError(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Read(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}

func TestSnapshotResource_Delete_APIError(t *testing.T) {
	r := &SnapshotResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("snapshot is busy")
			},
		},
	}

	schemaResp := getSnapshotResourceSchema(t)
	stateValue := createSnapshotResourceModelValue(snapshotModelParams{
		ID:              "tank/data@snap1",
		DatasetID:       "tank/data",
		Name:            "snap1",
		Hold:            false,
		Recursive:       false,
		CreateTXG:       "12345",
		UsedBytes:       float64(1024),
		ReferencedBytes: float64(2048),
	})

	req := resource.DeleteRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    stateValue,
		},
	}

	resp := &resource.DeleteResponse{}

	r.Delete(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}
```

**Step 6: Run all tests**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_Delete\|TestSnapshotResource_Create_InvalidJSON\|TestSnapshotResource_Read_APIError"`
Expected: All PASS

**Step 7: Commit Delete implementation**

```bash
git add internal/resources/snapshot.go internal/resources/snapshot_test.go
git commit -m "feat(snapshot): implement Delete with hold release and error tests"
```

---

## Task 7: Snapshot Resource - Import Implementation

**Files:**
- Modify: `internal/resources/snapshot_test.go`

**Step 1: Write test for ImportState**

Add to `snapshot_test.go`:

```go
func TestSnapshotResource_ImportState(t *testing.T) {
	r := NewSnapshotResource().(*SnapshotResource)

	req := resource.ImportStateRequest{
		ID: "tank/data@snap1",
	}

	schemaResp := getSnapshotResourceSchema(t)
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.ImportState(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotResourceModel
	resp.State.Get(context.Background(), &data)

	if data.ID.ValueString() != "tank/data@snap1" {
		t.Errorf("expected ID 'tank/data@snap1', got %q", data.ID.ValueString())
	}
}
```

**Step 2: Run test to verify it passes**

Run: `go test ./internal/resources/ -v -run "TestSnapshotResource_ImportState"`
Expected: PASS (already implemented via ImportStatePassthroughID)

**Step 3: Commit Import test**

```bash
git add internal/resources/snapshot_test.go
git commit -m "test(snapshot): add ImportState test"
```

---

## Task 8: Snapshots Data Source - Scaffold and Schema

**Files:**
- Create: `internal/datasources/snapshots.go`
- Modify: `internal/provider/provider.go:158-163`

**Step 1: Create snapshots.go with scaffold**

```go
package datasources

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var _ datasource.DataSource = &SnapshotsDataSource{}
var _ datasource.DataSourceWithConfigure = &SnapshotsDataSource{}

// SnapshotsDataSource defines the data source implementation.
type SnapshotsDataSource struct {
	client client.Client
}

// SnapshotsDataSourceModel describes the data source data model.
type SnapshotsDataSourceModel struct {
	DatasetID   types.String    `tfsdk:"dataset_id"`
	Recursive   types.Bool      `tfsdk:"recursive"`
	NamePattern types.String    `tfsdk:"name_pattern"`
	Snapshots   []SnapshotModel `tfsdk:"snapshots"`
}

// SnapshotModel represents a snapshot in the list.
type SnapshotModel struct {
	ID              types.String `tfsdk:"id"`
	Name            types.String `tfsdk:"name"`
	DatasetID       types.String `tfsdk:"dataset_id"`
	UsedBytes       types.Int64  `tfsdk:"used_bytes"`
	ReferencedBytes types.Int64  `tfsdk:"referenced_bytes"`
	Hold            types.Bool   `tfsdk:"hold"`
}

// NewSnapshotsDataSource creates a new SnapshotsDataSource.
func NewSnapshotsDataSource() datasource.DataSource {
	return &SnapshotsDataSource{}
}

func (d *SnapshotsDataSource) Metadata(ctx context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_snapshots"
}

func (d *SnapshotsDataSource) Schema(ctx context.Context, req datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Retrieves snapshots for a dataset.",
		Attributes: map[string]schema.Attribute{
			"dataset_id": schema.StringAttribute{
				Description: "Dataset ID to query snapshots for.",
				Required:    true,
			},
			"recursive": schema.BoolAttribute{
				Description: "Include child dataset snapshots. Default: false.",
				Optional:    true,
			},
			"name_pattern": schema.StringAttribute{
				Description: "Glob pattern to filter snapshot names.",
				Optional:    true,
			},
			"snapshots": schema.ListNestedAttribute{
				Description: "List of snapshots.",
				Computed:    true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.StringAttribute{
							Description: "Snapshot ID (dataset@name).",
							Computed:    true,
						},
						"name": schema.StringAttribute{
							Description: "Snapshot name.",
							Computed:    true,
						},
						"dataset_id": schema.StringAttribute{
							Description: "Parent dataset ID.",
							Computed:    true,
						},
						"used_bytes": schema.Int64Attribute{
							Description: "Space consumed by snapshot.",
							Computed:    true,
						},
						"referenced_bytes": schema.Int64Attribute{
							Description: "Space referenced by snapshot.",
							Computed:    true,
						},
						"hold": schema.BoolAttribute{
							Description: "Whether snapshot is held.",
							Computed:    true,
						},
					},
				},
			},
		},
	}
}

func (d *SnapshotsDataSource) Configure(ctx context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected client.Client, got: %T.", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *SnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	// TODO: implement
	resp.Diagnostics.AddError("Not Implemented", "Read not yet implemented")
}
```

**Step 2: Register data source in provider**

In `internal/provider/provider.go`, add to DataSources function:

```go
func (p *TrueNASProvider) DataSources(ctx context.Context) []func() datasource.DataSource {
	return []func() datasource.DataSource{
		datasources.NewPoolDataSource,
		datasources.NewDatasetDataSource,
		datasources.NewSnapshotsDataSource,  // ADD THIS LINE
	}
}
```

**Step 3: Verify it compiles**

Run: `go build ./...`
Expected: Success

**Step 4: Commit scaffold**

```bash
git add internal/datasources/snapshots.go internal/provider/provider.go
git commit -m "feat(snapshots): add data source scaffold and schema"
```

---

## Task 9: Snapshots Data Source - Tests and Read Implementation

**Files:**
- Create: `internal/datasources/snapshots_test.go`
- Modify: `internal/datasources/snapshots.go`

**Step 1: Create test file with schema tests**

```go
package datasources

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewSnapshotsDataSource(t *testing.T) {
	ds := NewSnapshotsDataSource()
	if ds == nil {
		t.Fatal("expected non-nil data source")
	}

	var _ datasource.DataSource = ds
	var _ datasource.DataSourceWithConfigure = ds.(*SnapshotsDataSource)
}

func TestSnapshotsDataSource_Metadata(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_snapshots" {
		t.Errorf("expected TypeName 'truenas_snapshots', got %q", resp.TypeName)
	}
}

func TestSnapshotsDataSource_Schema(t *testing.T) {
	ds := NewSnapshotsDataSource()

	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(context.Background(), req, resp)

	if resp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify dataset_id is required
	datasetIDAttr, ok := resp.Schema.Attributes["dataset_id"]
	if !ok {
		t.Fatal("expected 'dataset_id' attribute in schema")
	}
	if !datasetIDAttr.IsRequired() {
		t.Error("expected 'dataset_id' attribute to be required")
	}

	// Verify snapshots is computed
	snapshotsAttr, ok := resp.Schema.Attributes["snapshots"]
	if !ok {
		t.Fatal("expected 'snapshots' attribute in schema")
	}
	if !snapshotsAttr.IsComputed() {
		t.Error("expected 'snapshots' attribute to be computed")
	}
}

func TestSnapshotsDataSource_Configure_Success(t *testing.T) {
	ds := NewSnapshotsDataSource().(*SnapshotsDataSource)

	mockClient := &client.MockClient{}

	req := datasource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func getSnapshotsDataSourceSchema(t *testing.T) datasource.SchemaResponse {
	t.Helper()
	ds := NewSnapshotsDataSource()
	schemaReq := datasource.SchemaRequest{}
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), schemaReq, schemaResp)
	return *schemaResp
}

func TestSnapshotsDataSource_Read_Success(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[
					{
						"id": "tank/data@snap1",
						"name": "snap1",
						"dataset": "tank/data",
						"holds": {},
						"properties": {
							"used": {"parsed": 1024},
							"referenced": {"parsed": 2048}
						}
					},
					{
						"id": "tank/data@snap2",
						"name": "snap2",
						"dataset": "tank/data",
						"holds": {"terraform": true},
						"properties": {
							"used": {"parsed": 512},
							"referenced": {"parsed": 1024}
						}
					}
				]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 2 {
		t.Errorf("expected 2 snapshots, got %d", len(data.Snapshots))
	}
}

func TestSnapshotsDataSource_Read_Empty(t *testing.T) {
	ds := &SnapshotsDataSource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getSnapshotsDataSourceSchema(t)

	configValue := tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"dataset_id":   tftypes.String,
			"recursive":    tftypes.Bool,
			"name_pattern": tftypes.String,
			"snapshots":    tftypes.List{ElementType: tftypes.Object{}},
		},
	}, map[string]tftypes.Value{
		"dataset_id":   tftypes.NewValue(tftypes.String, "tank/data"),
		"recursive":    tftypes.NewValue(tftypes.Bool, nil),
		"name_pattern": tftypes.NewValue(tftypes.String, nil),
		"snapshots":    tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{}}, nil),
	})

	req := datasource.ReadRequest{
		Config: tfsdk.Config{
			Schema: schemaResp.Schema,
			Raw:    configValue,
		},
	}

	resp := &datasource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	ds.Read(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	var data SnapshotsDataSourceModel
	resp.State.Get(context.Background(), &data)

	if len(data.Snapshots) != 0 {
		t.Errorf("expected 0 snapshots, got %d", len(data.Snapshots))
	}
}
```

**Step 2: Run tests to verify Read fails**

Run: `go test ./internal/datasources/ -v -run "TestSnapshotsDataSource_Read"`
Expected: FAIL with "Not Implemented"

**Step 3: Implement Read**

Update `snapshots.go` Read method:

```go
// snapshotsQueryResponse represents a snapshot from pool.snapshot.query.
type snapshotsQueryResponse struct {
	ID         string                     `json:"id"`
	Name       string                     `json:"name"`
	Dataset    string                     `json:"dataset"`
	Holds      map[string]bool            `json:"holds"`
	Properties snapshotPropertiesResponse `json:"properties"`
}

type snapshotPropertiesResponse struct {
	Used       parsedValue `json:"used"`
	Referenced parsedValue `json:"referenced"`
}

type parsedValue struct {
	Parsed int64 `json:"parsed"`
}

func (d *SnapshotsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var data SnapshotsDataSourceModel

	resp.Diagnostics.Append(req.Config.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build filter for dataset
	datasetID := data.DatasetID.ValueString()
	filter := [][]any{{"dataset", "=", datasetID}}

	// If recursive, match exact dataset OR child datasets (dataset/)
	if !data.Recursive.IsNull() && data.Recursive.ValueBool() {
		filter = [][]any{
			{"OR", [][]any{
				{"dataset", "=", datasetID},
				{"dataset", "^", datasetID + "/"},
			}},
		}
	}

	result, err := d.client.Call(ctx, "pool.snapshot.query", filter)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Snapshots",
			fmt.Sprintf("Unable to query snapshots: %s", err.Error()),
		)
		return
	}

	var snapshots []snapshotsQueryResponse
	if err := json.Unmarshal(result, &snapshots); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Snapshots Response",
			fmt.Sprintf("Unable to parse snapshots response: %s", err.Error()),
		)
		return
	}

	// Filter by name pattern if specified
	namePattern := data.NamePattern.ValueString()

	data.Snapshots = make([]SnapshotModel, 0, len(snapshots))
	for _, snap := range snapshots {
		// Apply name pattern filter
		if namePattern != "" {
			matched, err := filepath.Match(namePattern, snap.Name)
			if err != nil {
				resp.Diagnostics.AddError(
					"Invalid Name Pattern",
					fmt.Sprintf("Invalid glob pattern %q: %s", namePattern, err.Error()),
				)
				return
			}
			if !matched {
				continue
			}
		}

		data.Snapshots = append(data.Snapshots, SnapshotModel{
			ID:              types.StringValue(snap.ID),
			Name:            types.StringValue(snap.Name),
			DatasetID:       types.StringValue(snap.Dataset),
			UsedBytes:       types.Int64Value(snap.Properties.Used.Parsed),
			ReferencedBytes: types.Int64Value(snap.Properties.Referenced.Parsed),
			Hold:            types.BoolValue(len(snap.Holds) > 0),
		})
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 4: Run tests to verify they pass**

Run: `go test ./internal/datasources/ -v -run "TestSnapshotsDataSource"`
Expected: All PASS

**Step 5: Commit data source**

```bash
git add internal/datasources/snapshots.go internal/datasources/snapshots_test.go
git commit -m "feat(snapshots): implement data source with filtering"
```

---

## Task 10: Dataset Resource - Add snapshot_id for Clone

**Files:**
- Modify: `internal/resources/dataset.go`
- Modify: `internal/resources/dataset_test.go`

**Step 1: Write failing test for clone**

Add to `dataset_test.go`:

```go
func TestDatasetResource_Create_WithSnapshotId(t *testing.T) {
	var cloneCalled bool
	var cloneParams map[string]any
	var createCalled bool

	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.clone" {
					cloneCalled = true
					cloneParams = params.(map[string]any)
					return json.RawMessage(`{"id": "tank/restored"}`), nil
				}
				if method == "pool.dataset.create" {
					createCalled = true
					return json.RawMessage(`{"id": "tank/restored"}`), nil
				}
				if method == "pool.dataset.query" {
					return json.RawMessage(`[{
						"id": "tank/restored",
						"name": "restored",
						"mountpoint": "/mnt/tank/restored",
						"compression": {"value": "lz4"},
						"quota": {"value": "0"},
						"refquota": {"value": "0"},
						"atime": {"value": "on"}
					}]`), nil
				}
				if method == "filesystem.stat" {
					return json.RawMessage(`{"mode": 493, "uid": 0, "gid": 0}`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)
	planValue := createDatasetResourceModelValue(datasetModelParams{
		Pool:       "tank",
		Path:       "restored",
		SnapshotID: "tank/data@snap1",
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	// Verify pool.snapshot.clone was called
	if !cloneCalled {
		t.Error("expected pool.snapshot.clone to be called")
	}

	// Verify pool.dataset.create was NOT called
	if createCalled {
		t.Error("expected pool.dataset.create to NOT be called when snapshot_id is set")
	}

	if cloneParams["snapshot"] != "tank/data@snap1" {
		t.Errorf("expected snapshot 'tank/data@snap1', got %v", cloneParams["snapshot"])
	}

	if cloneParams["dataset_dst"] != "tank/restored" {
		t.Errorf("expected dataset_dst 'tank/restored', got %v", cloneParams["dataset_dst"])
	}
}

func TestDatasetResource_Create_WithSnapshotId_APIError(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "pool.snapshot.clone" {
					return nil, errors.New("snapshot not found")
				}
				return nil, nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)
	planValue := createDatasetResourceModelValue(datasetModelParams{
		Pool:       "tank",
		Path:       "restored",
		SnapshotID: "tank/data@nonexistent",
	})

	req := resource.CreateRequest{
		Plan: tfsdk.Plan{
			Schema: schemaResp.Schema,
			Raw:    planValue,
		},
	}

	resp := &resource.CreateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
		},
	}

	r.Create(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for clone API error")
	}
}
```

**Step 2: Add snapshot_id to schema**

In `dataset.go`, add to Schema:

```go
"snapshot_id": schema.StringAttribute{
	Description: "Create dataset as clone from this snapshot. Mutually exclusive with other creation options.",
	Optional:    true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
},
```

Add to DatasetResourceModel:

```go
SnapshotID types.String `tfsdk:"snapshot_id"`
```

**Step 3: Modify Create to handle clone**

In `dataset.go` Create method, add before regular create:

```go
// If snapshot_id is set, use clone instead of create
if !data.SnapshotID.IsNull() && data.SnapshotID.ValueString() != "" {
	cloneParams := map[string]any{
		"snapshot":    data.SnapshotID.ValueString(),
		"dataset_dst": datasetID,
	}

	_, err := r.client.Call(ctx, "pool.snapshot.clone", cloneParams)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Clone Snapshot",
			fmt.Sprintf("Unable to clone snapshot to dataset: %s", err.Error()),
		)
		return
	}

	// Continue to read the created dataset...
}
```

**Step 4: Run test to verify it passes**

Run: `go test ./internal/resources/ -v -run "TestDatasetResource_Create_WithSnapshotId"`
Expected: PASS

**Step 5: Commit clone support**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): add snapshot_id for clone creation"
```

---

## Task 11: Documentation

**Files:**
- Create: `docs/resources/snapshot.md`
- Create: `docs/data-sources/snapshots.md`
- Modify: `docs/resources/dataset.md`

**Step 1: Create snapshot resource docs**

```markdown
# truenas_snapshot Resource

Manages a ZFS snapshot for pre-upgrade backups and point-in-time recovery.

## Example Usage

```hcl
resource "truenas_dataset" "app_data" {
  pool = "tank"
  path = "apps/myapp"
}

resource "truenas_snapshot" "pre_upgrade" {
  dataset_id = truenas_dataset.app_data.id
  name       = "pre-v2-upgrade"
  hold       = true
}
```

## Argument Reference

* `dataset_id` - (Required) Dataset ID to snapshot. Reference a truenas_dataset resource or data source.
* `name` - (Required) Snapshot name.
* `hold` - (Optional) Prevent automatic deletion. Default: false.
* `recursive` - (Optional) Include child datasets. Default: false.

## Attribute Reference

* `id` - Snapshot identifier (dataset@name).
* `createtxg` - Transaction group when created.
* `used_bytes` - Space consumed by snapshot.
* `referenced_bytes` - Space referenced by snapshot.

## Import

Snapshots can be imported using the snapshot ID:

```bash
terraform import truenas_snapshot.example "tank/data@snap1"
```
```

**Step 2: Create snapshots data source docs**

```markdown
# truenas_snapshots Data Source

Retrieves snapshots for a dataset.

## Example Usage

```hcl
data "truenas_snapshots" "backups" {
  dataset_id   = truenas_dataset.app_data.id
  name_pattern = "pre-*"
}

output "backup_count" {
  value = length(data.truenas_snapshots.backups.snapshots)
}
```

## Argument Reference

* `dataset_id` - (Required) Dataset ID to query snapshots for.
* `recursive` - (Optional) Include child dataset snapshots. Default: false.
* `name_pattern` - (Optional) Glob pattern to filter snapshot names.

## Attribute Reference

* `snapshots` - List of snapshot objects with:
  * `id` - Snapshot ID (dataset@name).
  * `name` - Snapshot name.
  * `dataset_id` - Parent dataset ID.
  * `used_bytes` - Space consumed.
  * `referenced_bytes` - Space referenced.
  * `hold` - Whether held.
```

**Step 3: Update dataset docs with snapshot_id**

Add to dataset.md:

```markdown
### Creating from Snapshot (Clone)

```hcl
resource "truenas_dataset" "restored" {
  pool        = "tank"
  path        = "apps/restored"
  snapshot_id = truenas_snapshot.backup.id
}
```

* `snapshot_id` - (Optional) Create dataset as clone from this snapshot.
```

**Step 4: Commit docs**

```bash
git add docs/
git commit -m "docs: add snapshot resource and data source documentation"
```

---

## Task 12: Run Full Test Suite

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All PASS

**Step 2: Run with coverage**

Run: `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out`
Expected: Coverage report showing snapshot coverage

**Step 3: Build provider**

Run: `go build ./...`
Expected: Success

**Step 4: Final commit**

```bash
git add -A
git commit -m "chore: final cleanup and verification"
```

---

## Summary

**Total Tasks:** 12
**Estimated Tests:** 32+

**Deliverables:**
- `internal/resources/snapshot.go` - Snapshot resource
- `internal/resources/snapshot_test.go` - Snapshot tests
- `internal/datasources/snapshots.go` - Snapshots data source
- `internal/datasources/snapshots_test.go` - Snapshots data source tests
- `internal/resources/dataset.go` - Modified for snapshot_id
- `internal/resources/dataset_test.go` - Clone tests
- `internal/provider/provider.go` - Register new resource/datasource
- `docs/resources/snapshot.md` - Resource documentation
- `docs/data-sources/snapshots.md` - Data source documentation
