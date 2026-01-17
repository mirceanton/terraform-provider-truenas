# truenas_cloudsync_credentials Data Source

Looks up existing cloud sync credentials by name.

## Example Usage

```hcl
data "truenas_cloudsync_credentials" "existing" {
  name = "aws-backup"
}

resource "truenas_cloudsync_task" "backup" {
  description   = "Backup using existing credentials"
  path          = "/mnt/tank/data"
  credentials   = data.truenas_cloudsync_credentials.existing.id
  direction     = "push"
  transfer_mode = "sync"

  schedule {
    minute = "0"
    hour   = "2"
  }

  s3 {
    bucket = "my-backup-bucket"
    folder = "truenas-backup"
  }
}
```

## Argument Reference

* `name` - (Required) Name of the credentials to look up.

## Attribute Reference

* `id` - Credential ID.
* `provider_type` - Provider type. One of: `s3`, `b2`, `gcs`, `azure`.
