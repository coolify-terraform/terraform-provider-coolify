# Docker Compose example: deploy a WordPress stack via Coolify
#
# Provisions:
# 1. A project
# 2. A Docker Compose application with WordPress + MariaDB

terraform {
  required_providers {
    coolify = {
      source = "SebTardifLabs/coolify"
    }
  }
}

provider "coolify" {
  endpoint = var.coolify_endpoint
  token    = var.coolify_token
}

# --- Variables ---

variable "coolify_endpoint" {
  description = "Coolify API endpoint"
  type        = string
}

variable "coolify_token" {
  description = "Coolify API token"
  type        = string
  sensitive   = true
}

variable "server_uuid" {
  description = "UUID of the server to deploy to"
  type        = string
}

variable "domain" {
  description = "Domain for the WordPress site"
  type        = string
  default     = ""
}

# --- Project ---

resource "coolify_project" "blog" {
  name        = "company-blog"
  description = "WordPress blog managed by Terraform"
}

# --- Docker Compose Application ---

resource "coolify_docker_compose_application" "wordpress" {
  name         = "wordpress-stack"
  project_uuid = coolify_project.blog.uuid
  server_uuid  = var.server_uuid
  fqdn         = var.domain != "" ? "https://${var.domain}" : null

  docker_compose_raw = <<-YAML
    services:
      wordpress:
        image: wordpress:6-apache
        ports:
          - "8080:80"
        environment:
          WORDPRESS_DB_HOST: mariadb:3306
          WORDPRESS_DB_USER: wordpress
          WORDPRESS_DB_PASSWORD: wp-secret-change-me
          WORDPRESS_DB_NAME: wordpress
        volumes:
          - wp_data:/var/www/html
        depends_on:
          - mariadb

      mariadb:
        image: mariadb:11
        environment:
          MYSQL_ROOT_PASSWORD: root-secret-change-me
          MYSQL_DATABASE: wordpress
          MYSQL_USER: wordpress
          MYSQL_PASSWORD: wp-secret-change-me
        volumes:
          - db_data:/var/lib/mysql

    volumes:
      wp_data:
      db_data:
  YAML
}

# --- Outputs ---

output "project_uuid" {
  value = coolify_project.blog.uuid
}

output "compose_app_uuid" {
  value = coolify_docker_compose_application.wordpress.uuid
}
