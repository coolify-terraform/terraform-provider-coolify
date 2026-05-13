package mysql

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	pg "github.com/SebTardifLabs/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &mysqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &mysqlDatabaseResource{}
	_ resource.ResourceWithImportState = &mysqlDatabaseResource{}
)

type mysqlDatabaseResource struct{ client *client.Client }

type mysqlDatabaseResourceModel struct {
	pg.CommonModel
	// Type-specific
	MysqlUser         types.String `tfsdk:"mysql_user"`
	MysqlPassword     types.String `tfsdk:"mysql_password"`
	MysqlDatabase     types.String `tfsdk:"mysql_database"`
	MysqlRootPassword types.String `tfsdk:"mysql_root_password"`
	MysqlConf         types.String `tfsdk:"mysql_conf"`
}

func NewResource() resource.Resource { return &mysqlDatabaseResource{} }

func (r *mysqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mysql_database"
}

func (r *mysqlDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a MySQL database resource on Coolify.",
		Attributes: pg.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
			"mysql_user":          schema.StringAttribute{MarkdownDescription: "The MySQL user name (maps to `MYSQL_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_password":      schema.StringAttribute{MarkdownDescription: "The MySQL user password (maps to `MYSQL_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_database":      schema.StringAttribute{MarkdownDescription: "The default database name (maps to `MYSQL_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_root_password": schema.StringAttribute{MarkdownDescription: "The MySQL root password (maps to `MYSQL_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_conf":          schema.StringAttribute{MarkdownDescription: "Custom MySQL configuration (base64-encoded `my.cnf` content).", Optional: true},
		}),
	}
}

func (r *mysqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = pg.ConfigureDatabase(req, resp)
}

func (r *mysqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mysqlDatabaseResourceModel
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_mysql_database"})

	input := client.CreateMysqlInput{ServerUUID: plan.ServerUUID.ValueString(), ProjectUUID: plan.ProjectUUID.ValueString(), EnvironmentName: plan.EnvironmentName.ValueString()}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.MysqlUser, plan.MysqlUser)
	flex.SetIfKnown(&input.MysqlPassword, plan.MysqlPassword)
	flex.SetIfKnown(&input.MysqlDatabase, plan.MysqlDatabase)
	flex.SetIfKnown(&input.MysqlRootPassword, plan.MysqlRootPassword)
	input.IsPublic = flex.BoolValueOrNull(plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)
	created, err := r.client.CreateDatabase(ctx, "mysql", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MySQL database", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := plan.ExtFields()
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if pg.HasExtendedFields(ext) || strSet(plan.MysqlConf) {
		update := client.UpdateDatabaseInput{}
		pg.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MysqlConf, plan.MysqlConf)
		if _, err := r.client.UpdateDatabase(ctx, created.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MySQL database extended fields", fmt.Sprintf("MySQL database %s: %s", created.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MySQL database after creation", fmt.Sprintf("MySQL database %s: %s", created.UUID, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mysqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_mysql_database", "uuid": state.UUID.ValueString()})

	db, err := pg.ReadDatabase(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MySQL database", fmt.Sprintf("MySQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
	if db == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	flattenDatabase(db, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *mysqlDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_mysql_database", "uuid": uuid})

	input := client.UpdateDatabaseInput{}
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.Description, plan.Description)
	flex.SetStrPtr(&input.Image, plan.Image)
	flex.SetBoolPtr(&input.IsPublic, plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)
	flex.SetStrPtr(&input.MysqlUser, plan.MysqlUser)
	flex.SetStrPtr(&input.MysqlPassword, plan.MysqlPassword)
	flex.SetStrPtr(&input.MysqlDatabase, plan.MysqlDatabase)
	flex.SetStrPtr(&input.MysqlRootPassword, plan.MysqlRootPassword)
	flex.SetStrPtr(&input.MysqlConf, plan.MysqlConf)
	pg.SetUpdateExtended(&input, plan.ExtFields())
	db, err := pg.UpdateDatabase(ctx, r.client, uuid, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MySQL database", fmt.Sprintf("MySQL database %s: %s", uuid, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *mysqlDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_mysql_database", "uuid": state.UUID.ValueString()})

	if err := pg.DeleteDatabase(ctx, r.client, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MySQL database", fmt.Sprintf("MySQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *mysqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pg.ImportDatabaseState(ctx, req, resp)
}

func flattenDatabase(db *client.Database, m *mysqlDatabaseResourceModel) {
	pg.FlattenDatabaseCommon(db, m.CommonPtrs())
	pg.FlattenDatabaseExtended(db, m.ExtFields())
	m.MysqlUser = flex.StringToFramework(db.MysqlUser)
	m.MysqlPassword = flex.StringToFramework(db.MysqlPassword)
	m.MysqlDatabase = flex.StringToFramework(db.MysqlDatabase)
	m.MysqlRootPassword = flex.StringToFramework(db.MysqlRootPassword)
	flex.SetStringIfConfigured(&m.MysqlConf, db.MysqlConf)
}
