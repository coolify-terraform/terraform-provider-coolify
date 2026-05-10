# Import a persistent storage on an application:
terraform import coolify_storage.example application:<application-uuid>:<storage-uuid>

# Import a persistent storage on a service:
terraform import coolify_storage.example service:<service-uuid>:<storage-uuid>

# Import a persistent storage on a database:
terraform import coolify_storage.example database:<database-uuid>:<storage-uuid>
