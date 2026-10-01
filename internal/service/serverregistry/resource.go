package serverregistry

import (
	"context"
	"fmt"
	"regexp"
	"strings"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

// registryHostPattern is DockerRegistryLogins::REGISTRY_PATTERN.
var registryHostPattern = regexp.MustCompile(`^[a-z0-9]([a-z0-9.\-]*[a-z0-9])?(:[0-9]{1,5})?$`)

var (
	_ resource.Resource                   = (*res)(nil)
	_ resource.ResourceWithConfigure      = (*res)(nil)
	_ resource.ResourceWithImportState    = (*res)(nil)
	_ resource.ResourceWithValidateConfig = (*res)(nil)
)

type res struct{ client *client.Client }

type model struct {
	ServerUUID types.String `tfsdk:"server_uuid"`
	Registry   types.String `tfsdk:"registry"`
	Username   types.String `tfsdk:"username"`
	Password   types.String `tfsdk:"password"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_docker_registry"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Logs a Coolify server into a Docker registry (`POST /servers/{uuid}/registries`). " +
			"Requires Coolify 4.4 tip (not v4.3.23 and not v4.4-rc.1). " +
			"The password is write-only. Destroy runs docker logout. " +
			"Docker Hub aliases are sent to Coolify as `docker.io`. Terraform keeps the registry string from configuration.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "Server UUID. Changing it replaces the login.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"registry": schema.StringAttribute{
				MarkdownDescription: "Registry host, optionally with a port (`ghcr.io`, `registry.example.com:5000`). Use `docker.io` for Docker Hub.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			},
			"username": schema.StringAttribute{
				MarkdownDescription: "Registry username. Must not contain spaces.",
				Required:            true,
				Validators:          []validator.String{usernameValidator{}},
			},
			"password": schema.StringAttribute{
				MarkdownDescription: "Registry password or access token. Coolify never returns it. After import, set it again in configuration.",
				Required:            true,
				Sensitive:           true,
			},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *res) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var cfg model
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() || cfg.Registry.IsUnknown() || cfg.Registry.IsNull() {
		return
	}
	host := client.NormalizeDockerRegistry(cfg.Registry.ValueString())
	if !registryHostPattern.MatchString(host) {
		resp.Diagnostics.AddAttributeError(path.Root("registry"), "Invalid registry host",
			"Enter a registry host, for example ghcr.io or registry.example.com:5000.")
	}
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := requireRegistryAPI(r.client); err != nil {
		resp.Diagnostics.AddError("Docker registry login requires Coolify 4.4", err.Error())
		return
	}
	if err := r.client.LoginServerDockerRegistry(ctx, plan.ServerUUID.ValueString(), plan.Registry.ValueString(), plan.Username.ValueString(), plan.Password.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error logging in to Docker registry", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, &plan, &resp.Diagnostics, func() { resp.State.RemoveResource(ctx) }, func(m model) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	})
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, &state, &resp.Diagnostics, func() { resp.State.RemoveResource(ctx) }, func(m model) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	})
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := requireRegistryAPI(r.client); err != nil {
		resp.Diagnostics.AddError("Docker registry login requires Coolify 4.4", err.Error())
		return
	}
	if err := r.client.LoginServerDockerRegistry(ctx, plan.ServerUUID.ValueString(), plan.Registry.ValueString(), plan.Username.ValueString(), plan.Password.ValueString()); err != nil {
		resp.Diagnostics.AddError("Error updating Docker registry login", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	r.readInto(ctx, &plan, &resp.Diagnostics, func() { resp.State.RemoveResource(ctx) }, func(m model) {
		resp.Diagnostics.Append(resp.State.Set(ctx, &m)...)
	})
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := r.client.LogoutServerDockerRegistry(ctx, state.ServerUUID.ValueString(), state.Registry.ValueString()); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error logging out of Docker registry", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	serverUUID, registry, ok := strings.Cut(req.ID, ":")
	if !ok || serverUUID == "" || registry == "" {
		resp.Diagnostics.AddError("Invalid import ID", "Use server_uuid:registry, for example aaaa0001-0001-4000-8000-000000000001:ghcr.io")
		return
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), serverUUID)...)
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("registry"), registry)...)
}

func (r *res) readInto(ctx context.Context, state *model, diags *diag.Diagnostics, remove func(), set func(model)) {
	rows, err := r.client.ListServerDockerRegistries(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			remove()
			return
		}
		diags.AddError("Error reading Docker registry login", err.Error())
		return
	}
	want := client.NormalizeDockerRegistry(state.Registry.ValueString())
	for _, row := range rows {
		if row.Registry == want && row.LoggedIn {
			if row.Username != "" {
				state.Username = types.StringValue(row.Username)
			}
			set(*state)
			tflog.Debug(ctx, "read docker registry login", map[string]interface{}{"registry": want})
			return
		}
	}
	remove()
}

func requireRegistryAPI(c *client.Client) error {
	if c == nil || c.SupportsCoolify44Tip() {
		return nil
	}
	return fmt.Errorf("coolify_server_docker_registry requires Coolify 4.4 or later (not v4.4-rc.1). This instance is %s", c.CoolifyVersion)
}

type usernameValidator struct{}

func (usernameValidator) Description(context.Context) string { return "username without spaces" }
func (usernameValidator) MarkdownDescription(context.Context) string {
	return "username without spaces"
}
func (usernameValidator) ValidateString(_ context.Context, req validator.StringRequest, resp *validator.StringResponse) {
	if req.ConfigValue.IsNull() || req.ConfigValue.IsUnknown() {
		return
	}
	if strings.ContainsAny(req.ConfigValue.ValueString(), " \t") {
		resp.Diagnostics.AddAttributeError(req.Path, "Invalid registry username", "The username must not contain spaces.")
	}
}
