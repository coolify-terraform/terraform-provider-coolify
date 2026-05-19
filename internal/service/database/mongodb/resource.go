package mongodb

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
	_ resource.Resource                = &res{}
	_ resource.ResourceWithConfigure   = &res{}
	_ resource.ResourceWithImportState = &res{}
)

type res struct{ client *client.Client }
type model struct {
	dbcommon.CommonModel
	// Type-specific
	MongoInitdbRootUsername types.String `tfsdk:"mongo_initdb_root_username"`
	MongoInitdbRootPassword types.String `tfsdk:"mongo_initdb_root_password"`
	MongoInitdbDatabase     types.String `tfsdk:"mongo_initdb_database"`
	MongoConf               types.String `tfsdk:"mongo_conf"`
	EnableSSL               types.Bool   `tfsdk:"enable_ssl"`
	SSLMode                 types.String `tfsdk:"ssl_mode"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_mongodb"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a MongoDB database resource on Coolify.", Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"mongo_initdb_root_username": schema.StringAttribute{MarkdownDescription: "The MongoDB root username (maps to `MONGO_INITDB_ROOT_USERNAME`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_initdb_root_password": schema.StringAttribute{MarkdownDescription: "The MongoDB root password (maps to `MONGO_INITDB_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_initdb_database":      schema.StringAttribute{MarkdownDescription: "The initial database name (maps to `MONGO_INITDB_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mongo_conf":                 schema.StringAttribute{MarkdownDescription: "Custom MongoDB configuration (base64-encoded `mongod.conf` content).", Optional: true},
		"enable_ssl":                 dbcommon.EnableSSLAttr(),
		"ssl_mode":                   dbcommon.SSLModeMongodbAttr(),
	})}
}
func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = dbcommon.ConfigureDatabase(req, resp)
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_mongodb"})

	in := client.CreateMongodbInput{ServerUUID: p.ServerUUID.ValueString(), ProjectUUID: p.ProjectUUID.ValueString(), EnvironmentName: p.EnvironmentName.ValueString()}
	flex.SetIfKnown(&in.Name, p.Name)
	flex.SetIfKnown(&in.Description, p.Description)
	flex.SetIfKnown(&in.Image, p.Image)
	flex.SetIfKnown(&in.MongoInitdbRootUsername, p.MongoInitdbRootUsername)
	flex.SetIfKnown(&in.MongoInitdbRootPassword, p.MongoInitdbRootPassword)
	flex.SetIfKnown(&in.MongoInitdbDatabase, p.MongoInitdbDatabase)
	in.IsPublic = flex.BoolValueOrNull(p.IsPublic)
	in.PublicPort = flex.Int64PtrFromFramework(p.PublicPort)
	in.InstantDeploy = flex.BoolValueOrNull(p.InstantDeploy)
	c, err := r.client.CreateDatabase(ctx, "mongodb", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MongoDB database",
			fmt.Sprintf("project %s, server %s: %s", p.ProjectUUID.ValueString(), p.ServerUUID.ValueString(), err))
		return
	}

	p.UUID = types.StringValue(c.UUID)
	dbcommon.NormalizeCommonCreateState(&p.CommonModel)
	flex.NormalizeUnknownString(&p.MongoInitdbRootUsername)
	flex.NormalizeUnknownString(&p.MongoInitdbRootPassword)
	flex.NormalizeUnknownString(&p.MongoInitdbDatabase)
	flex.NormalizeUnknownString(&p.MongoConf)
	flex.NormalizeUnknownString(&p.SSLMode)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := p.ExtFields().WithSSL(&p.EnableSSL, &p.SSLMode)
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if dbcommon.HasExtendedFields(ext) || strSet(p.MongoConf) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MongoConf, p.MongoConf)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MongoDB database extended fields", fmt.Sprintf("MongoDB database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "MongoDB database", c.UUID, err)
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_mongodb", "uuid": c.UUID})
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_database_mongodb", "uuid": s.UUID.ValueString()})

	db, err := dbcommon.ReadDatabase(ctx, r.client, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MongoDB database", fmt.Sprintf("MongoDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
	if db == nil {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_database_mongodb", "uuid": s.UUID.ValueString()})
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_mongodb", "uuid": s.UUID.ValueString()})

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
	dbcommon.SetUpdateExtendedDiff(&u, p.ExtFields().WithSSL(&p.EnableSSL, &p.SSLMode), s.ExtFields().WithSSL(&s.EnableSSL, &s.SSLMode))
	db, err := dbcommon.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
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
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_database_mongodb", "uuid": s.UUID.ValueString()})

	if err := dbcommon.DeleteDatabase(ctx, r.client, "coolify_database_mongodb", s.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MongoDB database", fmt.Sprintf("MongoDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields().WithSSL(&m.EnableSSL, &m.SSLMode))
	m.MongoInitdbRootUsername = flex.StringToFramework(db.MongoInitdbRootUsername)
	// Preserve password from plan/state when the API hides sensitive fields.
	if db.MongoInitdbRootPassword != "" {
		m.MongoInitdbRootPassword = types.StringValue(db.MongoInitdbRootPassword)
	} else if m.MongoInitdbRootPassword.IsUnknown() {
		m.MongoInitdbRootPassword = types.StringNull()
	}
	m.MongoInitdbDatabase = flex.StringToFramework(db.MongoInitdbDatabase)
	flex.SetStringOrClear(&m.MongoConf, db.MongoConf)
}
