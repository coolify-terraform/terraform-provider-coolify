variable "project_name" {
  description = "Name for the Coolify project"
  type        = string
}

variable "project_description" {
  description = "Project description"
  type        = string
  default     = "Managed by Terraform"
}

variable "server_uuid" {
  description = "UUID of the Coolify server to deploy to"
  type        = string
}

variable "db_name" {
  description = "PostgreSQL database name"
  type        = string
}

variable "db_image" {
  description = "Docker image for the PostgreSQL database"
  type        = string
  default     = "postgres:16"
}

variable "git_repo" {
  description = "Git repository URL for the application"
  type        = string
}

variable "git_branch" {
  description = "Git branch to deploy"
  type        = string
  default     = "main"
}
