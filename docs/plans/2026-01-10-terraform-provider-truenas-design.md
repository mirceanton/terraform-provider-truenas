# Terraform Provider TrueNAS - Design Document

**Date:** 2026-01-10
**Status:** Approved
**Scope:** v1.0 - Custom Docker Compose apps via SSH

## Overview

A Terraform provider for managing TrueNAS SCALE and Community Edition servers. The provider communicates with TrueNAS via SSH, executing `midclt` commands to interact with the TrueNAS middleware JSON-RPC API.

## Goals

- Deploy custom Docker Compose applications to TrueNAS
- Manage ZFS datasets and host paths for app storage
- Provide ergonomic HCL configuration (not raw JSON)
- Support fresh deployments initially, with import capability for datasets

## Non-Goals (v1)

- Catalog app support (deferred to v2)
- API/WebSocket authentication methods
- Pool creation/management (read-only)
- Network, certificate, or user management

---

## Architecture

### High-Level Structure

```
┌─────────────────────────────────────────────────────────┐
│                    Terraform Core                        │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                 Provider Layer                           │
│  internal/provider/provider.go                          │
│  - Schema definition (host, auth_method, ssh block)     │
│  - Resource/datasource registration                     │
│  - Client initialization                                │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│              Resources & Data Sources                    │
│  internal/resources/     internal/datasources/          │
│  - app.go                - pool.go                      │
│  - dataset.go            - dataset.go                   │
│  - host_path.go                                         │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                   Client Layer                           │
│  internal/client/                                        │
│  - ssh.go      (connection management)                  │
│  - midclt.go   (command execution, JSON parsing)        │
│  - jobs.go     (async job polling)                      │
│  - errors.go   (error types and mapping)                │
└─────────────────────────────────────────────────────────┘
                            │
                            ▼
┌─────────────────────────────────────────────────────────┐
│                   TrueNAS Server                         │
│  SSH → midclt → TrueNAS Middleware (JSON-RPC)           │
└─────────────────────────────────────────────────────────┘
```

### Design Principles

- **Idiomatic Terraform**: Separate resources with explicit dependencies, no magic side effects
- **Ergonomic HCL**: High-level configuration, not raw JSON passthrough
- **Smart operations**: In-place updates when possible, replace only when necessary
- **Helpful errors**: Parsed messages with actionable suggestions, raw error included

---

## Provider Configuration

```hcl
provider "truenas" {
  host        = "10.29.204.1"
  auth_method = "ssh"              # Required: "ssh" (future: "api", "websocket")

  ssh {
    port        = 522              # Optional, default 22
    user        = "admin"          # Optional, default "root"
    private_key = file("~/.ssh/truenas_ed25519")
  }
}
```

### Connection Behavior

- **Lazy initialization**: SSH connection established on first resource operation, not at provider configure time
- **Connection reuse**: Single SSH connection shared across all operations within a `terraform apply`
- **Graceful cleanup**: Connection closed when provider context ends

### Future Auth Methods

The `auth_method` discriminator allows adding new connection types without breaking changes:

```hcl
# Future example
provider "truenas" {
  host        = "10.29.204.1"
  auth_method = "api"

  api {
    key        = var.api_key
    tls_verify = true
  }
}
```

---

## Resources

### truenas_app

Manages custom Docker Compose applications.

```hcl
resource "truenas_app" "caddy" {
  name           = "caddy"
  compose_config = file("${path.module}/caddy/docker-compose.yml")

  volume {
    name      = "config"
    host_path = truenas_host_path.caddy_config.full_path
    read_only = false    # Optional, default false
  }

  volume {
    name      = "data"
    host_path = truenas_host_path.caddy_data.full_path
  }

  port {
    name        = "http"
    number      = 80
    bind_mode   = "published"    # Optional, default "published"
    host_ips    = []             # Optional, default [] (all interfaces)
  }

  port {
    name   = "https"
    number = 443
  }

  labels = {                     # Optional
    "traefik.enable" = "true"
  }

  timeouts {                     # Optional
    create = "10m"
    update = "10m"
    delete = "5m"
  }
}
```

**Attributes:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `name` | string | yes | Unique app name |
| `compose_config` | string | yes | Docker Compose YAML content |
| `volume` | block | no | Volume mount configuration |
| `port` | block | no | Port mapping configuration |
| `labels` | map | no | App labels |
| `timeouts` | block | no | Operation timeouts |

**Update Behavior:**

- `compose_config` change → in-place update via `app.update`
- `volume`/`port` change → in-place update
- `name` change → forces replacement (destroy + create)

### truenas_dataset

Manages ZFS datasets.

```hcl
resource "truenas_dataset" "caddy" {
  pool = data.truenas_pool.main.name
  path = "apps/caddy"

  # Optional ZFS properties
  compression  = "lz4"           # Optional
  quota        = "50G"           # Optional
  refquota     = "40G"           # Optional
  atime        = "off"           # Optional
}
```

**With parent reference (for Terraform-managed hierarchy):**

