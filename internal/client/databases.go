package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
)

type Database struct {
	UUID                    string `json:"uuid"`
	Name                    string `json:"name"`
	Description             string `json:"description,omitempty"`
	Type                    string `json:"type"`
	Image                   string `json:"image,omitempty"`
	IsPublic                bool   `json:"is_public"`
	PublicPort              *int64 `json:"public_port,omitempty"`
	ServerUUID              string `json:"server_uuid,omitempty"`
	ProjectUUID             string `json:"project_uuid,omitempty"`
	EnvironmentName         string `json:"environment_name,omitempty"`
	EnvironmentUUID         string `json:"environment_uuid,omitempty"`
	PostgresUser            string `json:"postgres_user,omitempty"`
	PostgresPassword        string `json:"postgres_password,omitempty"`
	PostgresDB              string `json:"postgres_db,omitempty"`
	MysqlUser               string `json:"mysql_user,omitempty"`
	MysqlPassword           string `json:"mysql_password,omitempty"`
	MysqlDatabase           string `json:"mysql_database,omitempty"`
	MysqlRootPassword       string `json:"mysql_root_password,omitempty"`
	MariadbUser             string `json:"mariadb_user,omitempty"`
	MariadbPassword         string `json:"mariadb_password,omitempty"`
	MariadbDatabase         string `json:"mariadb_database,omitempty"`
	MariadbRootPassword     string `json:"mariadb_root_password,omitempty"`
	MongoInitdbRootUsername string `json:"mongo_initdb_root_username,omitempty"`
	MongoInitdbRootPassword string `json:"mongo_initdb_root_password,omitempty"`
	MongoInitdbDatabase     string `json:"mongo_initdb_database,omitempty"`
	ClickhouseAdminUser     string `json:"clickhouse_admin_user,omitempty"`
	ClickhouseAdminPassword string `json:"clickhouse_admin_password,omitempty"`
	// Resource limits (shared across all DB types)
	LimitsMemory            string `json:"limits_memory,omitempty"`
	LimitsMemorySwap        string `json:"limits_memory_swap,omitempty"`
	LimitsMemorySwappiness  *int64 `json:"limits_memory_swappiness,omitempty"`
	LimitsMemoryReservation string `json:"limits_memory_reservation,omitempty"`
	LimitsCPUs              string `json:"limits_cpus,omitempty"`
	LimitsCPUSet            string `json:"limits_cpuset,omitempty"`
	LimitsCPUShares         *int64 `json:"limits_cpu_shares,omitempty"`
	// Container/network settings
	PortsMappings          string `json:"ports_mappings,omitempty"`
	CustomDockerRunOptions string `json:"custom_docker_run_options,omitempty"`
	PublicPortTimeout      *int64 `json:"public_port_timeout,omitempty"`
	Status                 string `json:"status,omitempty"`
	// Type-specific configs
	PostgresConf           string `json:"postgres_conf,omitempty"`
	PostgresInitdbArgs     string `json:"postgres_initdb_args,omitempty"`
	PostgresHostAuthMethod string `json:"postgres_host_auth_method,omitempty"`
	InitScripts            string `json:"init_scripts,omitempty"`
	MysqlConf              string `json:"mysql_conf,omitempty"`
	MariadbConf            string `json:"mariadb_conf,omitempty"`
	MongoConf              string `json:"mongo_conf,omitempty"`
	RedisConf              string `json:"redis_conf,omitempty"`
	ClickhouseDB           string `json:"clickhouse_db,omitempty"`
	KeydbConf              string `json:"keydb_conf,omitempty"`
	KeydbPassword          string `json:"keydb_password,omitempty"`
	DragonflyPassword      string `json:"dragonfly_password,omitempty"`
}
type CreatePostgresqlInput struct {
	ServerUUID       string `json:"server_uuid"`
	ProjectUUID      string `json:"project_uuid"`
	EnvironmentName  string `json:"environment_name"`
	EnvironmentUUID  string `json:"environment_uuid,omitempty"`
	Name             string `json:"name,omitempty"`
	Description      string `json:"description,omitempty"`
	Image            string `json:"image,omitempty"`
	PostgresUser     string `json:"postgres_user,omitempty"`
	PostgresPassword string `json:"postgres_password,omitempty"`
	PostgresDB       string `json:"postgres_db,omitempty"`
	IsPublic         *bool  `json:"is_public,omitempty"`
	PublicPort       *int64 `json:"public_port,omitempty"`
}
type CreateMysqlInput struct {
	ServerUUID        string `json:"server_uuid"`
	ProjectUUID       string `json:"project_uuid"`
	EnvironmentName   string `json:"environment_name"`
	EnvironmentUUID   string `json:"environment_uuid,omitempty"`
	Name              string `json:"name,omitempty"`
	Description       string `json:"description,omitempty"`
	Image             string `json:"image,omitempty"`
	MysqlRootPassword string `json:"mysql_root_password,omitempty"`
	MysqlUser         string `json:"mysql_user,omitempty"`
	MysqlPassword     string `json:"mysql_password,omitempty"`
	MysqlDatabase     string `json:"mysql_database,omitempty"`
	IsPublic          *bool  `json:"is_public,omitempty"`
	PublicPort        *int64 `json:"public_port,omitempty"`
}
type CreateMariadbInput struct {
	ServerUUID          string `json:"server_uuid"`
	ProjectUUID         string `json:"project_uuid"`
	EnvironmentName     string `json:"environment_name"`
	EnvironmentUUID     string `json:"environment_uuid,omitempty"`
	Name                string `json:"name,omitempty"`
	Description         string `json:"description,omitempty"`
	Image               string `json:"image,omitempty"`
	MariadbRootPassword string `json:"mariadb_root_password,omitempty"`
	MariadbUser         string `json:"mariadb_user,omitempty"`
	MariadbPassword     string `json:"mariadb_password,omitempty"`
	MariadbDatabase     string `json:"mariadb_database,omitempty"`
	IsPublic            *bool  `json:"is_public,omitempty"`
	PublicPort          *int64 `json:"public_port,omitempty"`
}
type CreateRedisInput struct {
	ServerUUID      string `json:"server_uuid"`
	ProjectUUID     string `json:"project_uuid"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Image           string `json:"image,omitempty"`
	IsPublic        *bool  `json:"is_public,omitempty"`
	PublicPort      *int64 `json:"public_port,omitempty"`
}
type CreateMongodbInput struct {
	ServerUUID              string `json:"server_uuid"`
	ProjectUUID             string `json:"project_uuid"`
	EnvironmentName         string `json:"environment_name"`
	EnvironmentUUID         string `json:"environment_uuid,omitempty"`
	Name                    string `json:"name,omitempty"`
	Description             string `json:"description,omitempty"`
	Image                   string `json:"image,omitempty"`
	MongoInitdbRootUsername string `json:"mongo_initdb_root_username,omitempty"`
	MongoInitdbRootPassword string `json:"mongo_initdb_root_password,omitempty"`
	MongoInitdbDatabase     string `json:"mongo_initdb_database,omitempty"`
	IsPublic                *bool  `json:"is_public,omitempty"`
	PublicPort              *int64 `json:"public_port,omitempty"`
}
type CreateClickhouseInput struct {
	ProjectUUID             string `json:"project_uuid"`
	ServerUUID              string `json:"server_uuid"`
	EnvironmentName         string `json:"environment_name"`
	EnvironmentUUID         string `json:"environment_uuid,omitempty"`
	Name                    string `json:"name,omitempty"`
	Description             string `json:"description,omitempty"`
	Image                   string `json:"image,omitempty"`
	IsPublic                *bool  `json:"is_public,omitempty"`
	PublicPort              *int64 `json:"public_port,omitempty"`
	ClickhouseAdminUser     string `json:"clickhouse_admin_user,omitempty"`
	ClickhouseAdminPassword string `json:"clickhouse_admin_password,omitempty"`
}

type CreateKeydbInput struct {
	ProjectUUID     string `json:"project_uuid"`
	ServerUUID      string `json:"server_uuid"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Image           string `json:"image,omitempty"`
	IsPublic        *bool  `json:"is_public,omitempty"`
	PublicPort      *int64 `json:"public_port,omitempty"`
}
type CreateDragonflyInput struct {
	ProjectUUID     string `json:"project_uuid"`
	ServerUUID      string `json:"server_uuid"`
	EnvironmentName string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
	Name            string `json:"name,omitempty"`
	Description     string `json:"description,omitempty"`
	Image           string `json:"image,omitempty"`
	IsPublic        *bool  `json:"is_public,omitempty"`
	PublicPort      *int64 `json:"public_port,omitempty"`
}

