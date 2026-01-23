# Rename cron_job stdout/stderr to capture_stdout/capture_stderr Implementation Plan

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:executing-plans to implement this plan task-by-task.

**Goal:** Rename misleading `stdout`/`stderr` attributes to `capture_stdout`/`capture_stderr` with inverted semantics for better UX.

**Architecture:** The TrueNAS API uses `stdout: true` to mean "hide output" and `stdout: false` to mean "mail output to user". We invert this in Terraform so `capture_stdout: true` means "capture and mail output" (sends `stdout: false` to API). This is a breaking change but makes the attribute names self-documenting.

**Tech Stack:** Go, Terraform Plugin Framework, TrueNAS midclt API

**Methodology:** TDD (Test-Driven Development) - Write failing tests first, then implement to make them pass.

**Coverage Baseline:**
- `internal/resources`: 87.9%
- `internal/api`: 86.0%

---

## Task 1: Update Test Helper Structs (RED setup)

**Files:**
- Modify: `internal/resources/cron_job_test.go:157-166` (cronJobModelParams struct)
- Modify: `internal/resources/cron_job_test.go:168-219` (createCronJobModelValue function)

**Step 1: Update cronJobModelParams struct**

Change lines 163-164 from:
```go
Stdout      bool
Stderr      bool
```

To:
```go
CaptureStdout bool
CaptureStderr bool
```

**Step 2: Update createCronJobModelValue function**

Change lines 186-188 from:
```go
"enabled":     tftypes.NewValue(tftypes.Bool, p.Enabled),
"stdout":      tftypes.NewValue(tftypes.Bool, p.Stdout),
"stderr":      tftypes.NewValue(tftypes.Bool, p.Stderr),
```

To:
```go
"enabled":        tftypes.NewValue(tftypes.Bool, p.Enabled),
"capture_stdout": tftypes.NewValue(tftypes.Bool, p.CaptureStdout),
"capture_stderr": tftypes.NewValue(tftypes.Bool, p.CaptureStderr),
```

**Step 3: Update objectType in createCronJobModelValue**

Change lines 206-214 from:
```go
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
```

To:
```go
objectType := tftypes.Object{
    AttributeTypes: map[string]tftypes.Type{
        "id":             tftypes.String,
        "user":           tftypes.String,
        "command":        tftypes.String,
        "description":    tftypes.String,
        "enabled":        tftypes.Bool,
        "capture_stdout": tftypes.Bool,
        "capture_stderr": tftypes.Bool,
        "schedule":       scheduleType,
    },
}
```

**Step 4: Verify test file compiles (with errors in test functions)**

Run: `go build ./internal/resources/...`
Expected: Build errors in test functions using old field names (Stdout/Stderr) - this is expected

---

## Task 2: Update Schema Test (RED)

**Files:**
- Modify: `internal/resources/cron_job_test.go:98-140` (TestCronJobResource_Schema)

**Step 1: Update schema attribute checks**

Change lines 128-133 from:
```go
if attrs["stdout"] == nil {
    t.Error("expected 'stdout' attribute")
}
if attrs["stderr"] == nil {
    t.Error("expected 'stderr' attribute")
}
```

To:
```go
if attrs["capture_stdout"] == nil {
    t.Error("expected 'capture_stdout' attribute")
}
if attrs["capture_stderr"] == nil {
    t.Error("expected 'capture_stderr' attribute")
}
```

**Step 2: Run schema test to verify it fails (RED)**

Run: `go test ./internal/resources -run TestCronJobResource_Schema -v`
Expected: FAIL - schema still has old attribute names

---

## Task 3: Update All Test Functions with New Field Names and Inverted Logic (RED)

**Files:**
- Modify: `internal/resources/cron_job_test.go` (multiple test functions)

**Step 1: Update TestCronJobResource_Create_Success (lines 221-360)**

The key insight: When Terraform has `capture_stdout: true`, the API should receive `stdout: false`.

Update planValue (lines 255-258) from:
```go
Stdout:      true,
Stderr:      false,
```

