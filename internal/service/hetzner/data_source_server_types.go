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
	_ datasource.DataSource              = (*serverTypesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverTypesDataSource)(nil)
)

type serverTypesDataSource struct {
	client *client.Client
}

type serverTypesDataSourceModel struct {
	ServerTypes []serverTypeModel `tfsdk:"server_types"`
}

type serverTypeModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Cores       types.Int64  `tfsdk:"cores"`
	Memory      types.Int64  `tfsdk:"memory"`
	Disk        types.Int64  `tfsdk:"disk"`
}

// NewServerTypesDataSource returns a new Hetzner server types data source instance.
func NewServerTypesDataSource() datasource.DataSource { return &serverTypesDataSource{} }

func (d *serverTypesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_server_types"
}

func (d *serverTypesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Hetzner server types.",
		Attributes: map[string]schema.Attribute{
			"server_types": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner server types.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{MarkdownDescription: "The numeric ID of the server type.", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "The name of the server type.", Computed: true},
						"description": schema.StringAttribute{MarkdownDescription: "The description of the server type.", Computed: true},
						"cores":       schema.Int64Attribute{MarkdownDescription: "The number of CPU cores.", Computed: true},
						"memory":      schema.Int64Attribute{MarkdownDescription: "The amount of memory in MB.", Computed: true},
						"disk":        schema.Int64Attribute{MarkdownDescription: "The disk size in GB.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *serverTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *serverTypesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	serverTypes, err := d.client.ListHetznerServerTypes(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner server types", err.Error())
		return
	}

	state := serverTypesDataSourceModel{
		ServerTypes: make([]serverTypeModel, len(serverTypes)),
	}
	for i, st := range serverTypes {
		state.ServerTypes[i] = serverTypeModel{
			ID:          types.Int64Value(st.ID),
			Name:        types.StringValue(st.Name),
			Description: types.StringValue(st.Description),
			Cores:       types.Int64Value(st.Cores),
			Memory:      types.Int64Value(st.Memory),
			Disk:        types.Int64Value(st.Disk),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
