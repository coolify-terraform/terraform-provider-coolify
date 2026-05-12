variable "s3_access_key" {
  type      = string
  sensitive = true
}

variable "s3_secret_key" {
  type      = string
  sensitive = true
}

resource "coolify_s3_storage" "example" {
  name        = "my-backup-storage"
  description = "S3 storage for database backups"
  endpoint    = "https://s3.amazonaws.com"
  bucket      = "coolify-backups"
  region      = "us-east-1"
  access_key  = var.s3_access_key
  secret_key  = var.s3_secret_key
}
