package postgresql

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = &postgresqlDatabaseResource{}
	_ resource.ResourceWithConfigure   = &postgresqlDatabaseResource{}
	_ resource.ResourceWithImportState = &postgresqlDatabaseResource{}
)

type postgresqlDatabaseResource struct{ client *client.Client }

type postgresqlDatabaseResourceModel struct {
	UUID             types.String `tfsdk:"uuid"`
	Name             types.String `tfsdk:"name"`
	Description      types.String `tfsdk:"description"`
	ProjectUUID      types.String `tfsdk:"project_uuid"`
	ServerUUID       types.String `tfsdk:"server_uuid"`
	EnvironmentName  types.String `tfsdk:"environment_name"`
	Image            types.String `tfsdk:"image"`
	IsPublic         types.Bool   `tfsdk:"is_public"`
	PublicPort       types.Int64  `tfsdk:"public_port"`
	PostgresUser     types.String `tfsdk:"postgres_user"`
	PostgresPassword types.String `tfsdk:"postgres_password"`
	PostgresDB       types.String `tfsdk:"postgres_db"`
}

func NewResource() resource.Resource { return &postgresqlDatabaseResource{} }

func (r *postgresqlDatabaseResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_postgresql_database"
}

func (r *postgresqlDatabaseResource) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a PostgreSQL database resource on Coolify.",
		Attributes: CommonDatabaseAttrs(map[string]schema.Attribute{
			"postgres_user": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL user.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_password": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL password.", Optional: true, Computed: true, Sensitive: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
			"postgres_db": schema.StringAttribute{
				MarkdownDescription: "The PostgreSQL database name.", Optional: true, Computed: true,
				PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			},
		}),
	}
}

func (r *postgresqlDatabaseResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Resource Configure Type", fmt.Sprintf("Expected *client.Client, got: %T.", req.ProviderData))
		return
	}
	r.client = c
}

func (r *postgresqlDatabaseResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan postgresqlDatabaseResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	input := client.CreatePostgresqlInput{
		ServerUUID:      plan.ServerUUID.ValueString(),
		ProjectUUID:     plan.ProjectUUID.ValueString(),
		EnvironmentName: plan.EnvironmentName.ValueString(),
	}
	SetIfKnown(&input.Name, plan.Name)
	SetIfKnown(&input.Description, plan.Description)
	SetIfKnown(&input.Image, plan.Image)
	SetIfKnown(&input.PostgresUser, plan.PostgresUser)
	SetIfKnown(&input.PostgresPassword, plan.PostgresPassword)
	SetIfKnown(&input.PostgresDB, plan.PostgresDB)

	created, err := r.client.CreatePostgresqlDatabase(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating PostgreSQL database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgreSQL database after creation", err.Error())
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
	db, err := r.client.GetDatabase(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading PostgreSQL database", err.Error())
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
	input := client.UpdateDatabaseInput{}
	SetStrPtr(&input.Name, plan.Name)
	SetStrPtr(&input.Description, plan.Description)
	SetStrPtr(&input.Image, plan.Image)
	SetBoolPtr(&input.IsPublic, plan.IsPublic)
	SetInt64Ptr(&input.PublicPort, plan.PublicPort)
	SetStrPtr(&input.PostgresUser, plan.PostgresUser)
	SetStrPtr(&input.PostgresPassword, plan.PostgresPassword)
	SetStrPtr(&input.PostgresDB, plan.PostgresDB)
	if _, err := r.client.UpdateDatabase(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating PostgreSQL database", err.Error())
		return
	}
	db, err := r.client.GetDatabase(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading PostgreSQL database after update", err.Error())
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
	if err := r.client.DeleteDatabase(ctx, state.UUID.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting PostgreSQL database", err.Error())
		return
	}
}

func (r *postgresqlDatabaseResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func flattenDatabase(db *client.Database, m *postgresqlDatabaseResourceModel) {
	m.UUID = types.StringValue(db.UUID)
	m.Name = types.StringValue(db.Name)
	m.Image = StringOrNull(db.Image)
	m.IsPublic = types.BoolValue(db.IsPublic)
	m.PublicPort = Int64PtrToFW(db.PublicPort)
	m.PostgresUser = StringOrNull(db.PostgresUser)
	m.PostgresPassword = StringOrNull(db.PostgresPassword)
	m.PostgresDB = StringOrNull(db.PostgresDB)
	m.Description = StringOrNull(db.Description)
	m.ProjectUUID = StringOrNull(db.ProjectUUID)
	m.ServerUUID = StringOrNull(db.ServerUUID)
	m.EnvironmentName = StringOrNull(db.EnvironmentName)
}

// --- shared helpers ---

// CommonDatabaseAttrs returns the shared schema attributes for all database types.
func CommonDatabaseAttrs(extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the database.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":             schema.StringAttribute{MarkdownDescription: "The name of the database resource.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"description":      schema.StringAttribute{MarkdownDescription: "A description of the database.", Optional: true},
		"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this database belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the database on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"environment_name": schema.StringAttribute{MarkdownDescription: "The environment name. Defaults to `production`. Changing this forces a new resource.", Optional: true, Computed: true, Default: stringdefault.StaticString("production"), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"image":            schema.StringAttribute{MarkdownDescription: "The Docker image to use.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"is_public":        schema.BoolAttribute{MarkdownDescription: "Whether the database is publicly accessible.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"public_port": schema.Int64Attribute{MarkdownDescription: "The public port for the database, if publicly accessible.", Optional: true, Validators: []validator.Int64{
			int64validator.Between(1, 65535),
		}},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}

// SetIfKnown sets dst to the string value if v is known and non-null.
func SetIfKnown(dst *string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		*dst = v.ValueString()
	}
}

// SetStrPtr sets dst to a pointer to the string value if v is known and non-null.
func SetStrPtr(dst **string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		s := v.ValueString()
		*dst = &s
	}
}

// SetBoolPtr sets dst to a pointer to the bool value if v is known and non-null.
func SetBoolPtr(dst **bool, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		b := v.ValueBool()
		*dst = &b
	}
}

// SetInt64Ptr sets dst to a pointer to the int64 value if v is known and non-null.
func SetInt64Ptr(dst **int64, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		i := v.ValueInt64()
		*dst = &i
	}
}

// StringOrNull returns a types.String that is null when s is empty.
func StringOrNull(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// Int64PtrToFW converts a *int64 to a types.Int64.
func Int64PtrToFW(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}
