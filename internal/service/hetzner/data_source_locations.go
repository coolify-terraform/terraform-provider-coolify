package hetzner

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*locationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*locationsDataSource)(nil)
)

type locationsDataSource struct {
	client *client.Client
}

type locationsDataSourceModel struct {
	Locations []locationModel `tfsdk:"locations"`
}

type locationModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	City        types.String `tfsdk:"city"`
	Country     types.String `tfsdk:"country"`
}

// NewLocationsDataSource returns a new Hetzner locations data source instance.
func NewLocationsDataSource() datasource.DataSource { return &locationsDataSource{} }

func (d *locationsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_locations"
}

func (d *locationsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Hetzner datacenter locations.",
		Attributes: map[string]schema.Attribute{
			"locations": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner locations.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{MarkdownDescription: "The numeric ID of the location.", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "The name of the location.", Computed: true},
						"description": schema.StringAttribute{MarkdownDescription: "The description of the location.", Computed: true},
						"city":        schema.StringAttribute{MarkdownDescription: "The city of the location.", Computed: true},
						"country":     schema.StringAttribute{MarkdownDescription: "The country of the location.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *locationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *locationsDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	locations, err := d.client.ListHetznerLocations(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner locations", err.Error())
		return
	}

	state := locationsDataSourceModel{
		Locations: make([]locationModel, len(locations)),
	}
	for i, loc := range locations {
		state.Locations[i] = locationModel{
			ID:          types.Int64Value(loc.ID),
			Name:        types.StringValue(loc.Name),
			Description: types.StringValue(loc.Description),
			City:        types.StringValue(loc.City),
			Country:     types.StringValue(loc.Country),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
