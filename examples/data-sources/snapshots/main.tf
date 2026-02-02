# List all snapshots for a dataset
data "truenas_snapshots" "backups" {
  dataset_id = "tank/data"
}

# Output snapshot information
output "snapshot_count" {
  value = length(data.truenas_snapshots.backups.snapshots)
}

output "snapshot_ids" {
  value = [for s in data.truenas_snapshots.backups.snapshots : s.id]
}