type UpdateDatabaseInput struct {
	Name                    *string `json:"name,omitempty"`
	Description             *string `json:"description,omitempty"`
	Image                   *string `json:"image,omitempty"`
	IsPublic                *bool   `json:"is_public,omitempty"`
	PublicPort              *int64  `json:"public_port,omitempty"`
	PostgresUser            *string `json:"postgres_user,omitempty"`
	PostgresPassword        *string `json:"postgres_password,omitempty"`
	PostgresDB              *string `json:"postgres_db,omitempty"`
	MysqlUser               *string `json:"mysql_user,omitempty"`
	MysqlPassword           *string `json:"mysql_password,omitempty"`
	MysqlDatabase           *string `json:"mysql_database,omitempty"`
	MysqlRootPassword       *string `json:"mysql_root_password,omitempty"`
	MariadbUser             *string `json:"mariadb_user,omitempty"`
	MariadbPassword         *string `json:"mariadb_password,omitempty"`
	MariadbDatabase         *string `json:"mariadb_database,omitempty"`
	MariadbRootPassword     *string `json:"mariadb_root_password,omitempty"`
	MongoInitdbRootUsername *string `json:"mongo_initdb_root_username,omitempty"`
	MongoInitdbRootPassword *string `json:"mongo_initdb_root_password,omitempty"`
	MongoInitdbDatabase     *string `json:"mongo_initdb_database,omitempty"`
	ClickhouseAdminUser     *string `json:"clickhouse_admin_user,omitempty"`
	ClickhouseAdminPassword *string `json:"clickhouse_admin_password,omitempty"`
	// Resource limits
	LimitsMemory            *string `json:"limits_memory,omitempty"`
	LimitsMemorySwap        *string `json:"limits_memory_swap,omitempty"`
	LimitsMemorySwappiness  *int64  `json:"limits_memory_swappiness,omitempty"`
	LimitsMemoryReservation *string `json:"limits_memory_reservation,omitempty"`
	LimitsCPUs              *string `json:"limits_cpus,omitempty"`
	LimitsCPUSet            *string `json:"limits_cpuset,omitempty"`
	LimitsCPUShares         *int64  `json:"limits_cpu_shares,omitempty"`
	// Container/network settings
	PortsMappings          *string `json:"ports_mappings,omitempty"`
	CustomDockerRunOptions *string `json:"custom_docker_run_options,omitempty"`
	PublicPortTimeout      *int64  `json:"public_port_timeout,omitempty"`
	// Type-specific configs
	PostgresConf           *string `json:"postgres_conf,omitempty"`
	PostgresInitdbArgs     *string `json:"postgres_initdb_args,omitempty"`
	PostgresHostAuthMethod *string `json:"postgres_host_auth_method,omitempty"`
	InitScripts            *string `json:"init_scripts,omitempty"`
	MysqlConf              *string `json:"mysql_conf,omitempty"`
	MariadbConf            *string `json:"mariadb_conf,omitempty"`
	MongoConf              *string `json:"mongo_conf,omitempty"`
	RedisConf              *string `json:"redis_conf,omitempty"`
	ClickhouseDB           *string `json:"clickhouse_db,omitempty"`
	KeydbConf              *string `json:"keydb_conf,omitempty"`
	KeydbPassword          *string `json:"keydb_password,omitempty"`
	DragonflyPassword      *string `json:"dragonfly_password,omitempty"`
}

