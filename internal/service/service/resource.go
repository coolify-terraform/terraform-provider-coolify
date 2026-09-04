package service

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-log/tflog"
)

var (
	_ resource.Resource                   = &serviceResource{}
	_ resource.ResourceWithConfigure      = &serviceResource{}
	_ resource.ResourceWithImportState    = &serviceResource{}
	_ resource.ResourceWithValidateConfig = &serviceResource{}
)

type serviceResource struct {
	client *client.Client
}

type serviceResourceModel struct {
	Timeouts    timeouts.Value `tfsdk:"timeouts"`
	UUID        types.String   `tfsdk:"uuid"`
	Name        types.String   `tfsdk:"name"`
	Description types.String   `tfsdk:"description"`
	ProjectUUID types.String   `tfsdk:"project_uuid"`
	ServerUUID  types.String   `tfsdk:"server_uuid"`
	// DestinationUUID is create-only. Coolify GET does not return it; preserve
	// from state after apply.
	DestinationUUID               types.String      `tfsdk:"destination_uuid"`
	EnvironmentName               types.String      `tfsdk:"environment_name"`
	Type                          types.String      `tfsdk:"type"`
	Status                        types.String      `tfsdk:"status"`
	DockerCompose                 types.String      `tfsdk:"docker_compose"`
	DockerComposeRaw              types.String      `tfsdk:"docker_compose_raw"`
	ConnectToNetwork              types.Bool        `tfsdk:"connect_to_docker_network"`
	IsContainerLabelEscapeEnabled types.Bool        `tfsdk:"is_container_label_escape_enabled"`
	ConfigHash                    types.String      `tfsdk:"config_hash"`
	InstantDeploy                 types.Bool        `tfsdk:"instant_deploy"`
	URLs                          []serviceURLModel `tfsdk:"urls"`
	ForceDomainOverride           types.Bool        `tfsdk:"force_domain_override"`
}

type serviceURLModel struct {
	Name types.String `tfsdk:"name"`
	URL  types.String `tfsdk:"url"`
}

func NewResource() resource.Resource {
	return &serviceResource{}
}

func (r *serviceResource) Metadata(_ context.Context, req resource.MetadataRequest, resp *resource.MetadataResponse) {
	resp.TypeName = req.ProviderTypeName + "_service"
}

