package mysql

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &mysqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &mysqlDatabaseResource{}
	_ resource.ResourceWithImportState = &mysqlDatabaseResource{}
)

type mysqlDatabaseResource struct{ client *client.Client }

type mysqlDatabaseResourceModel struct {
	UUID              types.String `tfsdk:"uuid"`
	Name              types.String `tfsdk:"name"`
	Description       types.String `tfsdk:"description"`
	ProjectUUID       types.String `tfsdk:"project_uuid"`
	ServerUUID        types.String `tfsdk:"server_uuid"`
	EnvironmentName   types.String `tfsdk:"environment_name"`
	Image             types.String `tfsdk:"image"`
	IsPublic          types.Bool   `tfsdk:"is_public"`
	PublicPort        types.Int64  `tfsdk:"public_port"`
	MysqlUser         types.String `tfsdk:"mysql_user"`
	MysqlPassword     types.String `tfsdk:"mysql_password"`
	MysqlDatabase     types.String `tfsdk:"mysql_database"`
	MysqlRootPassword types.String `tfsdk:"mysql_root_password"`
}

func NewResource() resource.Resource { return &mysqlDatabaseResource{} }

func (r *mysqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mysql_database"
}

func (r *mysqlDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a MySQL database resource on Coolify.",
		Attributes: postgresql.CommonDatabaseAttrs(map[string]schema.Attribute{
			"mysql_user":          schema.StringAttribute{MarkdownDescription: "The MySQL user.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_password":      schema.StringAttribute{MarkdownDescription: "The MySQL user password.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_database":      schema.StringAttribute{MarkdownDescription: "The MySQL database name.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
			"mysql_root_password": schema.StringAttribute{MarkdownDescription: "The MySQL root password.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		}),
	}
}

func (r *mysqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *mysqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan mysqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	input := client.CreateMysqlInput{ServerUUID: plan.ServerUUID.ValueString(), ProjectUUID: plan.ProjectUUID.ValueString(), EnvironmentName: plan.EnvironmentName.ValueString()}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.MysqlUser, plan.MysqlUser)
	flex.SetIfKnown(&input.MysqlPassword, plan.MysqlPassword)
	flex.SetIfKnown(&input.MysqlDatabase, plan.MysqlDatabase)
	flex.SetIfKnown(&input.MysqlRootPassword, plan.MysqlRootPassword)
	created, err := r.client.CreateMysqlDatabase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MySQL database", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MySQL database after creation", err.Error())
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
	db, err := r.client.GetDatabase(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading MySQL database", err.Error())
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
	if _, err := r.client.UpdateDatabase(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating MySQL database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading MySQL database after update", err.Error())
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
	if err := r.client.DeleteDatabase(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting MySQL database", err.Error())
		return
	}
}

func (r *mysqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func flattenDatabase(db *client.Database, m *mysqlDatabaseResourceModel) {
	m.UUID = types.StringValue(db.UUID)
	m.Name = types.StringValue(db.Name)
	m.Image = flex.StringToFramework(db.Image)
	m.IsPublic = types.BoolValue(db.IsPublic)
	m.PublicPort = flex.Int64PtrToFramework(db.PublicPort)
	m.MysqlUser = flex.StringToFramework(db.MysqlUser)
	m.MysqlPassword = flex.StringToFramework(db.MysqlPassword)
	m.MysqlDatabase = flex.StringToFramework(db.MysqlDatabase)
	m.MysqlRootPassword = flex.StringToFramework(db.MysqlRootPassword)
	m.Description = flex.StringToFramework(db.Description)
	m.ProjectUUID = flex.StringToFramework(db.ProjectUUID)
	m.ServerUUID = flex.StringToFramework(db.ServerUUID)
	m.EnvironmentName = flex.StringToFramework(db.EnvironmentName)
}
