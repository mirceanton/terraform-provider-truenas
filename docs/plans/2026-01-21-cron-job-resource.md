# Cron Job Resource Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Add a `truenas_cron_job` resource to manage scheduled tasks on TrueNAS via the cron job API.

**Architecture:** Implement a new Terraform resource following the existing CloudSync Task pattern. The resource will use the standard CRUD operations (`cronjob.query`, `cronjob.create`, `cronjob.update`, `cronjob.delete`) via the SSH/midclt client. The schedule block will be reused from the existing CloudSync implementation pattern.

**Tech Stack:** Go, Terraform Plugin Framework, existing SSH client infrastructure

---

## Files Overview

| Action | File | Purpose |
|--------|------|---------|
| Create | `internal/api/cron.go` | API response struct for cron jobs |
| Create | `internal/resources/cron_job.go` | Resource implementation with CRUD |
| Create | `internal/resources/cron_job_test.go` | Unit tests with 100% coverage |
| Create | `docs/resources/cron_job.md` | User documentation |
| Modify | `internal/provider/provider.go` | Register the new resource |

---

## Task 1: Create API Response Struct

**Files:**
- Create: `internal/api/cron.go`

**Step 1: Write the API struct file**

```go
package api

// CronJobResponse represents a cron job from the TrueNAS API.
type CronJobResponse struct {
	ID          int64            `json:"id"`
	User        string           `json:"user"`
	Command     string           `json:"command"`
	Description string           `json:"description"`
	Enabled     bool             `json:"enabled"`
	Stdout      bool             `json:"stdout"`
	Stderr      bool             `json:"stderr"`
	Schedule    ScheduleResponse `json:"schedule"`
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/api/cron.go
git commit -m "$(cat <<'EOF'
feat(api): add cron job API response struct

Add CronJobResponse struct for parsing TrueNAS cronjob.query responses.
Reuses existing ScheduleResponse for the schedule block.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 2: Create Cron Job Resource - Types and Scaffolding

**Files:**
- Create: `internal/resources/cron_job.go`

**Step 1: Write the resource scaffolding with types**

```go
package resources

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"

	"github.com/deevus/terraform-provider-truenas/internal/api"
	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &CronJobResource{}
	_ resource.ResourceWithConfigure   = &CronJobResource{}
	_ resource.ResourceWithImportState = &CronJobResource{}
)

// CronJobResourceModel describes the resource data model.
type CronJobResourceModel struct {
	ID          types.String   `tfsdk:"id"`
	User        types.String   `tfsdk:"user"`
	Command     types.String   `tfsdk:"command"`
	Description types.String   `tfsdk:"description"`
	Enabled     types.Bool     `tfsdk:"enabled"`
	Stdout      types.Bool     `tfsdk:"stdout"`
	Stderr      types.Bool     `tfsdk:"stderr"`
	Schedule    *ScheduleBlock `tfsdk:"schedule"`
}

// CronJobResource defines the resource implementation.
type CronJobResource struct {
	client client.Client
}

// NewCronJobResource creates a new CronJobResource.
func NewCronJobResource() resource.Resource {
	return &CronJobResource{}
}

func (r *CronJobResource) Metadata(ctx context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_cron_job"
}

