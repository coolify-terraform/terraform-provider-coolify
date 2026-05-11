# Import a database backup using the composite ID format.
# Note: <backup-id> is the numeric "id" attribute (e.g. 42), NOT the backup's "uuid".
terraform import coolify_database_backup.example <database-uuid>:<backup-id>