To (we want to capture stderr but not stdout):
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update params verification (lines 308-313) - verify API receives inverted values:
```go
if params["stdout"] != true {
    t.Errorf("expected stdout true (inverted from capture_stdout false), got %v", params["stdout"])
}
if params["stderr"] != false {
    t.Errorf("expected stderr false (inverted from capture_stderr true), got %v", params["stderr"])
}
```

Update state verification (lines 354-359):
```go
if resultData.CaptureStdout.ValueBool() != false {
    t.Errorf("expected capture_stdout false, got %v", resultData.CaptureStdout.ValueBool())
}
if resultData.CaptureStderr.ValueBool() != true {
    t.Errorf("expected capture_stderr true, got %v", resultData.CaptureStderr.ValueBool())
}
```

**Step 2: Update TestCronJobResource_Create_APIError (lines 362-411)**

Update planValue (lines 377-378) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

**Step 3: Update TestCronJobResource_Read_Success (lines 413-510)**

Mock returns `stdout: true, stderr: false` which means capture_stdout: false, capture_stderr: true.

Update stateValue (lines 438-440) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update state verification (lines 486-491):
```go
if resultData.CaptureStdout.ValueBool() != false {
    t.Errorf("expected capture_stdout false (inverted from API stdout true), got %v", resultData.CaptureStdout.ValueBool())
}
if resultData.CaptureStderr.ValueBool() != true {
    t.Errorf("expected capture_stderr true (inverted from API stderr false), got %v", resultData.CaptureStderr.ValueBool())
}
```

**Step 4: Update TestCronJobResource_Read_NotFound (lines 512-562)**

Update stateValue (lines 528-530) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

**Step 5: Update TestCronJobResource_Read_APIError (lines 564-609)**

Update stateValue (lines 580-582) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

**Step 6: Update TestCronJobResource_Update_Success (lines 611-789)**

Mock response has `stdout: false, stderr: true` → capture_stdout: true, capture_stderr: false.

Update stateValue (lines 652-654) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update planValue (lines 670-672) from:
```go
Stdout:      false,
Stderr:      true,
```

To:
```go
CaptureStdout: true,
CaptureStderr: false,
```

Update params verification (lines 725-730) - verify API receives inverted values:
```go
if capturedUpdateData["stdout"] != false {
    t.Errorf("expected stdout false (inverted from capture_stdout true), got %v", capturedUpdateData["stdout"])
}
if capturedUpdateData["stderr"] != true {
    t.Errorf("expected stderr true (inverted from capture_stderr false), got %v", capturedUpdateData["stderr"])
}
```

Update state verification (lines 769-773):
```go
if resultData.CaptureStdout.ValueBool() != true {
    t.Errorf("expected capture_stdout true, got %v", resultData.CaptureStdout.ValueBool())
}
if resultData.CaptureStderr.ValueBool() != false {
    t.Errorf("expected capture_stderr false, got %v", resultData.CaptureStderr.ValueBool())
}
```

**Step 7: Update TestCronJobResource_Update_APIError (lines 791-860)**

Update stateValue (lines 810-812) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update planValue (lines 828-830) from:
```go
Stdout:      false,
Stderr:      true,
```

To:
```go
CaptureStdout: true,
CaptureStderr: false,
```

**Step 8: Update TestCronJobResource_Delete_Success (lines 862-916)**

Update stateValue (lines 883-885) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

**Step 9: Update TestCronJobResource_Delete_APIError (lines 918-959)**

Update stateValue (lines 933-935) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

**Step 10: Update TestCronJobResource_Create_CustomSchedule (lines 961-1072)**

Update planValue (lines 994-995) from:
```go
Stdout:      true,
Stderr:      true,
```

To:
```go
CaptureStdout: false,
CaptureStderr: false,
```

Note: Mock returns `stdout: true, stderr: true` which is `capture_stdout: false, capture_stderr: false`.

**Step 11: Update TestCronJobResource_Update_ScheduleOnly (lines 1074-1222)**

Update stateValue (lines 1110-1112) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update planValue (lines 1128-1130) from:
```go
Stdout:      true,
Stderr:      false,
```

To:
```go
CaptureStdout: false,
CaptureStderr: true,
```