func (r *CronJobResource) Schema(ctx context.Context, req resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		Description: "Manages cron jobs for scheduled task execution.",
		Attributes: map[string]schema.Attribute{
			"id": schema.StringAttribute{
				Description: "Cron job ID.",
				Computed:    true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"user": schema.StringAttribute{
				Description: "User to run the command as.",
				Required:    true,
			},
			"command": schema.StringAttribute{
				Description: "Command to execute.",
				Required:    true,
			},
			"description": schema.StringAttribute{
				Description: "Job description.",
				Optional:    true,
				Computed:    true,
				Default:     stringdefault.StaticString(""),
			},
			"enabled": schema.BoolAttribute{
				Description: "Enable the cron job.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stdout": schema.BoolAttribute{
				Description: "Redirect stdout to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
			"stderr": schema.BoolAttribute{
				Description: "Redirect stderr to syslog.",
				Optional:    true,
				Computed:    true,
				Default:     booldefault.StaticBool(true),
			},
		},
		Blocks: map[string]schema.Block{
			"schedule": schema.SingleNestedBlock{
				Description: "Cron schedule for the job.",
				Attributes: map[string]schema.Attribute{
					"minute": schema.StringAttribute{
						Description: "Minute (0-59 or cron expression).",
						Required:    true,
					},
					"hour": schema.StringAttribute{
						Description: "Hour (0-23 or cron expression).",
						Required:    true,
					},
					"dom": schema.StringAttribute{
						Description: "Day of month (1-31 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"month": schema.StringAttribute{
						Description: "Month (1-12 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
					"dow": schema.StringAttribute{
						Description: "Day of week (0-6 or cron expression).",
						Optional:    true,
						Computed:    true,
						Default:     stringdefault.StaticString("*"),
					},
				},
			},
		},
	}
}

func (r *CronJobResource) Configure(ctx context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
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

func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	// TODO: implement in next task
}

func (r *CronJobResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("id"), req, resp)
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): add cron job resource scaffolding

Add CronJobResource with types, schema, and empty CRUD methods.
Reuses ScheduleBlock from CloudSync for cron scheduling.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 3: Implement Helper Functions

**Files:**
- Modify: `internal/resources/cron_job.go`

**Step 1: Add queryCronJob helper function**

Add after ImportState method:

```go
// queryCronJob queries a cron job by ID and returns the response.
func (r *CronJobResource) queryCronJob(ctx context.Context, id int64) (*api.CronJobResponse, error) {
	filter := [][]any{{"id", "=", id}}
	result, err := r.client.Call(ctx, "cronjob.query", filter)
	if err != nil {
		return nil, err
	}

	var jobs []api.CronJobResponse
	if err := json.Unmarshal(result, &jobs); err != nil {
		return nil, fmt.Errorf("parse response: %w", err)
	}

	if len(jobs) == 0 {
		return nil, nil
	}

	return &jobs[0], nil
}
```

**Step 2: Add buildCronJobParams helper function**

Add after queryCronJob:

```go
// buildCronJobParams builds the API params from the resource model.
func buildCronJobParams(data *CronJobResourceModel) map[string]any {
	params := map[string]any{
		"user":        data.User.ValueString(),
		"command":     data.Command.ValueString(),
		"description": data.Description.ValueString(),
		"enabled":     data.Enabled.ValueBool(),
		"stdout":      data.Stdout.ValueBool(),
		"stderr":      data.Stderr.ValueBool(),
	}

	if data.Schedule != nil {
		params["schedule"] = map[string]any{
			"minute": data.Schedule.Minute.ValueString(),
			"hour":   data.Schedule.Hour.ValueString(),
			"dom":    data.Schedule.Dom.ValueString(),
			"month":  data.Schedule.Month.ValueString(),
			"dow":    data.Schedule.Dow.ValueString(),
		}
	}

	return params
}
```

**Step 3: Add mapCronJobToModel helper function**

Add after buildCronJobParams:

```go
// mapCronJobToModel maps an API response to the resource model.
func mapCronJobToModel(job *api.CronJobResponse, data *CronJobResourceModel) {
	data.ID = types.StringValue(strconv.FormatInt(job.ID, 10))
	data.User = types.StringValue(job.User)
	data.Command = types.StringValue(job.Command)
	data.Description = types.StringValue(job.Description)
	data.Enabled = types.BoolValue(job.Enabled)
	data.Stdout = types.BoolValue(job.Stdout)
	data.Stderr = types.BoolValue(job.Stderr)

	if data.Schedule != nil {
		data.Schedule.Minute = types.StringValue(job.Schedule.Minute)
		data.Schedule.Hour = types.StringValue(job.Schedule.Hour)
		data.Schedule.Dom = types.StringValue(job.Schedule.Dom)
		data.Schedule.Month = types.StringValue(job.Schedule.Month)
		data.Schedule.Dow = types.StringValue(job.Schedule.Dow)
	}
}
```

**Step 4: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 5: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): add cron job helper functions

Add queryCronJob, buildCronJobParams, and mapCronJobToModel helpers
for API interaction and data mapping.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 4: Implement Create Method

**Files:**
- Modify: `internal/resources/cron_job.go`

**Step 1: Implement Create method**

Replace the Create method:

```go
func (r *CronJobResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var data CronJobResourceModel

	resp.Diagnostics.Append(req.Plan.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Build params
	params := buildCronJobParams(&data)

	// Call API
	result, err := r.client.Call(ctx, "cronjob.create", params)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Create Cron Job",
			fmt.Sprintf("Unable to create cron job: %s", err.Error()),
		)
		return
	}

	// Parse response to get ID
	var createResp struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal(result, &createResp); err != nil {
		resp.Diagnostics.AddError(
			"Unable to Parse Response",
			fmt.Sprintf("Unable to parse create response: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	job, err := r.queryCronJob(ctx, createResp.ID)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Cron job created but unable to read: %s", err.Error()),
		)
		return
	}

	if job == nil {
		resp.Diagnostics.AddError(
			"Cron Job Not Found",
			"Cron job was created but could not be found.",
		)
		return
	}

	// Set state from response
	mapCronJobToModel(job, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): implement cron job Create method

Implement Create with API call, response parsing, and state mapping.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 5: Implement Read Method

**Files:**
- Modify: `internal/resources/cron_job.go`

**Step 1: Implement Read method**

Replace the Read method:

```go
func (r *CronJobResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var data CronJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	job, err := r.queryCronJob(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Unable to query cron job: %s", err.Error()),
		)
		return
	}

	if job == nil {
		// Cron job was deleted outside Terraform
		resp.State.RemoveResource(ctx)
		return
	}

	mapCronJobToModel(job, &data)

	resp.Diagnostics.Append(resp.State.Set(ctx, &data)...)
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): implement cron job Read method

