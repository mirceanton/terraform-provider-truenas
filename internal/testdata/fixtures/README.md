# TrueNAS API Response Fixtures

This directory contains JSON fixture files representing realistic TrueNAS API responses.
These fixtures are used for unit testing the Terraform provider without requiring a real
TrueNAS server.

## Available Fixtures

### System Information
- `system_info.json` - Response from `system.info` API call

### Applications
- `app_query.json` - Response from `app.query` (multiple apps)
- `app_query_single.json` - Response from `app.query` with name filter (single app)

### Pools
- `pool_query.json` - Response from `pool.query` (multiple pools)

### Datasets
- `pool_dataset_query.json` - Response from `pool.dataset.query`
- `pool_dataset_create.json` - Response from `pool.dataset.create`

### Filesystem
- `filesystem_stat.json` - Response from `filesystem.stat`

### Snapshots
- `pool_snapshot_query.json` - Response from `pool.snapshot.query` (or `zfs.snapshot.query`)

## API Documentation References

These fixtures are based on the TrueNAS SCALE API documentation:
- https://api.truenas.com/v25.10/

## Usage Example

```go
import (
    "embed"
    "encoding/json"
)

//go:embed testdata/fixtures/*.json
var fixtures embed.FS

func loadFixture(name string) ([]byte, error) {
    return fixtures.ReadFile("testdata/fixtures/" + name)
}

func TestAppQuery(t *testing.T) {
    data, _ := loadFixture("app_query_single.json")

    var apps []appAPIResponse
    json.Unmarshal(data, &apps)

    // Use apps in test...
}
```

## Property Value Structure

Many TrueNAS API responses use a consistent property value structure:

```json
{
  "property_name": {
    "parsed": <native_type>,
    "rawvalue": "string",
    "value": "string",
    "source": "LOCAL|INHERITED|DEFAULT|NONE",
    "source_info": null | "parent_dataset"
  }
}
```

- `parsed`: Value converted to appropriate Go type (int64, bool, string)
- `rawvalue`: Raw string from ZFS
- `value`: Human-readable formatted value
- `source`: Where the value comes from
- `source_info`: Additional source context (e.g., parent dataset for inherited values)

## Adding New Fixtures

When adding new fixtures:

1. Reference the TrueNAS API documentation for the correct response structure
2. Cross-reference with existing Go structs in `internal/api/`, `internal/resources/`,
   and `internal/datasources/`
3. Include realistic sample data that exercises common scenarios
4. Document the fixture in this README
