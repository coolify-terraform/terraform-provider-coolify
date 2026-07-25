package vultr

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
	ID          types.String `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	DateCreated types.String `tfsdk:"date_created"`
}

func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }
func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_vultr_ssh_keys"
}
func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists Vultr SSH keys for a cloud provider token. Requires Coolify >= v4.2.0.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": cloudProviderTokenUUIDAttribute("SSH keys"),
			"ssh_keys": schema.ListNestedAttribute{Computed: true, NestedObject: schema.NestedAttributeObject{Attributes: map[string]schema.Attribute{
				"id":           schema.StringAttribute{Computed: true, MarkdownDescription: "SSH key ID."},
				"name":         schema.StringAttribute{Computed: true, MarkdownDescription: "SSH key name."},
				"date_created": schema.StringAttribute{Computed: true, MarkdownDescription: "Creation date."},
			}}},
		},
		Blocks: map[string]schema.Block{"filter": filter.Block()},
	}
}
func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = configureVultrDataSourceClient(req, resp)
}
func (d *sshKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sshKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}
	items, ok := readFilteredTokenList(ctx, config.CloudProviderTokenUUID.ValueString(), config.Filters,
		"coolify_vultr_ssh_keys", "Error listing Vultr SSH keys", resp, d.client, d.client.ListVultrSSHKeys,
		func(k client.VultrSSHKey, field string) (string, bool) {
			if field == "id" {
				return k.ID, true
			}
			if field == "name" {
				return k.Name, true
			}
			return "", false
		})
	if !ok {
		return
	}
	state := sshKeysDataSourceModel{CloudProviderTokenUUID: config.CloudProviderTokenUUID, Filters: config.Filters, SSHKeys: make([]sshKeyModel, len(items))}
	for i, k := range items {
		state.SSHKeys[i] = sshKeyModel{ID: types.StringValue(k.ID), Name: types.StringValue(k.Name), DateCreated: types.StringValue(k.DateCreated)}
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