Update params verification (lines 1176-1181):
```go
if capturedUpdateData["stdout"] != true {
    t.Errorf("expected stdout true (inverted from capture_stdout false), got %v", capturedUpdateData["stdout"])
}
if capturedUpdateData["stderr"] != false {
    t.Errorf("expected stderr false (inverted from capture_stderr true), got %v", capturedUpdateData["stderr"])
}
```

**Step 12: Update TestCronJobResource_Create_DisabledJob (lines 1224-1316)**

Mock returns `stdout: false, stderr: false` → capture_stdout: true, capture_stderr: true.

Update planValue (lines 1256-1258) from:
```go
Stdout:      false,
Stderr:      false,
```

To:
```go
CaptureStdout: true,
CaptureStderr: true,
```

Update params verification (lines 1296-1301):
```go
if params["stdout"] != false {
    t.Errorf("expected stdout false (inverted from capture_stdout true), got %v", params["stdout"])
}
if params["stderr"] != false {
    t.Errorf("expected stderr false (inverted from capture_stderr true), got %v", params["stderr"])
}
```

**Step 13: Run all tests to verify they fail (RED)**

Run: `go test ./internal/resources -run TestCronJobResource -v`
Expected: FAIL - implementation still uses old names and no inversion logic

---

## Task 4: Update Resource Model and Schema (GREEN)

**Files:**
- Modify: `internal/resources/cron_job.go:28-36` (CronJobResourceModel struct)
- Modify: `internal/resources/cron_job.go:84-95` (Schema attributes)

**Step 1: Update CronJobResourceModel struct**

Change lines 34-35 from:
```go
Stdout      types.Bool     `tfsdk:"stdout"`
Stderr      types.Bool     `tfsdk:"stderr"`
```

To:
```go
CaptureStdout types.Bool   `tfsdk:"capture_stdout"`
CaptureStderr types.Bool   `tfsdk:"capture_stderr"`
```

**Step 2: Update Schema attributes**

Change lines 84-95 from:
```go
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
```

To:
```go
"capture_stdout": schema.BoolAttribute{
    Description: "Capture standard output and mail to user account.",
    Optional:    true,
    Computed:    true,
    Default:     booldefault.StaticBool(false),
},
"capture_stderr": schema.BoolAttribute{
    Description: "Capture error output and mail to user account.",
    Optional:    true,
    Computed:    true,
    Default:     booldefault.StaticBool(false),
},
```

**Step 3: Build to verify syntax**

Run: `go build ./...`
Expected: Build errors in buildCronJobParams and mapCronJobToModel (expected, we fix next)

---

## Task 5: Update buildCronJobParams with Inverted Logic (GREEN)

**Files:**
- Modify: `internal/resources/cron_job.go:353-374` (buildCronJobParams function)

**Step 1: Update buildCronJobParams to invert boolean values**

Change lines 359-360 from:
```go
"stdout":      data.Stdout.ValueBool(),
"stderr":      data.Stderr.ValueBool(),
```

To:
```go
"stdout":      !data.CaptureStdout.ValueBool(),
"stderr":      !data.CaptureStderr.ValueBool(),
```

Note: `capture_stdout: true` → `stdout: false` (mail output), `capture_stdout: false` → `stdout: true` (hide output)

**Step 2: Build to verify syntax**

Run: `go build ./...`
Expected: Build error in mapCronJobToModel (expected, we fix next)

---

## Task 6: Update mapCronJobToModel with Inverted Logic (GREEN)

**Files:**
- Modify: `internal/resources/cron_job.go:377-393` (mapCronJobToModel function)

**Step 1: Update mapCronJobToModel to invert boolean values from API**

Change lines 383-384 from:
```go
data.Stdout = types.BoolValue(job.Stdout)
data.Stderr = types.BoolValue(job.Stderr)
```

To:
```go
data.CaptureStdout = types.BoolValue(!job.Stdout)
data.CaptureStderr = types.BoolValue(!job.Stderr)
```

Note: API `stdout: false` → `capture_stdout: true`, API `stdout: true` → `capture_stdout: false`

**Step 2: Build to verify compilation**

Run: `go build ./...`
Expected: SUCCESS

**Step 3: Run all tests to verify they pass (GREEN)**

Run: `go test ./internal/resources -run TestCronJobResource -v`
Expected: PASS - all tests should now pass