func (c *Client) ListDatabases(ctx context.Context) ([]Database, error) {
	var d []Database
	if err := c.do(ctx, http.MethodGet, "/api/v1/databases", nil, &d); err != nil {
		return nil, fmt.Errorf("listing databases: %w", err)
	}
	return d, nil
}
func (c *Client) GetDatabase(ctx context.Context, uuid string) (*Database, error) {
	var d Database
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s", url.PathEscape(uuid)), nil, &d); err != nil {
		return nil, fmt.Errorf("getting database %s: %w", uuid, err)
	}
	return &d, nil
}

// CreateDatabase creates a database of the given type (postgresql, mysql,
// mariadb, redis, mongodb, clickhouse, keydb, dragonfly). The input struct
// is type-specific but serialized as JSON.
func (c *Client) CreateDatabase(ctx context.Context, dbType string, input any) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/"+url.PathEscape(dbType), input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating %s database: %w", dbType, err)
	}
	return &d, nil
}
func (c *Client) UpdateDatabase(ctx context.Context, uuid string, input UpdateDatabaseInput) (*Database, error) {
	var d Database
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s", url.PathEscape(uuid)), input, &d); err != nil {
		return nil, fmt.Errorf("updating database %s: %w", uuid, err)
	}
	return &d, nil
}
func (c *Client) DeleteDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("deleting database %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StartDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/start", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("starting database %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StopDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/stop", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("stopping database %s: %w", uuid, err)
	}
	return nil
}

