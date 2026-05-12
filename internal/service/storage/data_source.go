package storage

import (
	"context"
	"fmt"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*storageListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*storageListDataSource)(nil)
)

type storageListDataSource struct {
	client *client.Client
}

type storageListModel struct {
	ApplicationUUID types.String       `tfsdk:"application_uuid"`
	ServiceUUID     types.String       `tfsdk:"service_uuid"`
	DatabaseUUID    types.String       `tfsdk:"database_uuid"`
	Storages        []storageItemModel `tfsdk:"storages"`
}

type storageItemModel struct {
	UUID      types.String `tfsdk:"uuid"`
	Name      types.String `tfsdk:"name"`
	MountPath types.String `tfsdk:"mount_path"`
	HostPath  types.String `tfsdk:"host_path"`
}

// NewListDataSource returns a new storageListDataSource instance.
func NewListDataSource() datasource.DataSource { return &storageListDataSource{} }

func (d *storageListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_storages"
}

func (d *storageListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all persistent storages for a Coolify application, service, or database.",
		Attributes: map[string]schema.Attribute{
			"application_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the application. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
				Validators: []validator.String{
					stringvalidator.ExactlyOneOf(
						path.MatchRoot("service_uuid"),
						path.MatchRoot("database_uuid"),
					),
				},
			},
			"service_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
			},
			"database_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the database. Exactly one of `application_uuid`, `service_uuid`, or `database_uuid` must be provided.",
				Optional:            true,
			},
			"storages": schema.ListNestedAttribute{
				MarkdownDescription: "The list of persistent storages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid":       schema.StringAttribute{MarkdownDescription: "The UUID of the persistent storage.", Computed: true},
						"name":       schema.StringAttribute{MarkdownDescription: "The name of the persistent storage.", Computed: true},
						"mount_path": schema.StringAttribute{MarkdownDescription: "The mount path inside the container.", Computed: true},
						"host_path":  schema.StringAttribute{MarkdownDescription: "The host path (empty for Docker volumes).", Computed: true},
					},
				},
			},
		},
	}
}

func (d *storageListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	if req.ProviderData == nil {
		return
	}
	c, ok := req.ProviderData.(*client.Client)
	if !ok {
		resp.Diagnostics.AddError("Unexpected Data Source Configure Type", fmt.Sprintf("Expected *client.Client, got: %T", req.ProviderData))
		return
	}
	d.client = c
}

func (d *storageListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config storageListModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

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
		resp.Diagnostics.AddError("Configuration Error", "One of application_uuid, service_uuid, or database_uuid must be set")
		return
	}

	storages, err := d.client.ListStorages(ctx, parentType, parentUUID)
	if err != nil {
		resp.Diagnostics.AddError("Error listing persistent storages", err.Error())
		return
	}

	items := make([]storageItemModel, len(storages))
	for i, s := range storages {
		items[i] = storageItemModel{
			UUID:      types.StringValue(s.UUID),
			Name:      types.StringValue(s.Name),
			MountPath: types.StringValue(s.MountPath),
			HostPath:  types.StringValue(s.HostPath),
		}
	}
	config.Storages = items

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
