package database

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

type CommonModel struct {
	Timeouts                timeouts.Value `tfsdk:"timeouts"`
	UUID                    types.String   `tfsdk:"uuid"`
	Name                    types.String   `tfsdk:"name"`
	Description             types.String   `tfsdk:"description"`
	ProjectUUID             types.String   `tfsdk:"project_uuid"`
	ServerUUID              types.String   `tfsdk:"server_uuid"`
	EnvironmentName         types.String   `tfsdk:"environment_name"`
	Image                   types.String   `tfsdk:"image"`
	IsPublic                types.Bool     `tfsdk:"is_public"`
	PublicPort              types.Int64    `tfsdk:"public_port"`
	LimitsMemory            types.String   `tfsdk:"limits_memory"`
	LimitsMemorySwap        types.String   `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness  types.Int64    `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation types.String   `tfsdk:"limits_memory_reservation"`
	LimitsCPUs              types.String   `tfsdk:"limits_cpus"`
	LimitsCPUSet            types.String   `tfsdk:"limits_cpuset"`
	LimitsCPUShares         types.Int64    `tfsdk:"limits_cpu_shares"`
	PortsMappings           types.String   `tfsdk:"ports_mappings"`
	CustomDockerRunOptions  types.String   `tfsdk:"custom_docker_run_options"`
	PublicPortTimeout       types.Int64    `tfsdk:"public_port_timeout"`
	IsLogDrainEnabled       types.Bool     `tfsdk:"is_log_drain_enabled"`
	IsIncludeTimestamps     types.Bool     `tfsdk:"is_include_timestamps"`
	HealthCheckEnabled      types.Bool     `tfsdk:"health_check_enabled"`
	HealthCheckInterval     types.Int64    `tfsdk:"health_check_interval"`
	HealthCheckTimeout      types.Int64    `tfsdk:"health_check_timeout"`
	HealthCheckRetries      types.Int64    `tfsdk:"health_check_retries"`
	HealthCheckStartPeriod  types.Int64    `tfsdk:"health_check_start_period"`
	Status                  types.String   `tfsdk:"status"`
	InternalDBUrl           types.String   `tfsdk:"internal_db_url"`
	InstantDeploy           types.Bool     `tfsdk:"instant_deploy"`
}

// DatabaseCommonPtrs groups pointers to the core database model fields
// used by FlattenDatabaseCommon.
type DatabaseCommonPtrs struct {
	UUID, Name, Description, Image   *types.String
	ProjectUUID, ServerUUID, EnvName *types.String
	IsPublic                         *types.Bool
	PublicPort                       *types.Int64
}

// ExtFields returns a DatabaseExtendedPtrs pointing to the extended fields
// in this CommonModel.
func (m *CommonModel) ExtFields() DatabaseExtendedPtrs {
	return DatabaseExtendedPtrs{
		LimitsMemory:            &m.LimitsMemory,
		LimitsMemorySwap:        &m.LimitsMemorySwap,
		LimitsMemorySwappiness:  &m.LimitsMemorySwappiness,
		LimitsMemoryReservation: &m.LimitsMemoryReservation,
		LimitsCPUs:              &m.LimitsCPUs,
		LimitsCPUSet:            &m.LimitsCPUSet,
		LimitsCPUShares:         &m.LimitsCPUShares,
		PortsMappings:           &m.PortsMappings,
		CustomDockerRunOptions:  &m.CustomDockerRunOptions,
		PublicPortTimeout:       &m.PublicPortTimeout,
		IsLogDrainEnabled:       &m.IsLogDrainEnabled,
		IsIncludeTimestamps:     &m.IsIncludeTimestamps,
		HealthCheckEnabled:      &m.HealthCheckEnabled,
		HealthCheckInterval:     &m.HealthCheckInterval,
		HealthCheckTimeout:      &m.HealthCheckTimeout,
		HealthCheckRetries:      &m.HealthCheckRetries,
		HealthCheckStartPeriod:  &m.HealthCheckStartPeriod,
		Status:                  &m.Status,
		InternalDBUrl:           &m.InternalDBUrl,
		InstantDeploy:           &m.InstantDeploy,
	}
}

