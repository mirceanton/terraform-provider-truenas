# truenas_cloudsync_credentials Resource

Manages cloud sync credentials for S3, Backblaze B2, Google Cloud Storage, or Azure Blob Storage.

## Example Usage

### S3 Credentials

```hcl
resource "truenas_cloudsync_credentials" "aws" {
  name = "aws-backup"

  s3 {
    access_key_id     = var.aws_access_key
    secret_access_key = var.aws_secret_key
    region            = "us-east-1"
  }
}
```

### Backblaze B2 Credentials

```hcl
resource "truenas_cloudsync_credentials" "b2" {
  name = "backblaze-backup"

  b2 {
    account = var.b2_account_id
    key     = var.b2_application_key
  }
}
```

### Google Cloud Storage Credentials

```hcl
resource "truenas_cloudsync_credentials" "gcs" {
  name = "gcs-backup"

  gcs {
    service_account_credentials = file("service-account.json")
  }
}
```

### Azure Blob Storage Credentials

```hcl
resource "truenas_cloudsync_credentials" "azure" {
  name = "azure-backup"

  azure {
    account = var.azure_account
    key     = var.azure_key
  }
}
```

## Argument Reference

* `name` - (Required) Credential name.

Exactly one of the following provider blocks is required:

### s3 Block

* `access_key_id` - (Required) AWS access key ID.
* `secret_access_key` - (Required) AWS secret access key.
* `endpoint` - (Optional) Custom S3 endpoint for S3-compatible storage.
* `region` - (Optional) AWS region.

### b2 Block

* `account` - (Required) Backblaze B2 account ID.
* `key` - (Required) Backblaze B2 application key.

### gcs Block

* `service_account_credentials` - (Required) Google Cloud service account credentials JSON.

### azure Block

* `account` - (Required) Azure storage account name.
* `key` - (Required) Azure storage account key.

## Attribute Reference

* `id` - Credential ID.

## Import

Cloud sync credentials can be imported using the credential ID:

```bash
terraform import truenas_cloudsync_credentials.example 1
```