func (r *serviceResource) Schema(ctx context.Context, _ resource.SchemaRequest, resp *resource.SchemaResponse) {
	resp.Schema = schema.Schema{
		MarkdownDescription: "Manages a service resource on Coolify. A service can be created from the Coolify catalog (using `type`) or from a custom Docker Compose file (using `docker_compose_raw`). These two fields are mutually exclusive.",
		Attributes: map[string]schema.Attribute{
			"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
			"uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the service.",
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"name": schema.StringAttribute{
				MarkdownDescription: "The name of the service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"description": schema.StringAttribute{
				MarkdownDescription: "A description of the service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"project_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the project this service belongs to. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"server_uuid": schema.StringAttribute{
				MarkdownDescription: "The UUID of the server to deploy the service on. Changing this forces a new resource.",
				Required:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"destination_uuid": schema.StringAttribute{
				MarkdownDescription: "UUID of the Coolify destination (Docker network) on the server. Create-only; " +
					"changing forces a new resource. Coolify requires this when the server has multiple destinations " +
					"and ignores a mismatched value when only one destination exists. Supported on service create for " +
					"all Coolify versions this provider supports (v4.1.0+; field is in create `$allowedFields`, not " +
					"update). When omitted on a multi-destination server, the client auto-resolves after Coolify " +
					"returns the multi-destination error (prefers network `coolify`). Manage destinations with " +
					"`coolify_destination`. Import cannot recover this value (GET does not return destination UUID); " +
					"re-adding it after import forces replacement unless you set it in state or omit the attribute.",
				Optional: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
				Validators: []validator.String{
					validate.UUID(),
				},
			},
			"environment_name": schema.StringAttribute{
				MarkdownDescription: "The environment name. Defaults to `production`. Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				Default:             stringdefault.StaticString("production"),
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
				},
			},
			"type": schema.StringAttribute{
				MarkdownDescription: "The service type from the Coolify service catalog (e.g., `plausible`, `uptime-kuma`, `minio`). Mutually exclusive with `docker_compose_raw`. See the full list in the Coolify UI under Services > New Service, or in the [Coolify source](https://github.com/coollabsio/coolify/tree/v4.x/templates/service). Changing this forces a new resource.",
				Optional:            true,
				Computed:            true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.RequiresReplace(),
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"status": schema.StringAttribute{
				MarkdownDescription: "The current status of the service (e.g., `running`, `stopped`, `exited`). Read-only.",
				Computed:            true,
			},
			"docker_compose": schema.StringAttribute{
				MarkdownDescription: "The parsed Docker Compose configuration. Requires API token with `read:sensitive` permission.",
				Computed:            true,
				Sensitive:           true,
			},
			"docker_compose_raw": schema.StringAttribute{
				MarkdownDescription: "The raw Docker Compose YAML content. Can be used instead of `type` to create a service from a custom compose file, or to customize a catalog service after creation. " +
					"The provider accepts plain YAML or pre-encoded base64; encoding is handled automatically. Requires API token with `read:sensitive` permission.",
				Optional:  true,
				Computed:  true,
				Sensitive: true,
				PlanModifiers: []planmodifier.String{
					stringplanmodifier.UseStateForUnknown(),
				},
			},
			"connect_to_docker_network": schema.BoolAttribute{
				MarkdownDescription: "Whether the service containers connect to the Coolify Docker network.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"is_container_label_escape_enabled": schema.BoolAttribute{
				MarkdownDescription: "Whether container label escaping is enabled for this service.",
				Optional:            true,
				Computed:            true,
				PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
			},
			"config_hash": schema.StringAttribute{
				MarkdownDescription: "Hash of the current service configuration. Changes when the compose or settings are modified.",
				Computed:            true,
			},
			"instant_deploy": schema.BoolAttribute{
				MarkdownDescription: "Whether to immediately deploy the service after creation. When `true`, Coolify starts the service containers right away. When `false` (default), the service is created but not started.",
				Optional:            true,
				Computed:            true,
				Default:             booldefault.StaticBool(false),
			},
			"urls": schema.ListNestedAttribute{
				MarkdownDescription: "Domain URL mappings for service containers. Each entry maps a compose service name to one or more comma-separated URLs (e.g., `https://app.example.com`). " +
					"Read-back matches Coolify application FQDNs by container name and keeps this list's configured order. " +
					"Coolify GET may return applications in a different order (typically compose or database insertion order).",
				Optional: true,
				NestedObject: schema.NestedAttributeObject{
					Attributes: map[string]schema.Attribute{
						"name": schema.StringAttribute{
							MarkdownDescription: "The service container name as defined in docker-compose (e.g., `web`, `api`).",
							Required:            true,
						},
						"url": schema.StringAttribute{
							MarkdownDescription: "Comma-separated list of URLs to assign to this container (e.g., `https://app.example.com,https://www.example.com`).",
							Optional:            true,
						},
					},
				},
			},
			"force_domain_override": schema.BoolAttribute{
				MarkdownDescription: "Force domain assignment even if conflicts with other resources are detected. Only relevant when `urls` is set.",
				Optional:            true,
			},
		},
	}
}

func (r *serviceResource) Configure(_ context.Context, req resource.ConfigureRequest, resp *resource.ConfigureResponse) {
	r.client = flex.ConfigureClient(req, &resp.Diagnostics)
}

// ValidateConfig checks that exactly one of type or docker_compose_raw is set.
// We use ValidateConfig instead of stringvalidator.ExactlyOneOf because type
// is Optional+Computed with UseStateForUnknown. ExactlyOneOf operates at the
// attribute level and would misfire when the computed value is populated from
// state, incorrectly rejecting configs that only set docker_compose_raw.
//
// Unknown values (from variables, data sources, or other resources) must not
// be treated as absent: ValidateConfig runs before those values are resolved.
func (r *serviceResource) ValidateConfig(ctx context.Context, req resource.ValidateConfigRequest, resp *resource.ValidateConfigResponse) {
	var model serviceResourceModel
	resp.Diagnostics.Append(req.Config.Get(ctx, &model)...)
	if resp.Diagnostics.HasError() {
		return
	}

	// Values that come from variables, data sources or other resources are
	// still unknown when ValidateConfig runs. Treat unknown as "set": the
	// attribute is present in the config, only its value is not resolved yet.
	// Without this, a config like docker_compose_raw = var.compose fails
	// validation with "One of type or docker_compose_raw must be set".
	if model.Type.IsUnknown() || model.DockerComposeRaw.IsUnknown() {
		return
	}

	hasType := !model.Type.IsNull()
	hasCompose := !model.DockerComposeRaw.IsNull()

	if hasType && hasCompose {
		resp.Diagnostics.AddAttributeError(
			path.Root("type"),
			"Conflicting attributes",
			"\"type\" and \"docker_compose_raw\" are mutually exclusive. Use \"type\" to deploy from the Coolify catalog, or \"docker_compose_raw\" to deploy a custom Docker Compose stack.",
		)
	}
	if !hasType && !hasCompose {
		resp.Diagnostics.AddError(
			"Missing required attribute",
			"One of \"type\" or \"docker_compose_raw\" must be set.",
		)
	}
}

func (r *serviceResource) Create(ctx context.Context, req resource.CreateRequest, resp *resource.CreateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	createTimeout, diags := plan.Timeouts.Create(ctx, 10*time.Minute)
	resp.Diagnostics.Append(diags...)
	if resp.Diagnostics.HasError() {
		return
	}

	ctx, cancel := context.WithTimeout(ctx, createTimeout)
	defer cancel()

	tflog.Debug(ctx, "creating resource", map[string]interface{}{"resource_type": "coolify_service"})

	input := client.CreateServiceInput{
		ServerUUID:      plan.ServerUUID.ValueString(),
		ProjectUUID:     plan.ProjectUUID.ValueString(),
		EnvironmentName: plan.EnvironmentName.ValueString(),
	}
	// Catalog type or custom compose (mutually exclusive, validated in ValidateConfig).
	if !plan.Type.IsNull() && !plan.Type.IsUnknown() {
		input.Type = plan.Type.ValueString()
	}
	if !plan.DockerComposeRaw.IsNull() && !plan.DockerComposeRaw.IsUnknown() {
		encoded := flex.EnsureBase64(plan.DockerComposeRaw.ValueString())
		input.DockerComposeRaw = &encoded
	}
	flex.SetIfKnown(&input.Name, plan.Name)
	flex.SetIfKnown(&input.Description, plan.Description)
	flex.SetIfKnown(&input.DestinationUUID, plan.DestinationUUID)
	input.InstantDeploy = flex.BoolValueOrNull(plan.InstantDeploy)
	input.ConnectToNetwork = flex.BoolValueOrNull(plan.ConnectToNetwork)
	input.IsContainerLabelEscapeEnabled = flex.BoolValueOrNull(plan.IsContainerLabelEscapeEnabled)
	input.URLs = expandServiceURLs(plan.URLs)
	input.ForceDomainOverride = flex.BoolValueOrNull(plan.ForceDomainOverride)
	created, err := r.client.CreateService(ctx, input)
	if err != nil {
		resp.Diagnostics.AddError("Error creating service",
			fmt.Sprintf("project %s, server %s: %s", plan.ProjectUUID.ValueString(), plan.ServerUUID.ValueString(), err))
		return
	}

	plan.UUID = types.StringValue(created.UUID)
	if plan.Name.IsUnknown() {
		plan.Name = types.StringNull()
	}
	if plan.Description.IsUnknown() {
		plan.Description = types.StringNull()
	}
	if plan.Type.IsUnknown() {
		plan.Type = types.StringNull()
	}
	if plan.DockerComposeRaw.IsUnknown() {
		plan.DockerComposeRaw = types.StringNull()
	}

	// Save partial state so the resource is tracked even if the read-back fails.
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}

	svc, err := r.client.GetService(ctx, created.UUID)
	if err != nil {
		resp.Diagnostics.AddError(
			flex.CreateReadBackFailedSummary("Service"),
			flex.CreateReadBackFailedDetail("service", created.UUID, err),
		)
		return
	}

	flattenService(svc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
	tflog.Debug(ctx, "created resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": created.UUID})
}

func (r *serviceResource) Read(ctx context.Context, req resource.ReadRequest, resp *resource.ReadResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}
	tflog.Debug(ctx, "reading resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": state.UUID.ValueString()})

	svc, err := r.client.GetService(ctx, state.UUID.ValueString())
	if err != nil {
		if client.IsNotFound(err) {
			tflog.Debug(ctx, "resource not found, removing from state", map[string]interface{}{"resource_type": "coolify_service", "uuid": state.UUID.ValueString()})
			resp.State.RemoveResource(ctx)
			return
		}
		resp.Diagnostics.AddError("Error reading service", fmt.Sprintf("service %s: %s", state.UUID.ValueString(), err))
		return
	}

	flattenService(svc, &state)
	resp.Diagnostics.Append(resp.State.Set(ctx, &state)...)
}

