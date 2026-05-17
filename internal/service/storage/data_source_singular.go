package storage

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*storageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageDataSource)(nil)
)

type storageDataSource struct {
	client *client.Client
}

type storageDataSourceModel struct {
	UUID            types.String `tfsdk:"uuid"`
	ApplicationUUID types.String `tfsdk:"application_uuid"`
	ServiceUUID     types.String `tfsdk:"service_uuid"`
	DatabaseUUID    types.String `tfsdk:"database_uuid"`
	Name            types.String `tfsdk:"name"`
	MountPath       types.String `tfsdk:"mount_path"`
	HostPath        types.String `tfsdk:"host_path"`
}

// NewDataSource returns a new singular storage data source.
func NewDataSource() datasource.DataSource { return &storageDataSource{} }

func (d *storageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storage"
}

func (d *storageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Retrieves a single persistent storage by UUID from a Coolify application, service, or database.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the persistent storage.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators: []validator.String{
					validate.UUID(),
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
						path.MatchRoot("database_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators:          []validator.String{validate.UUID()},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the persistent storage.",
				Computed:            true,
			},
			"mount_path": schema.StringAttribute{
				MarkdownDescription: "The mount path inside the container.",
				Computed:            true,
			},
			"host_path": schema.StringAttribute{
				MarkdownDescription: "The host path (empty for Docker volumes).",
				Computed:            true,
			},
		},
	}
}

func (d *storageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *storageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config storageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_storage"})

	var parentType, parentUUID string
	//nolint:gocritic // if-else chain dispatches to different parent types; switch not applicable
	if !config.ApplicationUUID.IsNull() {
		parentType = "applications"
		parentUUID = config.ApplicationUUID.ValueString()
	} else if !config.ServiceUUID.IsNull() {
		parentType = "services"
		parentUUID = config.ServiceUUID.ValueString()
	} else if !config.DatabaseUUID.IsNull() {
		parentType = "databases"
		parentUUID = config.DatabaseUUID.ValueString()
	} else {
		resp.Diagnostics.AddError("Configuration error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	storages, err := d.client.ListStorages(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error reading storage", err.Error())
		return
	}

	uuid := config.UUID.ValueString()
	for _, s := range storages {
		if s.UUID == uuid {
			config.Name = types.StringValue(s.Name)
			config.MountPath = types.StringValue(s.MountPath)
			config.HostPath = types.StringValue(s.HostPath)
			resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
			return
		}
	}

	resp.Diagnostics.AddError("Error reading storage",
		fmt.Sprintf("Storage with UUID %q not found", uuid))
}
