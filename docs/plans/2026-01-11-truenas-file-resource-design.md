# truenas_file Resource Design

**Date:** 2026-01-11
**Status:** Draft

## Overview

Add a `truenas_file` resource to the TrueNAS Terraform provider for deploying configuration files (Caddyfile, nginx.conf, app configs) to TrueNAS.

## Motivation

Currently, the provider can create datasets, host paths, and apps, but cannot deploy configuration files. Users must either:
- Use Ansible alongside Terraform (hybrid approach)
- Bake configs into custom Docker images (complex, requires registry)
- Use provisioners (non-idiomatic, breaks drift detection)

A native `truenas_file` resource follows the Kubernetes `ConfigMap` pattern: treat config as data, let the provider handle delivery.

## Design Decisions

### Content Source
- **Decision:** `content` attribute only (string)
- **Rationale:** Terraform's `templatefile()` and `file()` functions already handle local file reading and templating. No need to duplicate in the provider.

### File Delivery Mechanism
- **Decision:** SFTP over existing SSH connection
- **Rationale:** Purpose-built for file transfer, handles binary safely, reuses existing SSH infrastructure. Uses `github.com/pkg/sftp` library.

### Change Detection
- **Decision:** SHA256 checksum comparison
- **Rationale:** Reliable drift detection without storing full content in state. Compute hash locally, compare with remote file hash on Read.

### Path Specification
- **Decision:** Two modes, mutually exclusive
  1. `host_path` + `relative_path` - Reference managed `truenas_host_path` resource
  2. `path` - Standalone full path for unmanaged directories
- **Rationale:** Encourages proper resource composition while allowing flexibility for pre-existing paths.

### Nested Directories
- **Decision:** `relative_path` supports subdirs (e.g., `config/Caddyfile`), auto-creates intermediate directories within the managed `host_path`
- **Rationale:** Convenient for nested configs without requiring a `truenas_host_path` for every subdirectory. Safe because it's scoped to within a managed path.

### Permissions
- **Decision:** Inherit from `host_path` by default, fall back to sensible defaults (`0644`, root) for standalone `path` mode. Can override explicitly.
- **Rationale:** DRY when you want consistent ownership; explicit when you need it.

### Delete Behavior
- **Decision:** Delete file only, leave parent directories intact
- **Rationale:** Idiomatic Terraform behavior. Each resource manages only its own object. Parent cleanup handled by `truenas_host_path` or external tooling.

## Schema

```hcl
resource "truenas_file" "example" {
  # Option A: Reference managed host_path
  host_path     = truenas_host_path.config.id  # optional
  relative_path = "subdir/config.txt"          # optional, allows nested paths

  # Option B: Standalone path (for unmanaged directories)
  path = "/mnt/storage/existing/config.txt"    # optional

  # Content (required)
  content = templatefile("./config.tftpl", {
    server_ip = var.server_ip
    port      = var.port
  })

  # Permissions (optional)
  mode = "0644"  # defaults: inherit from host_path, or "0644"
  uid  = 0       # defaults: inherit from host_path, or 0
  gid  = 0       # defaults: inherit from host_path, or 0
}
```

### Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `host_path` | string | No* | ID of `truenas_host_path` resource |
| `relative_path` | string | No* | Path relative to host_path (can include subdirs) |
| `path` | string | No* | Full absolute path (for unmanaged directories) |
| `content` | string | Yes | File content (use `templatefile()` or `file()`) |
| `mode` | string | No | Unix permissions (e.g., "0644") |
| `uid` | number | No | Owner user ID |
| `gid` | number | No | Owner group ID |

*Must provide either `host_path` + `relative_path` OR `path`, not both.

### Computed Attributes

| Attribute | Type | Description |
|-----------|------|-------------|
| `id` | string | Full file path (same as `path` output) |
| `path` | string | Full resolved file path |
| `checksum` | string | SHA256 hash of file content |

## Validation Rules

1. Must provide `content`
2. Must provide either:
   - `host_path` AND `relative_path`, OR
   - `path` (standalone)
3. Cannot provide both `host_path` and `path`
4. `relative_path` cannot start with `/`
5. `path` must be absolute (start with `/`)
6. `relative_path` cannot contain `..` (path traversal)

## Implementation

### Client Interface Extension

```go
type Client interface {
    // Existing methods
    Call(ctx context.Context, method string, params any) (json.RawMessage, error)
    CallAndWait(ctx context.Context, method string, params any) (json.RawMessage, error)

    // New SFTP methods
    WriteFile(ctx context.Context, path string, content []byte, mode os.FileMode) error
    ReadFile(ctx context.Context, path string) ([]byte, error)
    DeleteFile(ctx context.Context, path string) error
    FileExists(ctx context.Context, path string) (bool, error)
    MkdirAll(ctx context.Context, path string, mode os.FileMode) error

    Close() error
}
```

### Resource Operations

| Operation | Behavior |
|-----------|----------|
| **Create** | Resolve path → MkdirAll for parents (if relative_path) → WriteFile → SetPermissions |
| **Read** | FileExists check → ReadFile → compute SHA256 → compare with state |
| **Update** | WriteFile (overwrite) → SetPermissions if changed |
| **Delete** | DeleteFile (file only, not parents) |
| **Import** | Read file at import ID, populate all state attributes |

### Error Handling