```hcl
resource "truenas_dataset" "apps" {
  pool = "storage"
  path = "apps"
}

resource "truenas_dataset" "caddy" {
  parent = truenas_dataset.apps.id
  name   = "caddy"
}
```

**Attributes:**

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `pool` | string | yes* | Pool name (required if no `parent`) |
| `path` | string | yes* | Full dataset path (required if no `parent`) |
| `parent` | string | no | Parent dataset ID |
| `name` | string | yes* | Dataset name relative to parent |
| `compression` | string | no | ZFS compression algorithm |
| `quota` | string | no | Dataset quota |

### truenas_host_path

Manages directories within datasets for app volume mounts.

```hcl
resource "truenas_host_path" "caddy_config" {
  dataset = truenas_dataset.caddy.id
  path    = "config"             # Relative to dataset mount point
  owner   = 568                  # Optional, default 568 (apps user)
  group   = 568                  # Optional, default 568
  mode    = "0755"               # Optional, default "0755"
}
```

**Computed Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `full_path` | string | Absolute path (e.g., `/mnt/storage/apps/caddy/config`) |

---

## Data Sources

### truenas_pool

Read-only pool information.

```hcl
data "truenas_pool" "main" {
  name = "storage"
}
```

**Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Pool ID |
| `name` | string | Pool name |
| `path` | string | Mount path (e.g., `/mnt/storage`) |
| `status` | string | Pool health status |
| `available_bytes` | number | Available space |
| `used_bytes` | number | Used space |

### truenas_dataset

Read existing dataset information.

```hcl
data "truenas_dataset" "apps" {
  pool = "storage"
  path = "apps"
}
```

**Attributes:**

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Dataset ID |
| `mount_path` | string | Mount path (e.g., `/mnt/storage/apps`) |
| `compression` | string | Current compression setting |
| `used_bytes` | number | Space used |
| `available_bytes` | number | Space available |

---

## Client Layer

### Interface

```go
// internal/client/client.go
type Client interface {
    // Execute midclt command, return parsed JSON
    Call(ctx context.Context, method string, params any) (json.RawMessage, error)

    // Execute and wait for job completion with polling
    CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error)

    // Close connection
    Close() error
}
```

### SSH Implementation

Uses `golang.org/x/crypto/ssh` with a custom wrapper:

- Key-based authentication
- Connection reuse across operations
- Automatic reconnection on connection loss
- Credential masking in logs

### Job Polling

TrueNAS operations are async - they return job IDs that must be polled:

```
midclt call app.create '{...}'
       │
       ▼ Returns job_id immediately
       │
Poll:  midclt call core.get_jobs '[["id","=",job_id]]'
       │
       ▼ Exponential backoff: 1s, 2s, 4s, 8s... max 30s
       │
       ▼ Until: state = "SUCCESS" | "FAILED" | timeout
```

**Polling Configuration:**

- Initial interval: 1 second
- Max interval: 30 seconds
- Backoff multiplier: 2x
- Default timeout: 5 minutes (configurable via `timeouts` block)

---

## Error Handling

### Error Structure

```go
type TrueNASError struct {
    Code       string    // e.g., "EINVAL", "ENOENT", "EFAULT"
    Message    string    // Raw error from middleware
    Field      string    // Which field caused error (if applicable)
    JobID      int64     // For job-related errors
    Suggestion string    // Actionable guidance
}
```

### Common Error Mappings

| TrueNAS Error | User-Friendly Message |
|---------------|----------------------|
| `[EINVAL] ... Field was not expected` | Invalid configuration: {field}. Check schema compatibility. |
| `[ENOENT] Unable to locate` | Resource not found. It may have been deleted outside Terraform. |
| `[EFAULT] Failed 'up' action` | Container failed to start. Check compose_config and image availability. |
| `[EEXIST] already exists` | App '{name}' already exists. Import it or choose a different name. |
| Job timeout | Operation timed out after {duration}. Increase timeout or check TrueNAS server. |
| SSH connection failed | Cannot connect to {host}:{port}. Verify SSH credentials and network. |

### Error Output Format

```
Error: Failed to create app "caddy"

Container failed to start. Check compose_config and image availability.

Suggestion: Verify the image exists and is accessible. Check TrueNAS app logs
for details: midclt call app.logs caddy

Raw error: [EFAULT] Failed 'up' action for app caddy: image not found
Job ID: 12345
```

---

## State Management

### Drift Detection

- Provider reads current state via `midclt call app.query` during refresh
- Compares with Terraform state, reports differences in plan
- Respects `lifecycle { ignore_changes }` for user-managed attributes

### Update Strategy

Smart detection of what changed:

| Change | Action |
|--------|--------|
| `compose_config` | In-place update via `app.update` |
| `volume`, `port`, `labels` | In-place update |
| `name` | Replace (destroy + create) |

### Deletion Behavior

Follows resource dependencies:
- `truenas_app` deletion only removes the app, not host paths
- `truenas_host_path` deletion removes the directory
- `truenas_dataset` deletion removes the dataset (fails if not empty)