Implement Read with query and state removal for deleted resources.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 6: Implement Update Method

**Files:**
- Modify: `internal/resources/cron_job.go`

**Step 1: Implement Update method**

Replace the Update method:

```go
func (r *CronJobResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var state CronJobResourceModel
	var plan CronJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Parse ID from state
	id, err := strconv.ParseInt(state.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", state.ID.ValueString(), err.Error()),
		)
		return
	}

	// Build update params
	params := buildCronJobParams(&plan)

	// Call API with []any{id, params}
	_, err = r.client.Call(ctx, "cronjob.update", []any{id, params})
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Update Cron Job",
			fmt.Sprintf("Unable to update cron job: %s", err.Error()),
		)
		return
	}

	// Query to get full state
	job, err := r.queryCronJob(ctx, id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Read Cron Job",
			fmt.Sprintf("Cron job updated but unable to read: %s", err.Error()),
		)
		return
	}

	if job == nil {
		resp.Diagnostics.AddError(
			"Cron Job Not Found",
			"Cron job was updated but could not be found.",
		)
		return
	}

	// Set state from response
	mapCronJobToModel(job, &plan)

	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): implement cron job Update method

Implement Update with API call using []any{id, params} pattern.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 7: Implement Delete Method

**Files:**
- Modify: `internal/resources/cron_job.go`

**Step 1: Implement Delete method**

Replace the Delete method:

```go
func (r *CronJobResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var data CronJobResourceModel

	resp.Diagnostics.Append(req.State.Get(ctx, &data)...)
	if resp.Diagnostics.HasError() {
		return
	}

	id, err := strconv.ParseInt(data.ID.ValueString(), 10, 64)
	if err != nil {
		resp.Diagnostics.AddError(
			"Invalid ID",
			fmt.Sprintf("Unable to parse ID %q: %s", data.ID.ValueString(), err.Error()),
		)
		return
	}

	_, err = r.client.Call(ctx, "cronjob.delete", id)
	if err != nil {
		resp.Diagnostics.AddError(
			"Unable to Delete Cron Job",
			fmt.Sprintf("Unable to delete cron job: %s", err.Error()),
		)
		return
	}
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job.go
git commit -m "$(cat <<'EOF'
feat(resources): implement cron job Delete method

Implement Delete with API call to cronjob.delete.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 8: Register Resource in Provider

**Files:**
- Modify: `internal/provider/provider.go:167-177`

**Step 1: Add resource to Resources function**

In `internal/provider/provider.go`, find the Resources function and add the new resource:

```go
func (p *TrueNASProvider) Resources(ctx context.Context) []func() resource.Resource {
	return []func() resource.Resource{
		resources.NewDatasetResource,
		resources.NewHostPathResource,
		resources.NewAppResource,
		resources.NewFileResource,
		resources.NewSnapshotResource,
		resources.NewCloudSyncCredentialsResource,
		resources.NewCloudSyncTaskResource,
		resources.NewCronJobResource,
	}
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/provider/provider.go
git commit -m "$(cat <<'EOF'
feat(provider): register cron job resource

Add NewCronJobResource to provider's Resources function.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 9: Create Test File - Basic Tests

**Files:**
- Create: `internal/resources/cron_job_test.go`

**Step 1: Write basic test scaffolding**

```go
package resources

import (
	"context"
	"encoding/json"
	"errors"
	"math/big"
	"testing"

	"github.com/deevus/terraform-provider-truenas/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
)

func TestNewCronJobResource(t *testing.T) {
	r := NewCronJobResource()
	if r == nil {
		t.Fatal("NewCronJobResource returned nil")
	}

	_, ok := r.(*CronJobResource)
	if !ok {
		t.Fatalf("expected *CronJobResource, got %T", r)
	}

	// Verify interface implementations
	var _ resource.Resource = r
	var _ resource.ResourceWithConfigure = r.(*CronJobResource)
	var _ resource.ResourceWithImportState = r.(*CronJobResource)
}

func TestCronJobResource_Metadata(t *testing.T) {
	r := NewCronJobResource()

	req := resource.MetadataRequest{
		ProviderTypeName: "truenas",
	}
	resp := &resource.MetadataResponse{}

	r.Metadata(context.Background(), req, resp)

	if resp.TypeName != "truenas_cron_job" {
		t.Errorf("expected TypeName 'truenas_cron_job', got %q", resp.TypeName)
	}
}

func TestCronJobResource_Configure_Success(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	mockClient := &client.MockClient{}

	req := resource.ConfigureRequest{
		ProviderData: mockClient,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}

	if r.client == nil {
		t.Error("expected client to be set")
	}
}

func TestCronJobResource_Configure_NilProviderData(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics)
	}
}

