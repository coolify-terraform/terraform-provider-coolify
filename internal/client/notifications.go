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

// WebhookNotificationSettings is the current team's generic webhook notification config.
// Requires Coolify >= v4.3.0.
type WebhookNotificationSettings struct {
	ID      int    `json:"id"`
	TeamID  int    `json:"team_id"`
	Enabled bool   `json:"webhook_enabled"`
	Webhook string `json:"webhook_url,omitempty"`

	DeploymentSuccess    bool `json:"deployment_success_webhook_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_webhook_notifications"`
	StatusChange         bool `json:"status_change_webhook_notifications"`
	BackupSuccess        bool `json:"backup_success_webhook_notifications"`
	BackupFailure        bool `json:"backup_failure_webhook_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_webhook_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_webhook_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_webhook_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_webhook_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_webhook_notifications"`
	ServerReachable      bool `json:"server_reachable_webhook_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_webhook_notifications"`
	ServerPatch          bool `json:"server_patch_webhook_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_webhook_notifications"`
}

// UpdateWebhookNotificationInput is the PATCH body for webhook notifications.
type UpdateWebhookNotificationInput struct {
	Enabled *bool   `json:"webhook_enabled,omitempty"`
	Webhook *string `json:"webhook_url,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_webhook_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_webhook_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_webhook_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_webhook_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_webhook_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_webhook_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_webhook_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_webhook_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_webhook_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_webhook_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_webhook_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_webhook_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_webhook_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_webhook_notifications,omitempty"`
}

// PushoverNotificationSettings is the current team's Pushover notification config.
// Requires Coolify >= v4.3.0.
type PushoverNotificationSettings struct {
	ID       int    `json:"id"`
	TeamID   int    `json:"team_id"`
	Enabled  bool   `json:"pushover_enabled"`
	UserKey  string `json:"pushover_user_key,omitempty"`
	APIToken string `json:"pushover_api_token,omitempty"`

	DeploymentSuccess    bool `json:"deployment_success_pushover_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_pushover_notifications"`
	StatusChange         bool `json:"status_change_pushover_notifications"`
	BackupSuccess        bool `json:"backup_success_pushover_notifications"`
	BackupFailure        bool `json:"backup_failure_pushover_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_pushover_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_pushover_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_pushover_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_pushover_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_pushover_notifications"`
	ServerReachable      bool `json:"server_reachable_pushover_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_pushover_notifications"`
	ServerPatch          bool `json:"server_patch_pushover_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_pushover_notifications"`
}

// UpdatePushoverNotificationInput is the PATCH body for Pushover notifications.
type UpdatePushoverNotificationInput struct {
	Enabled  *bool   `json:"pushover_enabled,omitempty"`
	UserKey  *string `json:"pushover_user_key,omitempty"`
	APIToken *string `json:"pushover_api_token,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_pushover_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_pushover_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_pushover_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_pushover_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_pushover_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_pushover_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_pushover_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_pushover_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_pushover_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_pushover_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_pushover_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_pushover_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_pushover_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_pushover_notifications,omitempty"`
}

// GetWebhookNotifications returns the current team's webhook settings.
func (c *Client) GetWebhookNotifications(ctx context.Context) (*WebhookNotificationSettings, error) {
	var r WebhookNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/webhook", nil, &r); err != nil {
		return nil, fmt.Errorf("getting webhook notifications: %w", err)
	}
	return &r, nil
}

// UpdateWebhookNotifications updates the current team's webhook settings.
func (c *Client) UpdateWebhookNotifications(ctx context.Context, input UpdateWebhookNotificationInput) (*WebhookNotificationSettings, error) {
	var r WebhookNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/webhook", input, &r); err != nil {
		return nil, fmt.Errorf("updating webhook notifications: %w", err)
	}
	return &r, nil
}

// GetPushoverNotifications returns the current team's Pushover settings.
func (c *Client) GetPushoverNotifications(ctx context.Context) (*PushoverNotificationSettings, error) {
	var r PushoverNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/pushover", nil, &r); err != nil {
		return nil, fmt.Errorf("getting pushover notifications: %w", err)
	}
	return &r, nil
}

