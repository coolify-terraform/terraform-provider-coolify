package client

import (
	"context"
	"fmt"
	"net/http"
)

// DiscordNotificationSettings is the current team's Discord notification config.
// Webhook URL is encrypted/hidden unless the token can read sensitive fields.
// Requires Coolify >= v4.3.0.
type DiscordNotificationSettings struct {
	ID          int    `json:"id"`
	TeamID      int    `json:"team_id"`
	Enabled     bool   `json:"discord_enabled"`
	Webhook     string `json:"discord_webhook_url,omitempty"`
	PingEnabled bool   `json:"discord_ping_enabled"`

	DeploymentSuccess    bool `json:"deployment_success_discord_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_discord_notifications"`
	StatusChange         bool `json:"status_change_discord_notifications"`
	BackupSuccess        bool `json:"backup_success_discord_notifications"`
	BackupFailure        bool `json:"backup_failure_discord_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_discord_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_discord_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_discord_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_discord_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_discord_notifications"`
	ServerReachable      bool `json:"server_reachable_discord_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_discord_notifications"`
	ServerPatch          bool `json:"server_patch_discord_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_discord_notifications"`
}

// UpdateDiscordNotificationInput is the PATCH body for Discord notifications.
// Only non-nil fields are sent.
type UpdateDiscordNotificationInput struct {
	Enabled     *bool   `json:"discord_enabled,omitempty"`
	Webhook     *string `json:"discord_webhook_url,omitempty"`
	PingEnabled *bool   `json:"discord_ping_enabled,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_discord_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_discord_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_discord_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_discord_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_discord_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_discord_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_discord_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_discord_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_discord_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_discord_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_discord_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_discord_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_discord_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_discord_notifications,omitempty"`
}

// SlackNotificationSettings is the current team's Slack notification config.
// Requires Coolify >= v4.3.0.
type SlackNotificationSettings struct {
	ID      int    `json:"id"`
	TeamID  int    `json:"team_id"`
	Enabled bool   `json:"slack_enabled"`
	Webhook string `json:"slack_webhook_url,omitempty"`

	DeploymentSuccess    bool `json:"deployment_success_slack_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_slack_notifications"`
	StatusChange         bool `json:"status_change_slack_notifications"`
	BackupSuccess        bool `json:"backup_success_slack_notifications"`
	BackupFailure        bool `json:"backup_failure_slack_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_slack_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_slack_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_slack_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_slack_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_slack_notifications"`
	ServerReachable      bool `json:"server_reachable_slack_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_slack_notifications"`
	ServerPatch          bool `json:"server_patch_slack_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_slack_notifications"`
}

// UpdateSlackNotificationInput is the PATCH body for Slack notifications.
type UpdateSlackNotificationInput struct {
	Enabled *bool   `json:"slack_enabled,omitempty"`
	Webhook *string `json:"slack_webhook_url,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_slack_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_slack_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_slack_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_slack_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_slack_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_slack_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_slack_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_slack_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_slack_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_slack_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_slack_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_slack_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_slack_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_slack_notifications,omitempty"`
}

// GetDiscordNotifications returns the current team's Discord settings.
// Requires Coolify >= v4.3.0.
func (c *Client) GetDiscordNotifications(ctx context.Context) (*DiscordNotificationSettings, error) {
	var r DiscordNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/discord", nil, &r); err != nil {
		return nil, fmt.Errorf("getting discord notifications: %w", err)
	}
	return &r, nil
}

// UpdateDiscordNotifications updates the current team's Discord settings.
// Requires Coolify >= v4.3.0.
func (c *Client) UpdateDiscordNotifications(ctx context.Context, input UpdateDiscordNotificationInput) (*DiscordNotificationSettings, error) {
	var r DiscordNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/discord", input, &r); err != nil {
		return nil, fmt.Errorf("updating discord notifications: %w", err)
	}
	return &r, nil
}

// GetSlackNotifications returns the current team's Slack settings.
// Requires Coolify >= v4.3.0.
func (c *Client) GetSlackNotifications(ctx context.Context) (*SlackNotificationSettings, error) {
	var r SlackNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/slack", nil, &r); err != nil {
		return nil, fmt.Errorf("getting slack notifications: %w", err)
	}
	return &r, nil
}

// UpdateSlackNotifications updates the current team's Slack settings.
// Requires Coolify >= v4.3.0.
func (c *Client) UpdateSlackNotifications(ctx context.Context, input UpdateSlackNotificationInput) (*SlackNotificationSettings, error) {
	var r SlackNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/slack", input, &r); err != nil {
		return nil, fmt.Errorf("updating slack notifications: %w", err)
	}
	return &r, nil
}
