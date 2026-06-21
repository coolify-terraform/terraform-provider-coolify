//nolint:dupl // schema and state mapping differ; list/filter logic is in data_source_common.go
package hetzner

import (
	"context"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

var (
	_ datasource.DataSource              = (*imagesDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*imagesDataSource)(nil)
)

type imagesDataSource struct {
	client *client.Client
}

type imagesDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Images                 []imageModel    `tfsdk:"images"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type imageModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Description types.String `tfsdk:"description"`
}

// NewImagesDataSource returns a new Hetzner images data source instance.
func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_images"
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Hetzner cloud images for a given cloud provider token.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("images"),
			"images": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner images.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{MarkdownDescription: "The numeric ID of the image.", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "The name of the image.", Computed: true},
						"description": schema.StringAttribute{MarkdownDescription: "The description of the image.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureHetznerDataSourceClient(req, resp)
}

func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config imagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	images, ok := readFilteredTokenList(
		ctx,
		config.CloudProviderTokenUUID.ValueString(),
		config.Filters,
		"coolify_hetzner_images",
		"Error listing Hetzner images",
		resp,
		d.client,
		d.client.ListHetznerImages,
		func(img client.HetznerImage, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(img.ID), true
			case "name":
				return img.Name, true
			case "description":
				return img.Description, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}

	state := imagesDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		Images:                 make([]imageModel, len(images)),
	}
	for i, img := range images {
		state.Images[i] = imageModel{
			ID:          types.Int64Value(img.ID),
			Name:        types.StringValue(img.Name),
			Description: types.StringValue(img.Description),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
