package sqlite

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	dbcommon "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// sqliteDatabasesPattern is StandaloneSqlite::DATABASES_PATTERN.
var sqliteDatabasesPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._-]*(,[A-Za-z0-9][A-Za-z0-9._-]*)*$`)

var (
	_ resource.Resource                   = &res{}
	_ resource.ResourceWithConfigure      = &res{}
	_ resource.ResourceWithImportState    = &res{}
	_ resource.ResourceWithValidateConfig = &res{}
)

type res struct{ client *client.Client }

type model struct {
	dbcommon.CommonModel
	SqliteDatabases types.String `tfsdk:"sqlite_databases"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_database_sqlite"
}

func (r *res) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a SQLite database on Coolify. Requires Coolify 4.4 tip (not v4.3.23 and not v4.4-rc.1). " +
			"Coolify rejects public networking on this type: leave `is_public` false and omit `public_port` and `public_port_timeout`. " +
			"`sqlite_databases` is a comma-separated list of database file names. Coolify's default is `database.sqlite`.",
		Attributes: dbcommon.CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
			"sqlite_databases": schema.StringAttribute{
				MarkdownDescription: "Comma-separated SQLite database file names (for example `app.sqlite,cache.sqlite`). " +
					"Coolify defaults to `database.sqlite` when omitted. Requires Coolify 4.4 tip. Each name must match `[A-Za-z0-9][A-Za-z0-9._-]*`.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
				Validators:    []validator.String{stringvalidator.RegexMatches(sqliteDatabasesPattern, "must be a comma-separated list of SQLite database file names")},
			},
		}),
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = dbcommon.ConfigureDatabase(req, resp)
}

func (r *res) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg model
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	public := !cfg.IsPublic.IsUnknown() && !cfg.IsPublic.IsNull() && cfg.IsPublic.ValueBool()
	portSet := !cfg.PublicPort.IsUnknown() && !cfg.PublicPort.IsNull()
	timeoutSet := !cfg.PublicPortTimeout.IsUnknown() && !cfg.PublicPortTimeout.IsNull()
	if public || portSet || timeoutSet {
		resp.Diagnostics.AddAttributeError(
			path.Root("is_public"),
			"SQLite databases are not published on a host port",
			"Coolify rejects is_public, public_port, and public_port_timeout on standalone SQLite. Leave is_public at false and omit public_port and public_port_timeout.",
		)
	}
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var p model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := requireSQLiteAPI(r.client); err != nil {
		resp.Diagnostics.AddError("SQLite databases require Coolify 4.4", err.Error())
		return
	}
	createTimeout, diags := p.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_database_sqlite"})

	var in client.CreateSqliteInput
	dbcommon.PopulateBaseCreateInput(&in.CreateDatabaseBaseInput, &p.CommonModel)
	// POST /databases/sqlite 422s these keys even when is_public is false.
	in.IsPublic = nil
	in.PublicPort = nil
	flex.SetIfKnown(&in.SqliteDatabases, p.SqliteDatabases)
	created, err := r.client.CreateDatabase(ctx, "sqlite", in)
	if err != nil {
		resp.Diagnostics.AddError("Error creating SQLite database",
			fmt.Sprintf("project %s, server %s: %s", p.ProjectUUID.ValueString(), p.ServerUUID.ValueString(), err))
		return
	}

	p.UUID = types.StringValue(created.UUID)
	dbcommon.NormalizeCommonCreateState(&p.CommonModel)
	flex.NormalizeUnknownString(&p.SqliteDatabases)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	if resp.Diagnostics.HasError() {
		return
	}

	if dbcommon.HasExtendedFields(p.ExtFields()) {
		update := client.UpdateDatabaseInput{}
		dbcommon.SetUpdateExtended(&update, p.ExtFields())
		update.IsPublic = nil
		update.PublicPort = nil
		update.PublicPortTimeout = nil
		if _, err := r.client.UpdateDatabase(ctx, created.UUID, update); err != nil {
			dbcommon.RecoverCreateAfterExtendedError(ctx, r.client, created.UUID, "SQLite database", err, resp, func(db *client.Database) {
				flattenDatabase(db, &p)
				resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
			})
			return
		}
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		dbcommon.AddCreateReadBackError(resp, "SQLite database", created.UUID, err)
		return
	}
	flattenDatabase(db, &p)
	resp.Diagnostics.Append(resp.State.Set(ctx, &p)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_database_sqlite", "uuid": created.UUID})
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var s model
	resp.Diagnostics.Append(req.State.Get(ctx, &s)...)
	if resp.Diagnostics.HasError() {
		return
	}
	dbcommon.ReadDatabaseState(ctx, r.client, "coolify_database_sqlite", s.UUID.ValueString(), resp, func(db *client.Database) {
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
	if err := requireSQLiteAPI(r.client); err != nil {
		resp.Diagnostics.AddError("SQLite databases require Coolify 4.4", err.Error())
		return
	}
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_database_sqlite", "uuid": s.UUID.ValueString()})

	u := client.UpdateDatabaseInput{
		Name:            flex.StringIfChanged(p.Name, s.Name),
		Description:     flex.StringIfChanged(p.Description, s.Description),
		Image:           flex.StringIfChanged(p.Image, s.Image),
		SqliteDatabases: flex.StringIfChanged(p.SqliteDatabases, s.SqliteDatabases),
	}
	dbcommon.SetUpdateExtendedDiff(&u, p.ExtFields(), s.ExtFields())
	u.IsPublic = nil
	u.PublicPort = nil
	u.PublicPortTimeout = nil
	db, err := dbcommon.UpdateDatabase(ctx, r.client, s.UUID.ValueString(), u)
	if err != nil {
		resp.Diagnostics.AddError("Error updating SQLite database", fmt.Sprintf("SQLite database %s: %s", s.UUID.ValueString(), err))
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
	dbcommon.DeleteDatabaseState(ctx, r.client, "coolify_database_sqlite", s.UUID.ValueString(), s.Timeouts, resp)
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	dbcommon.ImportDatabaseState(ctx, r.client, req, resp)
}

func flattenDatabase(db *client.Database, m *model) {
	dbcommon.FlattenDatabaseCommon(db, m.CommonPtrs())
	dbcommon.FlattenDatabaseExtended(db, m.ExtFields())
	if db.SqliteDatabases != "" {
		m.SqliteDatabases = types.StringValue(db.SqliteDatabases)
	} else if m.SqliteDatabases.IsUnknown() {
		m.SqliteDatabases = types.StringNull()
	}
}

func requireSQLiteAPI(c *client.Client) error {
	if c == nil || c.SupportsCoolify44Tip() {
		return nil
	}
	return fmt.Errorf("coolify_database_sqlite requires Coolify 4.4 or later (not v4.4-rc.1). This instance is %s", c.CoolifyVersion)
}