func (r *serviceResource) Update(ctx context.Context, req resource.UpdateRequest, resp *resource.UpdateResponse) {
	var plan serviceResourceModel
	resp.Diagnostics.Append(req.Plan.Get(ctx, &plan)...)
	if resp.Diagnostics.HasError() {
		return
	}
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	uuid := state.UUID.ValueString()
	tflog.Debug(ctx, "updating resource", map[string]interface{}{"resource_type": "coolify_service", "uuid": uuid})

	// Auto-encode docker_compose_raw before comparing/sending.
	var encodedComposeRaw *string
	if raw := flex.StringIfChanged(plan.DockerComposeRaw, state.DockerComposeRaw); raw != nil {
		encoded := flex.EnsureBase64(*raw)
		encodedComposeRaw = &encoded
	}
	input := client.UpdateServiceInput{
		Name:                          flex.StringIfChanged(plan.Name, state.Name),
		Description:                   flex.StringIfChanged(plan.Description, state.Description),
		DockerComposeRaw:              encodedComposeRaw,
		ConnectToNetwork:              flex.BoolIfChanged(plan.ConnectToNetwork, state.ConnectToNetwork),
		IsContainerLabelEscapeEnabled: flex.BoolIfChanged(plan.IsContainerLabelEscapeEnabled, state.IsContainerLabelEscapeEnabled),
		URLs:                          expandServiceURLs(plan.URLs),
		ForceDomainOverride:           flex.BoolValueOrNull(plan.ForceDomainOverride),
	}
	if _, err := r.client.UpdateService(ctx, uuid, input); err != nil {
		resp.Diagnostics.AddError("Error updating service", fmt.Sprintf("service %s: %s", uuid, err))
		return
	}

	svc, err := r.client.GetService(ctx, uuid)
	if err != nil {
		resp.Diagnostics.AddError("Error reading service after update", fmt.Sprintf("service %s: %s", uuid, err))
		return
	}

	flattenService(svc, &plan)
	resp.Diagnostics.Append(resp.State.Set(ctx, &plan)...)
}

