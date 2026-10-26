package serviceapplication

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

type listDataSource struct{ client *client.Client }

type listModel struct {
	ServiceUUID  types.String    `tfsdk:"service_uuid"`
	Applications []listItemModel `tfsdk:"applications"`
	Filters      []filter.Config `tfsdk:"filter"`
}

type listItemModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	HumanName types.String `tfsdk:"human_name"`
	FQDN      types.String `tfsdk:"fqdn"`
	Status    types.String `tfsdk:"status"`
}

func NewListDataSource() datasource.DataSource { return &listDataSource{} }

func (d *listDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service_applications"
}

func (d *listDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists applications inside one compose service. Use an application's `uuid` with `coolify_service_application`.",
		Attributes: map[string]schema.Attribute{
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the parent service.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"applications": schema.ListNestedAttribute{
				MarkdownDescription: "Compose applications on the service.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"uuid":       schema.StringAttribute{MarkdownDescription: "Service application UUID.", Computed: true},
					"name":       schema.StringAttribute{MarkdownDescription: "Compose service name.", Computed: true},
					"human_name": schema.StringAttribute{MarkdownDescription: "Display name.", Computed: true},
					"fqdn":       schema.StringAttribute{MarkdownDescription: "Stored FQDN.", Computed: true},
					"status":     schema.StringAttribute{MarkdownDescription: "Runtime status.", Computed: true},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *listDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *listDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config listModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, err := d.client.ListServiceApplications(ctx, config.ServiceUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing service applications", err.Error())
		return
	}
	filtered := filter.Apply(ctx, items, config.Filters, func(item client.ServiceApplication, field string) (string, bool) {
		switch field {
		case "uuid":
			return item.UUID, true
		case "name":
			return item.Name, true
		case "human_name":
			return item.HumanName, true
		case "status":
			return item.Status, true
		default:
			return "", false
		}
	})
	state := listModel{ServiceUUID: config.ServiceUUID, Filters: config.Filters, Applications: make([]listItemModel, len(filtered))}
	for i, item := range filtered {
		state.Applications[i] = listItemModel{
			UUID: types.StringValue(item.UUID), Name: types.StringValue(item.Name),
			HumanName: types.StringValue(item.HumanName), FQDN: types.StringValue(item.FQDN), Status: types.StringValue(item.Status),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
