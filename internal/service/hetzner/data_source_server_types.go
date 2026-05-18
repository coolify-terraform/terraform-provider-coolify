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
	_ datasource.DataSource              = (*serverTypesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*serverTypesDataSource)(nil)
)

type serverTypesDataSource struct {
	client *client.Client
}

type serverTypesDataSourceModel struct {
	CloudProviderTokenUUID types.String      `tfsdk:"cloud_provider_token_uuid"`
	ServerTypes            []serverTypeModel `tfsdk:"server_types"`
	Filters                []filter.Config   `tfsdk:"filter"`
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
		MarkdownDescription: "Lists all available Hetzner server types for a given cloud provider token.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the cloud provider token to use for listing Hetzner server types.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *serverTypesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *serverTypesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverTypesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_hetzner_server_types"})

	serverTypes, err := d.client.ListHetznerServerTypes(ctx, config.CloudProviderTokenUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner server types", err.Error())
		return
	}

	serverTypes = filter.Apply(ctx, serverTypes, config.Filters, func(st client.HetznerServerType, field string) (string, bool) {
		switch field {
		case "id":
			return filter.Int64ToString(st.ID), true
		case "name":
			return st.Name, true
		case "description":
			return st.Description, true
		case "cores":
			return filter.Int64ToString(st.Cores), true
		case "memory":
			return filter.Int64ToString(st.Memory), true
		case "disk":
			return filter.Int64ToString(st.Disk), true
		default:
			return "", false
		}
	})

	state := serverTypesDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		ServerTypes:            make([]serverTypeModel, len(serverTypes)),
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