func TestCronJobResource_Configure_WrongType(t *testing.T) {
	r := NewCronJobResource().(*CronJobResource)

	req := resource.ConfigureRequest{
		ProviderData: "not a client",
	}
	resp := &resource.ConfigureResponse{}

	r.Configure(context.Background(), req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for wrong ProviderData type")
	}
}

func TestCronJobResource_Schema(t *testing.T) {
	r := NewCronJobResource()

	ctx := context.Background()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}

	r.Schema(ctx, schemaReq, schemaResp)

	if schemaResp.Schema.Description == "" {
		t.Error("expected non-empty schema description")
	}

	// Verify required attributes exist
	attrs := schemaResp.Schema.Attributes
	if attrs["id"] == nil {
		t.Error("expected 'id' attribute")
	}
	if attrs["user"] == nil {
		t.Error("expected 'user' attribute")
	}
	if attrs["command"] == nil {
		t.Error("expected 'command' attribute")
	}
	if attrs["description"] == nil {
		t.Error("expected 'description' attribute")
	}
	if attrs["enabled"] == nil {
		t.Error("expected 'enabled' attribute")
	}
	if attrs["stdout"] == nil {
		t.Error("expected 'stdout' attribute")
	}
	if attrs["stderr"] == nil {
		t.Error("expected 'stderr' attribute")
	}

	// Verify blocks exist
	blocks := schemaResp.Schema.Blocks
	if blocks["schedule"] == nil {
		t.Error("expected 'schedule' block")
	}
}
```

**Step 2: Run tests to verify**

Run: `go test ./internal/resources/... -run TestCronJob -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job basic tests

Add tests for NewCronJobResource, Metadata, Configure, and Schema.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 10: Add Test Helpers

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add test helper functions**

Add after the Schema test:

