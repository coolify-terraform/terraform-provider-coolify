# Format: application|service|database:<parent_uuid>:<storage_uuid>
# Coolify has no GET for the schedule. Before the next apply, set enabled,
# save_s3, disable_local_backup, stop_during_backup, and every retention
# attribute. timeout and missing_backup_notification_days can stay omitted.
terraform import coolify_storage_backup.app_data 'application:00000000-0000-4000-8000-000000000001:00000000-0000-4000-8000-000000000002'
