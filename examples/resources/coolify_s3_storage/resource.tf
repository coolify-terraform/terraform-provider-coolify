variable "s3_access_key" {
  type      = string
  sensitive = true
}

variable "s3_secret_key" {
  type      = string
  sensitive = true
}

resource "coolify_s3_storage" "example" {
  name        = "my-s3-storage"
  description = "S3 storage for database backups"
  endpoint    = "https://s3.us-east-1.amazonaws.com"
  bucket      = "my-coolify-backups"
  region      = "us-east-1"
  key         = var.s3_access_key
  secret      = var.s3_secret_key
}
