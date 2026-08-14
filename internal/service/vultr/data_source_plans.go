package vultr

import (
	"context"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*plansDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*plansDataSource)(nil)
)

type plansDataSource struct{ client *client.Client }
type plansDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Plans                  []planModel     `tfsdk:"plans"`
	Filters                []filter.Config `tfsdk:"filter"`
}
type planModel struct {
	ID          types.String  `tfsdk:"id"`
	VCPUCount   types.Int64   `tfsdk:"vcpu_count"`
	RAM         types.Int64   `tfsdk:"ram"`
	Disk        types.Int64   `tfsdk:"disk"`
	MonthlyCost types.Float64 `tfsdk:"monthly_cost"`
	Type        types.String  `tfsdk:"type"`
}

func NewPlansDataSource() datasource.DataSource { return &plansDataSource{} }
func (d *plansDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vultr_plans"
}
func (d *plansDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Vultr plans for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("plans"),
			"plans": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":           schema.StringAttribute{Computed: true, MarkdownDescription: "Plan ID."},
				"vcpu_count":   schema.Int64Attribute{Computed: true, MarkdownDescription: "vCPU count."},
				"ram":          schema.Int64Attribute{Computed: true, MarkdownDescription: "RAM in MB."},
				"disk":         schema.Int64Attribute{Computed: true, MarkdownDescription: "Disk in GB."},
				"monthly_cost": schema.Float64Attribute{Computed: true, MarkdownDescription: "Monthly cost USD."},
				"type":         schema.StringAttribute{Computed: true, MarkdownDescription: "Plan type."},
			}}},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}
func (d *plansDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureVultrDataSourceClient(req, resp)
}
func (d *plansDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config plansDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_vultr_plans", "Error listing Vultr plans", resp, d.client.ListVultrPlans,
		func(p client.VultrPlan, field string) (string, bool) {
			if field == "id" {
				return p.ID, true
			}
			if field == "type" {
				return p.Type, true
			}
			return "", false
		})
	if !ok {
		return
	}
	state := plansDataSourceModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, Plans: make([]planModel, len(items))}
	for i, p := range items {
		state.Plans[i] = planModel{ID: types.StringValue(p.ID), VCPUCount: types.Int64Value(p.VCPUCount), RAM: types.Int64Value(p.RAM), Disk: types.Int64Value(p.Disk), MonthlyCost: types.Float64Value(p.MonthlyCost), Type: types.StringValue(p.Type)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
