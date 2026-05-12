package redis

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
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_redis_database"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a Redis database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(ctx, nil)}
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_redis_database"})
	in := client.CreateRedisInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	flex.SetIfKnown(&in.Name, p.Name)
	flex.SetIfKnown(&in.Description, p.Description)
	flex.SetIfKnown(&in.Image, p.Image)
	in.IsPublic = flex.BoolValueOrNull(p.IsPublic)
	in.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	c, err := r.client.CreateDatabase(ctx, "redis", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating Redis database", err.Error())
		return
	}

	p.UUID = types.StringValue(c.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redis database", fmt.Sprintf("Redis database %s: %s", c.UUID, err))
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
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_redis_database", "uuid": s.UUID.ValueString()})

	db, err := pg.ReadDatabase(ctx, r.client, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading Redis database", fmt.Sprintf("Redis database %s: %s", s.UUID.ValueString(), err))
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_redis_database", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{}
	flex.SetStrPtr(&u.Name, p.Name)
	flex.SetStrPtr(&u.Description, p.Description)
	flex.SetStrPtr(&u.Image, p.Image)
	flex.SetBoolPtr(&u.IsPublic, p.IsPublic)
	u.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	db, err := pg.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating Redis database", fmt.Sprintf("Redis database %s: %s", s.UUID.ValueString(), err))
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
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_redis_database", "uuid": s.UUID.ValueString()})

	if err := pg.DeleteDatabase(ctx, r.client, s.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting Redis database", fmt.Sprintf("Redis database %s: %s", s.UUID.ValueString(), err))
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pg.ImportDatabaseState(ctx, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	pg.FlattenDatabaseCommon(db, &m.UUID, &m.Name, &m.Description, &m.Image, &m.ProjectUUID, &m.ServerUUID, &m.EnvironmentName, &m.IsPublic, &m.PublicPort)
}
