package notificationcommon

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ThreadModel is the 15 Telegram forum thread-id strings. Embed it in the
// Telegram resource model so tfsdk names stay promoted (thread_deployment_success, …).
// Method names are Thread-prefixed so they do not collide with EventModel
// when both are embedded.
type ThreadModel struct {
	ThreadDeploymentSuccess    types.String `tfsdk:"thread_deployment_success"`
	ThreadDeploymentFailure    types.String `tfsdk:"thread_deployment_failure"`
	ThreadStatusChange         types.String `tfsdk:"thread_status_change"`
	ThreadRestartLimitReached  types.String `tfsdk:"thread_restart_limit_reached"`
	ThreadBackupSuccess        types.String `tfsdk:"thread_backup_success"`
	ThreadBackupFailure        types.String `tfsdk:"thread_backup_failure"`
	ThreadScheduledTaskSuccess types.String `tfsdk:"thread_scheduled_task_success"`
	ThreadScheduledTaskFailure types.String `tfsdk:"thread_scheduled_task_failure"`
	ThreadDockerCleanupSuccess types.String `tfsdk:"thread_docker_cleanup_success"`
	ThreadDockerCleanupFailure types.String `tfsdk:"thread_docker_cleanup_failure"`
	ThreadServerDiskUsage      types.String `tfsdk:"thread_server_disk_usage"`
	ThreadServerReachable      types.String `tfsdk:"thread_server_reachable"`
	ThreadServerUnreachable    types.String `tfsdk:"thread_server_unreachable"`
	ThreadServerPatch          types.String `tfsdk:"thread_server_patch"`
	ThreadTraefikOutdated      types.String `tfsdk:"thread_traefik_outdated"`
}

// CreateThreadUpdate returns PATCH thread-id pointers for every known plan value.
func (t ThreadModel) CreateThreadUpdate() client.NotificationThreadUpdate {
	return client.NotificationThreadUpdate{
		ThreadDeploymentSuccess:    flex.StringValueOrNull(t.ThreadDeploymentSuccess),
		ThreadDeploymentFailure:    flex.StringValueOrNull(t.ThreadDeploymentFailure),
		ThreadStatusChange:         flex.StringValueOrNull(t.ThreadStatusChange),
		ThreadRestartLimitReached:  flex.StringValueOrNull(t.ThreadRestartLimitReached),
		ThreadBackupSuccess:        flex.StringValueOrNull(t.ThreadBackupSuccess),
		ThreadBackupFailure:        flex.StringValueOrNull(t.ThreadBackupFailure),
		ThreadScheduledTaskSuccess: flex.StringValueOrNull(t.ThreadScheduledTaskSuccess),
		ThreadScheduledTaskFailure: flex.StringValueOrNull(t.ThreadScheduledTaskFailure),
		ThreadDockerCleanupSuccess: flex.StringValueOrNull(t.ThreadDockerCleanupSuccess),
		ThreadDockerCleanupFailure: flex.StringValueOrNull(t.ThreadDockerCleanupFailure),
		ThreadServerDiskUsage:      flex.StringValueOrNull(t.ThreadServerDiskUsage),
		ThreadServerReachable:      flex.StringValueOrNull(t.ThreadServerReachable),
		ThreadServerUnreachable:    flex.StringValueOrNull(t.ThreadServerUnreachable),
		ThreadServerPatch:          flex.StringValueOrNull(t.ThreadServerPatch),
		ThreadTraefikOutdated:      flex.StringValueOrNull(t.ThreadTraefikOutdated),
	}
}

