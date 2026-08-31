package client

// NotificationThreads is the 15 Telegram forum thread IDs (channel-specific).
// Field names must match the Thread* fields on TelegramNotificationSettings
// and UpdateTelegramNotificationInput.
type NotificationThreads struct {
	ThreadDeploymentSuccess    string
	ThreadDeploymentFailure    string
	ThreadStatusChange         string
	ThreadRestartLimitReached  string
	ThreadBackupSuccess        string
	ThreadBackupFailure        string
	ThreadScheduledTaskSuccess string
	ThreadScheduledTaskFailure string
	ThreadDockerCleanupSuccess string
	ThreadDockerCleanupFailure string
	ThreadServerDiskUsage      string
	ThreadServerReachable      string
	ThreadServerUnreachable    string
	ThreadServerPatch          string
	ThreadTraefikOutdated      string
}

// NotificationThreadUpdate is the PATCH form of NotificationThreads (nil = omit).
type NotificationThreadUpdate struct {
	ThreadDeploymentSuccess    *string
	ThreadDeploymentFailure    *string
	ThreadStatusChange         *string
	ThreadRestartLimitReached  *string
	ThreadBackupSuccess        *string
	ThreadBackupFailure        *string
	ThreadScheduledTaskSuccess *string
	ThreadScheduledTaskFailure *string
	ThreadDockerCleanupSuccess *string
	ThreadDockerCleanupFailure *string
	ThreadServerDiskUsage      *string
	ThreadServerReachable      *string
	ThreadServerUnreachable    *string
	ThreadServerPatch          *string
	ThreadTraefikOutdated      *string
}

// ApplyThreadUpdate copies Telegram thread-id pointers onto an update input.
// dst must be a pointer to UpdateTelegramNotificationInput. Extra destination
// fields are left unchanged.
func ApplyThreadUpdate(dst any, th NotificationThreadUpdate) error {
	return copyFieldsByName(dst, th, false)
}

// ThreadsFrom copies Telegram thread IDs from TelegramNotificationSettings.
func ThreadsFrom(src any) (NotificationThreads, error) {
	var out NotificationThreads
	if err := copyFieldsByName(&out, src, true); err != nil {
		return NotificationThreads{}, err
	}
	return out, nil
}