// CommonPtrs returns a DatabaseCommonPtrs pointing to the core fields
// in this CommonModel.
func (m *CommonModel) CommonPtrs() DatabaseCommonPtrs {
	return DatabaseCommonPtrs{
		UUID: &m.UUID, Name: &m.Name, Description: &m.Description,
		Image: &m.Image, ProjectUUID: &m.ProjectUUID,
		ServerUUID: &m.ServerUUID, EnvName: &m.EnvironmentName,
		IsPublic: &m.IsPublic, PublicPort: &m.PublicPort,
	}
}

// CommonDatabaseAttrs returns the shared schema attributes for all database types.
func CommonDatabaseAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"timeouts":         timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the database.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":             schema.StringAttribute{MarkdownDescription: "The name of the database resource. Also used as the Docker container name and internal DNS hostname for inter-container communication.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"description":      schema.StringAttribute{MarkdownDescription: "A description of the database.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this database belongs to. Changing this forces a new resource.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the database on. Changing this forces a new resource.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"environment_name": schema.StringAttribute{MarkdownDescription: "The name of the environment within the project to deploy into. Coolify auto-creates a `production` environment per project; for other environments, create one first with `coolify_environment`. Defaults to `production`. Changing this forces a new resource.", Optional: true, Computed: true, Default: stringdefault.StaticString("production"), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"image":            schema.StringAttribute{MarkdownDescription: "The Docker image to use.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"is_public":        schema.BoolAttribute{MarkdownDescription: "When `true`, exposes the database on a port accessible via the server's IP address. When `false` (default), the database is only reachable from other containers on the same Docker network. Set `public_port` to choose a specific port.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"public_port": schema.Int64Attribute{MarkdownDescription: "The host port to expose the database on when `is_public` is `true`. If omitted, Coolify auto-assigns an available port. Ignored when `is_public` is `false`.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: []validator.Int64{
			int64validator.Between(1, 65535),
		}},
		// Resource limits
		"limits_memory":             schema.StringAttribute{MarkdownDescription: "Memory limit (e.g., `512m`, `2g`).", Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
		"limits_memory_swap":        schema.StringAttribute{MarkdownDescription: "Memory swap limit (e.g., `1g`).", Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
		"limits_memory_swappiness":  schema.Int64Attribute{MarkdownDescription: "Memory swappiness (0-100).", Optional: true, Computed: true, Default: int64default.StaticInt64(60)},
		"limits_memory_reservation": schema.StringAttribute{MarkdownDescription: "Memory reservation (e.g., `256m`).", Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
		"limits_cpus":               schema.StringAttribute{MarkdownDescription: "CPU limit (e.g., `0.5`, `2`).", Optional: true, Computed: true, Default: stringdefault.StaticString("0")},
		"limits_cpuset":             schema.StringAttribute{MarkdownDescription: "CPU set restriction (e.g., `0-3`, `0,2`).", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"limits_cpu_shares":         schema.Int64Attribute{MarkdownDescription: "CPU shares (relative weight).", Optional: true, Computed: true, Default: int64default.StaticInt64(1024)},
		// Container/network settings
		"ports_mappings": schema.StringAttribute{
			MarkdownDescription: "Port mappings in `host:container` format, comma-separated (e.g., `8080:5432`).",
			Optional:            true,
			Validators: []validator.String{
				validate.PortMappings(),
			},
		},
		"custom_docker_run_options": schema.StringAttribute{MarkdownDescription: "Custom Docker run options passed to the container.", Optional: true, Validators: []validator.String{validate.NoShellMetachars()}},
		"public_port_timeout":       schema.Int64Attribute{MarkdownDescription: "Timeout in seconds for public port allocation.", Optional: true},
		"is_log_drain_enabled":      schema.BoolAttribute{MarkdownDescription: "When `true`, sends container logs to the configured log drain. Defaults to `false`.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"is_include_timestamps":     IsIncludeTimestampsAttr(),
		"health_check_enabled":      schema.BoolAttribute{MarkdownDescription: "When `true`, enables the Docker health check probe for this database container. Defaults to `true`.", Optional: true, Computed: true, Default: booldefault.StaticBool(true)},
		"health_check_interval":     schema.Int64Attribute{MarkdownDescription: "Health check interval in seconds. Minimum `1`. Defaults to `15`.", Optional: true, Computed: true, Default: int64default.StaticInt64(15), Validators: []validator.Int64{int64validator.AtLeast(1)}},
		"health_check_timeout":      schema.Int64Attribute{MarkdownDescription: "Health check timeout in seconds. Minimum `1`. Defaults to `5`.", Optional: true, Computed: true, Default: int64default.StaticInt64(5), Validators: []validator.Int64{int64validator.AtLeast(1)}},
		"health_check_retries":      schema.Int64Attribute{MarkdownDescription: "Number of consecutive health check failures before the container is considered unhealthy. Minimum `1`. Defaults to `5`.", Optional: true, Computed: true, Default: int64default.StaticInt64(5), Validators: []validator.Int64{int64validator.AtLeast(1)}},
		"health_check_start_period": schema.Int64Attribute{MarkdownDescription: "Grace period in seconds before health checks start counting failures after container start. Minimum `0`. Defaults to `5`.", Optional: true, Computed: true, Default: int64default.StaticInt64(5), Validators: []validator.Int64{int64validator.AtLeast(0)}},
		"status":                    schema.StringAttribute{MarkdownDescription: "The current status of the database (e.g., `running`, `exited`).", Computed: true},
		"internal_db_url":           schema.StringAttribute{MarkdownDescription: "Internal connection URL for the database, accessible from other containers on the same server. Contains credentials; requires an API token with sensitive-data read permission.", Computed: true, Sensitive: true},
		"instant_deploy":            schema.BoolAttribute{MarkdownDescription: "Whether to immediately deploy the database after creation. When `true`, Coolify starts the database container right away. When `false` (default), the database is created but not started.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// IsIncludeTimestampsAttr returns the schema attribute for is_include_timestamps,
// shared by all database types.
func IsIncludeTimestampsAttr() schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: "When `true`, includes timestamps in container log output. Defaults to `false`.",
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	}
}

// EnableSSLAttr returns the schema attribute for enable_ssl, shared by all
// database types except ClickHouse.
func EnableSSLAttr() schema.BoolAttribute {
	return schema.BoolAttribute{
		MarkdownDescription: "When `true`, enables SSL/TLS encryption for database connections. Defaults to `false`.",
		Optional:            true,
		Computed:            true,
		Default:             booldefault.StaticBool(false),
	}
}

// SSLModePostgresqlAttr returns the ssl_mode schema attribute for PostgreSQL.
func SSLModePostgresqlAttr() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "The SSL connection mode for PostgreSQL. Only applies when `enable_ssl` is `true`. Valid values: `allow`, `prefer`, `require`, `verify-ca`, `verify-full`.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		Validators:          []validator.String{stringvalidator.OneOf("allow", "prefer", "require", "verify-ca", "verify-full")},
	}
}

// SSLModeMysqlAttr returns the ssl_mode schema attribute for MySQL/MariaDB.
func SSLModeMysqlAttr() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "The SSL connection mode. Only applies when `enable_ssl` is `true`. Valid values: `REQUIRED`, `DISABLED`, `PREFERRED`, `VERIFY_CA`, `VERIFY_IDENTITY`.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		Validators:          []validator.String{stringvalidator.OneOf("REQUIRED", "DISABLED", "PREFERRED", "VERIFY_CA", "VERIFY_IDENTITY")},
	}
}

// SSLModeMongodbAttr returns the ssl_mode schema attribute for MongoDB.
func SSLModeMongodbAttr() schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: "The SSL connection mode for MongoDB. Only applies when `enable_ssl` is `true`. Valid values: `allow`, `prefer`, `require`, `verify-ca`, `verify-full`.",
		Optional:            true,
		Computed:            true,
		PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		Validators:          []validator.String{stringvalidator.OneOf("allow", "prefer", "require", "verify-ca", "verify-full")},
	}
}

