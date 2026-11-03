package integrationtoken

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*integrationTokenDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*integrationTokenDataSource)(nil)
)

type integrationTokenDataSource struct {
	client *client.Client
}

type integrationTokenDataModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	Provider      types.String `tfsdk:"integration_provider"`
	Capabilities  types.List   `tfsdk:"capabilities"`
	AutomaticDNS  types.Bool   `tfsdk:"automatic_dns"`
	ProxiedDNS    types.Bool   `tfsdk:"proxied_dns"`
	DNSZonesCount types.Int64  `tfsdk:"dns_zones_count"`
	ZonesSyncedAt types.String `tfsdk:"zones_synced_at"`
	CreatedAt     types.String `tfsdk:"created_at"`
	UpdatedAt     types.String `tfsdk:"updated_at"`
}

func NewDataSource() datasource.DataSource { return &integrationTokenDataSource{} }

func (d *integrationTokenDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_token"
}

func (d *integrationTokenDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Reads one Coolify integration token. Requires Coolify >= v4.6.0. The secret and metadata are not returned.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "Integration token UUID.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name":                 schema.StringAttribute{MarkdownDescription: "Name of the integration token.", Computed: true},
			"integration_provider": schema.StringAttribute{MarkdownDescription: "Integration provider. Coolify's API field is provider.", Computed: true},
			"capabilities":         schema.ListAttribute{MarkdownDescription: "Capabilities Coolify stored for this token.", Computed: true, ElementType: types.StringType},
			"automatic_dns":        schema.BoolAttribute{MarkdownDescription: "Cloudflare only. Whether Coolify creates DNS records for new domains.", Computed: true},
			"proxied_dns":          schema.BoolAttribute{MarkdownDescription: "Cloudflare only. Whether new DNS records are proxied.", Computed: true},
			"dns_zones_count":      schema.Int64Attribute{MarkdownDescription: "Number of Cloudflare DNS zones Coolify could read.", Computed: true},
			"zones_synced_at":      schema.StringAttribute{MarkdownDescription: "When Cloudflare zones were last read.", Computed: true},
			"created_at":           schema.StringAttribute{MarkdownDescription: "When Coolify created the token.", Computed: true},
			"updated_at":           schema.StringAttribute{MarkdownDescription: "When Coolify last updated the token.", Computed: true},
		},
	}
}

func (d *integrationTokenDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *integrationTokenDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var state integrationTokenDataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	got, err := d.client.GetIntegrationToken(ctx, state.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading integration token", fmt.Sprintf("token %s: %s", state.UUID.ValueString(), err))
		return
	}
	flattenData(got, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func flattenData(got *client.IntegrationToken, m *integrationTokenDataModel) {
	holder := integrationTokenModel{UUID: m.UUID}
	flattenIntegrationToken(got, &holder)
	m.UUID = holder.UUID
	m.Name = holder.Name
	m.Provider = holder.Provider
	m.Capabilities = holder.Capabilities
	m.AutomaticDNS = holder.AutomaticDNS
	m.ProxiedDNS = holder.ProxiedDNS
	m.DNSZonesCount = holder.DNSZonesCount
	m.ZonesSyncedAt = holder.ZonesSyncedAt
	m.CreatedAt = holder.CreatedAt
	m.UpdatedAt = holder.UpdatedAt
}
