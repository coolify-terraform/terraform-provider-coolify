package serverimagescan

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                = (*res)(nil)
	_ resource.ResourceWithConfigure   = (*res)(nil)
	_ resource.ResourceWithImportState = (*res)(nil)
)

type res struct{ client *client.Client }

type model struct {
	ServerUUID           types.String `tfsdk:"server_uuid"`
	IsImageScanEnabled   types.Bool   `tfsdk:"is_image_scan_enabled"`
	ScannerState         types.String `tfsdk:"scanner_state"`
	ScannerFailedStep    types.String `tfsdk:"scanner_failed_step"`
	ScannerBuiltAt       types.String `tfsdk:"scanner_built_at"`
	ScannerDiskUsedBytes types.Int64  `tfsdk:"scanner_disk_used_bytes"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_image_scanning"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages image vulnerability scanning for one Coolify server (`GET`/`PATCH /servers/{uuid}/image-scanning`). " +
			"Requires Coolify >= v4.7.0. Turning scanning on prepares the scanner on the server in the background. " +
			"Destroy turns scanning off. The scanner state fields are read-only.",
		Attributes: map[string]schema.Attribute{
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the server.",
				Required:            true,
				PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
				Validators:          []validator.String{validate.UUID()},
			},
			"is_image_scan_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether this server scans images of its resources. The server setting default is false. " +
					"Leaving it unset keeps the value Coolify already stored.",
				Optional:      true,
				Computed:      true,
				PlanModifiers: []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"scanner_state": schema.StringAttribute{
				MarkdownDescription: "Scanner state on the server: `pulling`, `updating_database`, `ready`, or `failed`. Null when Coolify has not reported one.",
				Computed:            true,
			},
			"scanner_failed_step": schema.StringAttribute{
				MarkdownDescription: "Step that failed when `scanner_state` is `failed`.",
				Computed:            true,
			},
			"scanner_built_at": schema.StringAttribute{
				MarkdownDescription: "Time of the vulnerability database on the server, as an RFC3339 timestamp.",
				Computed:            true,
			},
			"scanner_disk_used_bytes": schema.Int64Attribute{
				MarkdownDescription: "Disk space the scanner uses on the server, in bytes.",
				Computed:            true,
			},
		},
	}
}

func (r *res) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

func (r *res) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.client.SupportsCoolify47() {
		resp.Diagnostics.AddError("Coolify version cannot manage server image scanning", versionDetail(r.client))
		return
	}
	uuid := plan.ServerUUID.ValueString()
	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_server_image_scanning", "server_uuid": uuid})
	if err := patchIfSet(ctx, r.client, uuid, flex.BoolValueOrNull(plan.IsImageScanEnabled)); err != nil {
		resp.Diagnostics.AddError("Error updating server image scanning", err.Error())
		return
	}
	if err := readInto(ctx, r.client, uuid, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading server image scanning", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	uuid := state.ServerUUID.ValueString()
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_server_image_scanning", "server_uuid": uuid})
	if err := readInto(ctx, r.client, uuid, &state); err != nil {
		if client.IsNotFound(err) {
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading server image scanning", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *res) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan, state model
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if !r.client.SupportsCoolify47() {
		resp.Diagnostics.AddError("Coolify version cannot manage server image scanning", versionDetail(r.client))
		return
	}
	uuid := plan.ServerUUID.ValueString()
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_server_image_scanning", "server_uuid": uuid})
	if err := patchIfSet(ctx, r.client, uuid, flex.BoolIfChanged(plan.IsImageScanEnabled, state.IsImageScanEnabled)); err != nil {
		resp.Diagnostics.AddError("Error updating server image scanning", err.Error())
		return
	}
	if err := readInto(ctx, r.client, uuid, &plan); err != nil {
		resp.Diagnostics.AddError("Error reading server image scanning", err.Error())
		return
	}
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

func (r *res) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state model
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	if r.client == nil || !r.client.SupportsCoolify47() {
		return
	}
	off := false
	uuid := state.ServerUUID.ValueString()
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": "coolify_server_image_scanning", "server_uuid": uuid})
	if _, err := r.client.UpdateServerImageScanning(ctx, uuid, client.UpdateServerImageScanningInput{IsImageScanEnabled: &off}); err != nil && !client.IsNotFound(err) {
		resp.Diagnostics.AddError("Error disabling server image scanning", err.Error())
	}
}

func (r *res) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	if err := validate.ImportUUID(req.ID); err != nil {
		resp.Diagnostics.AddError("Invalid import ID", err.Error())
		return
	}
	resource.ImportStatePassthroughID(ctx, path.Root("server_uuid"), req, resp)
}

func patchIfSet(ctx context.Context, c *client.Client, uuid string, enabled *bool) error {
	if enabled == nil {
		return nil
	}
	_, err := c.UpdateServerImageScanning(ctx, uuid, client.UpdateServerImageScanningInput{IsImageScanEnabled: enabled})
	return err
}

func readInto(ctx context.Context, c *client.Client, uuid string, m *model) error {
	got, err := c.GetServerImageScanning(ctx, uuid)
	if err != nil {
		return err
	}
	flatten(got, m)
	m.ServerUUID = types.StringValue(uuid)
	return nil
}

func flatten(got *client.ServerImageScanning, m *model) {
	if got == nil || m == nil {
		return
	}
	m.IsImageScanEnabled = types.BoolValue(got.IsImageScanEnabled)
	m.ScannerState = stringOrNull(got.ScannerState)
	m.ScannerFailedStep = stringOrNull(got.ScannerFailedStep)
	m.ScannerBuiltAt = stringOrNull(got.ScannerBuiltAt)
	if got.ScannerDiskUsedBytes != nil {
		m.ScannerDiskUsedBytes = types.Int64Value(*got.ScannerDiskUsedBytes)
	} else {
		m.ScannerDiskUsedBytes = types.Int64Null()
	}
}

func stringOrNull(v string) types.String {
	if v == "" {
		return types.StringNull()
	}
	return types.StringValue(v)
}

func versionDetail(c *client.Client) string {
	ver := "unknown"
	if c != nil && c.CoolifyVersion != "" {
		ver = c.CoolifyVersion
	}
	return fmt.Sprintf("This Coolify instance (%s) has no /servers/{uuid}/image-scanning (added in v4.7.0).", ver)
}