```go
// Test helpers

func getCronJobResourceSchema(t *testing.T) resource.SchemaResponse {
	t.Helper()
	r := NewCronJobResource()
	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	r.Schema(context.Background(), schemaReq, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("failed to get schema: %v", schemaResp.Diagnostics)
	}
	return *schemaResp
}

// cronJobModelParams holds parameters for creating test model values.
type cronJobModelParams struct {
	ID          interface{}
	User        interface{}
	Command     interface{}
	Description interface{}
	Enabled     bool
	Stdout      bool
	Stderr      bool
	Schedule    *scheduleBlockParams
}

func createCronJobModelValue(p cronJobModelParams) tftypes.Value {
	// Define type structures
	scheduleType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"minute": tftypes.String,
			"hour":   tftypes.String,
			"dom":    tftypes.String,
			"month":  tftypes.String,
			"dow":    tftypes.String,
		},
	}

	// Build the values map
	values := map[string]tftypes.Value{
		"id":          tftypes.NewValue(tftypes.String, p.ID),
		"user":        tftypes.NewValue(tftypes.String, p.User),
		"command":     tftypes.NewValue(tftypes.String, p.Command),
		"description": tftypes.NewValue(tftypes.String, p.Description),
		"enabled":     tftypes.NewValue(tftypes.Bool, p.Enabled),
		"stdout":      tftypes.NewValue(tftypes.Bool, p.Stdout),
		"stderr":      tftypes.NewValue(tftypes.Bool, p.Stderr),
	}

	// Handle schedule block
	if p.Schedule != nil {
		values["schedule"] = tftypes.NewValue(scheduleType, map[string]tftypes.Value{
			"minute": tftypes.NewValue(tftypes.String, p.Schedule.Minute),
			"hour":   tftypes.NewValue(tftypes.String, p.Schedule.Hour),
			"dom":    tftypes.NewValue(tftypes.String, p.Schedule.Dom),
			"month":  tftypes.NewValue(tftypes.String, p.Schedule.Month),
			"dow":    tftypes.NewValue(tftypes.String, p.Schedule.Dow),
		})
	} else {
		values["schedule"] = tftypes.NewValue(scheduleType, nil)
	}

	// Create object type matching the schema
	objectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"id":          tftypes.String,
			"user":        tftypes.String,
			"command":     tftypes.String,
			"description": tftypes.String,
			"enabled":     tftypes.Bool,
			"stdout":      tftypes.Bool,
			"stderr":      tftypes.Bool,
			"schedule":    scheduleType,
		},
	}

	return tftypes.NewValue(objectType, values)
}
```

**Step 2: Run build to verify**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job test helpers

Add getCronJobResourceSchema and createCronJobModelValue helpers.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 11: Add Create Tests

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add Create success test**

Add after the test helpers:

```go
func TestCronJobResource_Create_Success(t *testing.T) {
	var capturedMethod string
	var capturedParams any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.create" {
					capturedMethod = method
					capturedParams = params
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 10,
						"user": "root",
						"command": "/usr/local/bin/backup.sh",
						"description": "Daily backup",
						"enabled": true,
						"stdout": true,
						"stderr": true,
						"schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	if capturedMethod != "cronjob.create" {
		t.Errorf("expected method 'cronjob.create', got %q", capturedMethod)
	}

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["user"] != "root" {
		t.Errorf("expected user 'root', got %v", params["user"])
	}
	if params["command"] != "/usr/local/bin/backup.sh" {
		t.Errorf("expected command '/usr/local/bin/backup.sh', got %v", params["command"])
	}

	// Verify schedule
	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}
	if schedule["minute"] != "0" {
		t.Errorf("expected schedule minute '0', got %v", schedule["minute"])
	}
	if schedule["hour"] != "2" {
		t.Errorf("expected schedule hour '2', got %v", schedule["hour"])
	}

	// Verify state was set
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.ID.ValueString() != "10" {
		t.Errorf("expected ID '10', got %q", resultData.ID.ValueString())
	}
}

func TestCronJobResource_Create_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

**Step 2: Run tests to verify**

Run: `go test ./internal/resources/... -run TestCronJobResource_Create -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job Create tests

Add tests for successful create and API error scenarios.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 12: Add Read Tests

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add Read tests**

Add after Create tests:

```go
func TestCronJobResource_Read_Success(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[{
					"id": 10,
					"user": "root",
					"command": "/usr/local/bin/backup.sh",
					"description": "Daily backup",
					"enabled": true,
					"stdout": true,
					"stderr": true,
					"schedule": {"minute": "0", "hour": "2", "dom": "*", "month": "*", "dow": "*"}
				}]`), nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	// Verify state was updated
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.User.ValueString() != "root" {
		t.Errorf("expected user 'root', got %q", resultData.User.ValueString())
	}
}

func TestCronJobResource_Read_NotFound(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return json.RawMessage(`[]`), nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Deleted job",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	// State should be removed (resource not found)
	if !resp.State.Raw.IsNull() {
		t.Error("expected state to be removed when resource not found")
	}
}

