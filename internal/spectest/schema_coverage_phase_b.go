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
	{ContractField: "custom_internal_name", Status: SkipInternal, Notes: "internal name"},
	// Deferred product settings (#626)
	{ContractField: "disable_build_cache", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "docker_images_to_keep", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "gpu_count", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "gpu_device_ids", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "gpu_driver", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "gpu_options", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "include_source_commit_in_build", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "inject_build_args_to_dockerfile", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_consistent_container_name_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_container_label_readonly_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_debug_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_env_sorting_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_git_lfs_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_git_shallow_clone_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_git_submodules_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_gpu_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_gzip_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_include_timestamps", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_log_drain_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_pr_deployments_public_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_raw_compose_deployment_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_stripprefix_enabled", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
	{ContractField: "is_swarm_only_worker_nodes", Status: SkipDeferred, Issue: 626, Notes: "not yet on application schema"},
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