Users control what gets deleted through explicit resource management.

---

## Project Structure

```
terraform-provider-truenas/
├── main.go
├── go.mod
├── go.sum
├── mise.toml                        # Tool versions
├── .goreleaser.yml                  # Release automation
│
├── .mise/
│   └── tasks/
│       ├── build
│       ├── test
│       ├── test-acc
│       ├── coverage
│       ├── install
│       ├── docs
│       └── lint
│
├── internal/
│   ├── provider/
│   │   ├── provider.go
│   │   └── provider_test.go
│   │
│   ├── client/
│   │   ├── client.go                # Interface
│   │   ├── ssh.go
│   │   ├── ssh_test.go
│   │   ├── midclt.go
│   │   ├── midclt_test.go
│   │   ├── jobs.go
│   │   ├── jobs_test.go
│   │   └── errors.go
│   │
│   ├── resources/
│   │   ├── app.go
│   │   ├── app_test.go
│   │   ├── dataset.go
│   │   ├── dataset_test.go
│   │   ├── host_path.go
│   │   └── host_path_test.go
│   │
│   └── datasources/
│       ├── pool.go
│       ├── pool_test.go
│       ├── dataset.go
│       └── dataset_test.go
│
├── examples/
│   ├── provider/
│   │   └── provider.tf
│   ├── resources/
│   │   ├── app/main.tf
│   │   ├── dataset/main.tf
│   │   └── host_path/main.tf
│   └── data-sources/
│       ├── pool/main.tf
│       └── dataset/main.tf
│
├── docs/
│   ├── index.md
│   ├── resources/
│   └── data-sources/
│
└── templates/
    └── index.md.tmpl
```

---

## Testing Strategy

### Test Coverage

100% coverage requirement with tiered testing:

**Unit Tests (every PR):**
- Mock SSH client for isolation
- Test HCL → JSON transformation
- Test error parsing and mapping
- Test job polling logic
- Fast execution (~30s)

**Acceptance Tests (TF_ACC=1):**
- Real TrueNAS server required
- Full CRUD lifecycle tests
- Drift detection verification
- Nightly CI or manual trigger

### Mock Client Pattern

```go
type MockClient struct {
    CallFunc func(ctx context.Context, method string, params any) (json.RawMessage, error)
}

func (m *MockClient) Call(ctx context.Context, method string, params any) (json.RawMessage, error) {
    return m.CallFunc(ctx, method, params)
}
```

---

## Logging

Uses `tflog` package with masked sensitive fields:

```go
tflog.SetField(ctx, "password", "")
tflog.SetField(ctx, "private_key", "")
tflog.MaskFieldValuesWithFieldKeys(ctx, "password", "private_key")

tflog.Debug(ctx, "Creating app", map[string]interface{}{
    "name": appName,
})

tflog.Trace(ctx, "midclt response", map[string]interface{}{
    "raw": response,  // Only at TRACE level
})
```

Respects `TF_LOG` environment variable levels.

---

## Implementation Technologies

| Component | Technology |
|-----------|------------|
| SDK | terraform-plugin-framework (latest) |
| SSH | golang.org/x/crypto/ssh |
| Testing | testing + terraform-plugin-testing |
| Logging | tflog |
| Task Runner | mise (file tasks) |
| Release | GoReleaser |

---

## Complete Example

```hcl
terraform {
  required_providers {
    truenas = {
      source  = "local/truenas/truenas"
      version = "~> 0.1"
    }
  }
}

provider "truenas" {
  host        = "10.29.204.1"
  auth_method = "ssh"

  ssh {
    port        = 522
    user        = "admin"
    private_key = file("~/.ssh/truenas_ed25519")
  }
}

# Reference existing pool
data "truenas_pool" "main" {
  name = "storage"
}

# Create dataset for app
resource "truenas_dataset" "caddy" {
  pool        = data.truenas_pool.main.name
  path        = "apps/caddy"
  compression = "lz4"
}

# Create directories for volumes
resource "truenas_host_path" "caddy_config" {
  dataset = truenas_dataset.caddy.id
  path    = "config"
  owner   = 568
}

resource "truenas_host_path" "caddy_data" {
  dataset = truenas_dataset.caddy.id
  path    = "data"
  owner   = 568
}

# Deploy app
resource "truenas_app" "caddy" {
  name           = "caddy"
  compose_config = file("${path.module}/docker-compose.yml")

  volume {
    name      = "config"
    host_path = truenas_host_path.caddy_config.full_path
    read_only = true
  }

  volume {
    name      = "data"
    host_path = truenas_host_path.caddy_data.full_path
  }

  port {
    name   = "http"
    number = 80
  }

  port {
    name   = "https"
    number = 443
  }
}
```

---

## Future Considerations (v2+)

- Catalog app support with schema discovery
- API/WebSocket authentication methods
- Import for existing apps
- Network configuration resources
- Certificate management
- User/group management
- Snapshot and replication resources
