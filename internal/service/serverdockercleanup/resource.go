package serverdockercleanup

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ resource.Resource                = (*res)(nil)
	_ resource.ResourceWithConfigure   = (*res)(nil)
	_ resource.ResourceWithImportState = (*res)(nil)
)

type res struct{ client *client.Client }

type model struct {
	ServerUUID                       types.String `tfsdk:"server_uuid"`
	DockerCleanupFrequency           types.String `tfsdk:"docker_cleanup_frequency"`
	DockerCleanupThreshold           types.Int64  `tfsdk:"docker_cleanup_threshold"`
	ForceDockerCleanup               types.Bool   `tfsdk:"force_docker_cleanup"`
	DeleteUnusedVolumes              types.Bool   `tfsdk:"delete_unused_volumes"`
	DeleteUnusedNetworks             types.Bool   `tfsdk:"delete_unused_networks"`
	DisableApplicationImageRetention types.Bool   `tfsdk:"disable_application_image_retention"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_docker_cleanup"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages the Coolify Docker cleanup schedule on a server. Requires Coolify >= v4.3.0. Destroy leaves the remote schedule in place.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"docker_cleanup_frequency": schema.StringAttribute{
				Optional:            true,
				Computed:            true,
				MarkdownDescription: "Cleanup frequency. Coolify accepts cron or human strings such as `@daily` or `daily`.",
				Validators:          []validator.String{validate.CoolifyFrequency()},
			},
			"docker_cleanup_threshold":            schema.Int64Attribute{Optional: true, Computed: true},
			"force_docker_cleanup":                schema.BoolAttribute{Optional: true, Computed: true},
			"delete_unused_volumes":               schema.BoolAttribute{Optional: true, Computed: true},
			"delete_unused_networks":              schema.BoolAttribute{Optional: true, Computed: true},
			"disable_application_image_retention": schema.BoolAttribute{Optional: true, Computed: true},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func boolPtrKnown(b types.Bool) *bool {
	if b.IsNull() || b.IsUnknown() {
		return nil
	}
	v := b.ValueBool()
	return &v
}

func toInput(m model) client.ServerDockerCleanup {
	in := client.ServerDockerCleanup{
		DockerCleanupFrequency:           m.DockerCleanupFrequency.ValueString(),
		ForceDockerCleanup:               boolPtrKnown(m.ForceDockerCleanup),
		DeleteUnusedVolumes:              boolPtrKnown(m.DeleteUnusedVolumes),
		DeleteUnusedNetworks:             boolPtrKnown(m.DeleteUnusedNetworks),
		DisableApplicationImageRetention: boolPtrKnown(m.DisableApplicationImageRetention),
	}
	if !m.DockerCleanupThreshold.IsNull() && !m.DockerCleanupThreshold.IsUnknown() {
		v := m.DockerCleanupThreshold.ValueInt64()
		in.DockerCleanupThreshold = &v
	}
	return in
}

func flatten(s *client.ServerDockerCleanup, m *model) {
	if s.DockerCleanupFrequency != "" {
		if m.DockerCleanupFrequency.IsNull() || m.DockerCleanupFrequency.IsUnknown() || m.DockerCleanupFrequency.ValueString() == "" {
			m.DockerCleanupFrequency = types.StringValue(s.DockerCleanupFrequency)
		}
	}
	if s.DockerCleanupThreshold != nil {
		m.DockerCleanupThreshold = types.Int64Value(*s.DockerCleanupThreshold)
	}
	if s.ForceDockerCleanup != nil {
		m.ForceDockerCleanup = types.BoolValue(*s.ForceDockerCleanup)
	} else if m.ForceDockerCleanup.IsUnknown() {
		m.ForceDockerCleanup = types.BoolValue(false)
	}
	if s.DeleteUnusedVolumes != nil {
		m.DeleteUnusedVolumes = types.BoolValue(*s.DeleteUnusedVolumes)
	} else if m.DeleteUnusedVolumes.IsUnknown() {
		m.DeleteUnusedVolumes = types.BoolValue(false)
	}
	if s.DeleteUnusedNetworks != nil {
		m.DeleteUnusedNetworks = types.BoolValue(*s.DeleteUnusedNetworks)
	} else if m.DeleteUnusedNetworks.IsUnknown() {
		m.DeleteUnusedNetworks = types.BoolValue(false)
	}
	if s.DisableApplicationImageRetention != nil {
		m.DisableApplicationImageRetention = types.BoolValue(*s.DisableApplicationImageRetention)
	} else if m.DisableApplicationImageRetention.IsUnknown() {
		m.DisableApplicationImageRetention = types.BoolValue(false)
	}
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerDockerCleanup(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying Docker cleanup schedule", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.GetServerDockerCleanup(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading Docker cleanup schedule", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
		return
	}
	flatten(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerDockerCleanup(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating Docker cleanup schedule", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
