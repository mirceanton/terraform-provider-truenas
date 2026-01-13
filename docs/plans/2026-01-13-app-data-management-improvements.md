# Custom App & Data Management Improvements

**Date:** 2026-01-13
**Status:** Draft

## Overview

Improve workflows for managing custom Docker Compose apps and their persistent data on TrueNAS by adding snapshot management, app lifecycle control, and enhanced dataset attributes.

## Motivation

The current provider enables deploying custom apps with storage, but lacks critical data management features:

1. **No snapshot support** - Cannot backup app data before updates or rollback on failure
2. **No app lifecycle control** - Cannot stop/start apps without destroying them
3. **No automated backups** - No way to schedule periodic snapshots
4. **Limited dataset attributes** - Missing ZFS tuning options (recordsize, sync, dedup)
5. **No runtime visibility** - Cannot see actual ports/volumes from active workloads

## Current Workflow Limitations

```hcl
# 1. Create storage foundation
resource "truenas_dataset" "apps" { pool = "tank"; path = "apps" }

# 2. Create app directory
resource "truenas_host_path" "myapp_config" {
  path = "/mnt/tank/apps/myapp/config"
}

# 3. Deploy config files
resource "truenas_file" "myapp_conf" {
  host_path     = truenas_host_path.myapp_config.id
  relative_path = "app.conf"
  content       = "..."
}

# 4. Deploy app with hardcoded path in YAML
resource "truenas_app" "myapp" {
  name           = "myapp"
  custom_app     = true
  compose_config = <<-EOF
    version: '3'
    services:
      web:
        volumes:
          - /mnt/tank/apps/myapp/config:/config  # Must match host_path manually
  EOF
}
```

**Pain Points:**
- No way to snapshot app data before updates
- Cannot stop/start apps for maintenance without full destroy/recreate
- Volume paths must be manually synchronized in compose YAML
- No visibility into runtime state (ports, actual mounts)
- No automated backup scheduling

---

## Proposed Improvements

### 1. Snapshot Resource (`truenas_snapshot`) - HIGH PRIORITY

**Why:** Enables backup of app data before updates, rollback on failure.

**API Methods:** `pool.snapshot.{query,create,delete,clone,rollback,hold,release}`

#### Schema

```hcl
resource "truenas_snapshot" "pre_update" {
  dataset = "tank/apps/myapp"
  name    = "pre-update-2024-01-13"

  # Optional: prevent auto-cleanup
  hold = true
}

# Clone to new dataset for testing
resource "truenas_snapshot" "clone_test" {
  dataset      = "tank/apps/myapp"
  name         = "test-snapshot"
  clone_target = "tank/apps/myapp-test"  # Creates new dataset from snapshot
}
```

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `dataset` | string | Yes | Dataset path (e.g., "tank/apps") |
| `name` | string | Yes | Snapshot name |
| `hold` | bool | No | Prevent auto-cleanup (default: false) |
| `clone_target` | string | No | If set, clone snapshot to this dataset |
| `recursive` | bool | No | Include child datasets (default: false) |

#### Computed Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Full snapshot identifier (dataset@name) |
| `createtxg` | string | Transaction group when created |
| `used_bytes` | number | Space consumed by snapshot |
| `referenced_bytes` | number | Space referenced by snapshot |

#### Operations

| Operation | API Method | Behavior |
|-----------|------------|----------|
| Create | `pool.snapshot.create` | Create snapshot, optionally hold |
| Read | `pool.snapshot.query` | Get snapshot details |
| Update | `pool.snapshot.hold/release` | Toggle hold status |
| Delete | `pool.snapshot.delete` | Remove snapshot |

#### Data Source

```hcl
data "truenas_snapshots" "app_backups" {
  dataset   = "tank/apps"
  recursive = true

  # Optional filters
  name_pattern = "pre-update-*"
}
```

---

### 2. App Lifecycle Operations - HIGH PRIORITY

**Why:** Cannot currently stop/start apps for maintenance without destroying them.

**API Methods:** `app.{start,stop,redeploy}`

#### Design Decision: `desired_state` attribute vs. separate resource

**Option A: Add `desired_state` attribute to existing resource**

```hcl
resource "truenas_app" "myapp" {
  name           = "myapp"
  custom_app     = true
  compose_config = "..."

  # NEW: Control desired state (default: "RUNNING")
  desired_state = "STOPPED"  # RUNNING, STOPPED
}
```

