package postgresql

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
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

var (
	_ resource.Resource                = &postgresqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &postgresqlDatabaseResource{}
	_ resource.ResourceWithImportState = &postgresqlDatabaseResource{}
)

type postgresqlDatabaseResource struct{ client *client.Client }

// CommonModel contains the fields shared by all database resource types.
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

type postgresqlDatabaseResourceModel struct {
	CommonModel
	// Type-specific
	PostgresUser           types.String `tfsdk:"postgres_user"`
	PostgresPassword       types.String `tfsdk:"postgres_password"`
	PostgresDB             types.String `tfsdk:"postgres_db"`
	PostgresConf           types.String `tfsdk:"postgres_conf"`
	PostgresInitdbArgs     types.String `tfsdk:"postgres_initdb_args"`
	PostgresHostAuthMethod types.String `tfsdk:"postgres_host_auth_method"`
	InitScripts            types.String `tfsdk:"init_scripts"`
}

func NewResource() resource.Resource { return &postgresqlDatabaseResource{} }

func (r *postgresqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_database"
}

func (r *postgresqlDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a PostgreSQL database resource on Coolify.",
		Attributes: CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
			"postgres_user": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL superuser name (maps to `POSTGRES_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_password": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL superuser password (maps to `POSTGRES_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_db": schema.StringAttribute{
				MarkdownDescription: "The default database name (maps to `POSTGRES_DB`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_conf": schema.StringAttribute{
				MarkdownDescription: "Custom PostgreSQL configuration (base64-encoded `postgresql.conf` content).",
				Optional:            true,
			},
			"postgres_initdb_args": schema.StringAttribute{
				MarkdownDescription: "Additional arguments passed to `initdb` (maps to `POSTGRES_INITDB_ARGS`).",
				Optional:            true,
			},
			"postgres_host_auth_method": schema.StringAttribute{
				MarkdownDescription: "Host authentication method (maps to `POSTGRES_HOST_AUTH_METHOD`, e.g. `trust`, `scram-sha-256`).",
				Optional:            true,
			},
			"init_scripts": schema.StringAttribute{
				MarkdownDescription: "Initialization scripts as a JSON array.",
				Optional:            true,
			},
		}),
	}
}

func (r *postgresqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = ConfigureDatabase(req, resp)
}

func (r *postgresqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_postgresql_database"})

	input := client.CreatePostgresqlInput{
		ServerUUID:      plan.ServerUUID.ValueString(),
		ProjectUUID:     plan.ProjectUUID.ValueString(),
		EnvironmentName: plan.EnvironmentName.ValueString(),
	}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.PostgresUser, plan.PostgresUser)
	flex.SetIfKnown(&input.PostgresPassword, plan.PostgresPassword)
	flex.SetIfKnown(&input.PostgresDB, plan.PostgresDB)
	input.IsPublic = flex.BoolValueOrNull(plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)

	created, err := r.client.CreateDatabase(ctx, "postgresql", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PostgreSQL database",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	NormalizeCommonCreateState(&plan.CommonModel)
	NormalizeUnknownString(&plan.PostgresUser)
	NormalizeUnknownString(&plan.PostgresPassword)
	NormalizeUnknownString(&plan.PostgresDB)
	NormalizeUnknownString(&plan.PostgresConf)
	NormalizeUnknownString(&plan.PostgresInitdbArgs)
	NormalizeUnknownString(&plan.PostgresHostAuthMethod)
	NormalizeUnknownString(&plan.InitScripts)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply extended fields that cannot be set during creation.
	ext := plan.ExtFields()
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	needsUpdate := HasExtendedFields(ext) || strSet(plan.PostgresConf) || strSet(plan.PostgresInitdbArgs) || strSet(plan.PostgresHostAuthMethod) || strSet(plan.InitScripts)
	if needsUpdate {
		update := client.UpdateDatabaseInput{}
		SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.PostgresConf, plan.PostgresConf)
		flex.SetStrPtr(&update.PostgresInitdbArgs, plan.PostgresInitdbArgs)
		flex.SetStrPtr(&update.PostgresHostAuthMethod, plan.PostgresHostAuthMethod)
		flex.SetStrPtr(&update.InitScripts, plan.InitScripts)
		if _, err := r.client.UpdateDatabase(ctx, created.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting PostgreSQL database extended fields", fmt.Sprintf("PostgreSQL database %s: %s", created.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		AddCreateReadBackError(resp, "PostgreSQL database", created.UUID, err)
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *postgresqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": state.UUID.ValueString()})

	db, err := ReadDatabase(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
	if db == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	flattenDatabase(db, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *postgresqlDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": uuid})

	input := client.UpdateDatabaseInput{
		Name:                   flex.StringIfChanged(plan.Name, state.Name),
		Description:            flex.StringIfChanged(plan.Description, state.Description),
		Image:                  flex.StringIfChanged(plan.Image, state.Image),
		IsPublic:               flex.BoolIfChanged(plan.IsPublic, state.IsPublic),
		PublicPort:             flex.Int64IfChanged(plan.PublicPort, state.PublicPort),
		PostgresUser:           flex.StringIfChanged(plan.PostgresUser, state.PostgresUser),
		PostgresPassword:       flex.StringIfChanged(plan.PostgresPassword, state.PostgresPassword),
		PostgresDB:             flex.StringIfChanged(plan.PostgresDB, state.PostgresDB),
		PostgresConf:           flex.StringIfChanged(plan.PostgresConf, state.PostgresConf),
		PostgresInitdbArgs:     flex.StringIfChanged(plan.PostgresInitdbArgs, state.PostgresInitdbArgs),
		PostgresHostAuthMethod: flex.StringIfChanged(plan.PostgresHostAuthMethod, state.PostgresHostAuthMethod),
		InitScripts:            flex.StringIfChanged(plan.InitScripts, state.InitScripts),
	}
	SetUpdateExtendedDiff(&input, plan.ExtFields(), state.ExtFields())
	db, err := UpdateDatabase(ctx, r.client, uuid, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", uuid, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *postgresqlDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": state.UUID.ValueString()})

	if err := DeleteDatabase(ctx, r.client, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *postgresqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ImportDatabaseState(ctx, req, resp)
}

func flattenDatabase(db *client.Database, m *postgresqlDatabaseResourceModel) {
	FlattenDatabaseCommon(db, m.CommonPtrs())
	FlattenDatabaseExtended(db, m.ExtFields())
	m.PostgresUser = flex.StringToFramework(db.PostgresUser)
	// Preserve password from plan/state when the API hides sensitive fields.
	if db.PostgresPassword != "" {
		m.PostgresPassword = types.StringValue(db.PostgresPassword)
	} else if m.PostgresPassword.IsUnknown() {
		m.PostgresPassword = types.StringNull()
	}
	m.PostgresDB = flex.StringToFramework(db.PostgresDB)
	flex.SetStringIfConfigured(&m.PostgresConf, db.PostgresConf)
	flex.SetStringIfConfigured(&m.PostgresInitdbArgs, db.PostgresInitdbArgs)
	flex.SetStringIfConfigured(&m.PostgresHostAuthMethod, db.PostgresHostAuthMethod)
	flex.SetStringIfConfigured(&m.InitScripts, db.InitScripts)
}

// --- shared helpers ---

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
			MarkdownDescription: "Port mappings in `host:container` format, comma-separated (e.g. `8080:5432`).",
			Optional:            true,
			Validators: []validator.String{
				validate.PortMappings(),
			},
		},
		"custom_docker_run_options": schema.StringAttribute{MarkdownDescription: "Custom Docker run options passed to the container.", Optional: true, Validators: []validator.String{validate.NoShellMetachars()}},
		"public_port_timeout":       schema.Int64Attribute{MarkdownDescription: "Timeout in seconds for public port allocation.", Optional: true},
		"status":                    schema.StringAttribute{MarkdownDescription: "The current status of the database (e.g. `running`, `exited`).", Computed: true},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// ConfigureDatabase extracts the API client from provider data.
// Returns nil when ProviderData is nil (expected during early configure).
func ConfigureDatabase(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return nil
	}
	return c
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
		return nil, err
	}

	db, err := c.GetDatabase(ctx, uuid)
	if err != nil {
		return nil, fmt.Errorf("reading database %s after update: %w", uuid, err)
	}
	return db, nil
}