func TestCronJobResource_Read_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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
```

**Step 2: Run tests to verify**

Run: `go test ./internal/resources/... -run TestCronJobResource_Read -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job Read tests

Add tests for successful read, not found, and API error scenarios.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 13: Add Update Tests

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add Update tests**

Add after Read tests:

```go
func TestCronJobResource_Update_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64
	var capturedUpdateData map[string]any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.update" {
					capturedMethod = method
					args := params.([]any)
					capturedID = args[0].(int64)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 10}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 10,
						"user": "root",
						"command": "/usr/local/bin/new-backup.sh",
						"description": "Updated backup",
						"enabled": true,
						"stdout": true,
						"stderr": true,
						"schedule": {"minute": "30", "hour": "3", "dom": "*", "month": "*", "dow": "*"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	// Updated plan
	planValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/new-backup.sh",
		Description: "Updated backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	if capturedMethod != "cronjob.update" {
		t.Errorf("expected method 'cronjob.update', got %q", capturedMethod)
	}

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}

	if capturedUpdateData["command"] != "/usr/local/bin/new-backup.sh" {
		t.Errorf("expected command '/usr/local/bin/new-backup.sh', got %v", capturedUpdateData["command"])
	}

	// Verify state was set
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Command.ValueString() != "/usr/local/bin/new-backup.sh" {
		t.Errorf("expected command '/usr/local/bin/new-backup.sh', got %q", resultData.Command.ValueString())
	}
}

func TestCronJobResource_Update_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("connection refused")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	planValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/new-backup.sh",
		Description: "Updated backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "30",
			Hour:   "3",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	if !resp.Diagnostics.HasError() {
		t.Fatal("expected error for API error")
	}
}
```

**Step 2: Run tests to verify**

Run: `go test ./internal/resources/... -run TestCronJobResource_Update -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job Update tests

Add tests for successful update and API error scenarios.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 14: Add Delete Tests

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add Delete tests**

Add after Update tests:

```go
func TestCronJobResource_Delete_Success(t *testing.T) {
	var capturedMethod string
	var capturedID int64

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				capturedMethod = method
				capturedID = params.(int64)
				return json.RawMessage(`true`), nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	if capturedMethod != "cronjob.delete" {
		t.Errorf("expected method 'cronjob.delete', got %q", capturedMethod)
	}

	if capturedID != 10 {
		t.Errorf("expected ID 10, got %d", capturedID)
	}
}

func TestCronJobResource_Delete_APIError(t *testing.T) {
	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				return nil, errors.New("job in use by active process")
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "10",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

**Step 2: Run tests to verify**

Run: `go test ./internal/resources/... -run TestCronJobResource_Delete -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job Delete tests

Add tests for successful delete and API error scenarios.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 15: Add Schedule Variation Tests

**Files:**
- Modify: `internal/resources/cron_job_test.go`

**Step 1: Add schedule variation tests**

Add after Delete tests:

```go
func TestCronJobResource_Create_CustomSchedule(t *testing.T) {
	var capturedParams any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 20}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 20,
						"user": "root",
						"command": "/usr/local/bin/workday.sh",
						"description": "Workday task",
						"enabled": true,
						"stdout": true,
						"stderr": true,
						"schedule": {"minute": "*/15", "hour": "9-17", "dom": "*", "month": "*", "dow": "1-5"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	// Custom schedule: every 15 minutes during business hours, weekdays only
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/workday.sh",
		Description: "Workday task",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "*/15",
			Hour:   "9-17",
			Dom:    "*",
			Month:  "*",
			Dow:    "1-5",
		},
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

	// Verify schedule params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	schedule, ok := params["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", params["schedule"])
	}

	if schedule["minute"] != "*/15" {
		t.Errorf("expected schedule minute '*/15', got %v", schedule["minute"])
	}
	if schedule["hour"] != "9-17" {
		t.Errorf("expected schedule hour '9-17', got %v", schedule["hour"])
	}
	if schedule["dow"] != "1-5" {
		t.Errorf("expected schedule dow '1-5', got %v", schedule["dow"])
	}

	// Verify state was set correctly
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Schedule.Minute.ValueString() != "*/15" {
		t.Errorf("expected state schedule minute '*/15', got %q", resultData.Schedule.Minute.ValueString())
	}
}

