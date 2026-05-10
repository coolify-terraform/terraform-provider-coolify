package client

import (
	"context"
	"fmt"
	"net/http"
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
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
}
type CreatePostgresqlInput struct {
	ServerUUID       string `json:"server_uuid"`
	ProjectUUID      string `json:"project_uuid"`
	EnvironmentName  string `json:"environment_name"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	EnvironmentName         string `json:"environment_name,omitempty"`
	EnvironmentUUID string `json:"environment_uuid,omitempty"`
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
	EnvironmentName string `json:"environment_name,omitempty"`
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
	EnvironmentName string `json:"environment_name,omitempty"`
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
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s", uuid), nil, &d); err != nil {
		return nil, fmt.Errorf("getting database %s: %w", uuid, err)
	}
	return &d, nil
}
func (c *Client) CreatePostgresqlDatabase(ctx context.Context, input CreatePostgresqlInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/postgresql", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating postgresql database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateMysqlDatabase(ctx context.Context, input CreateMysqlInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/mysql", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating mysql database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateMariadbDatabase(ctx context.Context, input CreateMariadbInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/mariadb", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating mariadb database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateRedisDatabase(ctx context.Context, input CreateRedisInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/redis", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating redis database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateMongodbDatabase(ctx context.Context, input CreateMongodbInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/mongodb", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating mongodb database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateClickhouseDatabase(ctx context.Context, input CreateClickhouseInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/clickhouse", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating clickhouse database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateKeydbDatabase(ctx context.Context, input CreateKeydbInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/keydb", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating keydb database: %w", err)
	}
	return &d, nil
}
func (c *Client) CreateDragonflyDatabase(ctx context.Context, input CreateDragonflyInput) (*Database, error) {
	var d Database
	if err := c.doWithStatus(ctx, http.MethodPost, "/api/v1/databases/dragonfly", input, &d, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating dragonfly database: %w", err)
	}
	return &d, nil
}
func (c *Client) UpdateDatabase(ctx context.Context, uuid string, input UpdateDatabaseInput) (*Database, error) {
	var d Database
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s", uuid), input, &d); err != nil {
		return nil, fmt.Errorf("updating database %s: %w", uuid, err)
	}
	return &d, nil
}
func (c *Client) DeleteDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s", uuid), nil, nil); err != nil {
		return fmt.Errorf("deleting database %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StartDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/start", uuid), nil, nil); err != nil {
		return fmt.Errorf("starting database %s: %w", uuid, err)
	}
	return nil
}
func (c *Client) StopDatabase(ctx context.Context, uuid string) error {
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/stop", uuid), nil, nil); err != nil {
		return fmt.Errorf("stopping database %s: %w", uuid, err)
	}
	return nil
}

// --- Database Backup types ---

type DatabaseBackup struct {
	ID           int    `json:"id"`
	UUID         string `json:"uuid"`
	DatabaseUUID string `json:"database_uuid"`
	Frequency    string `json:"frequency"`
	Enabled      bool   `json:"enabled"`
	S3StorageID  string `json:"s3_storage_id,omitempty"`
	DatabaseType string `json:"database_type,omitempty"`
	RetainDays   *int64 `json:"database_backup_retention_amount_locally,omitempty"`
}

type CreateDatabaseBackupInput struct {
	Frequency   string `json:"frequency"`
	Enabled     bool   `json:"enabled"`
	S3StorageID string `json:"s3_storage_id,omitempty"`
	RetainDays  *int64 `json:"database_backup_retention_amount_locally,omitempty"`
}

type UpdateDatabaseBackupInput struct {
	Frequency   *string `json:"frequency,omitempty"`
	Enabled     *bool   `json:"enabled,omitempty"`
	S3StorageID *string `json:"s3_storage_id,omitempty"`
	RetainDays  *int64  `json:"database_backup_retention_amount_locally,omitempty"`
}

func (c *Client) ListDatabaseBackups(ctx context.Context, dbUUID string) ([]DatabaseBackup, error) {
	var backups []DatabaseBackup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID), nil, &backups); err != nil {
		return nil, fmt.Errorf("listing backups for database %s: %w", dbUUID, err)
	}
	return backups, nil
}

func (c *Client) GetDatabaseBackup(ctx context.Context, dbUUID string, backupID int) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID), nil, &b); err != nil {
		return nil, fmt.Errorf("getting backup %d for database %s: %w", backupID, dbUUID, err)
	}
	return &b, nil
}

func (c *Client) CreateDatabaseBackup(ctx context.Context, dbUUID string, input CreateDatabaseBackupInput) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.doWithStatus(ctx, http.MethodPost, fmt.Sprintf("/api/v1/databases/%s/backups", dbUUID), input, &b, http.StatusCreated); err != nil {
		return nil, fmt.Errorf("creating backup for database %s: %w", dbUUID, err)
	}
	return &b, nil
}

func (c *Client) UpdateDatabaseBackup(ctx context.Context, dbUUID string, backupID int, input UpdateDatabaseBackupInput) (*DatabaseBackup, error) {
	var b DatabaseBackup
	if err := c.do(ctx, http.MethodPatch, fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID), input, &b); err != nil {
		return nil, fmt.Errorf("updating backup %d for database %s: %w", backupID, dbUUID, err)
	}
	return &b, nil
}

func (c *Client) DeleteDatabaseBackup(ctx context.Context, dbUUID string, backupID int) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s/backups/%d", dbUUID, backupID), nil, nil); err != nil {
		return fmt.Errorf("deleting backup %d for database %s: %w", backupID, dbUUID, err)
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
	var execs []BackupExecution
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/api/v1/databases/%s/backups/%s/executions", dbUUID, backupUUID), nil, &execs); err != nil {
		return nil, fmt.Errorf("listing backup executions for database %s backup %s: %w", dbUUID, backupUUID, err)
	}
	return execs, nil
}

// DeleteBackupExecution deletes a specific backup execution.
func (c *Client) DeleteBackupExecution(ctx context.Context, dbUUID, backupUUID, execUUID string) error {
	if err := c.do(ctx, http.MethodDelete, fmt.Sprintf("/api/v1/databases/%s/backups/%s/executions/%s", dbUUID, backupUUID, execUUID), nil, nil); err != nil {
		return fmt.Errorf("deleting backup execution %s for database %s backup %s: %w", execUUID, dbUUID, backupUUID, err)
	}
	return nil
}
