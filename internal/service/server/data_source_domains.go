package server

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &serverDomainsDataSource{}
	_ datasource.DataSourceWithConfigure = &serverDomainsDataSource{}
)

type serverDomainsDataSource struct {
	client *client.Client
}

type serverDomainsDataSourceModel struct {
	ServerUUID types.String        `tfsdk:"server_uuid"`
	Domains    []serverDomainModel `tfsdk:"domains"`
	Filters    []filter.Config     `tfsdk:"filter"`
}

type serverDomainModel struct {
	Domain types.String `tfsdk:"domain"`
	IP     types.String `tfsdk:"ip"`
}

func NewDomainsDataSource() datasource.DataSource {
	return &serverDomainsDataSource{}
}

func (d *serverDomainsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_domains"
}

func (d *serverDomainsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all domains configured on a Coolify server.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the server.",
				Required:            true,
			},
			"domains": schema.ListNestedAttribute{
				MarkdownDescription: "The list of domains configured on the server.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"domain": schema.StringAttribute{
							MarkdownDescription: "The domain name.",
							Computed:            true,
						},
						"ip": schema.StringAttribute{
							MarkdownDescription: "The IP address the domain points to.",
							Computed:            true,
						},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *serverDomainsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}

	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Data Source Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}

	d.client = c
}

func (d *serverDomainsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverDomainsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	domains, err := d.client.ListServerDomains(ctx, config.ServerUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing server domains", err.Error())
		return
	}

	domains = filter.Apply(domains, config.Filters, func(d client.ServerDomain, field string) (string, bool) {
		switch field {
		case "domain":
			return d.Domain, true
		case "ip":
			return d.IP, true
		default:
			return "", false
		}
	})

	for _, dom := range domains {
		config.Domains = append(config.Domains, serverDomainModel{
			Domain: types.StringValue(dom.Domain),
			IP:     types.StringValue(dom.IP),
		})
	}

	if config.Domains == nil {
		config.Domains = []serverDomainModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
