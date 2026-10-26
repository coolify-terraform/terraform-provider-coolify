package hostinger

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/filter"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func tokenAttribute(label string) schema.StringAttribute {
	return schema.StringAttribute{
		MarkdownDescription: fmt.Sprintf("UUID of a Hostinger cloud provider token used to list %s.", label),
		Required:            true,
		Validators:          []validator.String{validate.UUID()},
	}
}

func configure(req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) *client.Client {
	return flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

type dataCentersDataSource struct{ client *client.Client }

type dataCentersModel struct {
	CloudProviderTokenUUID types.String      `tfsdk:"cloud_provider_token_uuid"`
	DataCenters            []dataCenterModel `tfsdk:"data_centers"`
	Filters                []filter.Config   `tfsdk:"filter"`
}

type dataCenterModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Location  types.String `tfsdk:"location"`
	City      types.String `tfsdk:"city"`
	Continent types.String `tfsdk:"continent"`
}

func NewDataCentersDataSource() datasource.DataSource { return &dataCentersDataSource{} }

func (d *dataCentersDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostinger_data_centers"
}

func (d *dataCentersDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Hostinger VPS data centers for a cloud provider token. Requires Coolify >= v4.5.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": tokenAttribute("data centers"),
			"data_centers": schema.ListNestedAttribute{
				MarkdownDescription: "Hostinger data centers.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":        schema.Int64Attribute{MarkdownDescription: "Data center id.", Computed: true},
					"name":      schema.StringAttribute{MarkdownDescription: "Data center name.", Computed: true},
					"location":  schema.StringAttribute{MarkdownDescription: "Location label.", Computed: true},
					"city":      schema.StringAttribute{MarkdownDescription: "City.", Computed: true},
					"continent": schema.StringAttribute{MarkdownDescription: "Continent.", Computed: true},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *dataCentersDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configure(req, resp)
}

func (d *dataCentersDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config dataCentersModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_hostinger_data_centers", "Error listing Hostinger data centers", resp, d.client.ListHostingerDataCenters,
		func(item client.HostingerDataCenter, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(item.ID), true
			case "name":
				return item.Name, true
			case "location":
				return item.Location, true
			case "city":
				return item.City, true
			case "continent":
				return item.Continent, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := dataCentersModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, DataCenters: make([]dataCenterModel, len(items))}
	for i, item := range items {
		state.DataCenters[i] = dataCenterModel{
			ID: types.Int64Value(item.ID), Name: types.StringValue(item.Name), Location: types.StringValue(item.Location),
			City: types.StringValue(item.City), Continent: types.StringValue(item.Continent),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type catalogDataSource struct{ client *client.Client }

type catalogModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Items                  []catalogItem   `tfsdk:"items"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type catalogItem struct {
	ID       types.String        `tfsdk:"id"`
	Name     types.String        `tfsdk:"name"`
	Category types.String        `tfsdk:"category"`
	Prices   []catalogPriceModel `tfsdk:"prices"`
}

type catalogPriceModel struct {
	ID               types.String `tfsdk:"id"`
	Name             types.String `tfsdk:"name"`
	Currency         types.String `tfsdk:"currency"`
	Price            types.Int64  `tfsdk:"price"`
	FirstPeriodPrice types.Int64  `tfsdk:"first_period_price"`
	Period           types.Int64  `tfsdk:"period"`
	PeriodUnit       types.String `tfsdk:"period_unit"`
}

func NewCatalogDataSource() datasource.DataSource { return &catalogDataSource{} }

func (d *catalogDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostinger_catalog"
}

func (d *catalogDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Hostinger KVM VPS plans Coolify can purchase. Prices are integer cents. Requires Coolify >= v4.5.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": tokenAttribute("catalog items"),
			"items": schema.ListNestedAttribute{
				MarkdownDescription: "Catalog items whose id matches a KVM VPS plan.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":       schema.StringAttribute{MarkdownDescription: "Catalog item id. Pass this to `coolify_server_hostinger.item_id`.", Computed: true},
					"name":     schema.StringAttribute{MarkdownDescription: "Plan name.", Computed: true},
					"category": schema.StringAttribute{MarkdownDescription: "Hostinger category. Coolify requests VPS.", Computed: true},
					"prices": schema.ListNestedAttribute{
						MarkdownDescription: "Billing prices. `price` and `first_period_price` are cents.",
						Computed:            true,
						NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
							"id":                 schema.StringAttribute{MarkdownDescription: "Price id.", Computed: true},
							"name":               schema.StringAttribute{MarkdownDescription: "Price name.", Computed: true},
							"currency":           schema.StringAttribute{MarkdownDescription: "Currency code.", Computed: true},
							"price":              schema.Int64Attribute{MarkdownDescription: "Recurring price in cents.", Computed: true},
							"first_period_price": schema.Int64Attribute{MarkdownDescription: "First period price in cents.", Computed: true},
							"period":             schema.Int64Attribute{MarkdownDescription: "Billing period length.", Computed: true},
							"period_unit":        schema.StringAttribute{MarkdownDescription: "Period unit, such as month.", Computed: true},
						}},
					},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *catalogDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configure(req, resp)
}

func (d *catalogDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config catalogModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_hostinger_catalog", "Error listing Hostinger catalog", resp, d.client.ListHostingerCatalog,
		func(item client.HostingerCatalogItem, field string) (string, bool) {
			switch field {
			case "id":
				return item.ID, true
			case "name":
				return item.Name, true
			case "category":
				return item.Category, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := catalogModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, Items: make([]catalogItem, len(items))}
	for i, item := range items {
		prices := make([]catalogPriceModel, len(item.Prices))
		for j, price := range item.Prices {
			prices[j] = catalogPriceModel{
				ID: types.StringValue(price.ID), Name: types.StringValue(price.Name), Currency: types.StringValue(price.Currency),
				Price: types.Int64Value(price.Price), FirstPeriodPrice: types.Int64Value(price.FirstPeriodPrice),
				Period: types.Int64Value(price.Period), PeriodUnit: types.StringValue(price.PeriodUnit),
			}
		}
		state.Items[i] = catalogItem{ID: types.StringValue(item.ID), Name: types.StringValue(item.Name), Category: types.StringValue(item.Category), Prices: prices}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type templatesDataSource struct{ client *client.Client }

type templatesModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Templates              []templateModel `tfsdk:"templates"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type templateModel struct {
	ID            types.Int64  `tfsdk:"id"`
	Name          types.String `tfsdk:"name"`
	Description   types.String `tfsdk:"description"`
	Documentation types.String `tfsdk:"documentation"`
}

func NewTemplatesDataSource() datasource.DataSource { return &templatesDataSource{} }

func (d *templatesDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostinger_templates"
}

func (d *templatesDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Hostinger OS templates. Requires Coolify >= v4.5.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": tokenAttribute("templates"),
			"templates": schema.ListNestedAttribute{
				MarkdownDescription: "OS templates.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":            schema.Int64Attribute{MarkdownDescription: "Template id. Pass this to `coolify_server_hostinger.template_id`.", Computed: true},
					"name":          schema.StringAttribute{MarkdownDescription: "Template name.", Computed: true},
					"description":   schema.StringAttribute{MarkdownDescription: "Template description.", Computed: true},
					"documentation": schema.StringAttribute{MarkdownDescription: "Documentation URL or text from Hostinger.", Computed: true},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *templatesDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configure(req, resp)
}

func (d *templatesDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config templatesModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_hostinger_templates", "Error listing Hostinger templates", resp, d.client.ListHostingerTemplates,
		func(item client.HostingerTemplate, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(item.ID), true
			case "name":
				return item.Name, true
			case "description":
				return item.Description, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := templatesModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, Templates: make([]templateModel, len(items))}
	for i, item := range items {
		state.Templates[i] = templateModel{ID: types.Int64Value(item.ID), Name: types.StringValue(item.Name), Description: types.StringValue(item.Description), Documentation: types.StringValue(item.Documentation)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type sshKeysDataSource struct{ client *client.Client }

type sshKeysModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	SSHKeys                []sshKeyModel   `tfsdk:"ssh_keys"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type sshKeyModel struct {
	ID   types.Int64  `tfsdk:"id"`
	Name types.String `tfsdk:"name"`
	Key  types.String `tfsdk:"key"`
}

func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostinger_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists SSH public keys stored in the Hostinger account. Requires Coolify >= v4.5.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": tokenAttribute("SSH keys"),
			"ssh_keys": schema.ListNestedAttribute{
				MarkdownDescription: "Hostinger public keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":   schema.Int64Attribute{MarkdownDescription: "Public key id. Pass this in `coolify_server_hostinger.public_key_ids`.", Computed: true},
					"name": schema.StringAttribute{MarkdownDescription: "Key name.", Computed: true},
					"key":  schema.StringAttribute{MarkdownDescription: "Public key material.", Computed: true},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configure(req, resp)
}

func (d *sshKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sshKeysModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_hostinger_ssh_keys", "Error listing Hostinger SSH keys", resp, d.client.ListHostingerSSHKeys,
		func(item client.HostingerPublicKey, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(item.ID), true
			case "name":
				return item.Name, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := sshKeysModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, SSHKeys: make([]sshKeyModel, len(items))}
	for i, item := range items {
		state.SSHKeys[i] = sshKeyModel{ID: types.Int64Value(item.ID), Name: types.StringValue(item.Name), Key: types.StringValue(item.Key)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

type scriptsDataSource struct{ client *client.Client }

type scriptsModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	Scripts                []scriptModel   `tfsdk:"scripts"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type scriptModel struct {
	ID        types.Int64  `tfsdk:"id"`
	Name      types.String `tfsdk:"name"`
	Content   types.String `tfsdk:"content"`
	CreatedAt types.String `tfsdk:"created_at"`
	UpdatedAt types.String `tfsdk:"updated_at"`
}

func NewPostInstallScriptsDataSource() datasource.DataSource { return &scriptsDataSource{} }

func (d *scriptsDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hostinger_post_install_scripts"
}

func (d *scriptsDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Hostinger post-install scripts. Requires Coolify >= v4.5.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": tokenAttribute("post-install scripts"),
			"scripts": schema.ListNestedAttribute{
				MarkdownDescription: "Post-install scripts.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
					"id":         schema.Int64Attribute{MarkdownDescription: "Script id. Pass this to `coolify_server_hostinger.post_install_script_id`.", Computed: true},
					"name":       schema.StringAttribute{MarkdownDescription: "Script name.", Computed: true},
					"content":    schema.StringAttribute{MarkdownDescription: "Script content.", Computed: true},
					"created_at": schema.StringAttribute{MarkdownDescription: "When Hostinger created the script.", Computed: true},
					"updated_at": schema.StringAttribute{MarkdownDescription: "When Hostinger last updated the script.", Computed: true},
				}},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *scriptsDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configure(req, resp)
}

func (d *scriptsDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config scriptsModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := flex.ReadFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_hostinger_post_install_scripts", "Error listing Hostinger post-install scripts", resp, d.client.ListHostingerPostInstallScripts,
		func(item client.HostingerPostInstallScript, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(item.ID), true
			case "name":
				return item.Name, true
			default:
				return "", false
			}
		})
	if !ok {
		return
	}
	state := scriptsModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, Scripts: make([]scriptModel, len(items))}
	for i, item := range items {
		state.Scripts[i] = scriptModel{
			ID: types.Int64Value(item.ID), Name: types.StringValue(item.Name), Content: types.StringValue(item.Content),
			CreatedAt: types.StringValue(item.CreatedAt), UpdatedAt: types.StringValue(item.UpdatedAt),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
