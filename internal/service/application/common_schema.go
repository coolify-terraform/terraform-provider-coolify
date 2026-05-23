package application

import (
	"context"
	"regexp"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/validate"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework-validators/stringvalidator"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/booldefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/boolplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/int64default"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringdefault"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/stringplanmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
)

// CommonAppAttrs returns the shared schema attributes for all application types.
func CommonAppAttrs(ctx context.Context, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := coreAppAttrs(ctx)
	mergeAttrs(attrs, extendedBuildDeployAttrs())
	mergeAttrs(attrs, extendedHealthCheckAttrs())
	mergeAttrs(attrs, securityNetworkAttrs())
	mergeAttrs(attrs, extra)
	return attrs
}

func mergeAttrs(dst, src map[string]schema.Attribute) {
	for k, v := range src {
		dst[k] = v
	}
}

// gitAppAttrs returns the shared schema attributes for Git-backed
// application resources. Keep dockerfile_location scoped here because the
// Dockerfile application resource uses the same attribute name for different
// semantics.
func gitAppAttrs(ctx context.Context, gitRepositoryDescription string, extra map[string]schema.Attribute) map[string]schema.Attribute {
	attrs := gitAppSourceAttrs(gitRepositoryDescription)
	mergeAttrs(attrs, extra)
	mergeAttrs(attrs, gitAppCommandAttrs())

	return CommonAppAttrs(ctx, attrs)
}

func gitAppSourceAttrs(gitRepositoryDescription string) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"git_repository": schema.StringAttribute{
			MarkdownDescription: gitRepositoryDescription,
			Required:            true,
		},
		"git_branch": schema.StringAttribute{
			MarkdownDescription: "The Git branch to deploy (defaults to `main`).",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("main"),
		},
		"build_pack": schema.StringAttribute{
			MarkdownDescription: "The build pack type. Valid values: `nixpacks`, `dockerfile`, `dockercompose`, `static`, `railpack`.",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.OneOf("nixpacks", "dockerfile", "dockercompose", "static", "railpack"),
			},
		},
		"ports_exposes": schema.StringAttribute{
			MarkdownDescription: "The ports to expose, as a comma-separated list (e.g., `3000` or `3000,8080`).",
			Required:            true,
			Validators: []validator.String{
				stringvalidator.RegexMatches(regexp.MustCompile(`^\d+(,\d+)*$`), "must be a comma-separated list of port numbers (e.g., \"3000\" or \"3000,8080\")"),
			},
		},
		"dockerfile_location": schema.StringAttribute{
			MarkdownDescription: "The path to the Dockerfile, relative to the repository root.",
			Optional:            true,
		},
	}
}

func gitAppCommandAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"install_command": schema.StringAttribute{
			MarkdownDescription: "The command to run during the install phase.",
			Optional:            true,
		},
		"build_command": schema.StringAttribute{
			MarkdownDescription: "The command to run during the build phase.",
			Optional:            true,
		},
		"start_command": schema.StringAttribute{
			MarkdownDescription: "The command to run to start the application.",
			Optional:            true,
		},
	}
}

