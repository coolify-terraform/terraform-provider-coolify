package resourcelist

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &resourcesDataSource{}
	_ datasource.DataSourceWithConfigure = &resourcesDataSource{}
)

type resourcesDataSource struct {
	client *client.Client
}

type resourcesDataSourceModel struct {
	Resources []resourceItemModel `tfsdk:"resources"`
	Filters   []filter.Config     `tfsdk:"filter"`
}

type resourceItemModel struct {
	UUID   types.String `tfsdk:"uuid"`
	Name   types.String `tfsdk:"name"`
	Type   types.String `tfsdk:"type"`
	Status types.String `tfsdk:"status"`
}

func NewDataSource() datasource.DataSource {
	return &resourcesDataSource{}
}

func (d *resourcesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_resources"
}

func (d *resourcesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify resources.",
		Attributes: map[string]schema.Attribute{
			"resources": schema.ListNestedAttribute{
				MarkdownDescription: "The list of resources.",
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
						"status": schema.StringAttribute{
							MarkdownDescription: "The status of the resource.",
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

func (d *resourcesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError(
			"Unexpected Configure Type",
			fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData),
		)
		return
	}
	d.client = c
}

func (d *resourcesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config resourcesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_resources"})

	resources, err := d.client.ListResources(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing resources", err.Error())
		return
	}

	resources = filter.Apply(ctx, resources, config.Filters, func(r client.Resource, field string) (string, bool) {
		switch field {
		case "uuid":
			return r.UUID, true
		case "name":
			return r.Name, true
		case "type":
			return r.Type, true
		case "status":
			return r.Status, true
		default:
			return "", false
		}
	})

	var state resourcesDataSourceModel
	state.Filters = config.Filters
	for _, r := range resources {
		state.Resources = append(state.Resources, resourceItemModel{
			UUID:   types.StringValue(r.UUID),
			Name:   types.StringValue(r.Name),
			Type:   types.StringValue(r.Type),
			Status: types.StringValue(r.Status),
		})
	}

	if state.Resources == nil {
		state.Resources = []resourceItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
