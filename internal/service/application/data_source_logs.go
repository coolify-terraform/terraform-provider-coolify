package application

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
	"github.com/SebTardif/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = &applicationLogsDataSource{}
	_ datasource.DataSourceWithConfigure = &applicationLogsDataSource{}
)

type applicationLogsDataSource struct {
	client *client.Client
}

type applicationLogsDataSourceModel struct {
	UUID types.String          `tfsdk:"uuid"`
	Logs []applicationLogModel `tfsdk:"logs"`
}

type applicationLogModel struct {
	Line      types.String `tfsdk:"line"`
	Timestamp types.String `tfsdk:"timestamp"`
}

// NewLogsDataSource returns a new application logs data source.
func NewLogsDataSource() datasource.DataSource {
	return &applicationLogsDataSource{}
}

func (d *applicationLogsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_application_logs"
}

func (d *applicationLogsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves log lines for a Coolify application.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"logs": schema.ListNestedAttribute{
				MarkdownDescription: "The list of log lines from the application.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"line": schema.StringAttribute{
							MarkdownDescription: "The log line content.",
							Computed:            true,
						},
						"timestamp": schema.StringAttribute{
							MarkdownDescription: "The timestamp of the log line.",
							Computed:            true,
						},
					},
				},
			},
		},
	}
}

func (d *applicationLogsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *applicationLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationLogsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	logs, err := d.client.GetApplicationLogs(ctx, config.UUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error reading application logs", err.Error())
		return
	}

	for _, l := range logs {
		config.Logs = append(config.Logs, applicationLogModel{
			Line:      types.StringValue(l.Line),
			Timestamp: types.StringValue(l.Timestamp),
		})
	}

	if config.Logs == nil {
		config.Logs = []applicationLogModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
