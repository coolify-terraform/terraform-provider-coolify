package application

import (
	"context"
	"strings"

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
	_ datasource.DataSource              = &applicationLogsDataSource{}
	_ datasource.DataSourceWithConfigure = &applicationLogsDataSource{}
)

type applicationLogsDataSource struct {
	client *client.Client
}

type applicationLogsDataSourceModel struct {
	UUID    types.String          `tfsdk:"uuid"`
	Logs    []applicationLogModel `tfsdk:"logs"`
	Filters []filter.Config       `tfsdk:"filter"`
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *applicationLogsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *applicationLogsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config applicationLogsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_application_logs"})

	logs, err := d.client.GetApplicationLogs(ctx, config.UUID.ValueString())
	if err != nil {
		// 400: container not running. Timeout: container does not exist
		// and Coolify hangs waiting for docker logs. Both are expected
		// for applications that have not been deployed yet.
		errMsg := err.Error()
		if strings.Contains(errMsg, "status 400") ||
			strings.Contains(errMsg, "context deadline exceeded") ||
			strings.Contains(errMsg, "giving up") {
			config.Logs = []applicationLogModel{}
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
		resp.Diagnostics.AddError("Error reading application logs", errMsg)
		return
	}

	logs = filter.Apply(ctx, logs, config.Filters, func(l client.ApplicationLog, field string) (string, bool) {
		switch field {
		case "line":
			return l.Line, true
		case "timestamp":
			return l.Timestamp, true
		default:
			return "", false
		}
	})

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