// RestartDatabase restarts a database.
func (c *Client) RestartDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/restart", url.PathEscape(uuid)), nil, nil); err != nil {
		return fmt.Errorf("restarting database %s: %w", uuid, err)
	}
	return nil
}

// --- Database Backup types ---

type DatabaseBackup struct {
	ID                    int    `json:"id"`
	UUID                  string `json:"uuid"`
	DatabaseUUID          string `json:"database_uuid"`
	Frequency             string `json:"frequency"`
	Enabled               bool   `json:"enabled"`
	SaveS3                bool   `json:"save_s3,omitempty"`
	S3StorageID           string `json:"s3_storage_uuid,omitempty"`
	DatabaseType          string `json:"database_type,omitempty"`
	DatabasesToBackup     string `json:"databases_to_backup,omitempty"`
	DumpAll               bool   `json:"dump_all,omitempty"`
	RetainAmountLocally   *int64 `json:"database_backup_retention_amount_locally,omitempty"`
	RetainDaysLocally     *int64 `json:"database_backup_retention_days_locally,omitempty"`
	RetainMaxStorageLocal *int64 `json:"database_backup_retention_max_storage_locally,omitempty"`
	RetainAmountS3        *int64 `json:"database_backup_retention_amount_s3,omitempty"`
	RetainDaysS3          *int64 `json:"database_backup_retention_days_s3,omitempty"`
	RetainMaxStorageS3    *int64 `json:"database_backup_retention_max_storage_s3,omitempty"`
	Timeout               *int64 `json:"timeout,omitempty"`
}

type CreateDatabaseBackupInput struct {
	Frequency             string `json:"frequency"`
	Enabled               bool   `json:"enabled"`
	SaveS3                *bool  `json:"save_s3,omitempty"`
	S3StorageID           string `json:"s3_storage_uuid,omitempty"`
	DatabasesToBackup     string `json:"databases_to_backup,omitempty"`
	DumpAll               *bool  `json:"dump_all,omitempty"`
	BackupNow             *bool  `json:"backup_now,omitempty"`
	RetainAmountLocally   *int64 `json:"database_backup_retention_amount_locally,omitempty"`
	RetainDaysLocally     *int64 `json:"database_backup_retention_days_locally,omitempty"`
	RetainMaxStorageLocal *int64 `json:"database_backup_retention_max_storage_locally,omitempty"`
	RetainAmountS3        *int64 `json:"database_backup_retention_amount_s3,omitempty"`
	RetainDaysS3          *int64 `json:"database_backup_retention_days_s3,omitempty"`
	RetainMaxStorageS3    *int64 `json:"database_backup_retention_max_storage_s3,omitempty"`
	Timeout               *int64 `json:"timeout,omitempty"`
}

