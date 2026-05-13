package mariadb

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	pg "github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &res{}
	_ resource.ResourceWithConfigure   = &res{}
	_ resource.ResourceWithImportState = &res{}
)

type res struct{ client *client.Client }
type model struct {
	Timeouts        timeouts.Value `tfsdk:"timeouts"`
	UUID            types.String   `tfsdk:"uuid"`
	Name            types.String   `tfsdk:"name"`
	Description     types.String   `tfsdk:"description"`
	ProjectUUID     types.String   `tfsdk:"project_uuid"`
	ServerUUID      types.String   `tfsdk:"server_uuid"`
	EnvironmentName types.String   `tfsdk:"environment_name"`
	Image           types.String   `tfsdk:"image"`
	IsPublic        types.Bool     `tfsdk:"is_public"`
	PublicPort      types.Int64    `tfsdk:"public_port"`
	// Shared extended fields
	LimitsMemory            types.String `tfsdk:"limits_memory"`
	LimitsMemorySwap        types.String `tfsdk:"limits_memory_swap"`
	LimitsMemorySwappiness  types.Int64  `tfsdk:"limits_memory_swappiness"`
	LimitsMemoryReservation types.String `tfsdk:"limits_memory_reservation"`
	LimitsCPUs              types.String `tfsdk:"limits_cpus"`
	LimitsCPUSet            types.String `tfsdk:"limits_cpuset"`
	LimitsCPUShares         types.Int64  `tfsdk:"limits_cpu_shares"`
	PortsMappings           types.String `tfsdk:"ports_mappings"`
	CustomDockerRunOptions  types.String `tfsdk:"custom_docker_run_options"`
	PublicPortTimeout       types.Int64  `tfsdk:"public_port_timeout"`
	Status                  types.String `tfsdk:"status"`
	// Type-specific
	MariadbUser         types.String `tfsdk:"mariadb_user"`
	MariadbPassword     types.String `tfsdk:"mariadb_password"`
	MariadbDatabase     types.String `tfsdk:"mariadb_database"`
	MariadbRootPassword types.String `tfsdk:"mariadb_root_password"`
	MariadbConf         types.String `tfsdk:"mariadb_conf"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mariadb_database"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a MariaDB database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"mariadb_user":          schema.StringAttribute{MarkdownDescription: "The MariaDB user name (maps to `MARIADB_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_password":      schema.StringAttribute{MarkdownDescription: "The MariaDB user password (maps to `MARIADB_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_database":      schema.StringAttribute{MarkdownDescription: "The default database name (maps to `MARIADB_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_root_password": schema.StringAttribute{MarkdownDescription: "The MariaDB root password (maps to `MARIADB_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_conf":          schema.StringAttribute{MarkdownDescription: "Custom MariaDB configuration (base64-encoded `my.cnf` content).", Optional: true},
	})}
}
func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = pg.ConfigureDatabase(req, resp)
}
func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := p.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_mariadb_database"})

	in := client.CreateMariadbInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	flex.SetIfKnown(&in.Name, p.Name)
	flex.SetIfKnown(&in.Description, p.Description)
	flex.SetIfKnown(&in.Image, p.Image)
	flex.SetIfKnown(&in.MariadbUser, p.MariadbUser)
	flex.SetIfKnown(&in.MariadbPassword, p.MariadbPassword)
	flex.SetIfKnown(&in.MariadbDatabase, p.MariadbDatabase)
	flex.SetIfKnown(&in.MariadbRootPassword, p.MariadbRootPassword)
	in.IsPublic = flex.BoolValueOrNull(p.IsPublic)
	in.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	c, err := r.client.CreateDatabase(ctx, "mariadb", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MariaDB database", err.Error())
		return
	}

	p.UUID = types.StringValue(c.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := extFields(&p)
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if pg.HasExtendedFields(ext) || strSet(p.MariadbConf) {
		update := client.UpdateDatabaseInput{}
		pg.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MariadbConf, p.MariadbConf)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MariaDB database extended fields", fmt.Sprintf("MariaDB database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MariaDB database", fmt.Sprintf("MariaDB database %s: %s", c.UUID, err))
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_mariadb_database", "uuid": s.UUID.ValueString()})

	db, err := pg.ReadDatabase(ctx, r.client, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MariaDB database", fmt.Sprintf("MariaDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
	if db == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	flattenDatabase(db, &s)
	resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
}
func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var p model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_mariadb_database", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{}
	flex.SetStrPtr(&u.Name, p.Name)
	flex.SetStrPtr(&u.Description, p.Description)
	flex.SetStrPtr(&u.Image, p.Image)
	flex.SetBoolPtr(&u.IsPublic, p.IsPublic)
	u.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	flex.SetStrPtr(&u.MariadbUser, p.MariadbUser)
	flex.SetStrPtr(&u.MariadbPassword, p.MariadbPassword)
	flex.SetStrPtr(&u.MariadbDatabase, p.MariadbDatabase)
	flex.SetStrPtr(&u.MariadbRootPassword, p.MariadbRootPassword)
	flex.SetStrPtr(&u.MariadbConf, p.MariadbConf)
	pg.SetUpdateExtended(&u, extFields(&p))
	db, err := pg.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MariaDB database", fmt.Sprintf("MariaDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_mariadb_database", "uuid": s.UUID.ValueString()})

	if err := pg.DeleteDatabase(ctx, r.client, s.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MariaDB database", fmt.Sprintf("MariaDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pg.ImportDatabaseState(ctx, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	pg.FlattenDatabaseCommon(db, &m.UUID, &m.Name, &m.Description, &m.Image, &m.ProjectUUID, &m.ServerUUID, &m.EnvironmentName, &m.IsPublic, &m.PublicPort)
	pg.FlattenDatabaseExtended(db, extFields(m))
	m.MariadbUser = flex.StringToFramework(db.MariadbUser)
	m.MariadbPassword = flex.StringToFramework(db.MariadbPassword)
	m.MariadbDatabase = flex.StringToFramework(db.MariadbDatabase)
	m.MariadbRootPassword = flex.StringToFramework(db.MariadbRootPassword)
	flex.SetStringIfConfigured(&m.MariadbConf, db.MariadbConf)
}

func extFields(m *model) pg.DatabaseExtendedPtrs {
	return pg.DatabaseExtendedPtrs{
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