const deletePollingTimeoutWarningSummary = "Delete is still finishing in Coolify"

func addDeletePollingTimeoutWarning(resp *resource.DeleteResponse, resourceType, uuid string) {
	resp.Diagnostics.AddWarning(
		deletePollingTimeoutWarningSummary,
		fmt.Sprintf(
			"Coolify accepted deletion of %s %s, but the resource was still returned by the API when the provider stopped polling. Terraform removed it from state, but the remote resource may still exist temporarily. Wait a moment before retrying dependent operations if they still report it.",
			resourceType,
			uuid,
		),
	)
}

func deleteService(ctx context.Context, c *client.Client, resourceType, uuid string, resp *resource.DeleteResponse) {
	tflog.Debug(ctx, "deleting resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})

	if err := c.DeleteService(ctx, uuid); err != nil {
		if client.IsNotFound(err) {
			return
		}
		resp.Diagnostics.AddError("Error deleting service", fmt.Sprintf("service %s: %s", uuid, err))
		return
	}
	if !client.PollUntilDeleted(ctx, func() error { _, err := c.GetService(ctx, uuid); return err }) {
		tflog.Warn(ctx, "resource may still exist after polling timeout", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
		addDeletePollingTimeoutWarning(resp, resourceType, uuid)
	}
	tflog.Debug(ctx, "deleted resource", map[string]interface{}{"resource_type": resourceType, "uuid": uuid})
}

func (r *serviceResource) Delete(ctx context.Context, req resource.DeleteRequest, resp *resource.DeleteResponse) {
	var state serviceResourceModel
	resp.Diagnostics.Append(req.State.Get(ctx, &state)...)
	if resp.Diagnostics.HasError() {
		return
	}

	deleteService(ctx, r.client, "coolify_service", state.UUID.ValueString(), resp)
}

func (r *serviceResource) ImportState(ctx context.Context, req resource.ImportStateRequest, resp *resource.ImportStateResponse) {
	parsed, compound, err := validate.ParseCompoundImportID(req.ID)
	if err != nil {
		resp.Diagnostics.AddError("Invalid Import ID", err.Error())
		return
	}
	if compound {
		if err := r.client.ValidateResourceOnServer(ctx, parsed.ServerUUID, parsed.UUID, "service"); err != nil {
			resp.Diagnostics.AddError("Invalid compound import ID", err.Error())
			return
		}
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("project_uuid"), parsed.ProjectUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("server_uuid"), parsed.ServerUUID)...)
		resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("environment_name"), parsed.EnvironmentName)...)
	}
	resp.Diagnostics.Append(resp.State.SetAttribute(ctx, path.Root("uuid"), parsed.UUID)...)
	resp.Diagnostics.AddWarning(
		"Sensitive fields require token permissions",
		"The Coolify API hides docker_compose and docker_compose_raw unless the API token has \"root\" or \"read:sensitive\" permission. "+
			"If you see unexpected diffs after import, check your token's permissions in the Coolify dashboard under Security > API Tokens.",
	)
}

