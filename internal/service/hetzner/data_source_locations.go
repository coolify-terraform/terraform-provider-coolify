package hetzner

import (
	"context"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*locationsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*locationsDataSource)(nil)
)

type locationsDataSource struct {
	client *client.Client
}

type locationsDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Locations              []locationModel `tfsdk:"locations"`
	Filters                []filter.Config `tfsdk:"filter"`
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
		MarkdownDescription: "Lists all available Hetzner datacenter locations for a given cloud provider token.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the cloud provider token to use for listing Hetzner locations.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *locationsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *locationsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config locationsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_hetzner_locations"})

	locations, err := d.client.ListHetznerLocations(ctx, config.CloudProviderTokenUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner locations", err.Error())
		return
	}

	locations = filter.Apply(ctx, locations, config.Filters, func(loc client.HetznerLocation, field string) (string, bool) {
		switch field {
		case "id":
			return filter.Int64ToString(loc.ID), true
		case "name":
			return loc.Name, true
		case "description":
			return loc.Description, true
		case "city":
			return loc.City, true
		case "country":
			return loc.Country, true
		default:
			return "", false
		}
	})

	state := locationsDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Locations:              make([]locationModel, len(locations)),
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