// UpdatePushoverNotifications updates the current team's Pushover settings.
func (c *Client) UpdatePushoverNotifications(ctx context.Context, input UpdatePushoverNotificationInput) (*PushoverNotificationSettings, error) {
	var r PushoverNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/pushover", input, &r); err != nil {
		return nil, fmt.Errorf("updating pushover notifications: %w", err)
	}
	return &r, nil
}

// EmailNotificationSettings is the current team's email notification config.
// Requires Coolify >= v4.3.0. Secrets are encrypted/hidden without read:sensitive.
type EmailNotificationSettings struct {
	ID     int `json:"id"`
	TeamID int `json:"team_id"`

	SMTPEnabled      bool   `json:"smtp_enabled"`
	SMTPFromAddress  string `json:"smtp_from_address,omitempty"`
	SMTPFromName     string `json:"smtp_from_name,omitempty"`
	SMTPRecipients   string `json:"smtp_recipients,omitempty"`
	SMTPHost         string `json:"smtp_host,omitempty"`
	SMTPPort         *int   `json:"smtp_port,omitempty"`
	SMTPEncryption   string `json:"smtp_encryption,omitempty"`
	SMTPUsername     string `json:"smtp_username,omitempty"`
	SMTPPassword     string `json:"smtp_password,omitempty"`
	SMTPTimeout      *int   `json:"smtp_timeout,omitempty"`
	ResendEnabled    bool   `json:"resend_enabled"`
	ResendAPIKey     string `json:"resend_api_key,omitempty"`
	UseInstanceEmail bool   `json:"use_instance_email_settings"`

	DeploymentSuccess    bool `json:"deployment_success_email_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_email_notifications"`
	StatusChange         bool `json:"status_change_email_notifications"`
	BackupSuccess        bool `json:"backup_success_email_notifications"`
	BackupFailure        bool `json:"backup_failure_email_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_email_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_email_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_email_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_email_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_email_notifications"`
	ServerReachable      bool `json:"server_reachable_email_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_email_notifications"`
	ServerPatch          bool `json:"server_patch_email_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_email_notifications"`
}

// UpdateEmailNotificationInput is the PATCH body for email notifications.
type UpdateEmailNotificationInput struct {
	SMTPEnabled      *bool   `json:"smtp_enabled,omitempty"`
	SMTPFromAddress  *string `json:"smtp_from_address,omitempty"`
	SMTPFromName     *string `json:"smtp_from_name,omitempty"`
	SMTPRecipients   *string `json:"smtp_recipients,omitempty"`
	SMTPHost         *string `json:"smtp_host,omitempty"`
	SMTPPort         *int    `json:"smtp_port,omitempty"`
	SMTPEncryption   *string `json:"smtp_encryption,omitempty"`
	SMTPUsername     *string `json:"smtp_username,omitempty"`
	SMTPPassword     *string `json:"smtp_password,omitempty"`
	SMTPTimeout      *int    `json:"smtp_timeout,omitempty"`
	ResendEnabled    *bool   `json:"resend_enabled,omitempty"`
	ResendAPIKey     *string `json:"resend_api_key,omitempty"`
	UseInstanceEmail *bool   `json:"use_instance_email_settings,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_email_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_email_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_email_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_email_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_email_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_email_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_email_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_email_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_email_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_email_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_email_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_email_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_email_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_email_notifications,omitempty"`
}

