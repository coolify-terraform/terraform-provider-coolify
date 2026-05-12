# ACME Corp Database Evaluation

This scenario provisions all 8 database types supported by Coolify in a single
project. It validates that every engine creates correctly, reads back with the
right attributes, and destroys cleanly.

## Database Engines Tested

| Resource | Engine | Image |
|---|---|---|
| `coolify_postgresql_database` | PostgreSQL 16 | `postgres:16` |
| `coolify_mysql_database` | MySQL 8 | `mysql:8` |
| `coolify_mariadb_database` | MariaDB 11 | `mariadb:11` |
| `coolify_redis_database` | Redis 7 | `redis:7` |
| `coolify_mongodb_database` | MongoDB 7 | `mongo:7` |
| `coolify_clickhouse_database` | ClickHouse | `clickhouse/clickhouse-server:latest` |
| `coolify_keydb_database` | KeyDB | `eqalpha/keydb:latest` |
| `coolify_dragonfly_database` | Dragonfly | `docker.dragonflydb.io/dragonflydb/dragonfly:latest` |

## Prerequisites

- A running Coolify instance with API enabled
- A registered and reachable server
- Sufficient disk space for 8 database images (~3GB total)

## Running

```bash
export TF_VAR_coolify_endpoint="http://localhost:8000"
export TF_VAR_coolify_token="your-token"
export TF_VAR_server_uuid="your-server-uuid"

terraform test -verbose
```

First run takes 10-15 minutes to pull all images. Subsequent runs are faster.