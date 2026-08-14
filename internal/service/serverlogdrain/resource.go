package serverlogdrain

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
	ServerUUID         types.String `tfsdk:"server_uuid"`
	NewRelicEnabled    types.Bool   `tfsdk:"is_logdrain_newrelic_enabled"`
	NewRelicLicenseKey types.String `tfsdk:"logdrain_newrelic_license_key"`
	NewRelicBaseURI    types.String `tfsdk:"logdrain_newrelic_base_uri"`
	AxiomEnabled       types.Bool   `tfsdk:"is_logdrain_axiom_enabled"`
	AxiomDatasetName   types.String `tfsdk:"logdrain_axiom_dataset_name"`
	AxiomAPIKey        types.String `tfsdk:"logdrain_axiom_api_key"`
	CustomEnabled      types.Bool   `tfsdk:"is_logdrain_custom_enabled"`
	CustomConfig       types.String `tfsdk:"logdrain_custom_config"`
	CustomConfigParser types.String `tfsdk:"logdrain_custom_config_parser"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_log_drain"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages Coolify server log drain settings (New Relic, Axiom, or custom). Requires Coolify >= v4.3.0. Destroy disables all drains.",
		Attributes: map[string]schema.Attribute{
			"server_uuid":                   schema.StringAttribute{Required: true, PlanModifiers: []planmodifier.String{stringplanmodifier.RequiresReplace()}, Validators: []validator.String{validate.UUID()}},
			"is_logdrain_newrelic_enabled":  schema.BoolAttribute{Optional: true, Computed: true},
			"logdrain_newrelic_license_key": schema.StringAttribute{Optional: true, Sensitive: true},
			"logdrain_newrelic_base_uri":    schema.StringAttribute{Optional: true},
			"is_logdrain_axiom_enabled":     schema.BoolAttribute{Optional: true, Computed: true},
			"logdrain_axiom_dataset_name":   schema.StringAttribute{Optional: true},
			"logdrain_axiom_api_key":        schema.StringAttribute{Optional: true, Sensitive: true},
			"is_logdrain_custom_enabled":    schema.BoolAttribute{Optional: true, Computed: true},
			"logdrain_custom_config":        schema.StringAttribute{Optional: true, Sensitive: true},
			"logdrain_custom_config_parser": schema.StringAttribute{Optional: true},
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

func toInput(m model) client.ServerLogDrains {
	return client.ServerLogDrains{
		IsNewRelicEnabled:  boolPtrKnown(m.NewRelicEnabled),
		NewRelicLicenseKey: m.NewRelicLicenseKey.ValueString(),
		NewRelicBaseURI:    m.NewRelicBaseURI.ValueString(),
		IsAxiomEnabled:     boolPtrKnown(m.AxiomEnabled),
		AxiomDatasetName:   m.AxiomDatasetName.ValueString(),
		AxiomAPIKey:        m.AxiomAPIKey.ValueString(),
		IsCustomEnabled:    boolPtrKnown(m.CustomEnabled),
		CustomConfig:       m.CustomConfig.ValueString(),
		CustomConfigParser: m.CustomConfigParser.ValueString(),
	}
}

func flatten(s *client.ServerLogDrains, m *model) {
	if s.IsNewRelicEnabled != nil {
		m.NewRelicEnabled = types.BoolValue(*s.IsNewRelicEnabled)
	} else if m.NewRelicEnabled.IsUnknown() {
		m.NewRelicEnabled = types.BoolValue(false)
	}
	if s.NewRelicLicenseKey != "" {
		m.NewRelicLicenseKey = types.StringValue(s.NewRelicLicenseKey)
	}
	if s.NewRelicBaseURI != "" {
		m.NewRelicBaseURI = types.StringValue(s.NewRelicBaseURI)
	}
	if s.IsAxiomEnabled != nil {
		m.AxiomEnabled = types.BoolValue(*s.IsAxiomEnabled)
	} else if m.AxiomEnabled.IsUnknown() {
		m.AxiomEnabled = types.BoolValue(false)
	}
	if s.AxiomDatasetName != "" {
		m.AxiomDatasetName = types.StringValue(s.AxiomDatasetName)
	}
	if s.AxiomAPIKey != "" {
		m.AxiomAPIKey = types.StringValue(s.AxiomAPIKey)
	}
	if s.IsCustomEnabled != nil {
		m.CustomEnabled = types.BoolValue(*s.IsCustomEnabled)
	} else if m.CustomEnabled.IsUnknown() {
		m.CustomEnabled = types.BoolValue(false)
	}
	if s.CustomConfig != "" {
		m.CustomConfig = types.StringValue(s.CustomConfig)
	}
	if s.CustomConfigParser != "" {
		m.CustomConfigParser = types.StringValue(s.CustomConfigParser)
	}
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := r.client.UpdateServerLogDrains(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error applying server log drains", err.Error())
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
	got, err := r.client.GetServerLogDrains(ctx, state.ServerUUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server log drains", fmt.Sprintf("%s: %s", state.ServerUUID.ValueString(), err))
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
	got, err := r.client.UpdateServerLogDrains(ctx, plan.ServerUUID.ValueString(), toInput(plan))
	if err != nil {
		resp.Diagnostics.AddError("Error updating server log drains", err.Error())
		return
	}
	flatten(got, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	off := false
	disabled := client.ServerLogDrains{IsNewRelicEnabled: &off, IsAxiomEnabled: &off, IsCustomEnabled: &off}
	if _, err := r.client.UpdateServerLogDrains(ctx, state.ServerUUID.ValueString(), disabled); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling server log drains", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}
