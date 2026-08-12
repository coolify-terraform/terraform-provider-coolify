package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*s3StorageListDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*s3StorageListDataSource)(nil)
)

// s3StorageListDataSource lists all Coolify S3 storages.
type s3StorageListDataSource struct {
	client *client.Client
}

// s3StorageListDataSourceModel maps the list data source schema data.
type s3StorageListDataSourceModel struct {
	S3Storages []s3StorageItemModel `tfsdk:"s3_storages"`
	Filters    []filter.Config      `tfsdk:"filter"`
}

// s3StorageItemModel maps a single S3 storage in the list.
type s3StorageItemModel struct {
	UUID        types.String `tfsdk:"uuid"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
	Endpoint    types.String `tfsdk:"endpoint"`
	Bucket      types.String `tfsdk:"bucket"`
	Region      types.String `tfsdk:"region"`
	IsUsable    types.Bool   `tfsdk:"is_usable"`
}

// NewListDataSource returns a new S3 storages list data source instance.
func NewListDataSource() datasource.DataSource {
	return &s3StorageListDataSource{}
}

func (d *s3StorageListDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storages"
}

func (d *s3StorageListDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to list all Coolify S3 storages. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"s3_storages": schema.ListNestedAttribute{
				MarkdownDescription: "The list of S3 storages.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"uuid": schema.StringAttribute{
							MarkdownDescription: "The unique identifier of the S3 storage.",
							Computed:            true,
						},
						"name": schema.StringAttribute{
							MarkdownDescription: "A friendly name for the S3 storage.",
							Computed:            true,
						},
						"description": schema.StringAttribute{
							MarkdownDescription: "Description of the S3 storage.",
							Computed:            true,
						},
						"endpoint": schema.StringAttribute{
							MarkdownDescription: "S3-compatible endpoint URL.",
							Computed:            true,
						},
						"bucket": schema.StringAttribute{
							MarkdownDescription: "S3 bucket name.",
							Computed:            true,
						},
						"region": schema.StringAttribute{
							MarkdownDescription: "S3 region.",
							Computed:            true,
						},
						"is_usable": schema.BoolAttribute{
							MarkdownDescription: "Whether Coolify marks this storage as usable.",
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

func (d *s3StorageListDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *s3StorageListDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config s3StorageListDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_s3_storages"})

	storages, err := d.client.ListS3Storages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing S3 storages", fmt.Sprintf("Could not list s3 storages: %s", err))
		return
	}

	storages = filter.Apply(ctx, storages, config.Filters, func(s client.S3Storage, field string) (string, bool) {
		switch field {
		case "uuid":
			return s.UUID, true
		case "name":
			return s.Name, true
		case "endpoint":
			return s.Endpoint, true
		case "bucket":
			return s.Bucket, true
		case "region":
			return s.Region, true
		case "is_usable":
			usable := false
			if s.IsUsable != nil {
				usable = *s.IsUsable
			}
			return filter.BoolToString(usable), true
		default:
			return "", false
		}
	})

	var state s3StorageListDataSourceModel
	state.Filters = config.Filters
	for _, s := range storages {
		item := s3StorageItemModel{
			UUID:        types.StringValue(s.UUID),
			Name:        types.StringValue(s.Name),
			Description: flex.StringToFramework(s.Description),
			Endpoint:    types.StringValue(s.Endpoint),
			Bucket:      types.StringValue(s.Bucket),
			Region:      types.StringValue(s.Region),
		}
		if s.IsUsable != nil {
			item.IsUsable = types.BoolValue(*s.IsUsable)
		} else {
			item.IsUsable = types.BoolValue(false)
		}
		state.S3Storages = append(state.S3Storages, item)
	}

	if state.S3Storages == nil {
		state.S3Storages = []s3StorageItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