// ConfigureDatabase extracts the API client from provider data.
// Returns nil when ProviderData is nil (expected during early configure).
func ConfigureDatabase(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	return flex.ConfigureClient(req, &resp.Diagnostics)
}

// ReadDatabase fetches a database by UUID. Returns (nil, nil) when the
// database is not found (caller should remove the resource from state).
func ReadDatabase(ctx context.Context, c *client.Client, uuid string) (*client.Database, error) {
	db, err := c.GetDatabase(ctx, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return db, nil
}

// UpdateDatabase sends an update for the given database and reads back the
// result.
func UpdateDatabase(ctx context.Context, c *client.Client, uuid string, input client.UpdateDatabaseInput) (*client.Database, error) {
	if _, err := c.UpdateDatabase(ctx, uuid, input); err != nil {
		return nil, fmt.Errorf("updating database %s: %w", uuid, err)
	}

	db, err := c.GetDatabase(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("reading database %s after update: %w", uuid, err)
	}
	return db, nil
}

// DeleteDatabase removes a database by UUID, silently succeeding if already
// gone.
const deletePollingTimeoutWarningSummary = "Delete is still finishing in Coolify"

func addDeletePollingTimeoutWarning(resp *resource.DeleteResponse, resourceType, uuid string) {
	resp.Diagnostics.AddWarning(
		deletePollingTimeoutWarningSummary,
		fmt.Sprintf(
			"Coolify accepted deletion of %s %s, but the resource was still returned by the API when the provider stopped polling. Terraform removed it from state, but the remote resource may still exist temporarily. Wait a moment before retrying dependent operations if they still report it.",
			resourceType,
			uuid,
		),
	)
}

func DeleteDatabase(ctx context.Context, c *client.Client, resourceType, uuid string, resp *resource.DeleteResponse) error {
	if err := c.DeleteDatabase(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !client.PollUntilDeleted(ctx, func() error { _, err := c.GetDatabase(ctx, uuid); return err }) {
		tflog.Warn(ctx, "resource may still exist after polling timeout", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
		addDeletePollingTimeoutWarning(resp, resourceType, uuid)
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	return nil
}

// ReadDatabaseState reads a database by UUID and calls the flatten callback.
// If the database is not found, it removes the resource from state.
func ReadDatabaseState(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.ReadResponse,
	flatten func(*client.Database),
) {
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	db, err := ReadDatabase(ctx, c, uuid)
	if err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error reading %s", resourceType), fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
		return
	}
	if db == nil {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
		resp.State.RemoveResource(ctx)
		return
	}
	flatten(db)
}

// DeleteDatabaseState deletes a database by UUID and reports any error.
func DeleteDatabaseState(
	ctx context.Context,
	c *client.Client,
	resourceType string,
	uuid string,
	resp *resource.DeleteResponse,
) {
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	if err := DeleteDatabase(ctx, c, resourceType, uuid, resp); err != nil {
		resp.Diagnostics.AddError(fmt.Sprintf("Error deleting %s", resourceType), fmt.Sprintf("%s %s: %s", resourceType, uuid, err))
	}
}

func NormalizeCommonCreateState(m *CommonModel) {
	flex.NormalizeUnknownString(&m.Name)
	flex.NormalizeUnknownString(&m.Description)
	flex.NormalizeUnknownString(&m.EnvironmentName)
	flex.NormalizeUnknownString(&m.Image)
	flex.NormalizeUnknownBool(&m.IsPublic)
	flex.NormalizeUnknownInt64(&m.PublicPort)
	flex.NormalizeUnknownString(&m.Status)
}

func AddCreateReadBackError(resp *resource.CreateResponse, label, identifier string, err error) {
	resp.Diagnostics.AddError(
		fmt.Sprintf("%s created but refresh failed", label),
		fmt.Sprintf("Coolify created %s %s, but the provider could not read it back: Could not read %s %s after create: %s. The partial Terraform state was saved, so rerun terraform apply or terraform refresh after the API becomes reachable again.", label, identifier, label, identifier, err),
	)
}

// ImportDatabaseState validates the import ID as a UUID and passes it through
// as the "uuid" attribute.
func ImportDatabaseState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, compound, err := validate.ParseCompoundImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parsed.UUID)...)
	if compound {
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parsed.ProjectUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parsed.ServerUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parsed.EnvironmentName)...)
	}
}

