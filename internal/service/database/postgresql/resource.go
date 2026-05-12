package postgresql

import (
	"context"
	"fmt"
	"time"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = &postgresqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &postgresqlDatabaseResource{}
	_ resource.ResourceWithImportState = &postgresqlDatabaseResource{}
)

type postgresqlDatabaseResource struct{ client *client.Client }

type postgresqlDatabaseResourceModel struct {
	Timeouts         timeouts.Value `tfsdk:"timeouts"`
	UUID             types.String   `tfsdk:"uuid"`
	Name             types.String   `tfsdk:"name"`
	Description      types.String   `tfsdk:"description"`
	ProjectUUID      types.String   `tfsdk:"project_uuid"`
	ServerUUID       types.String   `tfsdk:"server_uuid"`
	EnvironmentName  types.String   `tfsdk:"environment_name"`
	Image            types.String   `tfsdk:"image"`
	IsPublic         types.Bool     `tfsdk:"is_public"`
	PublicPort       types.Int64    `tfsdk:"public_port"`
	PostgresUser     types.String   `tfsdk:"postgres_user"`
	PostgresPassword types.String   `tfsdk:"postgres_password"`
	PostgresDB       types.String   `tfsdk:"postgres_db"`
}

func NewResource() resource.Resource { return &postgresqlDatabaseResource{} }

func (r *postgresqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_database"
}