func TestCronJobResource_Update_ScheduleOnly(t *testing.T) {
	var capturedUpdateData map[string]any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.update" {
					args := params.([]any)
					capturedUpdateData = args[1].(map[string]any)
					return json.RawMessage(`{"id": 21}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 21,
						"user": "root",
						"command": "/usr/local/bin/backup.sh",
						"description": "Daily backup",
						"enabled": true,
						"stdout": true,
						"stderr": true,
						"schedule": {"minute": "*/30", "hour": "0", "dom": "*", "month": "*", "dow": "0,6"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)

	// Current state: daily at 2am
	stateValue := createCronJobModelValue(cronJobModelParams{
		ID:          "21",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "2",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
	})

	// Updated plan: every 30 minutes at midnight on weekends
	planValue := createCronJobModelValue(cronJobModelParams{
		ID:          "21",
		User:        "root",
		Command:     "/usr/local/bin/backup.sh",
		Description: "Daily backup",
		Enabled:     true,
		Stdout:      true,
		Stderr:      true,
		Schedule: &scheduleBlockParams{
			Minute: "*/30",
			Hour:   "0",
			Dom:    "*",
			Month:  "*",
			Dow:    "0,6",
		},
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

	// Verify schedule was updated
	schedule, ok := capturedUpdateData["schedule"].(map[string]any)
	if !ok {
		t.Fatalf("expected schedule to be map[string]any, got %T", capturedUpdateData["schedule"])
	}
	if schedule["minute"] != "*/30" {
		t.Errorf("expected schedule minute '*/30', got %v", schedule["minute"])
	}
	if schedule["hour"] != "0" {
		t.Errorf("expected schedule hour '0', got %v", schedule["hour"])
	}
	if schedule["dow"] != "0,6" {
		t.Errorf("expected schedule dow '0,6', got %v", schedule["dow"])
	}
}

func TestCronJobResource_Create_DisabledJob(t *testing.T) {
	var capturedParams any

	r := &CronJobResource{
		client: &client.MockClient{
			CallFunc: func(ctx context.Context, method string, params any) (json.RawMessage, error) {
				if method == "cronjob.create" {
					capturedParams = params
					return json.RawMessage(`{"id": 22}`), nil
				}
				if method == "cronjob.query" {
					return json.RawMessage(`[{
						"id": 22,
						"user": "root",
						"command": "/usr/local/bin/disabled.sh",
						"description": "Disabled job",
						"enabled": false,
						"stdout": false,
						"stderr": false,
						"schedule": {"minute": "0", "hour": "0", "dom": "*", "month": "*", "dow": "*"}
					}]`), nil
				}
				return nil, nil
			},
		},
	}

	schemaResp := getCronJobResourceSchema(t)
	planValue := createCronJobModelValue(cronJobModelParams{
		User:        "root",
		Command:     "/usr/local/bin/disabled.sh",
		Description: "Disabled job",
		Enabled:     false,
		Stdout:      false,
		Stderr:      false,
		Schedule: &scheduleBlockParams{
			Minute: "0",
			Hour:   "0",
			Dom:    "*",
			Month:  "*",
			Dow:    "*",
		},
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

	// Verify params
	params, ok := capturedParams.(map[string]any)
	if !ok {
		t.Fatalf("expected params to be map[string]any, got %T", capturedParams)
	}

	if params["enabled"] != false {
		t.Errorf("expected enabled false, got %v", params["enabled"])
	}
	if params["stdout"] != false {
		t.Errorf("expected stdout false, got %v", params["stdout"])
	}
	if params["stderr"] != false {
		t.Errorf("expected stderr false, got %v", params["stderr"])
	}

	// Verify state
	var resultData CronJobResourceModel
	resp.State.Get(context.Background(), &resultData)
	if resultData.Enabled.ValueBool() != false {
		t.Errorf("expected enabled false, got %v", resultData.Enabled.ValueBool())
	}
}
```

**Step 2: Run all cron job tests**

Run: `go test ./internal/resources/... -run TestCronJob -v`
Expected: All tests pass

**Step 3: Commit**

```bash
git add internal/resources/cron_job_test.go
git commit -m "$(cat <<'EOF'
test(resources): add cron job schedule variation tests

Add tests for custom schedules, schedule-only updates, and disabled jobs.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 16: Create User Documentation

**Files:**
- Create: `docs/resources/cron_job.md`

**Step 1: Write the documentation file**

```markdown
# truenas_cron_job Resource

Manages cron jobs for scheduled task execution on TrueNAS.

## Example Usage

### Basic Cron Job

```hcl
resource "truenas_cron_job" "backup" {
  user        = "root"
  command     = "/usr/local/bin/backup.sh"
  description = "Daily backup script"

  schedule {
    minute = "0"
    hour   = "2"
  }
}
```

### Every 15 Minutes During Business Hours

```hcl
resource "truenas_cron_job" "health_check" {
  user        = "root"
  command     = "/usr/local/bin/check-health.sh"
  description = "Health check"

  schedule {
    minute = "*/15"
    hour   = "9-17"
    dow    = "1-5"
  }
}
```

### Disabled Job (for testing)

```hcl
resource "truenas_cron_job" "test_job" {
  user        = "root"
  command     = "/usr/local/bin/test.sh"
  description = "Test job - disabled"
  enabled     = false
  stdout      = false
  stderr      = false

  schedule {
    minute = "0"
    hour   = "0"
  }
}
```

## Argument Reference

* `user` - (Required) User to run the command as.
* `command` - (Required) Command to execute.
* `description` - (Optional) Job description. Default: empty string.
* `enabled` - (Optional) Enable the cron job. Default: true.
* `stdout` - (Optional) Redirect stdout to syslog. Default: true.
* `stderr` - (Optional) Redirect stderr to syslog. Default: true.
* `schedule` - (Required) Cron schedule block. See below.

### Schedule Block

* `minute` - (Required) Minute (0-59 or cron expression like `*/15`).
* `hour` - (Required) Hour (0-23 or cron expression like `9-17`).
* `dom` - (Optional) Day of month (1-31 or cron expression). Default: `*`.
* `month` - (Optional) Month (1-12 or cron expression). Default: `*`.
* `dow` - (Optional) Day of week (0-6 where 0=Sunday, or `1-5` for weekdays). Default: `*`.

## Attribute Reference

* `id` - Cron job ID.

## Common Schedules

| Description | minute | hour | dom | month | dow |
|-------------|--------|------|-----|-------|-----|
| Every minute | `*` | `*` | `*` | `*` | `*` |
| Every hour | `0` | `*` | `*` | `*` | `*` |
| Daily at midnight | `0` | `0` | `*` | `*` | `*` |
| Daily at 2am | `0` | `2` | `*` | `*` | `*` |
| Weekly (Sunday midnight) | `0` | `0` | `*` | `*` | `0` |
| Monthly (1st at midnight) | `0` | `0` | `1` | `*` | `*` |
| Every 15 minutes | `*/15` | `*` | `*` | `*` | `*` |
| Every 6 hours | `0` | `*/6` | `*` | `*` | `*` |
| Weekdays at 9am | `0` | `9` | `*` | `*` | `1-5` |

## Import

Cron jobs can be imported using the job ID:

```bash
terraform import truenas_cron_job.example 10
```
```

**Step 2: Commit**

```bash
git add docs/resources/cron_job.md
git commit -m "$(cat <<'EOF'
docs: add cron job resource documentation

Add user documentation with examples and common schedule patterns.

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Task 17: Run Full Test Suite and Build

**Files:**
- None (verification only)

**Step 1: Run all tests**

Run: `go test ./... -v`
Expected: All tests pass

**Step 2: Run build**

Run: `go build ./...`
Expected: Build succeeds

**Step 3: Run linter (if available)**

Run: `golangci-lint run` (or `mise run lint` if configured)
Expected: No issues

**Step 4: Final commit if any cleanup needed**

If any issues found, fix them and commit:

```bash
git add -A
git commit -m "$(cat <<'EOF'
fix: address linting issues in cron job resource

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

---

## Summary

This plan implements the `truenas_cron_job` resource with:

1. **API Struct** - `CronJobResponse` in `internal/api/cron.go`
2. **Resource Implementation** - Full CRUD in `internal/resources/cron_job.go`
3. **Tests** - 100% coverage in `internal/resources/cron_job_test.go`
4. **Documentation** - User docs in `docs/resources/cron_job.md`
5. **Provider Registration** - Single line addition to `provider.go`

The implementation follows existing patterns (CloudSync Task) and reuses the `ScheduleBlock` type for cron scheduling.
