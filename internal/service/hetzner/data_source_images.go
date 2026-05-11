//nolint:dupl // shared hetzner list data source extraction tracked in #11
package hetzner

import (
	"context"
	"fmt"

	"github.com/SebTardif/terraform-provider-coolify/internal/client"
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
	Images []imageModel `tfsdk:"images"`
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
		MarkdownDescription: "Lists all available Hetzner cloud images.",
		Attributes: map[string]schema.Attribute{
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
	}
}

func (d *imagesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *imagesDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	images, err := d.client.ListHetznerImages(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner images", err.Error())
		return
	}

	state := imagesDataSourceModel{
		Images: make([]imageModel, len(images)),
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
