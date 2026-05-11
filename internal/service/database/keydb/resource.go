package keydb

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	pg "github.com/SebTardif/terraform-provider-coolify/internal/service/database/postgresql"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
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
	resp.TypeName = req.ProviderTypeName + "_keydb_database"
}
func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a KeyDB database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(nil)}
}
func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
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
	in := client.CreateKeydbInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	flex.SetIfKnown(&in.Name, p.Name)
	flex.SetIfKnown(&in.Description, p.Description)
	flex.SetIfKnown(&in.Image, p.Image)
	in.IsPublic = flex.BoolValueOrNull(p.IsPublic)
	in.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	c, err := r.client.CreateKeydbDatabase(ctx, in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating KeyDB database", err.Error())
		return
	}

	p.UUID = types.StringValue(c.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading KeyDB database", err.Error())
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
	db, err := r.client.GetDatabase(ctx, s.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading KeyDB database", err.Error())
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
	u := client.UpdateDatabaseInput{}
	flex.SetStrPtr(&u.Name, p.Name)
	flex.SetStrPtr(&u.Description, p.Description)
	flex.SetStrPtr(&u.Image, p.Image)
	flex.SetBoolPtr(&u.IsPublic, p.IsPublic)
	u.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	if _, err := r.client.UpdateDatabase(ctx, s.UUID.ValueString(), u); err != nil {
		resp.Diagnostics.AddError("Error updating KeyDB database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading KeyDB database", err.Error())
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
	if err := r.client.DeleteDatabase(ctx, s.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting KeyDB database", err.Error())
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	m.UUID = types.StringValue(db.UUID)
	m.Name = types.StringValue(db.Name)
	m.Image = flex.StringToFramework(db.Image)
	m.IsPublic = types.BoolValue(db.IsPublic)
	m.PublicPort = flex.Int64PtrToFramework(db.PublicPort)
	m.Description = flex.StringToFramework(db.Description)
	m.ProjectUUID = types.StringValue(db.ProjectUUID)
	m.ServerUUID = types.StringValue(db.ServerUUID)
	m.EnvironmentName = flex.StringToFramework(db.EnvironmentName)
}
