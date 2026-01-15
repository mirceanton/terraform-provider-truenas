# Design: `desired_state` Attribute for `truenas_app` Resource

## Overview

Add a `desired_state` attribute to the `truenas_app` resource enabling declarative control over app lifecycle (start/stop). The existing `state` attribute remains computed and read-only, reflecting actual state from the TrueNAS API.

## Schema Changes

```hcl
resource "truenas_app" "example" {
  name           = "myapp"
  custom_app     = true
  compose_config = file("docker-compose.yml")

  desired_state = "running"   # Optional, default "RUNNING"
  state_timeout = 300         # Optional, default 120 seconds
}

output "actual_state" {
  value = truenas_app.example.state  # Read-only, reflects reality
}
```

### New Attributes

| Attribute | Type | Required | Default | Description |
|-----------|------|----------|---------|-------------|
| `desired_state` | string | No | `"RUNNING"` | Target state: `running`/`RUNNING` or `stopped`/`STOPPED` (case-insensitive input, normalized to uppercase) |
| `state_timeout` | int | No | `120` | Seconds to wait for state transitions (range: 30-600) |

### Existing Attributes

| Attribute | Change |
|-----------|--------|
| `state` | No change - remains computed, read-only, uppercase |

## TrueNAS API

State control uses job-based methods:

- `app.start` - `[app_name]` - Starts a stopped app
- `app.stop` - `[app_name]` - Stops a running app

Both return job IDs; use `CallAndWait` to block until completion.

### App States

| State | Type | Description |
|-------|------|-------------|
| `RUNNING` | Stable | App is running normally |
| `STOPPED` | Stable | App is stopped |
| `DEPLOYING` | Transitional | App is being deployed/updated |
| `STARTING` | Transitional | App is starting |
| `STOPPING` | Transitional | App is stopping |
| `CRASHED` | Error | App crashed |

## Lifecycle Behavior

### Create

1. Call `app.create` as today
2. After creation completes, check if `desired_state` differs from actual state
3. If `desired_state = "STOPPED"` and app started as RUNNING, call `app.stop`
4. Default: apps start RUNNING, so no extra call needed when `desired_state = "RUNNING"`

### Read

1. Query app with `app.query` as today
2. Populate `state` (computed) from API response
3. If `state != desired_state`, Terraform shows a diff and plans an update

### Update

1. If `desired_state` changed or drifted:
   - `desired_state = "RUNNING"` and `state != "RUNNING"` → call `app.start`
   - `desired_state = "STOPPED"` and `state = "RUNNING"` → call `app.stop`
2. Wait for job completion (up to `state_timeout`)
3. Log warning if reconciling drift from external changes
4. Other attribute changes (`compose_config`) handled separately via `app.update`

### Delete

No change - `app.delete` handles any state.

## Transitional & Error State Handling

### Transitional States (DEPLOYING, STARTING, STOPPING)

- **Read**: Report transitional state in `state` attribute as-is
- **Update**: Wait (poll every 5s) until state becomes stable before attempting start/stop
- **Timeout**: If state remains transitional after `state_timeout`, return error

### CRASHED State

- If `desired_state = "RUNNING"` and `state = "CRASHED"`: Attempt `app.start`
- If start fails: Return error with details
- If `desired_state = "STOPPED"` and `state = "CRASHED"`: No action needed (already not running)

## Warnings & Diagnostics

### Drift Warning (during Update)

```
Warning: App state was externally changed

The app "myapp" was found in state STOPPED but desired_state is RUNNING.
Reconciling to desired state. To stop this app intentionally, set desired_state = "stopped".
```

### Validation Error

```
Error: Invalid desired_state value

desired_state must be "running" or "stopped", got "paused"
```

### Timeout Error

```
Error: Timeout waiting for app state

App "myapp" is stuck in DEPLOYING state after 300s.
This may indicate a deployment issue. Check TrueNAS UI for details.
```

### Start Failure on CRASHED App

```
Error: Failed to start crashed app

App "myapp" is in CRASHED state and failed to start: <API error details>
Check container logs in TrueNAS UI for the root cause.
```

## Implementation Details

### New Client Methods

```go
func (c *Client) StartApp(ctx context.Context, name string) error
func (c *Client) StopApp(ctx context.Context, name string) error
```

Both use `CallAndWait` since `app.start` and `app.stop` return jobs.

### State Normalization

```go
func normalizeDesiredState(s string) string {
    return strings.ToUpper(strings.TrimSpace(s))
}
```

### Plan Modifier

Use `planmodifier.String` that compares normalized values to avoid spurious diffs between `"running"` and `"RUNNING"`.

### Polling Helper

```go
func (r *AppResource) waitForStableState(ctx context.Context, name string, timeout time.Duration) (string, error) {
    // Poll every 5s until state is RUNNING, STOPPED, or CRASHED
    // Returns final state or error on timeout
}
```

## Summary

| Aspect | Decision |
|--------|----------|
| Attribute name | `desired_state` (optional, default `"RUNNING"`) |
| Valid values | `running`/`RUNNING`, `stopped`/`STOPPED` (case-insensitive) |
| Existing `state` | Kept as computed read-only |
| Timeout | `state_timeout` attribute, default 120s, range 30-600s |
| Poll rate | Hardcoded 5s |
| Drift handling | Reconcile with warning |
| Transitional states | Wait for stable state before acting |
| CRASHED state | Attempt start if desired=RUNNING |
