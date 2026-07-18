package clickhouse

import (
	"context"
	"fmt"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	dbcommon "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database"
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
	ClickhouseAdminUser     types.String `tfsdk:"clickhouse_admin_user"`
	ClickhouseAdminPassword types.String `tfsdk:"clickhouse_admin_password"`
	ClickhouseDB            types.String `tfsdk:"clickhouse_db"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_clickhouse"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a ClickHouse database resource on Coolify.", Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"clickhouse_admin_user":     schema.StringAttribute{MarkdownDescription: "The ClickHouse admin user name (maps to `CLICKHOUSE_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"clickhouse_admin_password": schema.StringAttribute{MarkdownDescription: "The ClickHouse admin password (maps to `CLICKHOUSE_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"clickhouse_db":             schema.StringAttribute{MarkdownDescription: "The default ClickHouse database name. If omitted, Coolify uses `default`.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
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

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_clickhouse"})

	createTimeout, diags := p.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	var in client.CreateClickhouseInput
	dbcommon.PopulateBaseCreateInput(&in.CreateDatabaseBaseInput, &p.CommonModel)
	flex.SetIfKnown(&in.ClickhouseAdminUser, p.ClickhouseAdminUser)
	flex.SetIfKnown(&in.ClickhouseAdminPassword, p.ClickhouseAdminPassword)
	c, err := r.client.CreateDatabase(ctx, "clickhouse", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating ClickHouse database",
			fmt.Sprintf("project %s, server %s: %s", p.ProjectUUID.ValueString(), p.ServerUUID.ValueString(), err))
		return
	}

	p.UUID = types.StringValue(c.UUID)
	dbcommon.NormalizeCommonCreateState(&p.CommonModel)
	flex.NormalizeUnknownString(&p.ClickhouseAdminUser)
	flex.NormalizeUnknownString(&p.ClickhouseAdminPassword)
	flex.NormalizeUnknownString(&p.ClickhouseDB)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := p.ExtFields()
	if dbcommon.HasExtendedFields(ext) || flex.StringValueConfigured(p.ClickhouseDB) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.ClickhouseDB, p.ClickhouseDB)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting ClickHouse database extended fields", fmt.Sprintf("ClickHouse database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "ClickHouse database", c.UUID, err)
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_clickhouse", "uuid": c.UUID})
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}

	dbcommon.ReadDatabaseState(ctx, r.client, "coolify_database_clickhouse", s.UUID.ValueString(), resp, func(db *client.Database) {
		flattenDatabase(db, &s)
		resp.Diagnostics.Append(resp.State.Set(ctx, &s)...)
	})
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

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_clickhouse", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{
		Name:                    flex.StringIfChanged(p.Name, s.Name),
		Description:             flex.StringIfChanged(p.Description, s.Description),
		Image:                   flex.StringIfChanged(p.Image, s.Image),
		IsPublic:                flex.BoolIfChanged(p.IsPublic, s.IsPublic),
		PublicPort:              flex.Int64IfChanged(p.PublicPort, s.PublicPort),
		ClickhouseAdminUser:     flex.StringIfChanged(p.ClickhouseAdminUser, s.ClickhouseAdminUser),
		ClickhouseAdminPassword: flex.StringIfChanged(p.ClickhouseAdminPassword, s.ClickhouseAdminPassword),
		ClickhouseDB:            flex.StringIfChanged(p.ClickhouseDB, s.ClickhouseDB),
	}
	dbcommon.SetUpdateExtendedDiff(&u, p.ExtFields(), s.ExtFields())
	db, err := dbcommon.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating ClickHouse database", fmt.Sprintf("ClickHouse database %s: %s", s.UUID.ValueString(), err))
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

	dbcommon.DeleteDatabaseState(ctx, r.client, "coolify_database_clickhouse", s.UUID.ValueString(), resp)
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, r.client, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields())
	m.ClickhouseAdminUser = flex.StringToFramework(db.ClickhouseAdminUser)
	// Preserve password from plan/state when the API hides sensitive fields.
	if db.ClickhouseAdminPassword != "" {
		m.ClickhouseAdminPassword = types.StringValue(db.ClickhouseAdminPassword)
	} else if m.ClickhouseAdminPassword.IsUnknown() {
		m.ClickhouseAdminPassword = types.StringNull()
	}
	m.ClickhouseDB = flex.StringToFramework(db.ClickhouseDB)
}