// DatabaseExtendedPtrs groups pointers to extended database model fields
// shared across all database types.
type DatabaseExtendedPtrs struct {
	LimitsMemory            *types.String
	LimitsMemorySwap        *types.String
	LimitsMemorySwappiness  *types.Int64
	LimitsMemoryReservation *types.String
	LimitsCPUs              *types.String
	LimitsCPUSet            *types.String
	LimitsCPUShares         *types.Int64
	PortsMappings           *types.String
	CustomDockerRunOptions  *types.String
	PublicPortTimeout       *types.Int64
	IsLogDrainEnabled       *types.Bool
	IsIncludeTimestamps     *types.Bool
	HealthCheckEnabled      *types.Bool
	HealthCheckInterval     *types.Int64
	HealthCheckTimeout      *types.Int64
	HealthCheckRetries      *types.Int64
	HealthCheckStartPeriod  *types.Int64
	EnableSSL               *types.Bool
	SSLMode                 *types.String
	Status                  *types.String
	InternalDBUrl           *types.String
	InstantDeploy           *types.Bool
}

// WithSSL returns a copy of the DatabaseExtendedPtrs with EnableSSL and SSLMode
// pointers set. Use this for database types that support SSL (all except ClickHouse).
func (f DatabaseExtendedPtrs) WithSSL(enableSSL *types.Bool, sslMode *types.String) DatabaseExtendedPtrs {
	f.EnableSSL = enableSSL
	f.SSLMode = sslMode
	return f
}

