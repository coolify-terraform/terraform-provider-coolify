package service

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*serviceListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serviceListDataSource)(nil)
)

// serviceListDataSource is the data source implementation for listing all Coolify services.
type serviceListDataSource struct {
	client *client.Client
}

// serviceListDataSourceModel maps the data source schema data.
type serviceListDataSourceModel struct {
	Services []serviceItemModel `tfsdk:"services"`
}

// serviceItemModel maps a single service in the list.
type serviceItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Type        types.String `tfsdk:"type"`
}

// NewListDataSource returns a new services list data source instance.
func NewListDataSource() datasource.DataSource {
	return &serviceListDataSource{}
}

func (d *serviceListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_services"
}

func (d *serviceListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify services.",
		Attributes: map[string]schema.Attribute{
			"services": schema.ListNestedAttribute{
				MarkdownDescription: "The list of services.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the service.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the service.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "A description of the service.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "The type of the service.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *serviceListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *serviceListDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	services, err := d.client.ListServices(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing services", fmt.Sprintf("Could not list services: %s", err))
		return
	}

	var state serviceListDataSourceModel
	for _, s := range services {
		item := serviceItemModel{
			UUID: types.StringValue(s.UUID),
			Name: types.StringValue(s.Name),
			Type: types.StringValue(s.Type),
		}
		item.Description = flex.StringToFramework(s.Description)
		state.Services = append(state.Services, item)
	}

	if state.Services == nil {
		state.Services = []serviceItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
