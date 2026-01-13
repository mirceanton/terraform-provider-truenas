# Deprecate host_path Resource, Improve Dataset Schema - Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Deprecate the `truenas_host_path` resource and improve the `truenas_dataset` schema by consolidating `name` into `path` and renaming `mount_path` to `full_path`.

**Architecture:** Add deprecation warnings to `host_path` schema. In `dataset`, keep both old and new attributes functional - `name` maps to `path` internally, `mount_path` syncs with `full_path`. Existing configs continue working while showing deprecation warnings.

**Tech Stack:** Go, Terraform Plugin Framework, TrueNAS API via midclt

---

## Task 1: Add `full_path` Computed Attribute to Dataset Model

**Files:**
- Modify: `internal/resources/dataset.go:28-43` (DatasetResourceModel struct)

**Step 1: Write test for `full_path` attribute existence**

Add to `internal/resources/dataset_test.go`:

```go
func TestDatasetResource_Schema_FullPathExists(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	fullPathAttr, ok := resp.Schema.Attributes["full_path"]
	if !ok {
		t.Fatal("expected 'full_path' attribute in schema")
	}
	if !fullPathAttr.IsComputed() {
		t.Error("expected 'full_path' attribute to be computed")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_FullPathExists`
Expected: FAIL with "expected 'full_path' attribute in schema"

**Step 3: Add `full_path` field to model**

In `internal/resources/dataset.go`, add to `DatasetResourceModel` struct after `MountPath`:

```go
FullPath     types.String `tfsdk:"full_path"`
```

**Step 4: Add `full_path` to schema**

In `internal/resources/dataset.go` `Schema()` method, add after `mount_path`:

```go
"full_path": schema.StringAttribute{
	Description: "Full filesystem path to the mounted dataset (e.g., '/mnt/tank/data').",
	Computed:    true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
	},
},
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_FullPathExists`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): add full_path computed attribute to schema"
```

---

## Task 2: Update `mapDatasetToModel` to Sync Both `mount_path` and `full_path`

**Files:**
- Modify: `internal/resources/dataset.go:97-104` (mapDatasetToModel function)

**Step 1: Write test verifying both attributes are populated**

Add to `internal/resources/dataset_test.go`:

```go
func TestDatasetResource_Read_BothMountPathAndFullPath(t *testing.T) {
	r := &DatasetResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": "storage/apps",
					"name": "storage/apps",
					"mountpoint": "/mnt/storage/apps",
					"compression": {"value": "lz4"},
					"quota": {"value": "0"},
					"refquota": {"value": "0"},
					"atime": {"value": "on"}
				}]`), nil
			},
		},
	}

	schemaResp := getDatasetResourceSchema(t)
	stateValue := createDatasetResourceModelWithFullPath("storage/apps", "storage", "apps", nil, nil, "/mnt/storage/apps", "/mnt/storage/apps", "lz4", nil, nil, nil, nil, nil, nil, nil)

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

	var model DatasetResourceModel
	diags := resp.State.Get(context.Background(), &model)
	if diags.HasError() {
		t.Fatalf("failed to get state: %v", diags)
	}

	if model.MountPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected MountPath '/mnt/storage/apps', got %q", model.MountPath.ValueString())
	}
	if model.FullPath.ValueString() != "/mnt/storage/apps" {
		t.Errorf("expected FullPath '/mnt/storage/apps', got %q", model.FullPath.ValueString())
	}
}
```

**Step 2: Add helper function for test**

Add to `internal/resources/dataset_test.go`:

```go
// createDatasetResourceModelWithFullPath creates a tftypes.Value including full_path
func createDatasetResourceModelWithFullPath(id, pool, path, parent, name, mountPath, fullPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"pool":          tftypes.NewValue(tftypes.String, pool),
		"path":          tftypes.NewValue(tftypes.String, path),
		"parent":        tftypes.NewValue(tftypes.String, parent),
		"name":          tftypes.NewValue(tftypes.String, name),
		"mount_path":    tftypes.NewValue(tftypes.String, mountPath),
		"full_path":     tftypes.NewValue(tftypes.String, fullPath),
		"compression":   tftypes.NewValue(tftypes.String, compression),
		"quota":         tftypes.NewValue(tftypes.String, quota),
		"refquota":      tftypes.NewValue(tftypes.String, refquota),
		"atime":         tftypes.NewValue(tftypes.String, atime),
		"mode":          tftypes.NewValue(tftypes.String, mode),
		"uid":           tftypes.NewValue(tftypes.Number, uid),
		"gid":           tftypes.NewValue(tftypes.Number, gid),
		"force_destroy": tftypes.NewValue(tftypes.Bool, forceDestroy),
	})
}
```

