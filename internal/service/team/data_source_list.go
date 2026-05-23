package team

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &teamsDataSource{}
	_ datasource.DataSourceWithConfigure = &teamsDataSource{}
)

type teamsDataSource struct {
	client *client.Client
}

type teamsDataSourceModel struct {
	Teams   []teamsItemModel `tfsdk:"teams"`
	Filters []filter.Config  `tfsdk:"filter"`
}

type teamsItemModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

func NewListDataSource() datasource.DataSource {
	return &teamsDataSource{}
}

func (d *teamsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_teams"
}

func (d *teamsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify teams.",
		Attributes: map[string]schema.Attribute{
			"teams": schema.ListNestedAttribute{
				MarkdownDescription: "The list of teams.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The numeric ID of the team.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the team.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "The description of the team.",
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

func (d *teamsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *teamsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config teamsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_teams"})

	teams, err := d.client.ListTeams(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing teams", err.Error())
		return
	}

	teams = filter.Apply(ctx, teams, config.Filters, func(t client.Team, field string) (string, bool) {
		switch field {
		case "id":
			return filter.IntToString(t.ID), true
		case "name":
			return t.Name, true
		case "description":
			return t.Description, true
		default:
			return "", false
		}
	})

	var state teamsDataSourceModel
	state.Filters = config.Filters
	for _, t := range teams {
		state.Teams = append(state.Teams, teamsItemModel{
			ID:          types.Int64Value(int64(t.ID)),
			Name:        types.StringValue(t.Name),
			Description: types.StringValue(t.Description),
		})
	}

	if state.Teams == nil {
		state.Teams = []teamsItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