**Rationale:** Simpler for users, keeps app config and state together.

**Option B: Separate lifecycle resource**

```hcl
resource "truenas_app_lifecycle" "myapp" {
  app_name = truenas_app.myapp.name
  action   = "stop"  # start, stop, restart, redeploy
}
```

**Rationale:** Better separation of concerns, lifecycle changes don't affect app config.

**Decision:** Option A (`desired_state` attribute) - more idiomatic Terraform pattern (see `aws_instance.instance_state`).

#### Implementation

| Current State | Desired State | Action |
|---------------|---------------|--------|
| RUNNING | RUNNING | None |
| RUNNING | STOPPED | Call `app.stop` |
| STOPPED | STOPPED | None |
| STOPPED | RUNNING | Call `app.start` |
| * | * (config change) | Call `app.update` + wait |

---

### 3. Snapshot Task Resource (`truenas_snapshot_task`) - MEDIUM PRIORITY

**Why:** Automated backup scheduling for app data protection.

**API Methods:** `pool.snapshottask.{query,create,update,delete}`

#### Schema

```hcl
resource "truenas_snapshot_task" "daily_backup" {
  dataset   = "tank/apps"
  recursive = true
  enabled   = true

  schedule {
    minute = "0"
    hour   = "2"
    dom    = "*"
    month  = "*"
    dow    = "*"
  }

  lifetime_value = 7
  lifetime_unit  = "DAY"  # HOUR, DAY, WEEK, MONTH, YEAR

  naming_schema = "auto-%Y-%m-%d_%H-%M"
}
```

#### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `dataset` | string | Yes | Target dataset path |
| `recursive` | bool | No | Include child datasets (default: true) |
| `enabled` | bool | No | Task enabled (default: true) |
| `schedule` | block | Yes | Cron-style schedule |
| `lifetime_value` | number | Yes | Retention period value |
| `lifetime_unit` | string | Yes | HOUR, DAY, WEEK, MONTH, YEAR |
| `naming_schema` | string | Yes | Snapshot name template (strftime) |
| `allow_empty` | bool | No | Create snapshot even if no changes (default: false) |

---

### 4. App active_workloads Sync - MEDIUM PRIORITY

**Why:** Visibility into runtime state (actual ports, volumes, network).

**Current TODO:** `internal/resources/app.go:248`

#### Investigation Required

Query an existing app with `retrieve_config: true` and inspect the `active_workloads` structure to determine:
1. What fields are available
2. Which fields are stable across app restarts
3. How to map container-level info to Terraform attributes

#### Proposed Schema Additions

```hcl
resource "truenas_app" "myapp" {
  # ... existing attributes ...

  # NEW: Computed from active_workloads (read-only)
  active_ports = [
    {
      container_port = 80
      host_port      = 8080
      protocol       = "tcp"
    }
  ]

  active_volumes = [
    {
      source      = "/mnt/tank/apps/myapp/config"
      destination = "/config"
      read_only   = false
    }
  ]
}
```

---

### 5. Dataset Attribute Enhancements - LOW EFFORT

**Why:** Quick wins for storage tuning.

#### Missing Attributes

| Attribute | Type | API Field | Description |
|-----------|------|-----------|-------------|
| `recordsize` | string | `recordsize.value` | Block size (128K default) |
| `sync` | string | `sync.value` | Sync behavior (standard, always, disabled) |
| `dedup` | string | `dedup.value` | Deduplication (off, on, verify) |
| `readonly` | bool | `readonly.value` | Read-only mode |
| `copies` | int | `copies.value` | Data copies (1-3) |
| `special_small_blocks` | int | `special_small_blocks.value` | Threshold for special vdev |

#### Implementation Notes

These follow the same pattern as existing `compression`, `quota`, `atime` attributes:
- Optional in schema
- Only included in API calls when explicitly set
- Sync from API on Read only if previously set (prevents drift from defaults)

---

## Improved Workflow (After Implementation)

