output "postgresql_uuid" {
  value = coolify_postgresql_database.pg.uuid
}

output "mysql_uuid" {
  value = coolify_mysql_database.mysql.uuid
}

output "mariadb_uuid" {
  value = coolify_mariadb_database.mariadb.uuid
}

output "redis_uuid" {
  value = coolify_redis_database.redis.uuid
}

output "mongodb_uuid" {
  value = coolify_mongodb_database.mongo.uuid
}

output "clickhouse_uuid" {
  value = coolify_clickhouse_database.clickhouse.uuid
}

output "keydb_uuid" {
  value = coolify_keydb_database.keydb.uuid
}

output "dragonfly_uuid" {
  value = coolify_dragonfly_database.dragonfly.uuid
}