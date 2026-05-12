# ACME Corp Database Evaluation
#
# Provisions all 8 database types supported by Coolify to validate
# that every engine creates, reads back, and destroys correctly.

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

# --- Project ---

resource "coolify_project" "eval" {
  name        = "acme-db-eval"
  description = "Database engine evaluation"
}

# --- All 8 Database Engines ---

resource "coolify_postgresql_database" "pg" {
  name             = "eval-postgres"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "postgres:16"
  postgres_user    = "eval"
  postgres_db      = "eval_db"
  is_public        = false
}

resource "coolify_mysql_database" "mysql" {
  name             = "eval-mysql"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "mysql:8"
  mysql_user       = "eval"
  mysql_database   = "eval_db"
}

resource "coolify_mariadb_database" "mariadb" {
  name             = "eval-mariadb"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "mariadb:11"
  mariadb_user     = "eval"
  mariadb_database = "eval_db"
}

resource "coolify_redis_database" "redis" {
  name             = "eval-redis"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "redis:7"
}

resource "coolify_mongodb_database" "mongo" {
  name             = "eval-mongo"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "mongo:7"
}

resource "coolify_clickhouse_database" "clickhouse" {
  name             = "eval-clickhouse"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "clickhouse/clickhouse-server:latest"
}

resource "coolify_keydb_database" "keydb" {
  name             = "eval-keydb"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "eqalpha/keydb:latest"
}

resource "coolify_dragonfly_database" "dragonfly" {
  name             = "eval-dragonfly"
  project_uuid     = coolify_project.eval.uuid
  server_uuid      = var.server_uuid
  environment_name = "production"
  image            = "docker.dragonflydb.io/dragonflydb/dragonfly:latest"
}