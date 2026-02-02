# Create a snapshot before making changes
resource "truenas_snapshot" "backup" {
  dataset_id = truenas_dataset.data.id
  name       = "pre-upgrade-backup"
}
