# truenas_cron_job Resource

Manages cron jobs for scheduled task execution on TrueNAS.

## Example Usage

### Basic Daily Backup Script

```hcl
resource "truenas_cron_job" "daily_backup" {
  user        = "root"
  command     = "/mnt/tank/scripts/backup.sh"
  description = "Daily backup script"

  schedule {
    minute = "0"
    hour   = "2"
  }
}
```

### Complex Schedule with Custom Options

```hcl
resource "truenas_cron_job" "weekly_maintenance" {
  user        = "admin"
  command     = "/usr/local/bin/maintenance.sh --verbose"
  description = "Weekly system maintenance"
  enabled     = true
  stdout      = false
  stderr      = false

  schedule {
    minute = "30"
    hour   = "3"
    dom    = "*"
    month  = "*"
    dow    = "0"
  }
}
```

### Hourly Health Check

```hcl
resource "truenas_cron_job" "health_check" {
  user        = "root"
  command     = "/mnt/tank/scripts/health-check.sh | logger -t health-check"
  description = "Hourly health check"
  stdout      = true
  stderr      = true

  schedule {
    minute = "0"
    hour   = "*"
  }
}
```

## Argument Reference

* `user` - (Required) User to run the command as.
* `command` - (Required) Command to execute.
* `description` - (Optional) Job description. Default: "".
* `enabled` - (Optional) Enable the cron job. Default: true.
* `stdout` - (Optional) Redirect stdout to syslog. Default: true.
* `stderr` - (Optional) Redirect stderr to syslog. Default: true.
* `schedule` - (Required) Schedule block. See Schedule below.

### schedule Block

* `minute` - (Required) Minute (0-59 or cron expression).
* `hour` - (Required) Hour (0-23 or cron expression).
* `dom` - (Optional) Day of month (1-31 or cron expression). Default: "*".
* `month` - (Optional) Month (1-12 or cron expression). Default: "*".
* `dow` - (Optional) Day of week (0-6, 0=Sunday, or cron expression). Default: "*".

## Attribute Reference

* `id` - Cron job ID.

## Import

Cron jobs can be imported using the job ID:

```bash
terraform import truenas_cron_job.example 1
```