**Step 3: Run test to verify it fails**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Read_BothMountPathAndFullPath`
Expected: FAIL (schema mismatch or full_path not populated)

**Step 4: Update mapDatasetToModel to set both attributes**

In `internal/resources/dataset.go`, modify `mapDatasetToModel`:

```go
func mapDatasetToModel(ds *datasetQueryResponse, data *DatasetResourceModel) {
	data.ID = types.StringValue(ds.ID)
	data.MountPath = types.StringValue(ds.Mountpoint)
	data.FullPath = types.StringValue(ds.Mountpoint)
	data.Compression = types.StringValue(ds.Compression.Value)
	data.Quota = types.StringValue(ds.Quota.Value)
	data.RefQuota = types.StringValue(ds.RefQuota.Value)
	data.Atime = types.StringValue(ds.Atime.Value)
}
```

**Step 5: Run test to verify it passes**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Read_BothMountPathAndFullPath`
Expected: PASS

**Step 6: Commit**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): sync full_path from API response"
```

---

## Task 3: Update Test Helpers to Include `full_path`

**Files:**
- Modify: `internal/resources/dataset_test.go:214-254` (createDatasetResourceModel functions)

**Step 1: Update existing helper to include full_path**

Modify `createDatasetResourceModel` and `createDatasetResourceModelWithPerms` to include `full_path`:

```go
func createDatasetResourceModel(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy interface{}) tftypes.Value {
	return createDatasetResourceModelWithPerms(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy, nil, nil, nil)
}

func createDatasetResourceModelWithPerms(id, pool, path, parent, name, mountPath, compression, quota, refquota, atime, forceDestroy, mode, uid, gid interface{}) tftypes.Value {
	return tftypes.NewValue(tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":            tftypes.String,
			"pool":          tftypes.String,
			"path":          tftypes.String,
			"parent":        tftypes.String,
			"name":          tftypes.String,
			"mount_path":    tftypes.String,
			"full_path":     tftypes.String,
			"compression":   tftypes.String,
			"quota":         tftypes.String,
			"refquota":      tftypes.String,
			"atime":         tftypes.String,
			"mode":          tftypes.String,
			"uid":           tftypes.Number,
			"gid":           tftypes.Number,
			"force_destroy": tftypes.Bool,
		},
	}, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, id),
		"pool":          tftypes.NewValue(tftypes.String, pool),
		"path":          tftypes.NewValue(tftypes.String, path),
		"parent":        tftypes.NewValue(tftypes.String, parent),
		"name":          tftypes.NewValue(tftypes.String, name),
		"mount_path":    tftypes.NewValue(tftypes.String, mountPath),
		"full_path":     tftypes.NewValue(tftypes.String, mountPath), // sync with mount_path
		"compression":   tftypes.NewValue(tftypes.String, compression),
		"quota":         tftypes.NewValue(tftypes.String, quota),
		"refquota":      tftypes.NewValue(tftypes.String, refquota),
		"atime":         tftypes.NewValue(tftypes.String, atime),
		"mode":          tftypes.NewValue(tftypes.String, mode),
		"uid":           tftypes.NewValue(tftypes.Number, uid),
		"gid":           tftypes.NewValue(tftypes.Number, gid),
		"force_destroy": tftypes.NewValue(tftypes.Bool, forceDestroy),
	})
}
```

**Step 2: Run all tests to verify no regressions**

Run: `go test -v ./internal/resources/... -run TestDataset`
Expected: All existing tests PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset_test.go
git commit -m "test(dataset): update test helpers to include full_path attribute"
```

---

## Task 4: Add Deprecation Warning to `mount_path` Attribute

**Files:**
- Modify: `internal/resources/dataset.go:154-160` (mount_path schema)

**Step 1: Write test for deprecation message**

Add to `internal/resources/dataset_test.go`:

```go
func TestDatasetResource_Schema_MountPathDeprecated(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	mountPathAttr, ok := resp.Schema.Attributes["mount_path"]
	if !ok {
		t.Fatal("expected 'mount_path' attribute in schema")
	}
	if mountPathAttr.GetDeprecationMessage() == "" {
		t.Error("expected 'mount_path' attribute to have deprecation message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_MountPathDeprecated`
Expected: FAIL with "expected 'mount_path' attribute to have deprecation message"

**Step 3: Add deprecation message to mount_path**

In `internal/resources/dataset.go`, update `mount_path` schema:

