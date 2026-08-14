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
	_ datasource.DataSource              = (*osDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*osDataSource)(nil)
)

type osDataSource struct{ client *client.Client }
type osDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	OperatingSystems       []osModel       `tfsdk:"operating_systems"`
	Filters                []filter.Config `tfsdk:"filter"`
}
type osModel struct {
	ID     types.Int64  `tfsdk:"id"`
	Name   types.String `tfsdk:"name"`
	Arch   types.String `tfsdk:"arch"`
	Family types.String `tfsdk:"family"`
}

func NewOSDataSource() datasource.DataSource { return &osDataSource{} }
func (d *osDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vultr_os"
}
func (d *osDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Vultr operating systems for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("operating systems"),
			"operating_systems": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":     schema.Int64Attribute{Computed: true, MarkdownDescription: "OS ID."},
				"name":   schema.StringAttribute{Computed: true, MarkdownDescription: "OS name."},
				"arch":   schema.StringAttribute{Computed: true, MarkdownDescription: "Architecture."},
				"family": schema.StringAttribute{Computed: true, MarkdownDescription: "OS family."},
			}}},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}
func (d *osDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureVultrDataSourceClient(req, resp)
}
func (d *osDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config osDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_vultr_os", "Error listing Vultr OS", resp, d.client.ListVultrOS,
		func(o client.VultrOS, field string) (string, bool) {
			if field == "id" {
				return filter.Int64ToString(o.ID), true
			}
			if field == "name" {
				return o.Name, true
			}
			if field == "family" {
				return o.Family, true
			}
			return "", false
		})
	if !ok {
		return
	}
	state := osDataSourceModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, OperatingSystems: make([]osModel, len(items))}
	for i, o := range items {
		state.OperatingSystems[i] = osModel{ID: types.Int64Value(o.ID), Name: types.StringValue(o.Name), Arch: types.StringValue(o.Arch), Family: types.StringValue(o.Family)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