// DeleteDatabase removes a database by UUID, silently succeeding if already
// gone.
func DeleteDatabase(ctx context.Context, c *client.Client, uuid string) error {
	if err := c.DeleteDatabase(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	client.PollUntilDeleted(ctx, func() error { _, err := c.GetDatabase(ctx, uuid); return err })
	return nil
}

func NormalizeUnknownString(v *types.String) {
	if v != nil && v.IsUnknown() {
		*v = types.StringNull()
	}
}

func NormalizeUnknownBool(v *types.Bool) {
	if v != nil && v.IsUnknown() {
		*v = types.BoolNull()
	}
}

func NormalizeUnknownInt64(v *types.Int64) {
	if v != nil && v.IsUnknown() {
		*v = types.Int64Null()
	}
}

func NormalizeCommonCreateState(m *CommonModel) {
	NormalizeUnknownString(&m.Name)
	NormalizeUnknownString(&m.Description)
	NormalizeUnknownString(&m.EnvironmentName)
	NormalizeUnknownString(&m.Image)
	NormalizeUnknownBool(&m.IsPublic)
	NormalizeUnknownInt64(&m.PublicPort)
	NormalizeUnknownString(&m.Status)
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
	// Accept both simple UUID and compound "project_uuid:server_uuid:environment_name:uuid" formats.
	parts := strings.SplitN(req.ID, ":", 4)
	switch len(parts) {
	case 1:
		if err := validate.ImportUUID(parts[0]); err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[0])...)
	case 4:
		if err := validate.ImportUUID(parts[0]); err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", "project_uuid: "+err.Error())
			return
		}
		if err := validate.ImportUUID(parts[1]); err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", "server_uuid: "+err.Error())
			return
		}
		if parts[2] == "" {
			resp.Diagnostics.AddError("Invalid Import ID", "environment_name must not be empty")
			return
		}
		if err := validate.ImportUUID(parts[3]); err != nil {
			resp.Diagnostics.AddError("Invalid Import ID", "uuid: "+err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parts[0])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parts[1])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parts[2])...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parts[3])...)
	default:
		resp.Diagnostics.AddError("Invalid Import ID",
			"Expected UUID or project_uuid:server_uuid:environment_name:uuid, got: "+req.ID)
		return
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
	Status                  *types.String
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
	flex.SetStringIfConfigured(f.PortsMappings, db.PortsMappings)
	flex.SetStringIfConfigured(f.CustomDockerRunOptions, db.CustomDockerRunOptions)
	flex.SetInt64IfConfigured(f.PublicPortTimeout, db.PublicPortTimeout)
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
	return strNonDefault(f.LimitsMemory, "0") || strNonDefault(f.LimitsMemorySwap, "0") ||
		strNonDefault(f.LimitsMemoryReservation, "0") || strNonDefault(f.LimitsCPUs, "0") ||
		strSet(f.LimitsCPUSet) ||
		intNonDefault(f.LimitsMemorySwappiness, 60) || intNonDefault(f.LimitsCPUShares, 1024) ||
		strSet(f.PortsMappings) || strSet(f.CustomDockerRunOptions) ||
		intSet(f.PublicPortTimeout)
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
