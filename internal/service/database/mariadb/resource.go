package mariadb

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
	MariadbUser         types.String `tfsdk:"mariadb_user"`
	MariadbPassword     types.String `tfsdk:"mariadb_password"`
	MariadbDatabase     types.String `tfsdk:"mariadb_database"`
	MariadbRootPassword types.String `tfsdk:"mariadb_root_password"`
	MariadbConf         types.String `tfsdk:"mariadb_conf"`
	EnableSSL           types.Bool   `tfsdk:"enable_ssl"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_mariadb"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a MariaDB database resource on Coolify.", Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"mariadb_user":          schema.StringAttribute{MarkdownDescription: "The MariaDB user name (maps to `MARIADB_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_password":      schema.StringAttribute{MarkdownDescription: "The MariaDB user password (maps to `MARIADB_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_database":      schema.StringAttribute{MarkdownDescription: "The default database name (maps to `MARIADB_DATABASE`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_root_password": schema.StringAttribute{MarkdownDescription: "The MariaDB root password (maps to `MARIADB_ROOT_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"mariadb_conf":          schema.StringAttribute{MarkdownDescription: "Custom MariaDB configuration (base64-encoded `my.cnf` content).", Optional: true},
		"enable_ssl":            dbcommon.EnableSSLAttr(),
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_mariadb"})

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
	in.InstantDeploy = flex.BoolValueOrNull(p.InstantDeploy)
	c, err := r.client.CreateDatabase(ctx, "mariadb", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating MariaDB database",
			fmt.Sprintf("project %s, server %s: %s", p.ProjectUUID.ValueString(), p.ServerUUID.ValueString(), err))
		return
	}

	p.UUID = types.StringValue(c.UUID)
	dbcommon.NormalizeCommonCreateState(&p.CommonModel)
	flex.NormalizeUnknownString(&p.MariadbUser)
	flex.NormalizeUnknownString(&p.MariadbPassword)
	flex.NormalizeUnknownString(&p.MariadbDatabase)
	flex.NormalizeUnknownString(&p.MariadbRootPassword)
	flex.NormalizeUnknownString(&p.MariadbConf)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := p.ExtFields().WithSSL(&p.EnableSSL, nil)
	strSet := func(v types.String) bool { return !v.IsNull() && !v.IsUnknown() }
	if dbcommon.HasExtendedFields(ext) || strSet(p.MariadbConf) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.MariadbConf, p.MariadbConf)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting MariaDB database extended fields", fmt.Sprintf("MariaDB database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "MariaDB database", c.UUID, err)
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_mariadb", "uuid": c.UUID})
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_database_mariadb", "uuid": s.UUID.ValueString()})

	db, err := dbcommon.ReadDatabase(ctx, r.client, s.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading MariaDB database", fmt.Sprintf("MariaDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
	if db == nil {
		tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_database_mariadb", "uuid": s.UUID.ValueString()})
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_mariadb", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{
		Name:                flex.StringIfChanged(p.Name, s.Name),
		Description:         flex.StringIfChanged(p.Description, s.Description),
		Image:               flex.StringIfChanged(p.Image, s.Image),
		IsPublic:            flex.BoolIfChanged(p.IsPublic, s.IsPublic),
		PublicPort:          flex.Int64IfChanged(p.PublicPort, s.PublicPort),
		MariadbUser:         flex.StringIfChanged(p.MariadbUser, s.MariadbUser),
		MariadbPassword:     flex.StringIfChanged(p.MariadbPassword, s.MariadbPassword),
		MariadbDatabase:     flex.StringIfChanged(p.MariadbDatabase, s.MariadbDatabase),
		MariadbRootPassword: flex.StringIfChanged(p.MariadbRootPassword, s.MariadbRootPassword),
		MariadbConf:         flex.StringIfChanged(p.MariadbConf, s.MariadbConf),
	}
	dbcommon.SetUpdateExtendedDiff(&u, p.ExtFields().WithSSL(&p.EnableSSL, nil), s.ExtFields().WithSSL(&s.EnableSSL, nil))
	db, err := dbcommon.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
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
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_database_mariadb", "uuid": s.UUID.ValueString()})

	if err := dbcommon.DeleteDatabase(ctx, r.client, "coolify_database_mariadb", s.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting MariaDB database", fmt.Sprintf("MariaDB database %s: %s", s.UUID.ValueString(), err))
		return
	}
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields().WithSSL(&m.EnableSSL, nil))
	m.MariadbUser = flex.StringToFramework(db.MariadbUser)
	// Preserve passwords from plan/state when the API hides sensitive fields.
	if db.MariadbPassword != "" {
		m.MariadbPassword = types.StringValue(db.MariadbPassword)
	} else if m.MariadbPassword.IsUnknown() {
		m.MariadbPassword = types.StringNull()
	}
	m.MariadbDatabase = flex.StringToFramework(db.MariadbDatabase)
	if db.MariadbRootPassword != "" {
		m.MariadbRootPassword = types.StringValue(db.MariadbRootPassword)
	} else if m.MariadbRootPassword.IsUnknown() {
		m.MariadbRootPassword = types.StringNull()
	}
	flex.SetStringOrClear(&m.MariadbConf, db.MariadbConf)
}
