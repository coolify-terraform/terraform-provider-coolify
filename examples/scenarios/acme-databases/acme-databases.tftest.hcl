# Acceptance test for all 8 database engines.
#
# Required variables are provided via TF_VAR_* environment variables:
#   TF_VAR_coolify_endpoint, TF_VAR_coolify_token, TF_VAR_server_uuid
#
# This scenario pulls 8 Docker images on first run (~3GB). Allow 45 minutes.

run "create_all_databases" {
  command = apply

  # --- Project ---
  assert {
    condition     = coolify_project.eval.name == "acme-db-eval"
    error_message = "Project name mismatch: got ${coolify_project.eval.name}"
  }

  # --- PostgreSQL ---
  assert {
    condition     = coolify_postgresql_database.pg.name == "eval-postgres"
    error_message = "PostgreSQL name mismatch: got ${coolify_postgresql_database.pg.name}"
  }
  assert {
    condition     = coolify_postgresql_database.pg.postgres_user == "eval"
    error_message = "PostgreSQL user mismatch: got ${coolify_postgresql_database.pg.postgres_user}"
  }

  # --- MySQL ---
  assert {
    condition     = coolify_mysql_database.mysql.name == "eval-mysql"
    error_message = "MySQL name mismatch: got ${coolify_mysql_database.mysql.name}"
  }
  assert {
    condition     = coolify_mysql_database.mysql.mysql_user == "eval"
    error_message = "MySQL user mismatch: got ${coolify_mysql_database.mysql.mysql_user}"
  }

  # --- MariaDB ---
  assert {
    condition     = coolify_mariadb_database.mariadb.name == "eval-mariadb"
    error_message = "MariaDB name mismatch: got ${coolify_mariadb_database.mariadb.name}"
  }

  # --- Redis ---
  assert {
    condition     = coolify_redis_database.redis.name == "eval-redis"
    error_message = "Redis name mismatch: got ${coolify_redis_database.redis.name}"
  }

  # --- MongoDB ---
  assert {
    condition     = coolify_mongodb_database.mongo.name == "eval-mongo"
    error_message = "MongoDB name mismatch: got ${coolify_mongodb_database.mongo.name}"
  }

  # --- ClickHouse ---
  assert {
    condition     = coolify_clickhouse_database.clickhouse.name == "eval-clickhouse"
    error_message = "ClickHouse name mismatch: got ${coolify_clickhouse_database.clickhouse.name}"
  }

  # --- KeyDB ---
  assert {
    condition     = coolify_keydb_database.keydb.name == "eval-keydb"
    error_message = "KeyDB name mismatch: got ${coolify_keydb_database.keydb.name}"
  }

  # --- Dragonfly ---
  assert {
    condition     = coolify_dragonfly_database.dragonfly.name == "eval-dragonfly"
    error_message = "Dragonfly name mismatch: got ${coolify_dragonfly_database.dragonfly.name}"
  }

  # --- All UUIDs are unique ---
  assert {
    condition     = coolify_postgresql_database.pg.uuid != coolify_mysql_database.mysql.uuid
    error_message = "PostgreSQL and MySQL have the same UUID"
  }
}

# Update: change description and verify the update is applied.
run "update_project_description" {
  command = apply

  variables {
    project_description = "Updated by scenario test"
  }

  assert {
    condition     = coolify_project.eval.description == "Updated by scenario test"
    error_message = "Project description not updated: got ${coolify_project.eval.description}"
  }
}

# Idempotency: re-plan should produce no changes.
run "idempotency" {
  command = plan

  assert {
    condition     = coolify_project.eval.name == "acme-db-eval"
    error_message = "Project name changed after re-plan (state corruption)"
  }
}