---

## Task 7: Run Full Test Suite and Verify Coverage

**Files:**
- None (verification only)

**Step 1: Run all tests**

Run: `mise run test`
Expected: All tests pass

**Step 2: Build provider**

Run: `go build ./...`
Expected: SUCCESS

**Step 3: Verify coverage has not regressed**

Run: `mise run coverage`
Expected:
- `internal/resources` ≥ 87.9%
- `internal/api` ≥ 86.0%

If coverage has decreased, add tests to cover any new code paths before proceeding.

---

## Task 8: Update Documentation

**Files:**
- Modify: `docs/resources/cron_job.md`

**Step 1: Update Argument Reference**

Change:
```markdown
* `stdout` - (Optional) Redirect stdout to syslog. Default: true.
* `stderr` - (Optional) Redirect stderr to syslog. Default: true.
```

To:
```markdown
* `capture_stdout` - (Optional) Capture standard output and mail to user account. Default: false.
* `capture_stderr` - (Optional) Capture error output and mail to user account. Default: false.
```

**Step 2: Update Complex Schedule example**

Change:
```hcl
stdout      = false
stderr      = false
```

To:
```hcl
capture_stdout = true
capture_stderr = true
```

Note: Old `stdout=false` meant "mail output", which is now `capture_stdout=true`.

**Step 3: Update Hourly Health Check example**

Remove the stdout/stderr lines entirely since defaults are now false (hide output):

```hcl
resource "truenas_cron_job" "health_check" {
  user        = "root"
  command     = "/mnt/tank/scripts/health-check.sh | logger -t health-check"
  description = "Hourly health check"

  schedule {
    minute = "0"
    hour   = "*"
  }
}
```

**Step 4: Verify documentation renders correctly**

Run: `cat docs/resources/cron_job.md`
Expected: Documentation is clear and consistent

---

## Task 9: Commit Changes

**Step 1: Stage all changes**

Run: `git add internal/resources/cron_job.go internal/resources/cron_job_test.go docs/resources/cron_job.md`

**Step 2: Commit**

Run:
```bash
git commit -m "$(cat <<'EOF'
feat(cron_job)!: rename stdout/stderr to capture_stdout/capture_stderr

BREAKING CHANGE: Renamed attributes with inverted semantics:
- stdout → capture_stdout (true = mail output, false = hide)
- stderr → capture_stderr (true = mail output, false = hide)
- Default changed from true to false

The TrueNAS API uses stdout=true to mean "hide output" which was
confusing. New attribute names are self-documenting: capture_stdout=true
means "capture and mail standard output to user".

Migration: Change stdout=true to capture_stdout=false (or remove, as
false is now default). Change stdout=false to capture_stdout=true.

Closes #12

Co-Authored-By: Claude Opus 4.5 <noreply@anthropic.com>
EOF
)"
```

Expected: Commit created successfully

**Step 3: Verify commit**

Run: `git log -1 --oneline`
Expected: Shows commit with breaking change message

---

## Task 10: Create Pull Request

**Step 1: Push branch to remote**

Run: `git push -u origin feature/rename-cron-stdout-stderr`
Expected: Branch pushed successfully

**Step 2: Create PR using tea**

Run:
```bash
tea pr create --title "feat(cron_job)!: rename stdout/stderr to capture_stdout/capture_stderr" --description "$(cat <<'EOF'
## Summary

- Renamed `stdout` → `capture_stdout` and `stderr` → `capture_stderr`
- Inverted semantics: `capture_stdout=true` means "mail output to user" (sends `stdout=false` to API)
- Changed default from `true` to `false` (hide output by default)

## Breaking Change

**Migration guide:**
- `stdout = true` → remove (false is now default) or `capture_stdout = false`
- `stdout = false` → `capture_stdout = true`
- Same for stderr/capture_stderr

## Why

The TrueNAS API uses `stdout=true` to mean "hide output" which is counterintuitive. The new attribute names are self-documenting: `capture_stdout=true` clearly means "capture and mail standard output to user".

Closes #12
EOF
)"
```

Expected: PR created, URL displayed

**Step 3: Verify PR**

Run: `tea pr list`
Expected: Shows the new PR linked to issue #12
