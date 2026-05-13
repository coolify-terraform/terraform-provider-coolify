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
	_ datasource.DataSource              = &serverResourcesDataSource{}
	_ datasource.DataSourceWithConfigure = &serverResourcesDataSource{}
)

type serverResourcesDataSource struct {
	client *client.Client
}

type serverResourcesDataSourceModel struct {
	ServerUUID types.String              `tfsdk:"server_uuid"`
	Resources  []serverResourceItemModel `tfsdk:"resources"`
	Filters    []filter.Config           `tfsdk:"filter"`
}

type serverResourceItemModel struct {
	UUID types.String `tfsdk:"uuid"`
	Name types.String `tfsdk:"name"`
	Type types.String `tfsdk:"type"`
}

func NewResourcesDataSource() datasource.DataSource {
	return &serverResourcesDataSource{}
}

func (d *serverResourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_resources"
}

func (d *serverResourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves all resources (apps, databases, services) deployed on a Coolify server.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the server.",
				Required:            true,
			},
			"resources": schema.ListNestedAttribute{
				MarkdownDescription: "The list of resources deployed on the server.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the resource.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the resource.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the resource.",
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

func (d *serverResourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serverResourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverResourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	resources, err := d.client.ListServerResources(ctx, config.ServerUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing server resources", err.Error())
		return
	}

	resources = filter.Apply(resources, config.Filters, func(r client.ServerResource, field string) (string, bool) {
		switch field {
		case "uuid":
			return r.UUID, true
		case "name":
			return r.Name, true
		case "type":
			return r.Type, true
		default:
			return "", false
		}
	})

	for _, r := range resources {
		config.Resources = append(config.Resources, serverResourceItemModel{
			UUID: types.StringValue(r.UUID),
			Name: types.StringValue(r.Name),
			Type: types.StringValue(r.Type),
		})
	}

	if config.Resources == nil {
		config.Resources = []serverResourceItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
