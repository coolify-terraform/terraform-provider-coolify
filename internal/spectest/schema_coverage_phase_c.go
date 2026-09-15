package spectest

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// applicationAllowListSchemaRegistry is Phase C: every ApplicationsController
// create_application and update_by_uuid $allowedFields entry maps to an
// application resource schema attribute or an explicit skip. Public allow-list
// fields must never use SkipInternal.
var applicationAllowListSchemaRegistry = []SchemaCoverageEntry{
	// Identity / placement
	{ContractField: "project_uuid", SchemaAttribute: "project_uuid", Status: StatusCovered},
	{ContractField: "environment_name", SchemaAttribute: "environment_name", Status: StatusCovered},
	{ContractField: "environment_uuid", Status: SkipNA, Notes: "provider uses environment_name on create; Coolify accepts either"},
	{ContractField: "server_uuid", SchemaAttribute: "server_uuid", Status: StatusCovered},
	{ContractField: "destination_uuid", SchemaAttribute: "destination_uuid", Status: StatusCovered},
	{ContractField: "type", Status: SkipNA, Notes: "Coolify create discriminator (public/private/github/dockerfile/dockerimage), not a Terraform attribute"},
	{ContractField: "name", SchemaAttribute: "name", Status: StatusCovered},
	{ContractField: "description", SchemaAttribute: "description", Status: StatusCovered},

	// Git / source
	{ContractField: "git_repository", SchemaAttribute: "git_repository", Status: StatusCovered},
	{ContractField: "git_branch", SchemaAttribute: "git_branch", Status: StatusCovered},
	{ContractField: "git_commit_sha", SchemaAttribute: "git_commit_sha", Status: StatusCovered},
	{ContractField: "private_key_uuid", SchemaAttribute: "private_key_uuid", Status: StatusCovered, Notes: "coolify_application_private_git"},
	{ContractField: "github_app_uuid", SchemaAttribute: "github_app_uuid", Status: StatusCovered, Notes: "coolify_application_github_app"},
	{ContractField: "build_pack", SchemaAttribute: "build_pack", Status: StatusCovered},
	{ContractField: "dockerfile", SchemaAttribute: "dockerfile", Status: StatusCovered},
	{ContractField: "dockerfile_location", SchemaAttribute: "dockerfile_location", Status: StatusCovered},
	{ContractField: "dockerfile_target_build", SchemaAttribute: "dockerfile_target_build", Status: StatusCovered},
	{ContractField: "docker_registry_image_name", SchemaAttribute: "docker_image", Status: StatusCovered, Notes: "coolify_application_docker_image remaps to docker_image"},
	{ContractField: "docker_registry_image_tag", SchemaAttribute: "docker_registry_image_tag", Status: StatusCovered},

	// Compose
	{ContractField: "docker_compose_location", SchemaAttribute: "docker_compose_location", Status: StatusCovered},
	{ContractField: "docker_compose_custom_start_command", SchemaAttribute: "docker_compose_custom_start_command", Status: StatusCovered},
	{ContractField: "docker_compose_custom_build_command", SchemaAttribute: "docker_compose_custom_build_command", Status: StatusCovered},
	{ContractField: "docker_compose_domains", SchemaAttribute: "docker_compose_domains", Status: StatusCovered},
	{ContractField: "docker_compose_raw", Status: SkipNA, Notes: "inline compose belongs to coolify_service / dockerfile-style resources, not git-backed application write"},

	// Commands / ports
	{ContractField: "install_command", SchemaAttribute: "install_command", Status: StatusCovered},
	{ContractField: "build_command", SchemaAttribute: "build_command", Status: StatusCovered},
	{ContractField: "start_command", SchemaAttribute: "start_command", Status: StatusCovered},
	{ContractField: "ports_exposes", SchemaAttribute: "ports_exposes", Status: StatusCovered},
	{ContractField: "ports_mappings", SchemaAttribute: "ports_mappings", Status: StatusCovered},
	{ContractField: "base_directory", SchemaAttribute: "base_directory", Status: StatusCovered},
	{ContractField: "publish_directory", SchemaAttribute: "publish_directory", Status: StatusCovered},
	{ContractField: "watch_paths", SchemaAttribute: "watch_paths", Status: StatusCovered},

	// Health checks
	{ContractField: "health_check_enabled", SchemaAttribute: "health_check_enabled", Status: StatusCovered},
	{ContractField: "health_check_type", SchemaAttribute: "health_check_type", Status: StatusCovered},
	{ContractField: "health_check_command", SchemaAttribute: "health_check_command", Status: StatusCovered},
	{ContractField: "health_check_path", SchemaAttribute: "health_check_path", Status: StatusCovered},
	{ContractField: "health_check_port", SchemaAttribute: "health_check_port", Status: StatusCovered},
	{ContractField: "health_check_host", SchemaAttribute: "health_check_host", Status: StatusCovered},
	{ContractField: "health_check_method", SchemaAttribute: "health_check_method", Status: StatusCovered},
	{ContractField: "health_check_return_code", SchemaAttribute: "health_check_return_code", Status: StatusCovered},
	{ContractField: "health_check_scheme", SchemaAttribute: "health_check_scheme", Status: StatusCovered},
	{ContractField: "health_check_response_text", SchemaAttribute: "health_check_response_text", Status: StatusCovered},
	{ContractField: "health_check_interval", SchemaAttribute: "health_check_interval", Status: StatusCovered},
	{ContractField: "health_check_timeout", SchemaAttribute: "health_check_timeout", Status: StatusCovered},
	{ContractField: "health_check_retries", SchemaAttribute: "health_check_retries", Status: StatusCovered},
	{ContractField: "health_check_start_period", SchemaAttribute: "health_check_start_period", Status: StatusCovered},

	// Limits
	{ContractField: "limits_memory", SchemaAttribute: "limits_memory", Status: StatusCovered},
	{ContractField: "limits_memory_swap", SchemaAttribute: "limits_memory_swap", Status: StatusCovered},
	{ContractField: "limits_memory_swappiness", SchemaAttribute: "limits_memory_swappiness", Status: StatusCovered},
	{ContractField: "limits_memory_reservation", SchemaAttribute: "limits_memory_reservation", Status: StatusCovered},
	{ContractField: "limits_cpus", SchemaAttribute: "limits_cpus", Status: StatusCovered},
	{ContractField: "limits_cpuset", SchemaAttribute: "limits_cpuset", Status: StatusCovered},
	{ContractField: "limits_cpu_shares", SchemaAttribute: "limits_cpu_shares", Status: StatusCovered},

	// Network / container
	{ContractField: "custom_labels", SchemaAttribute: "custom_labels", Status: StatusCovered},
	{ContractField: "custom_docker_run_options", SchemaAttribute: "custom_docker_run_options", Status: StatusCovered},
	{ContractField: "custom_network_aliases", SchemaAttribute: "custom_network_aliases", Status: StatusCovered},
	{ContractField: "custom_nginx_configuration", SchemaAttribute: "custom_nginx_configuration", Status: StatusCovered},
	{ContractField: "connect_to_docker_network", SchemaAttribute: "connect_to_docker_network", Status: StatusCovered},
	{ContractField: "force_domain_override", SchemaAttribute: "force_domain_override", Status: StatusCovered},
	{ContractField: "domains", SchemaAttribute: "domains", Status: StatusCovered},
	{ContractField: "noindex_domains", SchemaAttribute: "noindex_domains", Status: StatusCovered},
	{ContractField: "redirect", SchemaAttribute: "redirect", Status: StatusCovered},

	// Deploy / static
	{ContractField: "is_static", SchemaAttribute: "is_static", Status: StatusCovered},
	{ContractField: "is_spa", SchemaAttribute: "is_spa", Status: StatusCovered},
	{ContractField: "is_auto_deploy_enabled", SchemaAttribute: "is_auto_deploy_enabled", Status: StatusCovered},
	{ContractField: "is_force_https_enabled", SchemaAttribute: "is_force_https_enabled", Status: StatusCovered},
	{ContractField: "is_preview_deployments_enabled", SchemaAttribute: "is_preview_deployments_enabled", Status: StatusCovered},
	{ContractField: "is_preserve_repository_enabled", SchemaAttribute: "is_preserve_repository_enabled", Status: StatusCovered},
	{ContractField: "instant_deploy", SchemaAttribute: "instant_deploy", Status: StatusCovered},
	{ContractField: "autogenerate_domain", SchemaAttribute: "autogenerate_domain", Status: StatusCovered},
	{ContractField: "use_build_server", SchemaAttribute: "use_build_server", Status: StatusCovered},
	{ContractField: "use_build_secrets", SchemaAttribute: "use_build_secrets", Status: StatusCovered},
	{ContractField: "static_image", SchemaAttribute: "static_image", Status: StatusCovered},
	{ContractField: "preview_url_template", SchemaAttribute: "preview_url_template", Status: StatusCovered},
	{ContractField: "max_restart_count", SchemaAttribute: "max_restart_count", Status: StatusCovered},
	{ContractField: "pre_deployment_command", SchemaAttribute: "pre_deployment_command", Status: StatusCovered},
	{ContractField: "pre_deployment_command_container", SchemaAttribute: "pre_deployment_command_container", Status: StatusCovered},
	{ContractField: "post_deployment_command", SchemaAttribute: "post_deployment_command", Status: StatusCovered},
	{ContractField: "post_deployment_command_container", SchemaAttribute: "post_deployment_command_container", Status: StatusCovered},

	// Auth / webhooks
	{ContractField: "is_http_basic_auth_enabled", SchemaAttribute: "is_http_basic_auth_enabled", Status: StatusCovered},
	{ContractField: "http_basic_auth_username", SchemaAttribute: "http_basic_auth_username", Status: StatusCovered},
	{ContractField: "http_basic_auth_password", SchemaAttribute: "http_basic_auth_password", Status: StatusCovered},
	{ContractField: "manual_webhook_secret_github", SchemaAttribute: "manual_webhook_secret_github", Status: StatusCovered},
	{ContractField: "manual_webhook_secret_gitlab", SchemaAttribute: "manual_webhook_secret_gitlab", Status: StatusCovered},
	{ContractField: "manual_webhook_secret_bitbucket", SchemaAttribute: "manual_webhook_secret_bitbucket", Status: StatusCovered},
	{ContractField: "manual_webhook_secret_gitea", SchemaAttribute: "manual_webhook_secret_gitea", Status: StatusCovered},
	{ContractField: "is_container_label_escape_enabled", SchemaAttribute: "is_container_label_escape_enabled", Status: StatusCovered},

	// Application settings also on the allow-list
	{ContractField: "is_git_submodules_enabled", SchemaAttribute: "is_git_submodules_enabled", Status: StatusCovered},
	{ContractField: "is_git_lfs_enabled", SchemaAttribute: "is_git_lfs_enabled", Status: StatusCovered},
	{ContractField: "is_git_shallow_clone_enabled", SchemaAttribute: "is_git_shallow_clone_enabled", Status: StatusCovered},
	{ContractField: "disable_build_cache", SchemaAttribute: "disable_build_cache", Status: StatusCovered},
	{ContractField: "inject_build_args_to_dockerfile", SchemaAttribute: "inject_build_args_to_dockerfile", Status: StatusCovered},
	{ContractField: "include_source_commit_in_build", SchemaAttribute: "include_source_commit_in_build", Status: StatusCovered},
	{ContractField: "is_env_sorting_enabled", SchemaAttribute: "is_env_sorting_enabled", Status: StatusCovered},
	{ContractField: "is_pr_deployments_public_enabled", SchemaAttribute: "is_pr_deployments_public_enabled", Status: StatusCovered},
	{ContractField: "stop_grace_period", SchemaAttribute: "stop_grace_period", Status: StatusCovered},
	{ContractField: "docker_images_to_keep", SchemaAttribute: "docker_images_to_keep", Status: StatusCovered},
	{ContractField: "is_gzip_enabled", SchemaAttribute: "is_gzip_enabled", Status: StatusCovered},
	{ContractField: "is_stripprefix_enabled", SchemaAttribute: "is_stripprefix_enabled", Status: StatusCovered},
	{ContractField: "is_raw_compose_deployment_enabled", SchemaAttribute: "is_raw_compose_deployment_enabled", Status: StatusCovered},
	{ContractField: "is_log_drain_enabled", SchemaAttribute: "is_log_drain_enabled", Status: StatusCovered},
	{ContractField: "is_gpu_enabled", SchemaAttribute: "is_gpu_enabled", Status: StatusCovered},
	{ContractField: "gpu_driver", SchemaAttribute: "gpu_driver", Status: StatusCovered},
	{ContractField: "gpu_count", SchemaAttribute: "gpu_count", Status: StatusCovered},
	{ContractField: "gpu_device_ids", SchemaAttribute: "gpu_device_ids", Status: StatusCovered},
	{ContractField: "gpu_options", SchemaAttribute: "gpu_options", Status: StatusCovered},
	{ContractField: "is_consistent_container_name_enabled", SchemaAttribute: "is_consistent_container_name_enabled", Status: StatusCovered},
	{ContractField: "custom_internal_name", SchemaAttribute: "custom_internal_name", Status: StatusCovered},

	// Wrong surface
	{ContractField: "tags", Status: SkipNA, Notes: "managed by coolify_tag / application-tag attach, not application write schema"},
}

func coolifyPrivateGitApplicationResource() resource.Resource {
	return application.NewPrivateGitResource()
}

func coolifyGitHubAppApplicationResource() resource.Resource {
	return application.NewGitHubAppResource()
}

func coolifyDockerfileApplicationResource() resource.Resource {
	return application.NewDockerfileResource()
}

func coolifyDockerImageApplicationResource() resource.Resource {
	return application.NewDockerResource()
}

func applicationResourceSchemaUnion() (map[string]struct{}, error) {
	resources := []resource.Resource{
		coolifyApplicationResource(),
		coolifyPrivateGitApplicationResource(),
		coolifyGitHubAppApplicationResource(),
		coolifyDockerfileApplicationResource(),
		coolifyDockerImageApplicationResource(),
	}
	out := map[string]struct{}{}
	for _, r := range resources {
		attrs, err := resourceSchemaAttributeNames(r)
		if err != nil {
			return nil, err
		}
		for name := range attrs {
			out[name] = struct{}{}
		}
	}
	return out, nil
}