// coreAppAttrs returns the core schema attributes (identity, status, limits,
// existing health checks, auto-deploy).
func coreAppAttrs(ctx context.Context) map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"timeouts": timeouts.Attributes(ctx, timeouts.Opts{Create: true}),
		"uuid": schema.StringAttribute{
			MarkdownDescription: "The unique identifier of the application.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"name": schema.StringAttribute{
			MarkdownDescription: "The name of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"description": schema.StringAttribute{
			MarkdownDescription: "A description of the application.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"project_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the project this application belongs to. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"server_uuid": schema.StringAttribute{
			MarkdownDescription: "The UUID of the server to deploy the application on. Changing this forces a new resource.",
			Required:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
			Validators:          []validator.String{validate.UUID()},
		},
		"environment_name": schema.StringAttribute{
			MarkdownDescription: "The environment name for the application (defaults to `production`). Changing this forces a new resource.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("production"),
			PlanModifiers:       []planmodifier.String{stringplanmodifier.RequiresReplace()},
		},
		"domains": schema.StringAttribute{
			MarkdownDescription: "The fully qualified domain name for the application (must start with http:// or https://).",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
			Validators:          []validator.String{validate.Domains()},
		},
		"status": schema.StringAttribute{
			MarkdownDescription: "The current status of the application (e.g., running, stopped, exited). Read-only.",
			Computed:            true,
		},
		// Resource limits
		"limits_memory": schema.StringAttribute{
			MarkdownDescription: "Memory limit (e.g., `512m`, `2g`).",
			Optional:            true,
		},
		"limits_memory_swap": schema.StringAttribute{
			MarkdownDescription: "Memory swap limit (e.g., `1g`).",
			Optional:            true,
		},
		"limits_memory_swappiness": schema.Int64Attribute{
			MarkdownDescription: "Memory swappiness (0-100).",
			Optional:            true,
		},
		"limits_memory_reservation": schema.StringAttribute{
			MarkdownDescription: "Memory reservation (e.g., `256m`).",
			Optional:            true,
		},
		"limits_cpus": schema.StringAttribute{
			MarkdownDescription: "CPU limit (e.g., `0.5`, `2`).",
			Optional:            true,
		},
		"limits_cpuset": schema.StringAttribute{
			MarkdownDescription: "CPU set restriction (e.g., `0-3`, `0,2`).",
			Optional:            true,
		},
		"limits_cpu_shares": schema.Int64Attribute{
			MarkdownDescription: "CPU shares (relative weight).",
			Optional:            true,
		},
		// Health checks
		"health_check_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether health checks are enabled. Coolify defaults to `false` for new applications.",
			Optional:            true,
			Computed:            true,
			PlanModifiers:       []planmodifier.Bool{boolplanmodifier.UseStateForUnknown()},
		},
		"health_check_path": schema.StringAttribute{
			MarkdownDescription: "The URL path for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString("/"),
		},
		"health_check_port": schema.StringAttribute{
			MarkdownDescription: "The port for health checks.",
			Optional:            true,
		},
		"health_check_interval": schema.Int64Attribute{
			MarkdownDescription: "Health check interval in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		"health_check_timeout": schema.Int64Attribute{
			MarkdownDescription: "Health check timeout in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		"health_check_retries": schema.Int64Attribute{
			MarkdownDescription: "Number of health check retries.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(10),
		},
		"health_check_start_period": schema.Int64Attribute{
			MarkdownDescription: "Health check start period in seconds.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(5),
		},
		// Auto-deploy
		"is_auto_deploy_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether auto-deploy on push is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"redeploy_on_update": schema.BoolAttribute{
			MarkdownDescription: "When `true`, the application is automatically restarted after a Terraform update that changes any configuration field. This covers all non-immutable, non-computed attributes including `name`, `description`, network settings (`ports_exposes`, `ports_mappings`, `domains`), resource limits (`limits_*`), health checks, build settings (`build_pack`, `build_command`, `dockerfile_location`, `base_directory`), deployment commands, container settings (`custom_labels`, `custom_docker_run_options`, `custom_nginx_configuration`), security (`is_force_https_enabled`, HTTP basic auth), webhook secrets (`manual_webhook_secret_*`), auto-deploy and static site settings, and type-specific fields (e.g., `docker_image`, `github_app_uuid`). Only immutable fields (`project_uuid`, `server_uuid`, `environment_name`), computed-only fields (`status`, `preview_url_template`), and the `redeploy_on_update` flag itself are excluded. Defaults to `false`.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
	}
}

// extendedBuildDeployAttrs returns schema attributes for build, deploy, and static settings.
func extendedBuildDeployAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"base_directory": schema.StringAttribute{
			MarkdownDescription: "The base directory for the application source code.",
			Optional:            true,
		},
		"publish_directory": schema.StringAttribute{
			MarkdownDescription: "The directory to publish for static sites.",
			Optional:            true,
		},
		"dockerfile": schema.StringAttribute{
			MarkdownDescription: "Inline Dockerfile content (base64 encoded). For `coolify_application_dockerfile` resources, use `dockerfile_location` instead; this field is only used by Git-backed application types that embed a Dockerfile inline.",
			Optional:            true,
			Sensitive:           true,
		},
		"docker_registry_image_tag": schema.StringAttribute{
			MarkdownDescription: "The Docker registry image tag.",
			Optional:            true,
		},
		"docker_compose_domains": schema.StringAttribute{
			MarkdownDescription: "Domain mappings for Docker Compose services.",
			Optional:            true,
		},
		"git_commit_sha": schema.StringAttribute{
			MarkdownDescription: "The specific Git commit SHA to deploy.",
			Optional:            true,
		},
		"watch_paths": schema.StringAttribute{
			MarkdownDescription: "Paths to watch for changes (triggers auto-deploy).",
			Optional:            true,
		},
		"redirect": schema.StringAttribute{
			MarkdownDescription: "Domain redirect mode. Valid values: `www`, `non-www`, `both`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultRedirect),
			Validators:          []validator.String{stringvalidator.OneOf("www", "non-www", "both")},
		},
		"static_image": schema.StringAttribute{
			MarkdownDescription: "The Docker image to use for serving static sites.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultStaticImage),
		},
		"is_static": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a static site.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_spa": schema.BoolAttribute{
			MarkdownDescription: "Whether the application is a single-page application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"is_preserve_repository_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to preserve the full Git repository (instead of shallow clone).",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"use_build_server": schema.BoolAttribute{
			MarkdownDescription: "Whether to use a build server for building the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"instant_deploy": schema.BoolAttribute{
			MarkdownDescription: "Whether to immediately deploy the application after creation. When `true`, Coolify triggers a deployment right away. When `false` (default), the application is created but not deployed.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"preview_url_template": schema.StringAttribute{
			MarkdownDescription: "The URL template for preview deployments. Read-only until Coolify supports setting it on create or update.",
			Computed:            true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"pre_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run before deployment.",
			Optional:            true,
		},
		"pre_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the pre-deployment command in.",
			Optional:            true,
		},
		"post_deployment_command": schema.StringAttribute{
			MarkdownDescription: "Command to run after deployment.",
			Optional:            true,
		},
		"post_deployment_command_container": schema.StringAttribute{
			MarkdownDescription: "Container to run the post-deployment command in.",
			Optional:            true,
		},
	}
}

