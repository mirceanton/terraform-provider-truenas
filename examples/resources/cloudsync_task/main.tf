# Create a daily backup task to S3
resource "truenas_cloudsync_task" "backup" {
  description   = "Daily backup to S3"
  path          = "/mnt/tank/data"
  credentials   = truenas_cloudsync_credentials.s3.id
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

  enabled = true
}
