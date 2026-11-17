package serverimagescan

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-validators/float64validator"
	"github.com/hashicorp/terraform-plugin-framework-validators/int64validator"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/float64planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64planmodifier"
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
	ServerUUID            types.String  `tfsdk:"server_uuid"`
	IsImageScanEnabled    types.Bool    `tfsdk:"is_image_scan_enabled"`
	ImageScanMemoryMB     types.Int64   `tfsdk:"image_scan_memory_mb"`
	ImageScanSyftTimeout  types.Int64   `tfsdk:"image_scan_syft_timeout"`
	ImageScanGrypeTimeout types.Int64   `tfsdk:"image_scan_grype_timeout"`
	ImageScanCPUs         types.Float64 `tfsdk:"image_scan_cpus"`
	ScannerState          types.String  `tfsdk:"scanner_state"`
	ScannerFailedStep     types.String  `tfsdk:"scanner_failed_step"`
	ScannerBuiltAt        types.String  `tfsdk:"scanner_built_at"`
	ScannerDiskUsedBytes  types.Int64   `tfsdk:"scanner_disk_used_bytes"`
}

func NewResource() resource.Resource { return &res{} }

func (r *res) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_server_image_scanning"
}

func (r *res) Schema(_ context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages image vulnerability scanning for one Coolify server (`GET`/`PATCH /servers/{uuid}/image-scanning`). " +
			"Requires Coolify >= v4.7.0. Turning scanning on prepares the scanner on the server in the background. " +
			"Memory, CPU, and Syft/Grype time limits require Coolify >= v4.7.3. " +
			"Destroy turns scanning off. The scanner state fields are read-only.",
		Attributes: imageScanAttrs(),
	}
}