```go
"mount_path": schema.StringAttribute{
	Description:        "Filesystem mount path.",
	DeprecationMessage: "Use 'full_path' instead. This attribute will be removed in a future version.",
	Computed:           true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.UseStateForUnknown(),
	},
},
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_MountPathDeprecated`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): deprecate mount_path in favor of full_path"
```

---

## Task 5: Add Deprecation Warning to `name` Attribute

**Files:**
- Modify: `internal/resources/dataset.go:146-153` (name schema)

**Step 1: Write test for deprecation message**

Add to `internal/resources/dataset_test.go`:

```go
func TestDatasetResource_Schema_NameDeprecated(t *testing.T) {
	r := NewDatasetResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	nameAttr, ok := resp.Schema.Attributes["name"]
	if !ok {
		t.Fatal("expected 'name' attribute in schema")
	}
	if nameAttr.GetDeprecationMessage() == "" {
		t.Error("expected 'name' attribute to have deprecation message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_NameDeprecated`
Expected: FAIL with "expected 'name' attribute to have deprecation message"

**Step 3: Add deprecation message to name**

In `internal/resources/dataset.go`, update `name` schema:

```go
"name": schema.StringAttribute{
	Description:        "Dataset name. Use with 'parent' attribute.",
	DeprecationMessage: "Use 'path' instead with 'parent'. This attribute will be removed in a future version.",
	Optional:           true,
	PlanModifiers: []planmodifier.String{
		stringplanmodifier.RequiresReplace(),
	},
},
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/resources/... -run TestDatasetResource_Schema_NameDeprecated`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): deprecate name in favor of path with parent"
```

---

## Task 6: Update `getFullName` to Prefer `path` with `parent`

**Files:**
- Modify: `internal/resources/dataset.go:509-534` (getFullName function)

**Step 1: Write test for path+parent precedence over name+parent**

Add to `internal/resources/dataset_test.go`:

```go
func TestGetFullName_PathWithParent(t *testing.T) {
	// path with parent should work (new preferred way)
	model := DatasetResourceModel{
		Parent: stringValue("tank/data"),
		Path:   stringValue("apps"),
	}
	result := getFullName(&model)
	if result != "tank/data/apps" {
		t.Errorf("expected 'tank/data/apps', got %q", result)
	}
}

func TestGetFullName_PathOverName(t *testing.T) {
	// when both path and name are provided with parent, path takes precedence
	model := DatasetResourceModel{
		Parent: stringValue("tank/data"),
		Path:   stringValue("newpath"),
		Name:   stringValue("oldname"),
	}
	result := getFullName(&model)
	if result != "tank/data/newpath" {
		t.Errorf("expected 'tank/data/newpath', got %q", result)
	}
}
```

**Step 2: Run tests to verify current behavior**

Run: `go test -v ./internal/resources/... -run TestGetFullName_Path`
Expected: May FAIL depending on current implementation

**Step 3: Update getFullName to handle path+parent**

In `internal/resources/dataset.go`, update `getFullName`:

```go
func getFullName(data *DatasetResourceModel) string {
	// Mode 1: pool + path
	hasPool := !data.Pool.IsNull() && !data.Pool.IsUnknown() && data.Pool.ValueString() != ""
	hasPath := !data.Path.IsNull() && !data.Path.IsUnknown() && data.Path.ValueString() != ""

	if hasPool && hasPath {
		return fmt.Sprintf("%s/%s", data.Pool.ValueString(), data.Path.ValueString())
	}

	// Mode 2: parent + path (new preferred way) or parent + name (deprecated)
	hasParent := !data.Parent.IsNull() && !data.Parent.IsUnknown() && data.Parent.ValueString() != ""
	hasName := !data.Name.IsNull() && !data.Name.IsUnknown() && data.Name.ValueString() != ""

	if hasParent {
		// Prefer path over name when both are set
		if hasPath {
			return fmt.Sprintf("%s/%s", data.Parent.ValueString(), data.Path.ValueString())
		}
		if hasName {
			return fmt.Sprintf("%s/%s", data.Parent.ValueString(), data.Name.ValueString())
		}
	}

	// Invalid configuration
	return ""
}
```

**Step 4: Run tests to verify it passes**

Run: `go test -v ./internal/resources/... -run TestGetFullName`
Expected: All TestGetFullName tests PASS

**Step 5: Commit**

```bash
git add internal/resources/dataset.go internal/resources/dataset_test.go
git commit -m "feat(dataset): support path attribute with parent, prefer over name"
```

---

## Task 7: Update Dataset Description for New Schema

**Files:**
- Modify: `internal/resources/dataset.go:117` (schema description)
- Modify: `internal/resources/dataset.go:127-139` (pool and path descriptions)

**Step 1: Update schema descriptions**

In `internal/resources/dataset.go`:

```go
resp.Schema = schema.Schema{
	Description: "Manages a TrueNAS dataset. Use nested datasets instead of host_path for app storage.",
	// ...
}
```

Update `pool` description:

```go
"pool": schema.StringAttribute{
	Description: "Pool name. Use with 'path' attribute for pool-relative paths.",
	// ...
},
```

Update `path` description:

```go
"path": schema.StringAttribute{
	Description: "Dataset path. With 'pool': relative path in pool. With 'parent': child dataset name.",
	// ...
},
```

Update `parent` description:

```go
"parent": schema.StringAttribute{
	Description: "Parent dataset ID (e.g., 'tank/data'). Use with 'path' attribute.",
	// ...
},
```

**Step 2: Run all dataset tests**

Run: `go test -v ./internal/resources/... -run TestDataset`
Expected: All tests PASS

**Step 3: Commit**

```bash
git add internal/resources/dataset.go
git commit -m "docs(dataset): update schema descriptions for new path usage"
```

---

## Task 8: Add Deprecation Warning to host_path Resource

**Files:**
- Modify: `internal/resources/host_path.go:55-103` (Schema method)

**Step 1: Write test for deprecation message**

Add to `internal/resources/host_path_test.go`:

```go
func TestHostPathResource_Schema_Deprecated(t *testing.T) {
	r := NewHostPathResource()

	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	r.Schema(context.Background(), req, resp)

	if resp.Schema.DeprecationMessage == "" {
		t.Error("expected host_path schema to have deprecation message")
	}
}
```

**Step 2: Run test to verify it fails**

Run: `go test -v ./internal/resources/... -run TestHostPathResource_Schema_Deprecated`
Expected: FAIL with "expected host_path schema to have deprecation message"

**Step 3: Add deprecation message to schema**

In `internal/resources/host_path.go`, update `Schema()`:

```go
func (r *HostPathResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description:        "Manages a TrueNAS host path directory for app storage mounts.",
		DeprecationMessage: "Use truenas_dataset with nested datasets instead. host_path relies on SFTP which may not work with non-root SSH users. Datasets are created via the TrueNAS API and provide better ZFS integration.",
		Attributes: map[string]schema.Attribute{
			// ... existing attributes
		},
	}
}
```

**Step 4: Run test to verify it passes**

Run: `go test -v ./internal/resources/... -run TestHostPathResource_Schema_Deprecated`
Expected: PASS

**Step 5: Commit**

```bash
git add internal/resources/host_path.go internal/resources/host_path_test.go
git commit -m "feat(host_path): add deprecation warning recommending datasets"
```

---

## Task 9: Run Full Test Suite and Verify

**Files:**
- All files modified in previous tasks

**Step 1: Run all tests**

Run: `go test -v ./...`
Expected: All tests PASS

**Step 2: Run linter**

Run: `mise run lint` or `golangci-lint run`
Expected: No errors

**Step 3: Build provider**

Run: `go build ./...`
Expected: Build succeeds

**Step 4: Commit any fixes if needed**

```bash
git add -A
git commit -m "fix: address test or lint issues"
```

---

## Task 10: Test Backwards Compatibility Manually

**Files:**
- None (manual testing)

**Step 1: Create test config using deprecated attributes**

Create a test Terraform config using old-style attributes:

```hcl
# test-deprecated.tf
resource "truenas_dataset" "test_old_style" {
  parent = "storage/apps"
  name   = "test-deprecated"  # deprecated
}

output "mount_path" {
  value = truenas_dataset.test_old_style.mount_path  # deprecated
}
```

**Step 2: Run terraform plan**

Run: `terraform plan`
Expected: Plan succeeds with deprecation warnings for `name` and `mount_path`

**Step 3: Create test config using new attributes**

```hcl
# test-new.tf
resource "truenas_dataset" "test_new_style" {
  parent = "storage/apps"
  path   = "test-new"  # new preferred way
}

output "full_path" {
  value = truenas_dataset.test_new_style.full_path  # new
}
```

**Step 4: Run terraform plan**

Run: `terraform plan`
Expected: Plan succeeds without deprecation warnings

---

## Summary of Changes

| File | Changes |
|------|---------|
| `internal/resources/dataset.go` | Add `full_path` field and schema, deprecate `mount_path` and `name`, update `getFullName` to support `path+parent`, update descriptions |
| `internal/resources/dataset_test.go` | Add tests for `full_path`, deprecation messages, `path+parent` support, update test helpers |
| `internal/resources/host_path.go` | Add deprecation message to schema |
| `internal/resources/host_path_test.go` | Add test for schema deprecation |

## Backwards Compatibility

- Existing configs using `name` continue to work (with deprecation warning)
- Existing configs using `mount_path` continue to work (with deprecation warning)
- Both `mount_path` and `full_path` return the same value
- `getFullName` prefers `path` over `name` when both are set with `parent`