// FlattenDatabaseExtended sets the extended fields shared by all database types.
// Optional-only fields use setIfConfigured to avoid "inconsistent result after
// apply" errors when the API returns defaults for unconfigured fields.
func FlattenDatabaseExtended(db *client.Database, f DatabaseExtendedPtrs) {
	// Fields with schema defaults — always set from API so import works.
	*f.LimitsMemory = flex.StringToFramework(db.LimitsMemory)
	*f.LimitsMemorySwap = flex.StringToFramework(db.LimitsMemorySwap)
	*f.LimitsMemoryReservation = flex.StringToFramework(db.LimitsMemoryReservation)
	*f.LimitsCPUs = flex.StringToFramework(db.LimitsCPUs)
	// limits_cpuset has no schema Default. Resolve unknown after create,
	// preserve user value on normal read, populate on import.
	if db.LimitsCPUSet != "" {
		*f.LimitsCPUSet = types.StringValue(db.LimitsCPUSet)
	} else if f.LimitsCPUSet.IsUnknown() {
		*f.LimitsCPUSet = types.StringNull()
	}
	*f.LimitsMemorySwappiness = flex.Int64PtrToFramework(db.LimitsMemorySwappiness)
	*f.LimitsCPUShares = flex.Int64PtrToFramework(db.LimitsCPUShares)
	// Fields without defaults — only set when configured.
	flex.SetStringOrClear(f.PortsMappings, db.PortsMappings)
	flex.SetStringOrClear(f.CustomDockerRunOptions, db.CustomDockerRunOptions)
	flex.SetInt64IfConfigured(f.PublicPortTimeout, db.PublicPortTimeout)
	// Logging settings (boolean with default false — always set from API).
	*f.IsLogDrainEnabled = types.BoolValue(db.IsLogDrainEnabled)
	*f.IsIncludeTimestamps = types.BoolValue(db.IsIncludeTimestamps)
	// Health check settings (boolean + integer with defaults — always set from API).
	if db.HealthCheckEnabled != nil {
		*f.HealthCheckEnabled = types.BoolValue(*db.HealthCheckEnabled)
	}
	*f.HealthCheckInterval = flex.Int64PtrToFramework(db.HealthCheckInterval)
	*f.HealthCheckTimeout = flex.Int64PtrToFramework(db.HealthCheckTimeout)
	*f.HealthCheckRetries = flex.Int64PtrToFramework(db.HealthCheckRetries)
	*f.HealthCheckStartPeriod = flex.Int64PtrToFramework(db.HealthCheckStartPeriod)
	// SSL settings — always set from API.
	if f.EnableSSL != nil {
		*f.EnableSSL = types.BoolValue(db.EnableSSL)
	}
	if f.SSLMode != nil {
		*f.SSLMode = flex.StringToFramework(db.SSLMode)
	}
	// Status is Computed — always set.
	*f.Status = flex.StringToFramework(db.Status)
	// Internal DB URL is Computed + Sensitive — always set from API.
	if f.InternalDBUrl != nil {
		*f.InternalDBUrl = flex.StringToFramework(db.InternalDBUrl)
	}
	// instant_deploy is create-only and never returned by the API.
	// Preserve state value when set; default to false otherwise (import).
	if f.InstantDeploy != nil && (f.InstantDeploy.IsNull() || f.InstantDeploy.IsUnknown()) {
		*f.InstantDeploy = types.BoolValue(false)
	}
}

