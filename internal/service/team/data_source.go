package team

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &teamDataSource{}
	_ datasource.DataSourceWithConfigure = &teamDataSource{}
)

type teamDataSource struct{ client *client.Client }
type teamDataSourceModel struct {
	ID          types.Int64       `tfsdk:"id"`
	Name        types.String      `tfsdk:"name"`
	Description types.String      `tfsdk:"description"`
	Members     []teamMemberModel `tfsdk:"members"`
}
type teamMemberModel struct {
	ID    types.Int64  `tfsdk:"id"`
	Name  types.String `tfsdk:"name"`
	Email types.String `tfsdk:"email"`
}

func NewDataSource() datasource.DataSource { return &teamDataSource{} }
func (d *teamDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_team"
}
func (d *teamDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a Coolify team and its members.",
		Attributes: map[string]schema.Attribute{
			"id":          schema.Int64Attribute{MarkdownDescription: "The numeric ID of the team.", Required: true},
			"name":        schema.StringAttribute{MarkdownDescription: "The name of the team.", Computed: true},
			"description": schema.StringAttribute{MarkdownDescription: "The description of the team.", Computed: true},
			"members": schema.ListNestedAttribute{MarkdownDescription: "The members of the team.", Computed: true, NestedObject: schema.NestedAttributeObject{
				Attributes: map[string]schema.Attribute{
					"id":    schema.Int64Attribute{MarkdownDescription: "The numeric ID of the team member.", Computed: true},
					"name":  schema.StringAttribute{MarkdownDescription: "The name of the team member.", Computed: true},
					"email": schema.StringAttribute{MarkdownDescription: "The email address of the team member.", Computed: true},
				},
			}},
		},
	}
}
func (d *teamDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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
func (d *teamDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config teamDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	teamID := int(config.ID.ValueInt64())
	teamResp, err := d.client.GetTeam(ctx, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team", err.Error())
		return
	}
	membersResp, err := d.client.ListTeamMembers(ctx, teamID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading team members", err.Error())
		return
	}
	config.ID = types.Int64Value(int64(teamResp.ID))
	config.Name = types.StringValue(teamResp.Name)
	config.Description = types.StringValue(teamResp.Description)
	members := make([]teamMemberModel, len(membersResp))
	for i, m := range membersResp {
		members[i] = teamMemberModel{ID: types.Int64Value(int64(m.ID)), Name: types.StringValue(m.Name), Email: types.StringValue(m.Email)}
	}
	config.Members = members
	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
