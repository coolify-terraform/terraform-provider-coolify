package application

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*applicationListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*applicationListDataSource)(nil)
)

// applicationListDataSource is the data source implementation for listing all Coolify applications.
type applicationListDataSource struct {
	client *client.Client
}

// applicationListDataSourceModel maps the data source schema data.
type applicationListDataSourceModel struct {
	Applications []applicationItemModel `tfsdk:"applications"`
	Filters      []filter.Config        `tfsdk:"filter"`
}

// applicationItemModel maps a single application in the list.
type applicationItemModel struct {
	UUID          types.String `tfsdk:"uuid"`
	Name          types.String `tfsdk:"name"`
	FQDN          types.String `tfsdk:"fqdn"`
	GitRepository types.String `tfsdk:"git_repository"`
	GitBranch     types.String `tfsdk:"git_branch"`
	BuildPack     types.String `tfsdk:"build_pack"`
	Status        types.String `tfsdk:"status"`
}

// NewListDataSource returns a new applications list data source instance.
func NewListDataSource() datasource.DataSource {
	return &applicationListDataSource{}
}

func (d *applicationListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_applications"
}

func (d *applicationListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify applications.",
		Attributes: map[string]schema.Attribute{
			"applications": schema.ListNestedAttribute{
				MarkdownDescription: "The list of applications.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the application.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "The name of the application.",
							Computed:            true,
						},
						"fqdn": schema.StringAttribute{
							MarkdownDescription: "The fully qualified domain name of the application.",
							Computed:            true,
						},
						"git_repository": schema.StringAttribute{
							MarkdownDescription: "The Git repository URL of the application.",
							Computed:            true,
						},
						"git_branch": schema.StringAttribute{
							MarkdownDescription: "The Git branch used by the application.",
							Computed:            true,
						},
						"build_pack": schema.StringAttribute{
							MarkdownDescription: "The build pack type used by the application.",
							Computed:            true,
						},
						"status": schema.StringAttribute{
							MarkdownDescription: "The current status of the application.",
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

func (d *applicationListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *applicationListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_applications"})

	apps, err := d.client.ListApplications(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing applications", fmt.Sprintf("Could not list applications: %s", err))
		return
	}

	apps = filter.Apply(apps, config.Filters, func(a client.Application, field string) (string, bool) {
		switch field {
		case "uuid":
			return a.UUID, true
		case "name":
			return a.Name, true
		case "fqdn":
			return a.FQDN, true
		case "git_repository":
			return a.GitRepository, true
		case "git_branch":
			return a.GitBranch, true
		case "build_pack":
			return a.BuildPack, true
		case "status":
			return a.Status, true
		default:
			return "", false
		}
	})

	var state applicationListDataSourceModel
	state.Filters = config.Filters
	for _, a := range apps {
		item := applicationItemModel{
			UUID: types.StringValue(a.UUID),
		}
		item.Name = flex.StringToFramework(a.Name)
		item.FQDN = flex.StringToFramework(a.FQDN)
		item.GitRepository = flex.StringToFramework(a.GitRepository)
		item.GitBranch = flex.StringToFramework(a.GitBranch)
		item.BuildPack = flex.StringToFramework(a.BuildPack)
		item.Status = flex.StringToFramework(a.Status)
		state.Applications = append(state.Applications, item)
	}

	if state.Applications == nil {
		state.Applications = []applicationItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