func (r *postgresqlDatabaseResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a PostgreSQL database resource on Coolify.",
		Attributes: CommonDatabaseAttrs(ctx, map[string]schema.Attribute{
			"postgres_user": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL superuser name (maps to `POSTGRES_USER`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_password": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL superuser password (maps to `POSTGRES_PASSWORD`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_db": schema.StringAttribute{
				MarkdownDescription: "The default database name (maps to `POSTGRES_DB`). If omitted, Coolify auto-generates a value readable from state after creation.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		}),
	}
}

func (r *postgresqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = ConfigureDatabase(req, resp)
}

func (r *postgresqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}
	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_postgresql_database"})

	input := client.CreatePostgresqlInput{
		ServerUUID:      plan.ServerUUID.ValueString(),
		ProjectUUID:     plan.ProjectUUID.ValueString(),
		EnvironmentName: plan.EnvironmentName.ValueString(),
	}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.PostgresUser, plan.PostgresUser)
	flex.SetIfKnown(&input.PostgresPassword, plan.PostgresPassword)
	flex.SetIfKnown(&input.PostgresDB, plan.PostgresDB)
	input.IsPublic = flex.BoolValueOrNull(plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)

	created, err := r.client.CreateDatabase(ctx, "postgresql", input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PostgreSQL database", err.Error())
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgreSQL database after creation", fmt.Sprintf("PostgreSQL database %s: %s", created.UUID, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *postgresqlDatabaseResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": state.UUID.ValueString()})

	db, err := ReadDatabase(ctx, r.client, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
	if db == nil {
		resp.State.RemoveResource(ctx)
		return
	}
	flattenDatabase(db, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *postgresqlDatabaseResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.UUID.ValueString()

	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": uuid})

	input := client.UpdateDatabaseInput{}
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.Description, plan.Description)
	flex.SetStrPtr(&input.Image, plan.Image)
	flex.SetBoolPtr(&input.IsPublic, plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)
	flex.SetStrPtr(&input.PostgresUser, plan.PostgresUser)
	flex.SetStrPtr(&input.PostgresPassword, plan.PostgresPassword)
	flex.SetStrPtr(&input.PostgresDB, plan.PostgresDB)
	db, err := UpdateDatabase(ctx, r.client, uuid, input)
	if err != nil {
		resp.Diagnostics.AddError("Error updating PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", uuid, err))
		return
	}
	flattenDatabase(db, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *postgresqlDatabaseResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_postgresql_database", "uuid": state.UUID.ValueString()})

	if err := DeleteDatabase(ctx, r.client, state.UUID.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error deleting PostgreSQL database", fmt.Sprintf("PostgreSQL database %s: %s", state.UUID.ValueString(), err))
		return
	}
}

func (r *postgresqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	ImportDatabaseState(ctx, req, resp)
}

func flattenDatabase(db *client.Database, m *postgresqlDatabaseResourceModel) {
	FlattenDatabaseCommon(db, &m.UUID, &m.Name, &m.Description, &m.Image, &m.ProjectUUID, &m.ServerUUID, &m.EnvironmentName, &m.IsPublic, &m.PublicPort)
	m.PostgresUser = flex.StringToFramework(db.PostgresUser)
	m.PostgresPassword = flex.StringToFramework(db.PostgresPassword)
	m.PostgresDB = flex.StringToFramework(db.PostgresDB)
}

// --- shared helpers ---

// CommonDatabaseAttrs returns the shared schema attributes for all database types.
func CommonDatabaseAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"timeouts":         timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the database.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":             schema.StringAttribute{MarkdownDescription: "The name of the database resource. Also used as the Docker container name and internal DNS hostname for inter-container communication.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"description":      schema.StringAttribute{MarkdownDescription: "A description of the database.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this database belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the database on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"environment_name": schema.StringAttribute{MarkdownDescription: "The name of the environment within the project to deploy into. Coolify auto-creates a `production` environment per project; for other environments, create one first with `coolify_environment`. Defaults to `production`. Changing this forces a new resource.", Optional: true, Computed: true, Default: stringdefault.StaticString("production"), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"image":            schema.StringAttribute{MarkdownDescription: "The Docker image to use.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"is_public":        schema.BoolAttribute{MarkdownDescription: "When `true`, exposes the database on a port accessible via the server's IP address. When `false` (default), the database is only reachable from other containers on the same Docker network. Set `public_port` to choose a specific port.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"public_port": schema.Int64Attribute{MarkdownDescription: "The host port to expose the database on when `is_public` is `true`. If omitted, Coolify auto-assigns an available port. Ignored when `is_public` is `false`.", Optional: true, Computed: true, PlanModifiers: []planmodifier.Int64{int64planmodifier.UseStateForUnknown()}, Validators: []validator.Int64{
			int64validator.Between(1, 65535),
		}},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// ConfigureDatabase extracts the API client from provider data.
// Returns nil when ProviderData is nil (expected during early configure).
func ConfigureDatabase(req resource.ConfigureRequest, resp *resource.ConfigureResponse) *client.Client {
	if req.ProviderData == nil {
		return nil
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return nil
	}
	return c
}

// ReadDatabase fetches a database by UUID. Returns (nil, nil) when the
// database is not found (caller should remove the resource from state).
func ReadDatabase(ctx context.Context, c *client.Client, uuid string) (*client.Database, error) {
	db, err := c.GetDatabase(ctx, uuid)
	if err != nil {
		if client.IsNotFound(err) {
			return nil, nil
		}
		return nil, err
	}
	return db, nil
}

// UpdateDatabase sends an update for the given database and reads back the
// result.
func UpdateDatabase(ctx context.Context, c *client.Client, uuid string, input client.UpdateDatabaseInput) (*client.Database, error) {
	if _, err := c.UpdateDatabase(ctx, uuid, input); err != nil {
		return nil, err
	}
	return c.GetDatabase(ctx, uuid)
}

// DeleteDatabase removes a database by UUID, silently succeeding if already
// gone.
func DeleteDatabase(ctx context.Context, c *client.Client, uuid string) error {
	if err := c.DeleteDatabase(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return nil
		}
		return err
	}
	return nil
}

// ImportDatabaseState validates the import ID as a UUID and passes it through
// as the "uuid" attribute.
func ImportDatabaseState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

// FlattenDatabaseCommon sets the fields shared by all database resource types.
func FlattenDatabaseCommon(db *client.Database, uuid, name, description, image, projectUUID, serverUUID, envName *types.String, isPublic *types.Bool, publicPort *types.Int64) {
	*uuid = types.StringValue(db.UUID)
	*name = types.StringValue(db.Name)
	*image = flex.StringToFramework(db.Image)
	*isPublic = types.BoolValue(db.IsPublic)
	*publicPort = flex.Int64PtrToFramework(db.PublicPort)
	*description = flex.StringToFramework(db.Description)
	if db.ProjectUUID != "" {
		*projectUUID = types.StringValue(db.ProjectUUID)
	}
	if db.ServerUUID != "" {
		*serverUUID = types.StringValue(db.ServerUUID)
	}
	if db.EnvironmentName != "" {
		*envName = flex.StringToFramework(db.EnvironmentName)
	}
}