func flattenService(svc *client.Service, model *serviceResourceModel) {
	model.UUID = types.StringValue(svc.UUID)
	model.Name = flex.StringToFramework(svc.Name)
	model.Description = flex.StringToFramework(svc.Description)
	model.Status = flex.StringToFramework(svc.Status)
	// instant_deploy is create-only and never returned by the API.
	// Preserve state value when set; default to false otherwise (import).
	if model.InstantDeploy.IsNull() || model.InstantDeploy.IsUnknown() {
		model.InstantDeploy = types.BoolValue(false)
	}
	model.DockerCompose = flex.StringToFramework(svc.DockerCompose)
	// The API returns decoded YAML for docker_compose_raw, but the user may
	// have provided raw YAML or base64. Preserve the user's original value
	// when the decoded content matches, to avoid perpetual diffs.
	if svc.DockerComposeRaw != "" {
		if model.DockerComposeRaw.IsNull() || model.DockerComposeRaw.IsUnknown() {
			model.DockerComposeRaw = types.StringValue(svc.DockerComposeRaw)
		}
		// else: keep the user's configured value in state
	} else if model.DockerComposeRaw.IsUnknown() {
		// API didn't return it (catalog service) and user didn't set it.
		model.DockerComposeRaw = types.StringNull()
	}
	model.ConfigHash = flex.StringToFramework(svc.ConfigHash)
	if svc.ConnectToNetwork != nil {
		model.ConnectToNetwork = types.BoolValue(*svc.ConnectToNetwork)
	} else {
		model.ConnectToNetwork = types.BoolNull()
	}
	if svc.IsContainerLabelEscapeEnabled != nil {
		model.IsContainerLabelEscapeEnabled = types.BoolValue(*svc.IsContainerLabelEscapeEnabled)
	} else {
		model.IsContainerLabelEscapeEnabled = types.BoolNull()
	}

	// Immutable fields: only update if the API returns them because
	// Coolify may omit these from the GET response.
	if svc.Type != "" {
		model.Type = types.StringValue(svc.Type)
	}
	if svc.ProjectUUID != "" {
		model.ProjectUUID = types.StringValue(svc.ProjectUUID)
	}
	if svc.ServerUUID != "" {
		model.ServerUUID = types.StringValue(svc.ServerUUID)
	}
	if svc.EnvironmentName != "" {
		model.EnvironmentName = flex.StringToFramework(svc.EnvironmentName)
	}

	model.URLs = flattenServiceURLs(svc.Applications, model.URLs)
	// force_domain_override is request-only, never returned by the API.
	// Preserve the user's value in state; it is not read back.
}

