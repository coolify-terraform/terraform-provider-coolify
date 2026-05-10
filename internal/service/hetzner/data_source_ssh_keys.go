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
	_ datasource.DataSource              = (*sshKeysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeysDataSource)(nil)
)

type sshKeysDataSource struct {
	client *client.Client
}

type sshKeysDataSourceModel struct {
	SSHKeys []sshKeyModel `tfsdk:"ssh_keys"`
}

type sshKeyModel struct {
	ID          types.Int64  `tfsdk:"id"`
	Name        types.String `tfsdk:"name"`
	Fingerprint types.String `tfsdk:"fingerprint"`
}

// NewSSHKeysDataSource returns a new Hetzner SSH keys data source instance.
func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Hetzner SSH keys.",
		Attributes: map[string]schema.Attribute{
			"ssh_keys": schema.ListNestedAttribute{
				MarkdownDescription: "The list of Hetzner SSH keys.",
				Computed:            true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"id":          schema.Int64Attribute{MarkdownDescription: "The numeric ID of the SSH key.", Computed: true},
						"name":        schema.StringAttribute{MarkdownDescription: "The name of the SSH key.", Computed: true},
						"fingerprint": schema.StringAttribute{MarkdownDescription: "The fingerprint of the SSH key.", Computed: true},
					},
				},
			},
		},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
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

func (d *sshKeysDataSource) Read(ctx context.Context, _ datasource.ReadRequest, resp *datasource.ReadResponse) {
	keys, err := d.client.ListHetznerSSHKeys(ctx)
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner SSH keys", err.Error())
		return
	}

	state := sshKeysDataSourceModel{
		SSHKeys: make([]sshKeyModel, len(keys)),
	}
	for i, k := range keys {
		state.SSHKeys[i] = sshKeyModel{
			ID:          types.Int64Value(k.ID),
			Name:        types.StringValue(k.Name),
			Fingerprint: types.StringValue(k.Fingerprint),
		}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