```hcl
# 1. Storage foundation with tuning
resource "truenas_dataset" "apps" {
  pool        = "tank"
  path        = "apps"
  compression = "zstd"
  recordsize  = "128K"  # NEW
}

# 2. Automated backups
resource "truenas_snapshot_task" "apps_daily" {
  dataset   = truenas_dataset.apps.id
  recursive = true
  schedule { hour = "2"; minute = "0"; dom = "*"; month = "*"; dow = "*" }
  lifetime_value = 7
  lifetime_unit  = "DAY"
}

# 3. App directory
resource "truenas_host_path" "myapp_config" {
  path = "${truenas_dataset.apps.mount_path}/myapp/config"
}

# 4. Pre-update snapshot
resource "truenas_snapshot" "myapp_backup" {
  dataset = truenas_dataset.apps.id
  name    = "pre-update-${formatdate("YYYY-MM-DD", timestamp())}"
  hold    = true
}

# 5. Deploy app with lifecycle control
resource "truenas_app" "myapp" {
  name           = "myapp"
  custom_app     = true
  compose_config = file("compose.yaml")
  desired_state  = "RUNNING"  # NEW: Can set to STOPPED for maintenance

  depends_on = [truenas_snapshot.myapp_backup]
}
```

---

## Implementation Order

| Phase | Item | Effort | Value | Files |
|-------|------|--------|-------|-------|
| 1 | Snapshot resource | Medium | High | `internal/resources/snapshot.go`, `internal/datasources/snapshot.go` |
| 2 | App lifecycle | Low | High | `internal/resources/app.go` (modify) |
| 3 | Snapshot task resource | Medium | Medium | `internal/resources/snapshot_task.go` |
| 4 | Dataset attributes | Low | Medium | `internal/resources/dataset.go` (modify) |
| 5 | active_workloads sync | Medium | Medium | `internal/resources/app.go` (modify) |

---

## File Structure

```
internal/
├── resources/
│   ├── snapshot.go           # NEW: truenas_snapshot resource
│   ├── snapshot_task.go      # NEW: truenas_snapshot_task resource
│   ├── app.go                # MODIFY: add desired_state, active_workloads
│   └── dataset.go            # MODIFY: add missing attributes
├── datasources/
│   └── snapshot.go           # NEW: truenas_snapshots data source
```

---

## Testing Strategy

### Snapshot Resource Tests

**Unit Tests:**
- Create snapshot → verify API params
- Create with hold → verify hold API called
- Create with clone_target → verify clone API called
- Read existing snapshot → state populated
- Read missing snapshot → removed from state
- Update hold status → verify hold/release API
- Delete snapshot → verify delete API

**Integration Tests:**
- Full lifecycle on real TrueNAS
- Snapshot of dataset with app data
- Clone snapshot to new dataset
- Rollback (if exposing rollback operation)

### App Lifecycle Tests

**Unit Tests:**
- Create with desired_state=RUNNING → default behavior
- Create with desired_state=STOPPED → app.create then app.stop
- Update desired_state RUNNING→STOPPED → app.stop called
- Update desired_state STOPPED→RUNNING → app.start called
- Update config while STOPPED → config update, no state change

### Snapshot Task Tests

**Unit Tests:**
- Create task → verify schedule format in API params
- Update schedule → verify update API
- Delete task → verify delete API
- Enabled toggle → verify API call

---

## Verification

After implementation, test the complete workflow:

1. Create dataset with new attributes (recordsize, sync)
2. Create snapshot task for automated backups
3. Deploy app with `desired_state = "RUNNING"`
4. Create manual snapshot before config change
5. Update app compose config
6. Verify app restarts with new config
7. Test `desired_state = "STOPPED"` for maintenance
8. Test snapshot rollback if update fails
9. Verify automated snapshots are created by task

---

## Future Considerations

- **Rollback operation:** Could add `rollback_to` attribute that triggers `pool.snapshot.rollback` - needs careful design as it's destructive
- **Clone management:** Cloned datasets could be separate resources or attributes
- **Snapshot policies:** More complex retention rules beyond lifetime
- **App upgrade/rollback:** For catalog apps (not custom), support version management

---

## References

- [ZFS Snapshot Best Practices](https://docs.oracle.com/cd/E19253-01/819-5461/ghzsk/index.html)
- [TrueNAS Snapshot Tasks](https://www.truenas.com/docs/scale/scaletutorials/dataprotection/addingsnapshottasks/)
- [Terraform AWS Instance State](https://registry.terraform.io/providers/hashicorp/aws/latest/docs/resources/instance#instance_state)