type UpdateDatabaseBackupInput struct {
	Frequency             *string `json:"frequency,omitempty"`
	Enabled               *bool   `json:"enabled,omitempty"`
	SaveS3                *bool   `json:"save_s3,omitempty"`
	S3StorageID           *string `json:"s3_storage_uuid,omitempty"`
	DatabasesToBackup     *string `json:"databases_to_backup,omitempty"`
	DumpAll               *bool   `json:"dump_all,omitempty"`
	RetainAmountLocally   *int64  `json:"database_backup_retention_amount_locally,omitempty"`
	RetainDaysLocally     *int64  `json:"database_backup_retention_days_locally,omitempty"`
	RetainMaxStorageLocal *int64  `json:"database_backup_retention_max_storage_locally,omitempty"`
	RetainAmountS3        *int64  `json:"database_backup_retention_amount_s3,omitempty"`
	RetainDaysS3          *int64  `json:"database_backup_retention_days_s3,omitempty"`
	RetainMaxStorageS3    *int64  `json:"database_backup_retention_max_storage_s3,omitempty"`
	Timeout               *int64  `json:"timeout,omitempty"`
}

func (c *Client) ListDatabaseBackups(ctx context.Context, dbUUID string) ([]DatabaseBackup, error) {
	var backups []DatabaseBackup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups", url.PathEscape(dbUUID)), nil, &backups); err != nil {
		return nil, fmt.Errorf("listing backups for database %s: %w", dbUUID, err)
	}
	return backups, nil
}

func (c *Client) GetDatabaseBackup(ctx context.Context, dbUUID string, backupID int) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups/%d", url.PathEscape(dbUUID), backupID), nil, &b); err != nil {
		return nil, fmt.Errorf("getting backup %d for database %s: %w", backupID, dbUUID, err)
	}
	return &b, nil
}

func (c *Client) CreateDatabaseBackup(ctx context.Context, dbUUID string, input CreateDatabaseBackupInput) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/databases/%s/backups", url.PathEscape(dbUUID)), input, &b, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating backup for database %s: %w", dbUUID, err)
	}
	return &b, nil
}

func (c *Client) UpdateDatabaseBackup(ctx context.Context, dbUUID string, backupUUID string, input UpdateDatabaseBackupInput) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s/backups/%s", url.PathEscape(dbUUID), url.PathEscape(backupUUID)), input, &b); err != nil {
		return nil, fmt.Errorf("updating backup %s for database %s: %w", backupUUID, dbUUID, err)
	}
	return &b, nil
}

func (c *Client) DeleteDatabaseBackup(ctx context.Context, dbUUID string, backupUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s/backups/%s", url.PathEscape(dbUUID), url.PathEscape(backupUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting backup %s for database %s: %w", backupUUID, dbUUID, err)
	}
	return nil
}

// --- Backup Execution types ---

// BackupExecution represents a single execution of a database backup.
type BackupExecution struct {
	UUID      string `json:"uuid"`
	Status    string `json:"status"`
	CreatedAt string `json:"created_at"`
	Size      int64  `json:"size,omitempty"`
}

// ListBackupExecutions returns all executions for a database backup.
func (c *Client) ListBackupExecutions(ctx context.Context, dbUUID, backupUUID string) ([]BackupExecution, error) {
	var wrapper struct {
		Executions []BackupExecution `json:"executions"`
	}
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups/%s/executions", url.PathEscape(dbUUID), url.PathEscape(backupUUID)), nil, &wrapper); err != nil {
		return nil, fmt.Errorf("listing backup executions for database %s backup %s: %w", dbUUID, backupUUID, err)
	}
	return wrapper.Executions, nil
}

// DeleteBackupExecution deletes a specific backup execution.
func (c *Client) DeleteBackupExecution(ctx context.Context, dbUUID, backupUUID, execUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s/backups/%s/executions/%s", url.PathEscape(dbUUID), url.PathEscape(backupUUID), url.PathEscape(execUUID)), nil, nil); err != nil {
		return fmt.Errorf("deleting backup execution %s for database %s backup %s: %w", execUUID, dbUUID, backupUUID, err)
	}
	return nil
}
