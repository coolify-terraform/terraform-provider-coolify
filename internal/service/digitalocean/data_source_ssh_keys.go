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
	_ datasource.DataSource              = (*sshKeysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeysDataSource)(nil)
)

type sshKeysDataSource struct{ client *client.Client }

type sshKeysDataSourceModel struct {
	CloudProviderTokenUUID types.String    `tfsdk:"cloud_provider_token_uuid"`
	SSHKeys                []sshKeyModel   `tfsdk:"ssh_keys"`
	Filters                []filter.Config `tfsdk:"filter"`
}

type sshKeyModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_digitalocean_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists DigitalOcean SSH keys for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("SSH keys"),
			"ssh_keys": schema.ListNestedAttribute{
				MarkdownDescription: "The list of DigitalOcean SSH keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{MarkdownDescription: "Numeric SSH key ID.", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "SSH key name.", Computed: true},
						"fingerprint": schema.StringAttribute{MarkdownDescription: "SSH key fingerprint.", Computed: true},
					},
				},
			},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureDigitalOceanDataSourceClient(req, resp)
}

func (d *sshKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sshKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	keys, ok := readFilteredTokenList(
		ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_digitalocean_ssh_keys", "Error listing DigitalOcean SSH keys", resp, d.client,
		d.client.ListDigitalOceanSSHKeys,
		func(k client.DigitalOceanSSHKey, field string) (string, bool) {
			switch field {
			case "id":
				return filter.Int64ToString(k.ID), true
			case "name":
				return k.Name, true
			case "fingerprint":
				return k.Fingerprint, true
			default:
				return "", false
			}
		},
	)
	if !ok {
		return
	}
	state := sshKeysDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		SSHKeys:                make([]sshKeyModel, len(keys)),
	}
	for i, k := range keys {
		state.SSHKeys[i] = sshKeyModel{
			ID: types.Int64Value(k.ID), Name: types.StringValue(k.Name), Fingerprint: types.StringValue(k.Fingerprint),
		}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
