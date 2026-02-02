# Look up existing cloud sync credentials by name
data "truenas_cloudsync_credentials" "aws" {
  name = "aws-backup"
}

# Use the credentials ID in a cloud sync task
resource "truenas_cloudsync_task" "backup" {
  description = "Daily backup"
  path        = "/mnt/tank/data"
  credentials = data.truenas_cloudsync_credentials.aws.id
  direction   = "push"

  # ... rest of configuration
}