// SetUpdateExtended populates the extended fields in an UpdateDatabaseInput.
func SetUpdateExtended(input *client.UpdateDatabaseInput, f DatabaseExtendedPtrs) {
	flex.SetStrPtr(&input.LimitsMemory, *f.LimitsMemory)
	flex.SetStrPtr(&input.LimitsMemorySwap, *f.LimitsMemorySwap)
	flex.SetStrPtr(&input.LimitsMemoryReservation, *f.LimitsMemoryReservation)
	flex.SetStrPtr(&input.LimitsCPUs, *f.LimitsCPUs)
	flex.SetStrPtr(&input.LimitsCPUSet, *f.LimitsCPUSet)
	flex.SetStrPtr(&input.PortsMappings, *f.PortsMappings)
	flex.SetStrPtr(&input.CustomDockerRunOptions, *f.CustomDockerRunOptions)
	flex.SetInt64Ptr(&input.LimitsMemorySwappiness, *f.LimitsMemorySwappiness)
	flex.SetInt64Ptr(&input.LimitsCPUShares, *f.LimitsCPUShares)
	input.PublicPortTimeout = flex.Int64PtrFromFramework(*f.PublicPortTimeout)
	flex.SetBoolPtr(&input.IsLogDrainEnabled, *f.IsLogDrainEnabled)
	flex.SetBoolPtr(&input.IsIncludeTimestamps, *f.IsIncludeTimestamps)
	flex.SetBoolPtr(&input.HealthCheckEnabled, *f.HealthCheckEnabled)
	flex.SetInt64Ptr(&input.HealthCheckInterval, *f.HealthCheckInterval)
	flex.SetInt64Ptr(&input.HealthCheckTimeout, *f.HealthCheckTimeout)
	flex.SetInt64Ptr(&input.HealthCheckRetries, *f.HealthCheckRetries)
	flex.SetInt64Ptr(&input.HealthCheckStartPeriod, *f.HealthCheckStartPeriod)
	if f.EnableSSL != nil {
		flex.SetBoolPtr(&input.EnableSSL, *f.EnableSSL)
	}
	if f.SSLMode != nil {
		flex.SetStrPtr(&input.SSLMode, *f.SSLMode)
	}
}

