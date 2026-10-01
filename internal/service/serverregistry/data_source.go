package serverregistry

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*dataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*dataSource)(nil)
)

type dataSource struct{ client *client.Client }

type dataModel struct {
	ServerUUID types.String `tfsdk:"server_uuid"`
	Registries []loginModel `tfsdk:"registries"`
}

type loginModel struct {
	Registry types.String `tfsdk:"registry"`
	LoggedIn types.Bool   `tfsdk:"logged_in"`
	Username types.String `tfsdk:"username"`
	Source   types.String `tfsdk:"source"`
}

func NewDataSource() datasource.DataSource { return &dataSource{} }

func (d *dataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_docker_registries"
}

func (d *dataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Docker registry logins Coolify can see on a server. Requires Coolify 4.4 tip (not v4.3.23 and not v4.4-rc.1). Passwords are not returned.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "Server UUID.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"registries": schema.ListNestedAttribute{
				MarkdownDescription: "Registry rows, including registries used by resources that are not logged in.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"registry":  schema.StringAttribute{MarkdownDescription: "Registry host.", Computed: true},
					"logged_in": schema.BoolAttribute{MarkdownDescription: "Whether the server has saved credentials for this registry.", Computed: true},
					"username":  schema.StringAttribute{MarkdownDescription: "Saved username. Empty when the server is not logged in.", Computed: true},
					"source":    schema.StringAttribute{MarkdownDescription: "Where Coolify found the login.", Computed: true},
				}},
			},
		},
	}
}

func (d *dataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *dataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var cfg dataModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &cfg)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if err := requireRegistryAPI(d.client); err != nil {
		resp.Diagnostics.AddError("Docker registry list requires Coolify 4.4", err.Error())
		return
	}
	rows, err := d.client.ListServerDockerRegistries(ctx, cfg.ServerUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing Docker registries", fmt.Sprintf("server %s: %s", cfg.ServerUUID.ValueString(), err))
		return
	}
	cfg.Registries = make([]loginModel, 0, len(rows))
	for _, row := range rows {
		cfg.Registries = append(cfg.Registries, loginModel{
			Registry: types.StringValue(row.Registry),
			LoggedIn: types.BoolValue(row.LoggedIn),
			Username: flex.StringToFramework(row.Username),
			Source:   flex.StringToFramework(row.Source),
		})
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &cfg)...)
}