// DiffThreadUpdate returns PATCH thread-id pointers only for values that changed.
func (t ThreadModel) DiffThreadUpdate(state ThreadModel) client.NotificationThreadUpdate {
	return client.NotificationThreadUpdate{
		ThreadDeploymentSuccess:    flex.StringIfChanged(t.ThreadDeploymentSuccess, state.ThreadDeploymentSuccess),
		ThreadDeploymentFailure:    flex.StringIfChanged(t.ThreadDeploymentFailure, state.ThreadDeploymentFailure),
		ThreadStatusChange:         flex.StringIfChanged(t.ThreadStatusChange, state.ThreadStatusChange),
		ThreadRestartLimitReached:  flex.StringIfChanged(t.ThreadRestartLimitReached, state.ThreadRestartLimitReached),
		ThreadBackupSuccess:        flex.StringIfChanged(t.ThreadBackupSuccess, state.ThreadBackupSuccess),
		ThreadBackupFailure:        flex.StringIfChanged(t.ThreadBackupFailure, state.ThreadBackupFailure),
		ThreadScheduledTaskSuccess: flex.StringIfChanged(t.ThreadScheduledTaskSuccess, state.ThreadScheduledTaskSuccess),
		ThreadScheduledTaskFailure: flex.StringIfChanged(t.ThreadScheduledTaskFailure, state.ThreadScheduledTaskFailure),
		ThreadDockerCleanupSuccess: flex.StringIfChanged(t.ThreadDockerCleanupSuccess, state.ThreadDockerCleanupSuccess),
		ThreadDockerCleanupFailure: flex.StringIfChanged(t.ThreadDockerCleanupFailure, state.ThreadDockerCleanupFailure),
		ThreadServerDiskUsage:      flex.StringIfChanged(t.ThreadServerDiskUsage, state.ThreadServerDiskUsage),
		ThreadServerReachable:      flex.StringIfChanged(t.ThreadServerReachable, state.ThreadServerReachable),
		ThreadServerUnreachable:    flex.StringIfChanged(t.ThreadServerUnreachable, state.ThreadServerUnreachable),
		ThreadServerPatch:          flex.StringIfChanged(t.ThreadServerPatch, state.ThreadServerPatch),
		ThreadTraefikOutdated:      flex.StringIfChanged(t.ThreadTraefikOutdated, state.ThreadTraefikOutdated),
	}
}

// FlattenThreads writes API thread IDs into the model, preserving state when
// Coolify hides empty sensitive values.
func (t *ThreadModel) FlattenThreads(src client.NotificationThreads) {
	flex.SetStringPreserveEmpty(&t.ThreadDeploymentSuccess, src.ThreadDeploymentSuccess)
	flex.SetStringPreserveEmpty(&t.ThreadDeploymentFailure, src.ThreadDeploymentFailure)
	flex.SetStringPreserveEmpty(&t.ThreadStatusChange, src.ThreadStatusChange)
	flex.SetStringPreserveEmpty(&t.ThreadRestartLimitReached, src.ThreadRestartLimitReached)
	flex.SetStringPreserveEmpty(&t.ThreadBackupSuccess, src.ThreadBackupSuccess)
	flex.SetStringPreserveEmpty(&t.ThreadBackupFailure, src.ThreadBackupFailure)
	flex.SetStringPreserveEmpty(&t.ThreadScheduledTaskSuccess, src.ThreadScheduledTaskSuccess)
	flex.SetStringPreserveEmpty(&t.ThreadScheduledTaskFailure, src.ThreadScheduledTaskFailure)
	flex.SetStringPreserveEmpty(&t.ThreadDockerCleanupSuccess, src.ThreadDockerCleanupSuccess)
	flex.SetStringPreserveEmpty(&t.ThreadDockerCleanupFailure, src.ThreadDockerCleanupFailure)
	flex.SetStringPreserveEmpty(&t.ThreadServerDiskUsage, src.ThreadServerDiskUsage)
	flex.SetStringPreserveEmpty(&t.ThreadServerReachable, src.ThreadServerReachable)
	flex.SetStringPreserveEmpty(&t.ThreadServerUnreachable, src.ThreadServerUnreachable)
	flex.SetStringPreserveEmpty(&t.ThreadServerPatch, src.ThreadServerPatch)
	flex.SetStringPreserveEmpty(&t.ThreadTraefikOutdated, src.ThreadTraefikOutdated)
}
