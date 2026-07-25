package digitalocean

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

type imagesDataSource struct{ client *client.Client }

type imagesDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Images                 []imageModel    `tfsdk:"images"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type imageModel struct {
	ID           types.Int64  `tfsdk:"id"`
	Name         types.String `tfsdk:"name"`
	Distribution types.String `tfsdk:"distribution"`
	Slug         types.String `tfsdk:"slug"`
	Public       types.Bool   `tfsdk:"public"`
	Type         types.String `tfsdk:"type"`
}

func NewImagesDataSource() datasource.DataSource { return &imagesDataSource{} }

func (d *imagesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digitalocean_images"
}

func (d *imagesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists DigitalOcean images for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("images"),
			"images": schema.ListNestedAttribute{
				MarkdownDescription: "The list of DigitalOcean images.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":           schema.Int64Attribute{MarkdownDescription: "Numeric image ID.", Computed: true},
						"name":         schema.StringAttribute{MarkdownDescription: "Image name.", Computed: true},
						"distribution": schema.StringAttribute{MarkdownDescription: "OS distribution.", Computed: true},
						"slug":         schema.StringAttribute{MarkdownDescription: "Image slug.", Computed: true},
						"public":       schema.BoolAttribute{MarkdownDescription: "Whether the image is public.", Computed: true},
						"type":         schema.StringAttribute{MarkdownDescription: "Image type.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDigitalOceanDataSourceClient(req, resp)
}

func (d *imagesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config imagesDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	images, ok := readFilteredTokenList(
		ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_digitalocean_images", "Error listing DigitalOcean images", resp, d.client,
		d.client.ListDigitalOceanImages,
		func(img client.DigitalOceanImage, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(img.ID), true
			case "name":
				return img.Name, true
			case "slug":
				return img.Slug, true
			case "distribution":
				return img.Distribution, true
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
			ID: types.Int64Value(img.ID), Name: types.StringValue(img.Name), Distribution: types.StringValue(img.Distribution),
			Slug: types.StringValue(img.Slug), Public: types.BoolValue(img.Public), Type: types.StringValue(img.Type),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
