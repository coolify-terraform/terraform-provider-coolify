output "postgresql_uuid" {
  value = coolify_database_postgresql.pg.uuid
}

output "mysql_uuid" {
  value = coolify_database_mysql.mysql.uuid
}

output "mariadb_uuid" {
  value = coolify_database_mariadb.mariadb.uuid
}

output "redis_uuid" {
  value = coolify_database_redis.redis.uuid
}

output "mongodb_uuid" {
  value = coolify_database_mongodb.mongo.uuid
}

output "clickhouse_uuid" {
  value = coolify_database_clickhouse.clickhouse.uuid
}

output "keydb_uuid" {
  value = coolify_database_keydb.keydb.uuid
}

output "dragonfly_uuid" {
  value = coolify_database_dragonfly.dragonfly.uuid
}