// SetUpdateExtendedDiff populates the extended fields in an UpdateDatabaseInput,
// only including fields that differ between plan and state.
func SetUpdateExtendedDiff(input *client.UpdateDatabaseInput, plan, state DatabaseExtendedPtrs) {
	input.LimitsMemory = flex.StringIfChanged(*plan.LimitsMemory, *state.LimitsMemory)
	input.LimitsMemorySwap = flex.StringIfChanged(*plan.LimitsMemorySwap, *state.LimitsMemorySwap)
	input.LimitsMemoryReservation = flex.StringIfChanged(*plan.LimitsMemoryReservation, *state.LimitsMemoryReservation)
	input.LimitsCPUs = flex.StringIfChanged(*plan.LimitsCPUs, *state.LimitsCPUs)
	input.LimitsCPUSet = flex.StringIfChanged(*plan.LimitsCPUSet, *state.LimitsCPUSet)
	input.PortsMappings = flex.StringIfChanged(*plan.PortsMappings, *state.PortsMappings)
	input.CustomDockerRunOptions = flex.StringIfChanged(*plan.CustomDockerRunOptions, *state.CustomDockerRunOptions)
	input.LimitsMemorySwappiness = flex.Int64IfChanged(*plan.LimitsMemorySwappiness, *state.LimitsMemorySwappiness)
	input.LimitsCPUShares = flex.Int64IfChanged(*plan.LimitsCPUShares, *state.LimitsCPUShares)
	input.PublicPortTimeout = flex.Int64IfChanged(*plan.PublicPortTimeout, *state.PublicPortTimeout)
	input.IsLogDrainEnabled = flex.BoolIfChanged(*plan.IsLogDrainEnabled, *state.IsLogDrainEnabled)
	input.IsIncludeTimestamps = flex.BoolIfChanged(*plan.IsIncludeTimestamps, *state.IsIncludeTimestamps)
	input.HealthCheckEnabled = flex.BoolIfChanged(*plan.HealthCheckEnabled, *state.HealthCheckEnabled)
	input.HealthCheckInterval = flex.Int64IfChanged(*plan.HealthCheckInterval, *state.HealthCheckInterval)
	input.HealthCheckTimeout = flex.Int64IfChanged(*plan.HealthCheckTimeout, *state.HealthCheckTimeout)
	input.HealthCheckRetries = flex.Int64IfChanged(*plan.HealthCheckRetries, *state.HealthCheckRetries)
	input.HealthCheckStartPeriod = flex.Int64IfChanged(*plan.HealthCheckStartPeriod, *state.HealthCheckStartPeriod)
	if plan.EnableSSL != nil {
		input.EnableSSL = flex.BoolIfChanged(*plan.EnableSSL, *state.EnableSSL)
	}
	if plan.SSLMode != nil {
		input.SSLMode = flex.StringIfChanged(*plan.SSLMode, *state.SSLMode)
	}
}

// HasExtendedFields returns true if any extended field is configured (not
// null/unknown), indicating an Update is needed after Create.
func HasExtendedFields(f DatabaseExtendedPtrs) bool {
	return flex.StringPtrNonDefault(f.LimitsMemory, "0") || flex.StringPtrNonDefault(f.LimitsMemorySwap, "0") ||
		flex.StringPtrNonDefault(f.LimitsMemoryReservation, "0") || flex.StringPtrNonDefault(f.LimitsCPUs, "0") ||
		flex.StringPtrConfigured(f.LimitsCPUSet) ||
		flex.Int64PtrNonDefault(f.LimitsMemorySwappiness, 60) || flex.Int64PtrNonDefault(f.LimitsCPUShares, 1024) ||
		flex.StringPtrConfigured(f.PortsMappings) || flex.StringPtrConfigured(f.CustomDockerRunOptions) ||
		flex.Int64PtrConfigured(f.PublicPortTimeout) ||
		flex.BoolPtrNonDefault(f.IsLogDrainEnabled, false) ||
		flex.BoolPtrNonDefault(f.IsIncludeTimestamps, false) ||
		flex.BoolPtrNonDefault(f.HealthCheckEnabled, true) ||
		flex.Int64PtrNonDefault(f.HealthCheckInterval, 15) ||
		flex.Int64PtrNonDefault(f.HealthCheckTimeout, 5) ||
		flex.Int64PtrNonDefault(f.HealthCheckRetries, 5) ||
		flex.Int64PtrNonDefault(f.HealthCheckStartPeriod, 5) ||
		(f.EnableSSL != nil && flex.BoolPtrNonDefault(f.EnableSSL, false)) ||
		(f.SSLMode != nil && flex.StringPtrConfigured(f.SSLMode))
}

// FlattenDatabaseCommon sets the fields shared by all database resource types.
func FlattenDatabaseCommon(db *client.Database, f DatabaseCommonPtrs) {
	*f.UUID = types.StringValue(db.UUID)
	*f.Name = types.StringValue(db.Name)
	*f.Image = flex.StringToFramework(db.Image)
	*f.IsPublic = types.BoolValue(db.IsPublic)
	*f.PublicPort = flex.Int64PtrToFramework(db.PublicPort)
	*f.Description = flex.StringToFramework(db.Description)
	if db.ProjectUUID != "" {
		*f.ProjectUUID = types.StringValue(db.ProjectUUID)
	}
	if db.ServerUUID != "" {
		*f.ServerUUID = types.StringValue(db.ServerUUID)
	}
	if db.EnvironmentName != "" {
		*f.EnvName = flex.StringToFramework(db.EnvironmentName)
	}
}