// TelegramNotificationSettings is the current team's Telegram notification config.
// Requires Coolify >= v4.3.0.
type TelegramNotificationSettings struct {
	ID      int    `json:"id"`
	TeamID  int    `json:"team_id"`
	Enabled bool   `json:"telegram_enabled"`
	Token   string `json:"telegram_token,omitempty"`
	ChatID  string `json:"telegram_chat_id,omitempty"`

	DeploymentSuccess    bool `json:"deployment_success_telegram_notifications"`
	DeploymentFailure    bool `json:"deployment_failure_telegram_notifications"`
	StatusChange         bool `json:"status_change_telegram_notifications"`
	BackupSuccess        bool `json:"backup_success_telegram_notifications"`
	BackupFailure        bool `json:"backup_failure_telegram_notifications"`
	ScheduledTaskSuccess bool `json:"scheduled_task_success_telegram_notifications"`
	ScheduledTaskFailure bool `json:"scheduled_task_failure_telegram_notifications"`
	DockerCleanupSuccess bool `json:"docker_cleanup_success_telegram_notifications"`
	DockerCleanupFailure bool `json:"docker_cleanup_failure_telegram_notifications"`
	ServerDiskUsage      bool `json:"server_disk_usage_telegram_notifications"`
	ServerReachable      bool `json:"server_reachable_telegram_notifications"`
	ServerUnreachable    bool `json:"server_unreachable_telegram_notifications"`
	ServerPatch          bool `json:"server_patch_telegram_notifications"`
	TraefikOutdated      bool `json:"traefik_outdated_telegram_notifications"`

	ThreadDeploymentSuccess    string `json:"telegram_notifications_deployment_success_thread_id,omitempty"`
	ThreadDeploymentFailure    string `json:"telegram_notifications_deployment_failure_thread_id,omitempty"`
	ThreadStatusChange         string `json:"telegram_notifications_status_change_thread_id,omitempty"`
	ThreadBackupSuccess        string `json:"telegram_notifications_backup_success_thread_id,omitempty"`
	ThreadBackupFailure        string `json:"telegram_notifications_backup_failure_thread_id,omitempty"`
	ThreadScheduledTaskSuccess string `json:"telegram_notifications_scheduled_task_success_thread_id,omitempty"`
	ThreadScheduledTaskFailure string `json:"telegram_notifications_scheduled_task_failure_thread_id,omitempty"`
	ThreadDockerCleanupSuccess string `json:"telegram_notifications_docker_cleanup_success_thread_id,omitempty"`
	ThreadDockerCleanupFailure string `json:"telegram_notifications_docker_cleanup_failure_thread_id,omitempty"`
	ThreadServerDiskUsage      string `json:"telegram_notifications_server_disk_usage_thread_id,omitempty"`
	ThreadServerReachable      string `json:"telegram_notifications_server_reachable_thread_id,omitempty"`
	ThreadServerUnreachable    string `json:"telegram_notifications_server_unreachable_thread_id,omitempty"`
	ThreadServerPatch          string `json:"telegram_notifications_server_patch_thread_id,omitempty"`
	ThreadTraefikOutdated      string `json:"telegram_notifications_traefik_outdated_thread_id,omitempty"`
}

// UpdateTelegramNotificationInput is the PATCH body for Telegram notifications.
type UpdateTelegramNotificationInput struct {
	Enabled *bool   `json:"telegram_enabled,omitempty"`
	Token   *string `json:"telegram_token,omitempty"`
	ChatID  *string `json:"telegram_chat_id,omitempty"`

	DeploymentSuccess    *bool `json:"deployment_success_telegram_notifications,omitempty"`
	DeploymentFailure    *bool `json:"deployment_failure_telegram_notifications,omitempty"`
	StatusChange         *bool `json:"status_change_telegram_notifications,omitempty"`
	BackupSuccess        *bool `json:"backup_success_telegram_notifications,omitempty"`
	BackupFailure        *bool `json:"backup_failure_telegram_notifications,omitempty"`
	ScheduledTaskSuccess *bool `json:"scheduled_task_success_telegram_notifications,omitempty"`
	ScheduledTaskFailure *bool `json:"scheduled_task_failure_telegram_notifications,omitempty"`
	DockerCleanupSuccess *bool `json:"docker_cleanup_success_telegram_notifications,omitempty"`
	DockerCleanupFailure *bool `json:"docker_cleanup_failure_telegram_notifications,omitempty"`
	ServerDiskUsage      *bool `json:"server_disk_usage_telegram_notifications,omitempty"`
	ServerReachable      *bool `json:"server_reachable_telegram_notifications,omitempty"`
	ServerUnreachable    *bool `json:"server_unreachable_telegram_notifications,omitempty"`
	ServerPatch          *bool `json:"server_patch_telegram_notifications,omitempty"`
	TraefikOutdated      *bool `json:"traefik_outdated_telegram_notifications,omitempty"`

	ThreadDeploymentSuccess    *string `json:"telegram_notifications_deployment_success_thread_id,omitempty"`
	ThreadDeploymentFailure    *string `json:"telegram_notifications_deployment_failure_thread_id,omitempty"`
	ThreadStatusChange         *string `json:"telegram_notifications_status_change_thread_id,omitempty"`
	ThreadBackupSuccess        *string `json:"telegram_notifications_backup_success_thread_id,omitempty"`
	ThreadBackupFailure        *string `json:"telegram_notifications_backup_failure_thread_id,omitempty"`
	ThreadScheduledTaskSuccess *string `json:"telegram_notifications_scheduled_task_success_thread_id,omitempty"`
	ThreadScheduledTaskFailure *string `json:"telegram_notifications_scheduled_task_failure_thread_id,omitempty"`
	ThreadDockerCleanupSuccess *string `json:"telegram_notifications_docker_cleanup_success_thread_id,omitempty"`
	ThreadDockerCleanupFailure *string `json:"telegram_notifications_docker_cleanup_failure_thread_id,omitempty"`
	ThreadServerDiskUsage      *string `json:"telegram_notifications_server_disk_usage_thread_id,omitempty"`
	ThreadServerReachable      *string `json:"telegram_notifications_server_reachable_thread_id,omitempty"`
	ThreadServerUnreachable    *string `json:"telegram_notifications_server_unreachable_thread_id,omitempty"`
	ThreadServerPatch          *string `json:"telegram_notifications_server_patch_thread_id,omitempty"`
	ThreadTraefikOutdated      *string `json:"telegram_notifications_traefik_outdated_thread_id,omitempty"`
}

