package backup

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*executionsDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*executionsDataSource)(nil)
)

type executionsDataSource struct {
	client *client.Client
}

type executionsDataSourceModel struct {
	DatabaseUUID types.String     `tfsdk:"database_uuid"`
	BackupUUID   types.String     `tfsdk:"backup_uuid"`
	Executions   []executionModel `tfsdk:"executions"`
	Filters      []filter.Config  `tfsdk:"filter"`
}

type executionModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Status    types.String `tfsdk:"status"`
	CreatedAt types.String `tfsdk:"created_at"`
	Size      types.Int64  `tfsdk:"size"`
}

// NewExecutionsDataSource returns a new backup executions data source instance.
func NewExecutionsDataSource() datasource.DataSource { return &executionsDataSource{} }

func (d *executionsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_backup_executions"
}

func (d *executionsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all executions for a database backup.",
		Attributes: map[string]schema.Attribute{
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database.",
				Required:            true,
			},
			"backup_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the backup configuration.",
				Required:            true,
			},
			"executions": schema.ListNestedAttribute{
				MarkdownDescription: "The list of backup executions.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{MarkdownDescription: "The UUID of the execution.", Computed: true},
						"status":     schema.StringAttribute{MarkdownDescription: "The status of the execution.", Computed: true},
						"created_at": schema.StringAttribute{MarkdownDescription: "The creation timestamp.", Computed: true},
						"size":       schema.Int64Attribute{MarkdownDescription: "The size of the backup in bytes.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *executionsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *executionsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config executionsDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_backup_executions"})

	execs, err := d.client.ListBackupExecutions(ctx, config.DatabaseUUID.ValueString(), config.BackupUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing backup executions", err.Error())
		return
	}

	execs = filter.Apply(ctx, execs, config.Filters, func(e client.BackupExecution, field string) (string, bool) {
		switch field {
		case "uuid":
			return e.UUID, true
		case "status":
			return e.Status, true
		case "created_at":
			return e.CreatedAt, true
		case "size":
			return filter.Int64ToString(e.Size), true
		default:
			return "", false
		}
	})

	items := make([]executionModel, len(execs))
	for i, e := range execs {
		items[i] = executionModel{
			UUID:      types.StringValue(e.UUID),
			Status:    types.StringValue(e.Status),
			CreatedAt: types.StringValue(e.CreatedAt),
			Size:      types.Int64Value(e.Size),
		}
	}
	config.Executions = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
