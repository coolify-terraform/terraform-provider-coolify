package clickhouse

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	pg "github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &res{}
	_ resource.ResourceWithConfigure   = &res{}
	_ resource.ResourceWithImportState = &res{}
)

type res struct{ client *client.Client }
type model struct {
	UUID                    types.String `tfsdk:"uuid"`
	Name                    types.String `tfsdk:"name"`
	Description             types.String `tfsdk:"description"`
	ProjectUUID             types.String `tfsdk:"project_uuid"`
	ServerUUID              types.String `tfsdk:"server_uuid"`
	EnvironmentName         types.String `tfsdk:"environment_name"`
	Image                   types.String `tfsdk:"image"`
	IsPublic                types.Bool   `tfsdk:"is_public"`
	PublicPort              types.Int64  `tfsdk:"public_port"`
	ClickhouseAdminUser     types.String `tfsdk:"clickhouse_admin_user"`
	ClickhouseAdminPassword types.String `tfsdk:"clickhouse_admin_password"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_clickhouse_database"
}
func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a ClickHouse database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(map[string]schema.Attribute{
		"clickhouse_admin_user":     schema.StringAttribute{MarkdownDescription: "The ClickHouse admin user.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"clickhouse_admin_password": schema.StringAttribute{MarkdownDescription: "The ClickHouse admin password.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
	})}
}
func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", fmt.Sprintf("got %T", req.ProviderData))
		return
	}
	r.client = c
}
func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	in := client.CreateClickhouseInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	pg.SetIfKnown(&in.Name, p.Name)
	pg.SetIfKnown(&in.Description, p.Description)
	pg.SetIfKnown(&in.Image, p.Image)
	pg.SetIfKnown(&in.ClickhouseAdminUser, p.ClickhouseAdminUser)
	pg.SetIfKnown(&in.ClickhouseAdminPassword, p.ClickhouseAdminPassword)
	c, err := r.client.CreateClickhouseDatabase(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ClickHouse database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading ClickHouse database", err.Error())
		return
	}
	toModel(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	db, err := r.client.GetDatabase(ctx, s.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading ClickHouse database", err.Error())
		return
	}
	toModel(db, &s)
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
	u := client.UpdateDatabaseInput{}
	pg.SetStrPtr(&u.Name, p.Name)
	pg.SetStrPtr(&u.Description, p.Description)
	pg.SetStrPtr(&u.Image, p.Image)
	pg.SetBoolPtr(&u.IsPublic, p.IsPublic)
	pg.SetInt64Ptr(&u.PublicPort, p.PublicPort)
	pg.SetStrPtr(&u.ClickhouseAdminUser, p.ClickhouseAdminUser)
	pg.SetStrPtr(&u.ClickhouseAdminPassword, p.ClickhouseAdminPassword)
	if _, err := r.client.UpdateDatabase(ctx, s.UUID.ValueString(), u); err != nil {
		resp.Diagnostics.AddError("Error updating ClickHouse database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading ClickHouse database", err.Error())
		return
	}
	toModel(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
}
func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.DeleteDatabase(ctx, s.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting ClickHouse database", err.Error())
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
func toModel(db *client.Database, m *model) {
	m.UUID = types.StringValue(db.UUID)
	m.Name = types.StringValue(db.Name)
	m.Image = pg.StringOrNull(db.Image)
	m.IsPublic = types.BoolValue(db.IsPublic)
	m.PublicPort = pg.Int64PtrToFW(db.PublicPort)
	m.ClickhouseAdminUser = pg.StringOrNull(db.ClickhouseAdminUser)
	m.ClickhouseAdminPassword = pg.StringOrNull(db.ClickhouseAdminPassword)
	m.Description = pg.StringOrNull(db.Description)
	m.ProjectUUID = pg.StringOrNull(db.ProjectUUID)
	m.ServerUUID = pg.StringOrNull(db.ServerUUID)
	m.EnvironmentName = pg.StringOrNull(db.EnvironmentName)
}