// flattenServiceURLs reconstructs URL mappings from the service's applications.
// Only populates urls if the user configured them (current != nil). Coolify
// auto-assigns FQDNs to catalog services, so the API may return application
// FQDNs even when the user didn't set urls. Without this guard, the flatten
// causes "inconsistent result after apply" (plan had null, state has values).
//
// Coolify GET loads applications as an unordered hasMany, so API list order
// can differ from HCL. Match by container name and keep the configured order.
func flattenServiceURLs(apps []client.ServiceApplication, current []serviceURLModel) []serviceURLModel {
	if current == nil {
		// User didn't configure urls; don't populate from API.
		return nil
	}
	if len(apps) == 0 {
		return current
	}
	byName := make(map[string]client.ServiceApplication, len(apps))
	for _, app := range apps {
		if app.Name == "" {
			continue
		}
		byName[app.Name] = app
	}
	out := make([]serviceURLModel, 0, len(current))
	for _, cfg := range current {
		app, ok := byName[cfg.Name.ValueString()]
		if !ok || app.FQDN == "" {
			out = append(out, cfg)
			continue
		}
		url := types.StringValue(app.FQDN)
		if serviceURLEquivalent(cfg.URL.ValueString(), app.FQDN) {
			url = cfg.URL
		}
		out = append(out, serviceURLModel{
			Name: cfg.Name,
			URL:  url,
		})
	}
	return out
}

// serviceURLEquivalent treats comma-separated FQDN lists as the same when they
// contain the same URLs after trim and case fold, regardless of list order.
func serviceURLEquivalent(configured, api string) bool {
	if configured == api || strings.EqualFold(configured, api) {
		return true
	}
	return serviceURLTokenKey(configured) == serviceURLTokenKey(api)
}

func serviceURLTokenKey(s string) string {
	parts := strings.Split(s, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p == "" {
			continue
		}
		out = append(out, strings.ToLower(p))
	}
	sort.Strings(out)
	return strings.Join(out, ",")
}

// expandServiceURLs converts the Terraform model to the client input format.
func expandServiceURLs(urls []serviceURLModel) []client.ServiceURL {
	if len(urls) == 0 {
		return nil
	}
	result := make([]client.ServiceURL, len(urls))
	for i, u := range urls {
		result[i] = client.ServiceURL{
			Name: u.Name.ValueString(),
			URL:  u.URL.ValueString(),
		}
	}
	return result
}
