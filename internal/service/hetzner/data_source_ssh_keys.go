//nolint:dupl // shared hetzner list data source extraction tracked in #11
package hetzner

import (
	"context"

	"github.com/SebTardifLabs/terraform-provider-coolify/internal/client"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/filter"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/flex"
	"github.com/SebTardifLabs/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*sshKeysDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*sshKeysDataSource)(nil)
)

type sshKeysDataSource struct {
	client *client.Client
}

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

// NewSSHKeysDataSource returns a new Hetzner SSH keys data source instance.
func NewSSHKeysDataSource() datasource.DataSource { return &sshKeysDataSource{} }

func (d *sshKeysDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_hetzner_ssh_keys"
}

func (d *sshKeysDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Lists all available Hetzner SSH keys for a given cloud provider token.",
		Attributes: map[string]schema.Attribute{
			"cloud_provider_token_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the cloud provider token to use for listing Hetzner SSH keys.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
			},
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
		Blocks: map[string]schema.Block{
			"filter": filter.Block(),
		},
	}
}

func (d *sshKeysDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *sshKeysDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config sshKeysDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_hetzner_ssh_keys"})

	keys, err := d.client.ListHetznerSSHKeys(ctx, config.CloudProviderTokenUUID.ValueString())
	if err != nil {
		resp.Diagnostics.AddError("Error listing Hetzner SSH keys", err.Error())
		return
	}

	keys = filter.Apply(ctx, keys, config.Filters, func(k client.HetznerSSHKey, field string) (string, bool) {
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
	})

	state := sshKeysDataSourceModel{
		CloudProviderTokenUUID: config.CloudProviderTokenUUID,
		Filters:                config.Filters,
		SSHKeys:                make([]sshKeyModel, len(keys)),
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
