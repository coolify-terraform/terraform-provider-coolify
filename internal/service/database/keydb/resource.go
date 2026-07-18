package keydb

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
	KeydbPassword types.String `tfsdk:"keydb_password"`
	KeydbConf     types.String `tfsdk:"keydb_conf"`
	EnableSSL     types.Bool   `tfsdk:"enable_ssl"`
}

func NewResource() resource.Resource { return &res{} }
func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_keydb"
}
func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{MarkdownDescription: "Manages a KeyDB database resource on Coolify.", Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
		"keydb_password": schema.StringAttribute{MarkdownDescription: "The KeyDB password. If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"keydb_conf":     schema.StringAttribute{MarkdownDescription: "Custom KeyDB configuration (base64-encoded `keydb.conf` content).", Optional: true},
		"enable_ssl":     dbcommon.EnableSSLAttr(),
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
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_keydb"})

	var in client.CreateKeydbInput
	dbcommon.PopulateBaseCreateInput(&in.CreateDatabaseBaseInput, &p.CommonModel)
	flex.SetIfKnown(&in.KeydbPassword, p.KeydbPassword)
	c, err := r.client.CreateDatabase(ctx, "keydb", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating KeyDB database",
			fmt.Sprintf("project %s, server %s: %s", p.ProjectUUID.ValueString(), p.ServerUUID.ValueString(), err))
		return
	}

	p.UUID = types.StringValue(c.UUID)
	dbcommon.NormalizeCommonCreateState(&p.CommonModel)
	flex.NormalizeUnknownString(&p.KeydbPassword)
	flex.NormalizeUnknownString(&p.KeydbConf)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	ext := p.ExtFields().WithSSL(&p.EnableSSL, nil)
	if dbcommon.HasExtendedFields(ext) || flex.StringValueConfigured(p.KeydbConf) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, ext)
		flex.SetStrPtr(&update.KeydbPassword, p.KeydbPassword)
		flex.SetStrPtr(&update.KeydbConf, p.KeydbConf)
		if _, err := r.client.UpdateDatabase(ctx, c.UUID, update); err != nil {
			resp.Diagnostics.AddError("Error setting KeyDB database extended fields", fmt.Sprintf("KeyDB database %s: %s", c.UUID, err))
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, c.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "KeyDB database", c.UUID, err)
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_keydb", "uuid": c.UUID})
}
func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dbcommon.ReadDatabaseState(ctx, r.client, "coolify_database_keydb", s.UUID.ValueString(), resp, func(db *client.Database) {
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
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_keydb", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{
		Name:          flex.StringIfChanged(p.Name, s.Name),
		Description:   flex.StringIfChanged(p.Description, s.Description),
		Image:         flex.StringIfChanged(p.Image, s.Image),
		IsPublic:      flex.BoolIfChanged(p.IsPublic, s.IsPublic),
		PublicPort:    flex.Int64IfChanged(p.PublicPort, s.PublicPort),
		KeydbPassword: flex.StringIfChanged(p.KeydbPassword, s.KeydbPassword),
		KeydbConf:     flex.StringIfChanged(p.KeydbConf, s.KeydbConf),
	}
	dbcommon.SetUpdateExtendedDiff(&u, p.ExtFields().WithSSL(&p.EnableSSL, nil), s.ExtFields().WithSSL(&s.EnableSSL, nil))
	db, err := dbcommon.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating KeyDB database", fmt.Sprintf("KeyDB database %s: %s", s.UUID.ValueString(), err))
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
	dbcommon.DeleteDatabaseState(ctx, r.client, "coolify_database_keydb", s.UUID.ValueString(), resp)
}
func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, r.client, req, resp)
}
func flattenDatabase(db *client.Database, m *model) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields().WithSSL(&m.EnableSSL, nil))
	if db.KeydbPassword != "" {
		m.KeydbPassword = types.StringValue(db.KeydbPassword)
	} else if m.KeydbPassword.IsUnknown() {
		m.KeydbPassword = types.StringNull()
	}
	flex.SetStringOrClear(&m.KeydbConf, db.KeydbConf)
}
