package s3storage

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/datasource/schema"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ datasource.DataSource              = (*s3StorageDataSource)(nil)
	_ datasource.DataSourceWithConfigure = (*s3StorageDataSource)(nil)
)

// s3StorageDataSource is the data source for a single Coolify S3 storage.
type s3StorageDataSource struct {
	client *client.Client
}

// s3StorageDataSourceModel maps the data source schema data.
// Key and secret are omitted: Coolify hides them unless the token can read sensitive data.
type s3StorageDataSourceModel struct {
	UUID                 types.String `tfsdk:"uuid"`
	Name                 types.String `tfsdk:"name"`
	Description          types.String `tfsdk:"description"`
	Endpoint             types.String `tfsdk:"endpoint"`
	Bucket               types.String `tfsdk:"bucket"`
	Region               types.String `tfsdk:"region"`
	UsePathStyleEndpoint types.Bool   `tfsdk:"use_path_style_endpoint"`
	IsUsable             types.Bool   `tfsdk:"is_usable"`
	Type                 types.String `tfsdk:"type"`
	SkipTLSVerification  types.Bool   `tfsdk:"skip_tls_verification"`
	PrivateKeyUUID       types.String `tfsdk:"private_key_uuid"`
	SFTPHostKey          types.String `tfsdk:"sftp_host_key"`
}

// NewDataSource returns a new S3 storage data source instance.
func NewDataSource() datasource.DataSource {
	return &s3StorageDataSource{}
}

func (d *s3StorageDataSource) Metadata(_ context.Context, req datasource.MetadataRequest, resp *datasource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_s3_storage"
}

func (d *s3StorageDataSource) Schema(_ context.Context, _ datasource.SchemaRequest, resp *datasource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Use this data source to retrieve information about a single Coolify S3 storage by its UUID. Requires Coolify >= v4.3.0.",
		Attributes: map[string]schema.Attribute{
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The unique identifier of the S3 storage to look up.",
				Required:            true,
				Validators:          []validator.String{validate.UUID()},
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
			"use_path_style_endpoint": schema.BoolAttribute{
				MarkdownDescription: "Whether this storage uses path-style URLs. Null when the Coolify instance is older than v4.5.1 and does not return the field.",
				Computed:            true,
			},
			"is_usable": schema.BoolAttribute{
				MarkdownDescription: "Whether Coolify marks this storage as usable.",
				Computed:            true,
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "Storage type (`s3` or `sftp`). Null when the instance is older than v4.7.0 and does not return the field.",
				Computed:            true,
			},
			"skip_tls_verification": schema.BoolAttribute{
				MarkdownDescription: "Whether S3 TLS verification is skipped. Null when the instance does not return the field.",
				Computed:            true,
			},
			"private_key_uuid": schema.StringAttribute{
				MarkdownDescription: "SFTP private key UUID. Null when this storage is not SFTP or the instance does not return the field.",
				Computed:            true,
			},
			"sftp_host_key": schema.StringAttribute{
				MarkdownDescription: "SFTP host key recorded by Coolify. Sensitive. Null when absent.",
				Computed:            true,
				Sensitive:           true,
			},
		},
	}
}

func (d *s3StorageDataSource) Configure(_ context.Context, req datasource.ConfigureRequest, resp *datasource.ConfigureResponse) {
	d.client = flex.ConfigureDataSourceClient(req, &resp.Diagnostics)
}

func (d *s3StorageDataSource) Read(ctx context.Context, req datasource.ReadRequest, resp *datasource.ReadResponse) {
	var config s3StorageDataSourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &config)...)
	if resp.Diagnostics.HasError() {
		return
	}

	tflog.Debug(ctx, "reading data source", map[string]interface{}{"data_source_type": "coolify_s3_storage"})

	uuid := config.UUID.ValueString()
	s, err := d.client.GetS3Storage(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading S3 storage", fmt.Sprintf("Could not read s3 storage %s: %s", uuid, err))
		return
	}

	config.UUID = types.StringValue(s.UUID)
	config.Name = types.StringValue(s.Name)
	config.Description = flex.StringToFramework(s.Description)
	config.Endpoint = types.StringValue(s.Endpoint)
	config.Bucket = types.StringValue(s.Bucket)
	config.Region = types.StringValue(s.Region)
	if s.UsePathStyleEndpoint != nil {
		config.UsePathStyleEndpoint = types.BoolValue(*s.UsePathStyleEndpoint)
	} else {
		config.UsePathStyleEndpoint = types.BoolNull()
	}
	if s.IsUsable != nil {
		config.IsUsable = types.BoolValue(*s.IsUsable)
	} else {
		config.IsUsable = types.BoolValue(false)
	}
	if s.Type != "" {
		config.Type = types.StringValue(s.Type)
	} else {
		config.Type = types.StringNull()
	}
	if s.SkipTLSVerification != nil {
		config.SkipTLSVerification = types.BoolValue(*s.SkipTLSVerification)
	} else {
		config.SkipTLSVerification = types.BoolNull()
	}
	if s.PrivateKeyUUID != "" {
		config.PrivateKeyUUID = types.StringValue(s.PrivateKeyUUID)
	} else {
		config.PrivateKeyUUID = types.StringNull()
	}
	if s.SFTPHostKey != "" {
		config.SFTPHostKey = types.StringValue(s.SFTPHostKey)
	} else {
		config.SFTPHostKey = types.StringNull()
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &config)...)
}
