package postgresql

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
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
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.Image, plan.Image)
	flex.SetIfKnown(&input.PostgresUser, plan.PostgresUser)
	flex.SetIfKnown(&input.PostgresPassword, plan.PostgresPassword)
	flex.SetIfKnown(&input.PostgresDB, plan.PostgresDB)

	created, err := r.client.CreatePostgresqlDatabase(ctx, input)
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
	flex.SetStrPtr(&input.Name, plan.Name)
	flex.SetStrPtr(&input.Description, plan.Description)
	flex.SetStrPtr(&input.Image, plan.Image)
	flex.SetBoolPtr(&input.IsPublic, plan.IsPublic)
	input.PublicPort = flex.Int64PtrFromFramework(plan.PublicPort)
	flex.SetStrPtr(&input.PostgresUser, plan.PostgresUser)
	flex.SetStrPtr(&input.PostgresPassword, plan.PostgresPassword)
	flex.SetStrPtr(&input.PostgresDB, plan.PostgresDB)
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
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("uuid"), req, resp)
}

func flattenDatabase(db *client.Database, m *postgresqlDatabaseResourceModel) {
	m.UUID = types.StringValue(db.UUID)
	m.Name = types.StringValue(db.Name)
	m.Image = flex.StringToFramework(db.Image)
	m.IsPublic = types.BoolValue(db.IsPublic)
	m.PublicPort = flex.Int64PtrToFramework(db.PublicPort)
	m.PostgresUser = flex.StringToFramework(db.PostgresUser)
	m.PostgresPassword = flex.StringToFramework(db.PostgresPassword)
	m.PostgresDB = flex.StringToFramework(db.PostgresDB)
	m.Description = flex.StringToFramework(db.Description)
	m.ProjectUUID = flex.StringToFramework(db.ProjectUUID)
	m.ServerUUID = flex.StringToFramework(db.ServerUUID)
	m.EnvironmentName = flex.StringToFramework(db.EnvironmentName)
}

// --- shared helpers ---

// CommonDatabaseAttrs returns the shared schema attributes for all database types.
func CommonDatabaseAttrs(extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := map[string]schema.Attribute{
		"uuid":             schema.StringAttribute{MarkdownDescription: "The UUID of the database.", Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"name":             schema.StringAttribute{MarkdownDescription: "The name of the database resource. Also used as the Docker container name and internal DNS hostname for inter-container communication.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"description":      schema.StringAttribute{MarkdownDescription: "A description of the database.", Optional: true, Computed: true},
		"project_uuid":     schema.StringAttribute{MarkdownDescription: "The UUID of the project this database belongs to.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"server_uuid":      schema.StringAttribute{MarkdownDescription: "The UUID of the server to deploy the database on.", Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
		"environment_name": schema.StringAttribute{MarkdownDescription: "The name of the environment within the project to deploy into. Coolify auto-creates a `production` environment per project; for other environments, create one first with `coolify_environment`. Defaults to `production`. Changing this forces a new resource.", Optional: true, Computed: true, Default: stringdefault.StaticString("production"), PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}},
		"image":            schema.StringAttribute{MarkdownDescription: "The Docker image to use.", Optional: true, Computed: true, PlanModifiers: []planmodifier.String{stringplanmodifier.UseStateForUnknown()}},
		"is_public":        schema.BoolAttribute{MarkdownDescription: "When `true`, exposes the database on a port accessible via the server's IP address. When `false` (default), the database is only reachable from other containers on the same Docker network. Set `public_port` to choose a specific port.", Optional: true, Computed: true, Default: booldefault.StaticBool(false)},
		"public_port": schema.Int64Attribute{MarkdownDescription: "The host port to expose the database on when `is_public` is `true`. If omitted, Coolify auto-assigns an available port. Ignored when `is_public` is `false`.", Optional: true, Computed: true, Validators: []validator.Int64{
			int64validator.Between(1, 65535),
		}},
	}
	for k, v := range extra {
		attrs[k] = v
	}
	return attrs
}
