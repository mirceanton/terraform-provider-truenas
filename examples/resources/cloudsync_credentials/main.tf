# Create AWS S3 cloud sync credentials
resource "truenas_cloudsync_credentials" "s3" {
  name = "aws-backup"

  s3 {
    access_key_id     = var.aws_access_key
    secret_access_key = var.aws_secret_key
    region            = "us-west-2"
  }
}
