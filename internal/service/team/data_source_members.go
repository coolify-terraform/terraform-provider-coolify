package team

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &teamMembersDataSource{}
	_ datasource.DataSourceWithConfigure = &teamMembersDataSource{}
)

type teamMembersDataSource struct {
	client *client.Client
}

type teamMembersDataSourceModel struct {
	ID      types.Int64       `tfsdk:"id"`
	Members []teamMemberModel `tfsdk:"members"`
}

func NewMembersDataSource() datasource.DataSource {
	return &teamMembersDataSource{}
}

func (d *teamMembersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team_members"
}

func (d *teamMembersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list the members of a Coolify team. If no id is given, the current team's members are returned.",
		Attributes: map[string]schema.Attribute{
			"id": schema.Int64Attribute{
				MarkdownDescription: "The numeric ID of the team. If omitted, the current team is used.",
				Optional:            true,
			},
			"members": schema.ListNestedAttribute{
				MarkdownDescription: "The members of the team.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id": schema.Int64Attribute{
							MarkdownDescription: "The numeric ID of the team member.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the team member.",
							Computed:            true,
						},
						"email": schema.StringAttribute{
							MarkdownDescription: "The email address of the team member.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *teamMembersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *teamMembersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config teamMembersDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	var members []client.TeamMember
	var err error

	if config.ID.IsNull() || config.ID.IsUnknown() {
		members, err = d.client.GetCurrentTeamMembers(ctx)
		if err != nil {
			resp.Diagnostics.AddError("Error reading current team members", err.Error())
			return
		}
	} else {
		teamID := int(config.ID.ValueInt64())
		members, err = d.client.GetTeamMembers(ctx, teamID)
		if err != nil {
			resp.Diagnostics.AddError("Error reading team members", err.Error())
			return
		}
	}

	result := make([]teamMemberModel, len(members))
	for i, m := range members {
		result[i] = teamMemberModel{
			ID:    types.Int64Value(int64(m.ID)),
			Name:  types.StringValue(m.Name),
			Email: types.StringValue(m.Email),
		}
	}

	config.Members = result
	if config.Members == nil {
		config.Members = []teamMemberModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
