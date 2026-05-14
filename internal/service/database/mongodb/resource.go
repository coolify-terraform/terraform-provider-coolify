package mongodb

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
	_ resource.Resource                = &res{}
	_ resource.ResourceWithConfigure   = &res{}
	_ resource.ResourceWithImportState = &res{}
)

type res struct{ client *client.Client }
type model struct {
	pg.CommonModel
	// Type-specific
	MongoInitdbRootUsername types.String `tfsdk:"mongo_initdb_root_username"`
	MongoInitdbRootPassword types.String `tfsdk:"mongo_initdb_root_password"`
	MongoInitdbDatabase     types.String `tfsdk:"mongo_initdb_database"`
	MongoConf               types.String `tfsdk:"mongo_conf"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_mongodb_database"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a MongoDB database resource on Coolify.", Attributes: pg.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"mongo_initdb_root_username": schema.StringAttribute{MarkdownDescription: "The MongoDB root username (maps to `MONGO_INITDB_ROOT_USERNAME`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_initdb_root_password": schema.StringAttribute{MarkdownDescription: "The MongoDB root password (maps to `MONGO_INITDB_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_initdb_database":      schema.StringAttribute{MarkdownDescription: "The initial database name (maps to `MONGO_INITDB_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_conf":                 schema.StringAttribute{MarkdownDescription: "Custom MongoDB configuration (base64-encoded `mongod.conf` content).", Optional: true},
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_mongodb_database"})

	in := client.CreateMongodbInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	flex.SetIfKnown(&in.Name, p.Name)
	flex.SetIfKnown(&in.Description, p.Description)
	flex.SetIfKnown(&in.Image, p.Image)
	flex.SetIfKnown(&in.MongoInitdbRootUsername, p.MongoInitdbRootUsername)
	flex.SetIfKnown(&in.MongoInitdbRootPassword, p.MongoInitdbRootPassword)
	flex.SetIfKnown(&in.MongoInitdbDatabase, p.MongoInitdbDatabase)
	in.IsPublic = flex.BoolValueOrNull(p.IsPublic)
	in.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	c, err := r.client.CreateDatabase(ctx, "mongodb", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MongoDB database", err.Error())
		return
	}

	p.UUID = types.StringValue(c.UUID)
	pg.NormalizeCommonCreateState(&p.CommonModel)
	pg.NormalizeUnknownString(&p.MongoInitdbRootUsername)
	pg.NormalizeUnknownString(&p.MongoInitdbRootPassword)
	pg.NormalizeUnknownString(&p.MongoInitdbDatabase)
	pg.NormalizeUnknownString(&p.MongoConf)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := p.ExtFields()
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if pg.HasExtendedFields(ext) || strSet(p.MongoConf) {
		update := client.UpdateDatabaseInput{}
		pg.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MongoConf, p.MongoConf)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MongoDB database extended fields", fmt.Sprintf("MongoDB database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		pg.AddCreateReadBackError(resp, "MongoDB database", c.UUID, err)
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
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_mongodb_database", "uuid": s.UUID.ValueString()})

	db, err := pg.ReadDatabase(ctx, r.client, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MongoDB database", fmt.Sprintf("MongoDB database %s: %s", s.UUID.ValueString(), err))
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_mongodb_database", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{
		Name:                    flex.StringIfChanged(p.Name, s.Name),
		Description:             flex.StringIfChanged(p.Description, s.Description),
		Image:                   flex.StringIfChanged(p.Image, s.Image),
		IsPublic:                flex.BoolIfChanged(p.IsPublic, s.IsPublic),
		PublicPort:              flex.Int64IfChanged(p.PublicPort, s.PublicPort),
		MongoInitdbRootUsername: flex.StringIfChanged(p.MongoInitdbRootUsername, s.MongoInitdbRootUsername),
		MongoInitdbRootPassword: flex.StringIfChanged(p.MongoInitdbRootPassword, s.MongoInitdbRootPassword),
		MongoInitdbDatabase:     flex.StringIfChanged(p.MongoInitdbDatabase, s.MongoInitdbDatabase),
		MongoConf:               flex.StringIfChanged(p.MongoConf, s.MongoConf),
	}
	pg.SetUpdateExtendedDiff(&u, p.ExtFields(), s.ExtFields())
	db, err := pg.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating MongoDB database", fmt.Sprintf("MongoDB database %s: %s", s.UUID.ValueString(), err))
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
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_mongodb_database", "uuid": s.UUID.ValueString()})

	if err := pg.DeleteDatabase(ctx, r.client, s.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MongoDB database", fmt.Sprintf("MongoDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	pg.ImportDatabaseState(ctx, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	pg.FlattenDatabaseCommon(db, m.CommonPtrs())
	pg.FlattenDatabaseExtended(db, m.ExtFields())
	m.MongoInitdbRootUsername = flex.StringToFramework(db.MongoInitdbRootUsername)
	m.MongoInitdbRootPassword = flex.StringToFramework(db.MongoInitdbRootPassword)
	m.MongoInitdbDatabase = flex.StringToFramework(db.MongoInitdbDatabase)
	flex.SetStringIfConfigured(&m.MongoConf, db.MongoConf)
}
