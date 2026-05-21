package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	dbcommon "github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &postgresqlDatabaseResource{}
	_ resource.ResourceWithImportState = &postgresqlDatabaseResource{}
)

type postgresqlDatabaseResource struct{ client *client.Client }
type postgresqlDatabaseResourceModel struct {
	dbcommon.CommonModel
	// Type-specific
	PostgresUser           types.String `tfsdk:"postgres_user"`
	PostgresPassword       types.String `tfsdk:"postgres_password"`
	PostgresDB             types.String `tfsdk:"postgres_db"`
	PostgresConf           types.String `tfsdk:"postgres_conf"`
	PostgresInitdbArgs     types.String `tfsdk:"postgres_initdb_args"`
	PostgresHostAuthMethod types.String `tfsdk:"postgres_host_auth_method"`
	InitScripts            types.String `tfsdk:"init_scripts"`
	EnableSSL              types.Bool   `tfsdk:"enable_ssl"`
	SSLMode                types.String `tfsdk:"ssl_mode"`
}

func NewResource() resource.Resource { return &postgresqlDatabaseResource{} }

func (r *postgresqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_postgresql"
}

func (r *postgresqlDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a PostgreSQL database resource on Coolify.",
		Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
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
				MarkdownDescription: "Host authentication method (maps to `POSTGRES_HOST_AUTH_METHOD`, e.g., `trust`, `scram-sha-256`).",
				Optional:            true,
			},
			"init_scripts": schema.StringAttribute{
				MarkdownDescription: "Initialization scripts as a JSON array.",
				Optional:            true,
			},
			"enable_ssl": dbcommon.EnableSSLAttr(),
			"ssl_mode":   dbcommon.SSLModePostgresqlAttr(),
		}),
	}
}

func (r *postgresqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = dbcommon.ConfigureDatabase(req, resp)
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_postgresql"})

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
	input.InstantDeploy = flex.BoolValueOrNull(plan.InstantDeploy)

	created, err := r.client.CreateDatabase(ctx, "postgresql", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PostgreSQL database",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	dbcommon.NormalizeCommonCreateState(&plan.CommonModel)
	flex.NormalizeUnknownString(&plan.PostgresUser)
	flex.NormalizeUnknownString(&plan.PostgresPassword)
	flex.NormalizeUnknownString(&plan.PostgresDB)
	flex.NormalizeUnknownString(&plan.PostgresConf)
	flex.NormalizeUnknownString(&plan.PostgresInitdbArgs)
	flex.NormalizeUnknownString(&plan.PostgresHostAuthMethod)
	flex.NormalizeUnknownString(&plan.InitScripts)
	flex.NormalizeUnknownString(&plan.SSLMode)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Apply extended fields that cannot be set during creation.
	ext := plan.ExtFields().WithSSL(&plan.EnableSSL, &plan.SSLMode)
	needsUpdate := dbcommon.HasExtendedFields(ext) || flex.StringValueConfigured(plan.PostgresConf) || flex.StringValueConfigured(plan.PostgresInitdbArgs) || flex.StringValueConfigured(plan.PostgresHostAuthMethod) || flex.StringValueConfigured(plan.InitScripts)
	if needsUpdate {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
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
		dbcommon.AddCreateReadBackError(resp, "PostgreSQL database", created.UUID, err)
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_postgresql", "uuid": created.UUID})
}

func (r *postgresqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dbcommon.ReadDatabaseState(ctx, r.client, "coolify_database_postgresql", state.UUID.ValueString(), resp, func(db *client.Database) {
		flattenDatabase(db, &state)
		resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
	})
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_postgresql", "uuid": uuid})

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
	dbcommon.SetUpdateExtendedDiff(&input, plan.ExtFields().WithSSL(&plan.EnableSSL, &plan.SSLMode), state.ExtFields().WithSSL(&state.EnableSSL, &state.SSLMode))
	db, err := dbcommon.UpdateDatabase(ctx, r.client, uuid, input)
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
	dbcommon.DeleteDatabaseState(ctx, r.client, "coolify_database_postgresql", state.UUID.ValueString(), resp)
}

func (r *postgresqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, req, resp)
}

func flattenDatabase(db *client.Database, m *postgresqlDatabaseResourceModel) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields().WithSSL(&m.EnableSSL, &m.SSLMode))
	m.PostgresUser = flex.StringToFramework(db.PostgresUser)
	// Preserve password from plan/state when the API hides sensitive fields.
	if db.PostgresPassword != "" {
		m.PostgresPassword = types.StringValue(db.PostgresPassword)
	} else if m.PostgresPassword.IsUnknown() {
		m.PostgresPassword = types.StringNull()
	}
	m.PostgresDB = flex.StringToFramework(db.PostgresDB)
	flex.SetStringOrClear(&m.PostgresConf, db.PostgresConf)
	flex.SetStringOrClear(&m.PostgresInitdbArgs, db.PostgresInitdbArgs)
	flex.SetStringOrClear(&m.PostgresHostAuthMethod, db.PostgresHostAuthMethod)
	if len(db.InitScripts) > 0 {
		flex.SetStringOrClear(&m.InitScripts, string(db.InitScripts))
	}
}
