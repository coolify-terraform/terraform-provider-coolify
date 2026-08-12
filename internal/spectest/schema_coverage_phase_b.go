package spectest

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/scheduledtask"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// applicationSettingsSchemaRegistry is Phase B: every fillable Application
// settings_fields row must map to coolify_application schema or an explicit skip.
// Contract field names use Coolify settings column names (is_build_server_enabled).
var applicationSettingsSchemaRegistry = []SchemaCoverageEntry{
	// Covered on coolify_application (shared common schema)
	{ContractField: "connect_to_docker_network", SchemaAttribute: "connect_to_docker_network", Status: StatusCovered},
	{ContractField: "is_auto_deploy_enabled", SchemaAttribute: "is_auto_deploy_enabled", Status: StatusCovered},
	{ContractField: "is_build_server_enabled", SchemaAttribute: "use_build_server", Status: StatusCovered, Notes: "API write key use_build_server"},
	{ContractField: "is_container_label_escape_enabled", SchemaAttribute: "is_container_label_escape_enabled", Status: StatusCovered},
	{ContractField: "is_force_https_enabled", SchemaAttribute: "is_force_https_enabled", Status: StatusCovered},
	{ContractField: "is_preserve_repository_enabled", SchemaAttribute: "is_preserve_repository_enabled", Status: StatusCovered},
	{ContractField: "is_preview_deployments_enabled", SchemaAttribute: "is_preview_deployments_enabled", Status: StatusCovered},
	{ContractField: "is_spa", SchemaAttribute: "is_spa", Status: StatusCovered},
	{ContractField: "is_static", SchemaAttribute: "is_static", Status: StatusCovered},
	{ContractField: "stop_grace_period", SchemaAttribute: "stop_grace_period", Status: StatusCovered},
	{ContractField: "use_build_secrets", SchemaAttribute: "use_build_secrets", Status: StatusCovered},
	// Internal / not schema surface
	{ContractField: "application_id", Status: SkipInternal, Notes: "settings FK"},
	// Public APPLICATION_SETTING_FIELDS (v4.2.0+) and v4.3.0 additions.
	{ContractField: "custom_internal_name", SchemaAttribute: "custom_internal_name", Status: StatusCovered},
	{ContractField: "disable_build_cache", SchemaAttribute: "disable_build_cache", Status: StatusCovered},
	{ContractField: "docker_images_to_keep", SchemaAttribute: "docker_images_to_keep", Status: StatusCovered},
	{ContractField: "gpu_count", SchemaAttribute: "gpu_count", Status: StatusCovered},
	{ContractField: "gpu_device_ids", SchemaAttribute: "gpu_device_ids", Status: StatusCovered},
	{ContractField: "gpu_driver", SchemaAttribute: "gpu_driver", Status: StatusCovered},
	{ContractField: "gpu_options", SchemaAttribute: "gpu_options", Status: StatusCovered},
	{ContractField: "include_source_commit_in_build", SchemaAttribute: "include_source_commit_in_build", Status: StatusCovered},
	{ContractField: "inject_build_args_to_dockerfile", SchemaAttribute: "inject_build_args_to_dockerfile", Status: StatusCovered},
	{ContractField: "is_consistent_container_name_enabled", SchemaAttribute: "is_consistent_container_name_enabled", Status: StatusCovered},
	// Still UI-only (not on ApplicationsController APPLICATION_SETTING_FIELDS).
	{ContractField: "is_container_label_readonly_enabled", Status: SkipNA, Notes: "not in ApplicationsController APPLICATION_SETTING_FIELDS public allow-list"},
	{ContractField: "is_debug_enabled", Status: SkipNA, Notes: "not in ApplicationsController APPLICATION_SETTING_FIELDS public allow-list"},
	{ContractField: "is_env_sorting_enabled", SchemaAttribute: "is_env_sorting_enabled", Status: StatusCovered},
	{ContractField: "is_git_lfs_enabled", SchemaAttribute: "is_git_lfs_enabled", Status: StatusCovered},
	{ContractField: "is_git_shallow_clone_enabled", SchemaAttribute: "is_git_shallow_clone_enabled", Status: StatusCovered},
	{ContractField: "is_git_submodules_enabled", SchemaAttribute: "is_git_submodules_enabled", Status: StatusCovered},
	{ContractField: "is_gpu_enabled", SchemaAttribute: "is_gpu_enabled", Status: StatusCovered},
	{ContractField: "is_gzip_enabled", SchemaAttribute: "is_gzip_enabled", Status: StatusCovered},
	{ContractField: "is_include_timestamps", Status: SkipNA, Notes: "not in ApplicationsController APPLICATION_SETTING_FIELDS public allow-list"},
	{ContractField: "is_log_drain_enabled", SchemaAttribute: "is_log_drain_enabled", Status: StatusCovered},
	{ContractField: "is_pr_deployments_public_enabled", SchemaAttribute: "is_pr_deployments_public_enabled", Status: StatusCovered},
	{ContractField: "is_raw_compose_deployment_enabled", SchemaAttribute: "is_raw_compose_deployment_enabled", Status: StatusCovered},
	{ContractField: "is_stripprefix_enabled", SchemaAttribute: "is_stripprefix_enabled", Status: StatusCovered},
	{ContractField: "is_swarm_only_worker_nodes", Status: SkipNA, Notes: "not in ApplicationsController APPLICATION_SETTING_FIELDS public allow-list"},
}

// scheduledTaskSchemaRegistry covers every fillable ScheduledTask contract field.
var scheduledTaskSchemaRegistry = []SchemaCoverageEntry{
	{ContractField: "name", SchemaAttribute: "name", Status: StatusCovered},
	{ContractField: "command", SchemaAttribute: "command", Status: StatusCovered},
	{ContractField: "frequency", SchemaAttribute: "frequency", Status: StatusCovered},
	{ContractField: "enabled", SchemaAttribute: "enabled", Status: StatusCovered},
	{ContractField: "container", SchemaAttribute: "container", Status: StatusCovered},
	{ContractField: "timeout", SchemaAttribute: "timeout", Status: StatusCovered},
	{ContractField: "uuid", SchemaAttribute: "uuid", Status: StatusCovered},
	{ContractField: "application_id", Status: SkipInternal, Notes: "numeric FK; provider uses application_uuid"},
	{ContractField: "service_id", Status: SkipInternal, Notes: "numeric FK; provider uses service_uuid"},
}

func coolifyApplicationResource() resource.Resource {
	return application.NewResource()
}

func coolifyScheduledTaskResource() resource.Resource {
	return scheduledtask.NewResource()
}
