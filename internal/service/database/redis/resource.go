package redis

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	pg "github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &res{}
	_ resource.ResourceWithConfigure   = &res{}
	_ resource.ResourceWithImportState = &res{}
)

type res struct{ client *client.Client }
type model struct {
	UUID            types.String `tfsdk:"uuid"`
	Name            types.String `tfsdk:"name"`
	Description     types.String `tfsdk:"description"`
	ProjectUUID     types.String `tfsdk:"project_uuid"`
	ServerUUID      types.String `tfsdk:"server_uuid"`
	EnvironmentName types.String `tfsdk:"environment_name"`
	Image           types.String `tfsdk:"image"`
	IsPublic        types.Bool   `tfsdk:"is_public"`
	PublicPort      types.Int64  `tfsdk:"public_port"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_database"
}
func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a Redis database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(nil)}
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
	in := client.CreateRedisInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	pg.SetIfKnown(&in.Name, p.Name)
	pg.SetIfKnown(&in.Description, p.Description)
	pg.SetIfKnown(&in.Image, p.Image)
	c, err := r.client.CreateRedisDatabase(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Redis database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redis database", err.Error())
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
		resp.Diagnostics.AddError("Error reading Redis database", err.Error())
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
	if _, err := r.client.UpdateDatabase(ctx, s.UUID.ValueString(), u); err != nil {
		resp.Diagnostics.AddError("Error updating Redis database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redis database", err.Error())
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
		resp.Diagnostics.AddError("Error deleting Redis database", err.Error())
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
	if db.Description != "" {
		m.Description = types.StringValue(db.Description)
	}
	if db.ProjectUUID != "" {
		m.ProjectUUID = types.StringValue(db.ProjectUUID)
	}
	if db.ServerUUID != "" {
		m.ServerUUID = types.StringValue(db.ServerUUID)
	}
	if db.EnvironmentName != "" {
		m.EnvironmentName = types.StringValue(db.EnvironmentName)
	}
}
