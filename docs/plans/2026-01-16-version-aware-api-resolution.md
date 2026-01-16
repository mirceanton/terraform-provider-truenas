# Version-Aware API Resolution

## Problem

TrueNAS SCALE 25.10 renamed snapshot API methods:
- `zfs.snapshot.*` (pre-25.10) → `pool.snapshot.*` (25.10+)

The provider needs to support TrueNAS SCALE 24.x through current versions.

## Solution

Probe `system.version` once per terraform run, cache in-memory, and resolve API method names based on detected version.

## Design

### Version Struct (`internal/api/version.go`)

```go
type Flavor string

const (
    FlavorScale     Flavor = "SCALE"
    FlavorCommunity Flavor = "COMMUNITY"
    FlavorUnknown   Flavor = ""
)

type Version struct {
    Major  int
    Minor  int
    Patch  int
    Build  int
    Flavor Flavor
    Raw    string
}

func ParseVersion(raw string) (Version, error)
func (v Version) Compare(other Version) int
func (v Version) AtLeast(major, minor int) bool
```

Parses strings like:
- `TrueNAS-SCALE-24.10.2.4`
- `TrueNAS-25.04.2.4`

### Snapshot API (`internal/api/snapshot.go`)

**Shared response types** (used by both resource and datasource):

```go
type SnapshotResponse struct {
    ID         string             `json:"id"`
    Name       string             `json:"name"`
    Dataset    string             `json:"dataset"`
    Holds      map[string]any     `json:"holds"`
    Properties SnapshotProperties `json:"properties"`
}

type SnapshotProperties struct {
    CreateTXG  PropertyValue `json:"createtxg"`
    Used       ParsedValue   `json:"used"`
    Referenced ParsedValue   `json:"referenced"`
}

func (s *SnapshotResponse) HasHold() bool
```

**Method resolution:**

```go
const (
    MethodSnapshotCreate  = "create"
    MethodSnapshotQuery   = "query"
    MethodSnapshotDelete  = "delete"
    MethodSnapshotHold    = "hold"
    MethodSnapshotRelease = "release"
    MethodSnapshotClone   = "clone"
)

func ResolveSnapshotMethod(v Version, method string) string {
    prefix := "zfs.snapshot"
    if v.AtLeast(25, 10) {
        prefix = "pool.snapshot"
    }
    return prefix + "." + method
}
```

### Client Changes (`internal/client/`)

**Interface addition:**

```go
type Client interface {
    // ... existing methods ...
    GetVersion(ctx context.Context) (api.Version, error)
}
```

**Implementation (SSHClient):**

```go
type SSHClient struct {
    // ... existing fields ...
    version    api.Version
    versionErr error
    versionMu  sync.Once
}

func (c *SSHClient) GetVersion(ctx context.Context) (api.Version, error) {
    c.versionMu.Do(func() {
        result, err := c.Call(ctx, "system.version", nil)
        if err != nil {
            c.versionErr = fmt.Errorf("failed to detect TrueNAS version: %w", err)
            return
        }
        var raw string
        if err := json.Unmarshal(result, &raw); err != nil {
            c.versionErr = fmt.Errorf("failed to parse version response: %w", err)
            return
        }
        c.version, c.versionErr = api.ParseVersion(raw)
    })
    return c.version, c.versionErr
}
```

### Resource/Datasource Usage

```go
// In snapshot.go Create()
version, err := r.client.GetVersion(ctx)
if err != nil {
    resp.Diagnostics.AddError("TrueNAS Version Detection Failed", err.Error())
    return
}

method := api.ResolveSnapshotMethod(version, api.MethodSnapshotCreate)
_, err = r.client.Call(ctx, method, params)
```

## Error Handling

- Hard fail if `system.version` call fails
- Hard fail if version string cannot be parsed
- No fallback - version detection is required

## File Changes

| File | Change |
|------|--------|
| `internal/api/version.go` | New - Version struct and parsing |
| `internal/api/snapshot.go` | New - Shared types + method resolver |
| `internal/client/client.go` | Add `GetVersion()` to interface |
| `internal/client/ssh.go` | Implement version probing + caching |
| `internal/resources/snapshot.go` | Use `api.ResolveSnapshotMethod()`, import shared types |
| `internal/datasources/snapshots.go` | Use `api.ResolveSnapshotMethod()`, import shared types |

## Known Version Mappings

| Version | Snapshot API Prefix |
|---------|---------------------|
| < 25.10 | `zfs.snapshot` |
| >= 25.10 | `pool.snapshot` |

## Related

This design also enables future version-specific API handling as TrueNAS continues to evolve.
