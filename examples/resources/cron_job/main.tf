# Create a cron job that runs daily at midnight
resource "truenas_cron_job" "cleanup" {
  user        = "root"
  command     = "/usr/local/bin/cleanup.sh"
  description = "Daily cleanup task"
  enabled     = true

  schedule {
    minute = "0"
    hour   = "0"
  }
}
