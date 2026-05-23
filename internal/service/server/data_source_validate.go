package server

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = &serverValidateDataSource{}
	_ datasource.DataSourceWithConfigure = &serverValidateDataSource{}
)

type serverValidateDataSource struct {
	client *client.Client
}

type serverValidateDataSourceModel struct {
	UUID    types.String `tfsdk:"uuid"`
	Valid   types.Bool   `tfsdk:"valid"`
	Message types.String `tfsdk:"message"`
}

// NewValidateDataSource returns a new server validation data source.
func NewValidateDataSource() datasource.DataSource {
	return &serverValidateDataSource{}
}

func (d *serverValidateDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_validation"
}

func (d *serverValidateDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Validates the connectivity of a Coolify server.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to validate.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"valid": schema.BoolAttribute{
				MarkdownDescription: "Whether the server connectivity check passed.",
				Computed:            true,
			},
			"message": schema.StringAttribute{
				MarkdownDescription: "A message describing the validation result.",
				Computed:            true,
			},
		},
	}
}

func (d *serverValidateDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *serverValidateDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config serverValidateDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_server_validation"})

	v, err := d.client.ValidateServer(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error validating server", err.Error())
		return
	}

	config.Valid = types.BoolValue(v.Valid)
	config.Message = types.StringValue(v.Message)

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
