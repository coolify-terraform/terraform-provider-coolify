package database

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
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
	Status                  types.String   `tfsdk:"status"`
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
		Status:                  &m.Status,
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
		"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this database belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the database on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
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
		"status":                    schema.StringAttribute{MarkdownDescription: "The current status of the database (e.g., `running`, `exited`).", Computed: true},
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
		MarkdownDescription: "The SSL connection mode for MySQL. Only applies when `enable_ssl` is `true`. Valid values: `REQUIRED`, `DISABLED`, `PREFERRED`, `VERIFY_CA`, `VERIFY_IDENTITY`.",
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
func DeleteDatabase(ctx context.Context, c *client.Client, resourceType, uuid string) error {
	if err := c.DeleteDatabase(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	if !client.PollUntilDeleted(ctx, func() error { _, err := c.GetDatabase(ctx, uuid); return err }) {
		tflog.Warn(ctx, "resource may still exist after polling timeout", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
	return nil
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
	EnableSSL               *types.Bool
	SSLMode                 *types.String
	Status                  *types.String
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
	// SSL settings — always set from API.
	if f.EnableSSL != nil {
		*f.EnableSSL = types.BoolValue(db.EnableSSL)
	}
	if f.SSLMode != nil {
		*f.SSLMode = flex.StringToFramework(db.SSLMode)
	}
	// Status is Computed — always set.
	*f.Status = flex.StringToFramework(db.Status)
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
	// strNonDefault returns true when the user configured a value that
	// differs from the Coolify default. Fields whose schema Default matches
	// the API create-response default return false here, avoiding an
	// unnecessary PATCH after create.
	strNonDefault := func(v *types.String, dflt string) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueString() != dflt
	}
	intNonDefault := func(v *types.Int64, dflt int64) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != dflt
	}
	strSet := func(v *types.String) bool { return v != nil && !v.IsNull() && !v.IsUnknown() }
	intSet := func(v *types.Int64) bool { return v != nil && !v.IsNull() && !v.IsUnknown() }
	boolNonDefault := func(v *types.Bool, dflt bool) bool {
		return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueBool() != dflt
	}
	return strNonDefault(f.LimitsMemory, "0") || strNonDefault(f.LimitsMemorySwap, "0") ||
		strNonDefault(f.LimitsMemoryReservation, "0") || strNonDefault(f.LimitsCPUs, "0") ||
		strSet(f.LimitsCPUSet) ||
		intNonDefault(f.LimitsMemorySwappiness, 60) || intNonDefault(f.LimitsCPUShares, 1024) ||
		strSet(f.PortsMappings) || strSet(f.CustomDockerRunOptions) ||
		intSet(f.PublicPortTimeout) ||
		boolNonDefault(f.IsLogDrainEnabled, false) ||
		boolNonDefault(f.IsIncludeTimestamps, false) ||
		(f.EnableSSL != nil && boolNonDefault(f.EnableSSL, false)) ||
		(f.SSLMode != nil && strSet(f.SSLMode))
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
