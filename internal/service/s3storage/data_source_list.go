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
						"use_path_style_endpoint": schema.BoolAttribute{
							MarkdownDescription: "Whether this storage uses path-style URLs. Null when the Coolify instance is older than v4.5.1 and does not return the field.",
							Computed:            true,
						},
						"is_usable": schema.BoolAttribute{
							MarkdownDescription: "Whether Coolify marks this storage as usable.",
							Computed:            true,
						},
						"type": schema.StringAttribute{
							MarkdownDescription: "Storage type (`s3`, `sftp`, `ftps`, or `webdav`). Null when the instance is older than v4.7.0.",
							Computed:            true,
						},
						"skip_tls_verification": schema.BoolAttribute{
							MarkdownDescription: "Whether S3 TLS verification is skipped. Null when the instance does not return the field.",
							Computed:            true,
						},
						"private_key_uuid": schema.StringAttribute{
							MarkdownDescription: "SFTP private key UUID. Null when absent.",
							Computed:            true,
						},
						"sftp_host_key": schema.StringAttribute{
							MarkdownDescription: "SFTP host key recorded by Coolify. Sensitive. Null when absent.",
							Computed:            true,
							Sensitive:           true,
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

func s3StorageFilterField(s client.S3Storage, field string) (string, bool) {
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
	case "use_path_style_endpoint":
		if s.UsePathStyleEndpoint == nil {
			return "", false
		}
		return filter.BoolToString(*s.UsePathStyleEndpoint), true
	case "is_usable":
		usable := false
		if s.IsUsable != nil {
			usable = *s.IsUsable
		}
		return filter.BoolToString(usable), true
	case "type":
		return s.Type, s.Type != ""
	case "skip_tls_verification":
		if s.SkipTLSVerification == nil {
			return "", false
		}
		return filter.BoolToString(*s.SkipTLSVerification), true
	case "private_key_uuid":
		return s.PrivateKeyUUID, s.PrivateKeyUUID != ""
	default:
		return "", false
	}
}

func s3StorageListItem(s client.S3Storage) s3StorageItemModel {
	item := s3StorageItemModel{
		UUID:        types.StringValue(s.UUID),
		Name:        types.StringValue(s.Name),
		Description: flex.StringToFramework(s.Description),
		Endpoint:    types.StringValue(s.Endpoint),
		Bucket:      types.StringValue(s.Bucket),
		Region:      types.StringValue(s.Region),
	}
	if s.UsePathStyleEndpoint != nil {
		item.UsePathStyleEndpoint = types.BoolValue(*s.UsePathStyleEndpoint)
	} else {
		item.UsePathStyleEndpoint = types.BoolNull()
	}
	if s.IsUsable != nil {
		item.IsUsable = types.BoolValue(*s.IsUsable)
	} else {
		item.IsUsable = types.BoolValue(false)
	}
	if s.Type != "" {
		item.Type = types.StringValue(s.Type)
	} else {
		item.Type = types.StringNull()
	}
	if s.SkipTLSVerification != nil {
		item.SkipTLSVerification = types.BoolValue(*s.SkipTLSVerification)
	} else {
		item.SkipTLSVerification = types.BoolNull()
	}
	if s.PrivateKeyUUID != "" {
		item.PrivateKeyUUID = types.StringValue(s.PrivateKeyUUID)
	} else {
		item.PrivateKeyUUID = types.StringNull()
	}
	if s.SFTPHostKey != "" {
		item.SFTPHostKey = types.StringValue(s.SFTPHostKey)
	} else {
		item.SFTPHostKey = types.StringNull()
	}
	return item
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

	storages = filter.Apply(ctx, storages, config.Filters, s3StorageFilterField)

	var state s3StorageListDataSourceModel
	state.Filters = config.Filters
	for _, s := range storages {
		state.S3Storages = append(state.S3Storages, s3StorageListItem(s))
	}

	if state.S3Storages == nil {
		state.S3Storages = []s3StorageItemModel{}
	}

	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}
