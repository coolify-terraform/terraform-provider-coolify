package notificationcommon

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// EventModel is the 15 shared event bools. Embed it in channel resource models
// so tfsdk names stay promoted (deployment_success, …).
type EventModel struct {
	DeploymentSuccess    types.Bool `tfsdk:"deployment_success"`
	DeploymentFailure    types.Bool `tfsdk:"deployment_failure"`
	StatusChange         types.Bool `tfsdk:"status_change"`
	RestartLimitReached  types.Bool `tfsdk:"restart_limit_reached"`
	BackupSuccess        types.Bool `tfsdk:"backup_success"`
	BackupFailure        types.Bool `tfsdk:"backup_failure"`
	ScheduledTaskSuccess types.Bool `tfsdk:"scheduled_task_success"`
	ScheduledTaskFailure types.Bool `tfsdk:"scheduled_task_failure"`
	DockerCleanupSuccess types.Bool `tfsdk:"docker_cleanup_success"`
	DockerCleanupFailure types.Bool `tfsdk:"docker_cleanup_failure"`
	ServerDiskUsage      types.Bool `tfsdk:"server_disk_usage"`
	ServerReachable      types.Bool `tfsdk:"server_reachable"`
	ServerUnreachable    types.Bool `tfsdk:"server_unreachable"`
	ServerPatch          types.Bool `tfsdk:"server_patch"`
	TraefikOutdated      types.Bool `tfsdk:"traefik_outdated"`
}

// CreateUpdate returns PATCH event pointers for every known plan value.
func (e EventModel) CreateUpdate() client.NotificationEventUpdate {
	return client.NotificationEventUpdate{
		DeploymentSuccess:    flex.BoolValueOrNull(e.DeploymentSuccess),
		DeploymentFailure:    flex.BoolValueOrNull(e.DeploymentFailure),
		StatusChange:         flex.BoolValueOrNull(e.StatusChange),
		RestartLimitReached:  flex.BoolValueOrNull(e.RestartLimitReached),
		BackupSuccess:        flex.BoolValueOrNull(e.BackupSuccess),
		BackupFailure:        flex.BoolValueOrNull(e.BackupFailure),
		ScheduledTaskSuccess: flex.BoolValueOrNull(e.ScheduledTaskSuccess),
		ScheduledTaskFailure: flex.BoolValueOrNull(e.ScheduledTaskFailure),
		DockerCleanupSuccess: flex.BoolValueOrNull(e.DockerCleanupSuccess),
		DockerCleanupFailure: flex.BoolValueOrNull(e.DockerCleanupFailure),
		ServerDiskUsage:      flex.BoolValueOrNull(e.ServerDiskUsage),
		ServerReachable:      flex.BoolValueOrNull(e.ServerReachable),
		ServerUnreachable:    flex.BoolValueOrNull(e.ServerUnreachable),
		ServerPatch:          flex.BoolValueOrNull(e.ServerPatch),
		TraefikOutdated:      flex.BoolValueOrNull(e.TraefikOutdated),
	}
}

// DiffUpdate returns PATCH event pointers only for values that changed.
func (e EventModel) DiffUpdate(state EventModel) client.NotificationEventUpdate {
	return client.NotificationEventUpdate{
		DeploymentSuccess:    flex.BoolIfChanged(e.DeploymentSuccess, state.DeploymentSuccess),
		DeploymentFailure:    flex.BoolIfChanged(e.DeploymentFailure, state.DeploymentFailure),
		StatusChange:         flex.BoolIfChanged(e.StatusChange, state.StatusChange),
		RestartLimitReached:  flex.BoolIfChanged(e.RestartLimitReached, state.RestartLimitReached),
		BackupSuccess:        flex.BoolIfChanged(e.BackupSuccess, state.BackupSuccess),
		BackupFailure:        flex.BoolIfChanged(e.BackupFailure, state.BackupFailure),
		ScheduledTaskSuccess: flex.BoolIfChanged(e.ScheduledTaskSuccess, state.ScheduledTaskSuccess),
		ScheduledTaskFailure: flex.BoolIfChanged(e.ScheduledTaskFailure, state.ScheduledTaskFailure),
		DockerCleanupSuccess: flex.BoolIfChanged(e.DockerCleanupSuccess, state.DockerCleanupSuccess),
		DockerCleanupFailure: flex.BoolIfChanged(e.DockerCleanupFailure, state.DockerCleanupFailure),
		ServerDiskUsage:      flex.BoolIfChanged(e.ServerDiskUsage, state.ServerDiskUsage),
		ServerReachable:      flex.BoolIfChanged(e.ServerReachable, state.ServerReachable),
		ServerUnreachable:    flex.BoolIfChanged(e.ServerUnreachable, state.ServerUnreachable),
		ServerPatch:          flex.BoolIfChanged(e.ServerPatch, state.ServerPatch),
		TraefikOutdated:      flex.BoolIfChanged(e.TraefikOutdated, state.TraefikOutdated),
	}
}

// FlattenEvents writes API event bools into the model.
func (e *EventModel) FlattenEvents(src client.NotificationEvents) {
	e.DeploymentSuccess = types.BoolValue(src.DeploymentSuccess)
	e.DeploymentFailure = types.BoolValue(src.DeploymentFailure)
	e.StatusChange = types.BoolValue(src.StatusChange)
	e.RestartLimitReached = types.BoolValue(src.RestartLimitReached)
	e.BackupSuccess = types.BoolValue(src.BackupSuccess)
	e.BackupFailure = types.BoolValue(src.BackupFailure)
	e.ScheduledTaskSuccess = types.BoolValue(src.ScheduledTaskSuccess)
	e.ScheduledTaskFailure = types.BoolValue(src.ScheduledTaskFailure)
	e.DockerCleanupSuccess = types.BoolValue(src.DockerCleanupSuccess)
	e.DockerCleanupFailure = types.BoolValue(src.DockerCleanupFailure)
	e.ServerDiskUsage = types.BoolValue(src.ServerDiskUsage)
	e.ServerReachable = types.BoolValue(src.ServerReachable)
	e.ServerUnreachable = types.BoolValue(src.ServerUnreachable)
	e.ServerPatch = types.BoolValue(src.ServerPatch)
	e.TraefikOutdated = types.BoolValue(src.TraefikOutdated)
}