// GetEmailNotifications returns the current team's email settings. Coolify >= v4.3.0.
func (c *Client) GetEmailNotifications(ctx context.Context) (*EmailNotificationSettings, error) {
	var r EmailNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/email", nil, &r); err != nil {
		return nil, fmt.Errorf("getting email notifications: %w", err)
	}
	return &r, nil
}

// UpdateEmailNotifications updates the current team's email settings. Coolify >= v4.3.0.
func (c *Client) UpdateEmailNotifications(ctx context.Context, input UpdateEmailNotificationInput) (*EmailNotificationSettings, error) {
	var r EmailNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/email", input, &r); err != nil {
		return nil, fmt.Errorf("updating email notifications: %w", err)
	}
	return &r, nil
}

// GetTelegramNotifications returns the current team's Telegram settings. Coolify >= v4.3.0.
func (c *Client) GetTelegramNotifications(ctx context.Context) (*TelegramNotificationSettings, error) {
	var r TelegramNotificationSettings
	if err := c.do(ctx, http.MethodGet, "/api/v1/notifications/telegram", nil, &r); err != nil {
		return nil, fmt.Errorf("getting telegram notifications: %w", err)
	}
	return &r, nil
}

// UpdateTelegramNotifications updates the current team's Telegram settings. Coolify >= v4.3.0.
func (c *Client) UpdateTelegramNotifications(ctx context.Context, input UpdateTelegramNotificationInput) (*TelegramNotificationSettings, error) {
	var r TelegramNotificationSettings
	if err := c.do(ctx, http.MethodPatch, "/api/v1/notifications/telegram", input, &r); err != nil {
		return nil, fmt.Errorf("updating telegram notifications: %w", err)
	}
	return &r, nil
}

// NotificationUpdateJSONTags returns JSON keys on Update*NotificationInput for
// the given Coolify channel name (email, discord, slack, telegram, pushover,
// webhook). Used by spectest write coverage against channelConfig rules.
func NotificationUpdateJSONTags(channel string) map[string]struct{} {
	switch channel {
	case "email":
		return jsonTagsFromValue(UpdateEmailNotificationInput{})
	case "discord":
		return jsonTagsFromValue(UpdateDiscordNotificationInput{})
	case "slack":
		return jsonTagsFromValue(UpdateSlackNotificationInput{})
	case "telegram":
		return jsonTagsFromValue(UpdateTelegramNotificationInput{})
	case "pushover":
		return jsonTagsFromValue(UpdatePushoverNotificationInput{})
	case "webhook":
		return jsonTagsFromValue(UpdateWebhookNotificationInput{})
	default:
		return nil
	}
}
