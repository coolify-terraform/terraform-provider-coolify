package integrationtoken

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*integrationTokenListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*integrationTokenListDataSource)(nil)
)

type integrationTokenListDataSource struct {
	client *client.Client
}

type integrationTokenListModel struct {
	IntegrationTokens []integrationTokenDataModel `tfsdk:"integration_tokens"`
}

func NewListDataSource() datasource.DataSource { return &integrationTokenListDataSource{} }

func (d *integrationTokenListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_integration_tokens"
}

func (d *integrationTokenListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Coolify integration tokens for the current team. Requires Coolify >= v4.6.0. Secrets and metadata are not returned.",
		Attributes: map[string]schema.Attribute{
			"integration_tokens": schema.ListNestedAttribute{
				MarkdownDescription: "Integration tokens on the team.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"uuid":                 schema.StringAttribute{MarkdownDescription: "Integration token UUID.", Computed: true},
					"name":                 schema.StringAttribute{MarkdownDescription: "Name of the integration token.", Computed: true},
					"integration_provider": schema.StringAttribute{MarkdownDescription: "Integration provider.", Computed: true},
					"capabilities":         schema.ListAttribute{MarkdownDescription: "Stored capabilities.", Computed: true, ElementType: types.StringType},
					"automatic_dns":        schema.BoolAttribute{MarkdownDescription: "Cloudflare automatic DNS.", Computed: true},
					"proxied_dns":          schema.BoolAttribute{MarkdownDescription: "Cloudflare proxied DNS.", Computed: true},
					"dns_zones_count":      schema.Int64Attribute{MarkdownDescription: "Cloudflare zone count.", Computed: true},
					"zones_synced_at":      schema.StringAttribute{MarkdownDescription: "When Cloudflare zones were last read.", Computed: true},
					"created_at":           schema.StringAttribute{MarkdownDescription: "Created at.", Computed: true},
					"updated_at":           schema.StringAttribute{MarkdownDescription: "Updated at.", Computed: true},
				}},
			},
		},
	}
}

func (d *integrationTokenListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *integrationTokenListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	tokens, err := d.client.ListIntegrationTokens(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing integration tokens", err.Error())
		return
	}
	out := integrationTokenListModel{IntegrationTokens: make([]integrationTokenDataModel, len(tokens))}
	for i := range tokens {
		flattenData(&tokens[i], &out.IntegrationTokens[i])
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &out)...)
}