| Scenario | Behavior |
|----------|----------|
| File not found on Read | Remove from state (external deletion) |
| Permission denied | Clear error with path context |
| Parent doesn't exist (standalone `path` mode) | Error, do not auto-create |
| Parent doesn't exist (`host_path` mode) | Error if `host_path` resource doesn't exist |
| SFTP connection failure | Retry with backoff, then error |

### Dependencies

- `github.com/pkg/sftp` - SFTP client library
- Existing `golang.org/x/crypto/ssh` - already in use

## File Structure

```
internal/
├── client/
│   ├── client.go         # Interface (add SFTP methods)
│   ├── ssh.go            # Existing SSH client
│   ├── sftp.go           # New SFTP implementation
│   └── sftp_test.go      # SFTP unit tests
├── resources/
│   ├── file.go           # truenas_file resource
│   └── file_test.go      # Resource unit tests
```

## Testing Strategy

**Goal: 100% test coverage**

### Unit Tests (resources/file_test.go)

**Schema & Validation:**
- `host_path` + `relative_path` provided → valid
- `path` provided → valid
- Both `host_path` and `path` provided → error
- Neither provided → error
- `relative_path` without `host_path` → error
- `relative_path` starts with `/` → error
- `relative_path` contains `..` → error
- `path` is not absolute → error
- `content` missing → error

**Path Resolution:**
- `host_path` + `relative_path` → correct full path
- `host_path` + nested `relative_path` → correct full path
- Standalone `path` → unchanged

**Checksum:**
- Empty content → correct SHA256
- Non-empty content → correct SHA256
- Same content → same checksum
- Different content → different checksum

**Permission Inheritance:**
- With `host_path`, no explicit perms → inherits from host_path
- With `host_path`, explicit perms → uses explicit values
- Standalone `path`, no explicit perms → defaults (0644, 0, 0)
- Standalone `path`, explicit perms → uses explicit values

**CRUD Operations (with mock client):**
- Create success → state populated correctly
- Create with mkdir failure → error propagated
- Create with write failure → error propagated
- Read success → state matches remote
- Read file not found → resource removed from state
- Read checksum mismatch → triggers update
- Update content change → WriteFile called
- Update permission change → SetPermissions called
- Update no change → no calls
- Delete success → no error
- Delete file not found → no error (idempotent)
- Delete failure → error propagated
- Import success → state populated from remote

### Unit Tests (client/sftp_test.go)

**SFTP Operations (with mock SSH session):**
- WriteFile success
- WriteFile permission denied → error
- WriteFile parent not found → error
- ReadFile success
- ReadFile not found → error
- ReadFile permission denied → error
- DeleteFile success
- DeleteFile not found → no error
- DeleteFile permission denied → error
- FileExists true case
- FileExists false case
- FileExists error case
- MkdirAll success (single level)
- MkdirAll success (nested)
- MkdirAll already exists → no error
- MkdirAll permission denied → error

**Connection Handling:**
- SFTP session reuses existing SSH connection
- SFTP session creation failure → error
- Concurrent operations → thread-safe

### Integration Tests

- Full CRUD lifecycle on real TrueNAS
- Drift detection (modify file externally, verify Terraform detects)
- Auto-create subdirs for relative_path
- Permission inheritance and override
- Import existing file
- Error cases (missing parent in standalone mode, permission denied)

## Usage Examples

### With host_path (recommended)

```hcl
resource "truenas_host_path" "caddy_config" {
  path = "/mnt/storage/apps/caddy/config"
  mode = "0755"
  uid  = 0
  gid  = 0
}

resource "truenas_file" "caddyfile" {
  host_path     = truenas_host_path.caddy_config.id
  relative_path = "Caddyfile"
  content       = templatefile("${path.module}/templates/Caddyfile.tftpl", {
    server_ips    = var.server_ips
    service_ports = var.service_ports
  })
  # Inherits mode/uid/gid from host_path
}

resource "truenas_app" "caddy" {
  name           = "caddy"
  custom_app     = true
  compose_config = file("${path.module}/compose/caddy.yml")

  depends_on = [truenas_file.caddyfile]
}
```

### With nested relative_path

```hcl
resource "truenas_host_path" "app_root" {
  path = "/mnt/storage/apps/myapp"
}

resource "truenas_file" "config" {
  host_path     = truenas_host_path.app_root.id
  relative_path = "config/settings.json"  # Creates config/ subdir
  content       = jsonencode(var.app_settings)
  mode          = "0600"  # Override permissions
}
```

### Standalone path (unmanaged directory)

```hcl
resource "truenas_file" "hosts" {
  path    = "/etc/hosts.custom"
  content = templatefile("./hosts.tftpl", { entries = var.host_entries })
  mode    = "0644"
  uid     = 0
  gid     = 0
}
```

## Future Considerations

- **Binary files:** Current design handles text/binary via `content` attribute, but base64 encoding may be needed for true binary in HCL. Consider `content_base64` attribute if needed.
- **Large files:** SFTP handles streaming, but very large files may need chunking or progress reporting.
- **File templates on remote:** Not planned - Terraform-side templating is preferred.

## References

- [Kubernetes ConfigMap pattern](https://kubernetes.io/docs/concepts/configuration/configmap/)
- [Terraform local_file resource](https://registry.terraform.io/providers/hashicorp/local/latest/docs/resources/file)
- [github.com/pkg/sftp](https://github.com/pkg/sftp)
- [danitso/terraform-provider-sftp](https://github.com/danitso/terraform-provider-sftp)