// extendedHealthCheckAttrs returns schema attributes for extended health check settings.
func extendedHealthCheckAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"health_check_command": schema.StringAttribute{
			MarkdownDescription: "Custom health check command (used when type is `cmd`).",
			Optional:            true,
		},
		"health_check_host": schema.StringAttribute{
			MarkdownDescription: "The host for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckHost),
		},
		"health_check_method": schema.StringAttribute{
			MarkdownDescription: "The HTTP method for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckMeth),
			Validators:          []validator.String{stringvalidator.OneOf("GET", "HEAD", "POST", "OPTIONS")},
		},
		"health_check_response_text": schema.StringAttribute{
			MarkdownDescription: "Expected response text for health check validation.",
			Optional:            true,
		},
		"health_check_return_code": schema.Int64Attribute{
			MarkdownDescription: "Expected HTTP return code for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             int64default.StaticInt64(defaultHealthCheckCode),
		},
		"health_check_scheme": schema.StringAttribute{
			MarkdownDescription: "The URL scheme for health checks.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckSchm),
			Validators:          []validator.String{stringvalidator.OneOf("http", "https")},
		},
		"health_check_type": schema.StringAttribute{
			MarkdownDescription: "The type of health check. Valid values: `http`, `cmd`.",
			Optional:            true,
			Computed:            true,
			Default:             stringdefault.StaticString(defaultHealthCheckType),
			Validators:          []validator.String{stringvalidator.OneOf("http", "cmd")},
		},
	}
}

// securityNetworkAttrs returns schema attributes for network, security, auth, and webhook settings.
func securityNetworkAttrs() map[string]schema.Attribute {
	return map[string]schema.Attribute{
		"connect_to_docker_network": schema.BoolAttribute{
			MarkdownDescription: "Whether to connect the application to the Docker network.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"custom_docker_run_options": schema.StringAttribute{
			MarkdownDescription: "Custom Docker run options passed to the container.",
			Optional:            true,
			Validators: []validator.String{
				validate.NoShellMetachars(),
			},
		},
		"custom_labels": schema.StringAttribute{
			MarkdownDescription: "Custom Docker labels for the container, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"custom_network_aliases": schema.StringAttribute{
			MarkdownDescription: "Custom network aliases for the container.",
			Optional:            true,
		},
		"custom_nginx_configuration": schema.StringAttribute{
			MarkdownDescription: "Custom Nginx configuration for the application, **base64-encoded**. Use `base64encode()` in your configuration.",
			Optional:            true,
		},
		"ports_mappings": schema.StringAttribute{
			MarkdownDescription: "Port mappings in `host:container` format, comma-separated (e.g., `8080:80` or `8080:80,8443:443`).",
			Optional:            true,
			Validators: []validator.String{
				validate.PortMappings(),
			},
		},
		"force_domain_override": schema.BoolAttribute{
			MarkdownDescription: "Whether to force domain override.",
			Optional:            true,
		},
		"is_container_label_escape_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether container label escaping is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_force_https_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether to force HTTPS for the application.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(true),
		},
		"is_http_basic_auth_enabled": schema.BoolAttribute{
			MarkdownDescription: "Whether HTTP Basic Authentication is enabled.",
			Optional:            true,
			Computed:            true,
			Default:             booldefault.StaticBool(false),
		},
		"http_basic_auth_username": schema.StringAttribute{
			MarkdownDescription: "Username for HTTP Basic Authentication.",
			Optional:            true,
		},
		"http_basic_auth_password": schema.StringAttribute{
			MarkdownDescription: "Password for HTTP Basic Authentication.",
			Optional:            true,
			Sensitive:           true,
		},
		"manual_webhook_secret_bitbucket": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Bitbucket.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitea": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for Gitea.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_github": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitHub.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
		"manual_webhook_secret_gitlab": schema.StringAttribute{
			MarkdownDescription: "Manual webhook secret for GitLab.",
			Optional:            true,
			Computed:            true,
			Sensitive:           true,
			PlanModifiers:       []planmodifier.String{stringplanmodifier.UseStateForUnknown()},
		},
	}
}