func imageScanAttrs() map[string]schema.Attribute {
	const needs473 = "Requires Coolify >= v4.7.3. Setting this on Coolify v4.7.0 through v4.7.2 fails the apply. "
	attrs := map[string]schema.Attribute{
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
		"image_scan_memory_mb": schema.Int64Attribute{
			MarkdownDescription: "Memory limit of each scanner step, in MB. One of 1024, 2048, 4096, or 8192. Coolify's stored default is 1024. The server refuses a value above its own memory. " + needs473,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:          []validator.Int64{int64validator.OneOf(1024, 2048, 4096, 8192)},
		},
		"image_scan_syft_timeout": schema.Int64Attribute{
			MarkdownDescription: "Time limit of one Syft run (the package list of an image), in seconds. One of 300, 600, 1200, 1800, or 3600. Coolify's stored default is 600. " + needs473,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:          []validator.Int64{int64validator.OneOf(300, 600, 1200, 1800, 3600)},
		},
		"image_scan_grype_timeout": schema.Int64Attribute{
			MarkdownDescription: "Time limit of one Grype run (the vulnerability match), in seconds. One of 300, 600, 1200, 1800, or 3600. Coolify's stored default is 600. " + needs473,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Int64{int64planmodifier.UseStateForUnknown()},
			Validators:          []validator.Int64{int64validator.OneOf(300, 600, 1200, 1800, 3600)},
		},
		"image_scan_cpus": schema.Float64Attribute{
			MarkdownDescription: "CPU limit of each scanner step. One of 0.5, 1, 2, or 4. Coolify's stored default is 1. The server refuses a value above its CPU count. " + needs473,
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Float64{float64planmodifier.UseStateForUnknown()},
			Validators:          []validator.Float64{float64validator.OneOf(0.5, 1, 2, 4)},
		},
	}
	return attrs
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
	if rejectUnsupportedLimits(&resp.Diagnostics, r.client, plan, model{}, false) {
		return
	}
	if err := applyScanPatch(ctx, r.client, uuid, plan, model{}, false); err != nil {
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
	if rejectUnsupportedLimits(&resp.Diagnostics, r.client, plan, state, true) {
		return
	}
	if err := applyScanPatch(ctx, r.client, uuid, plan, state, true); err != nil {
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

func rejectUnsupportedLimits(diags *diag.Diagnostics, c *client.Client, plan, state model, updating bool) bool {
	input, _ := scanPatch(plan, state, updating)
	if !patchHasLimits(input) || c.SupportsImageScanLimits() {
		return false
	}
	diags.AddError("Coolify version cannot set image scan limits", limitsVersionDetail(c))
	return true
}

func applyScanPatch(ctx context.Context, c *client.Client, uuid string, plan, state model, updating bool) error {
	input, send := scanPatch(plan, state, updating)
	if !send {
		return nil
	}
	_, err := c.UpdateServerImageScanning(ctx, uuid, input)
	return err
}

func scanPatch(plan, state model, updating bool) (client.UpdateServerImageScanningInput, bool) {
	var in client.UpdateServerImageScanningInput
	if updating {
		in.IsImageScanEnabled = flex.BoolIfChanged(plan.IsImageScanEnabled, state.IsImageScanEnabled)
		in.ImageScanMemoryMB = flex.Int64IfChanged(plan.ImageScanMemoryMB, state.ImageScanMemoryMB)
		in.ImageScanSyftTimeout = flex.Int64IfChanged(plan.ImageScanSyftTimeout, state.ImageScanSyftTimeout)
		in.ImageScanGrypeTimeout = flex.Int64IfChanged(plan.ImageScanGrypeTimeout, state.ImageScanGrypeTimeout)
		in.ImageScanCPUs = flex.Float64IfChanged(plan.ImageScanCPUs, state.ImageScanCPUs)
	} else {
		in.IsImageScanEnabled = flex.BoolValueOrNull(plan.IsImageScanEnabled)
		in.ImageScanMemoryMB = flex.Int64PtrFromFramework(plan.ImageScanMemoryMB)
		in.ImageScanSyftTimeout = flex.Int64PtrFromFramework(plan.ImageScanSyftTimeout)
		in.ImageScanGrypeTimeout = flex.Int64PtrFromFramework(plan.ImageScanGrypeTimeout)
		in.ImageScanCPUs = flex.Float64IfChanged(plan.ImageScanCPUs, types.Float64Null())
	}
	send := in.IsImageScanEnabled != nil || patchHasLimits(in)
	return in, send
}

func patchHasLimits(in client.UpdateServerImageScanningInput) bool {
	return in.ImageScanMemoryMB != nil || in.ImageScanSyftTimeout != nil || in.ImageScanGrypeTimeout != nil || in.ImageScanCPUs != nil
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
	m.ImageScanMemoryMB = intFromAPI(got.ImageScanMemoryMB, m.ImageScanMemoryMB)
	m.ImageScanSyftTimeout = intFromAPI(got.ImageScanSyftTimeout, m.ImageScanSyftTimeout)
	m.ImageScanGrypeTimeout = intFromAPI(got.ImageScanGrypeTimeout, m.ImageScanGrypeTimeout)
	m.ImageScanCPUs = floatFromAPI(got.ImageScanCPUs, m.ImageScanCPUs)
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

func intFromAPI(v *int64, prev types.Int64) types.Int64 {
	if v != nil {
		return types.Int64Value(*v)
	}
	if !prev.IsNull() && !prev.IsUnknown() {
		return prev
	}
	return types.Int64Null()
}

func floatFromAPI(v *float64, prev types.Float64) types.Float64 {
	if v != nil {
		return types.Float64Value(*v)
	}
	if !prev.IsNull() && !prev.IsUnknown() {
		return prev
	}
	return types.Float64Null()
}

func limitsVersionDetail(c *client.Client) string {
	ver := "unknown"
	if c != nil && c.CoolifyVersion != "" {
		ver = c.CoolifyVersion
	}
	return fmt.Sprintf("This Coolify instance (%s) does not accept image scan memory, CPU, or time limits (added in Coolify v4.7.3). "+
		"Upgrade to Coolify v4.7.3 or later, or remove image_scan_memory_mb, image_scan_cpus, image_scan_syft_timeout, and image_scan_grype_timeout.", ver)
}

func versionDetail(c *client.Client) string {
	ver := "unknown"
	if c != nil && c.CoolifyVersion != "" {
		ver = c.CoolifyVersion
	}
	return fmt.Sprintf("This Coolify instance (%s) has no /servers/{uuid}/image-scanning (added in v4.7.0).", ver)
}
