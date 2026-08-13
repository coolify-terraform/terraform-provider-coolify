package notificationcommon_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/notificationcommon"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventJSONKey(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "deployment_failure_discord_notifications",
		notificationcommon.EventJSONKey("discord", "deployment_failure"))
	assert.Equal(t, "traefik_outdated_slack_notifications",
		notificationcommon.EventJSONKey("slack", "traefik_outdated"))
}

func TestEventStore_PutSnapshotAndApplyBody(t *testing.T) {
	t.Parallel()
	var e notificationcommon.EventStore
	e.DeploymentFailure = true
	e.ServerDiskUsage = true

	out := map[string]interface{}{"id": 1}
	e.PutSnapshot(out, "webhook")
	require.Len(t, out, 1+14)
	assert.Equal(t, true, out["deployment_failure_webhook_notifications"])
	assert.Equal(t, false, out["deployment_success_webhook_notifications"])
	assert.Equal(t, true, out["server_disk_usage_webhook_notifications"])

	body := map[string]interface{}{
		"deployment_failure_webhook_notifications": false,
		"status_change_webhook_notifications":      true,
		"ignored":                                  "x",
	}
	e.ApplyBody("webhook", body)
	assert.False(t, e.DeploymentFailure)
	assert.True(t, e.StatusChange)
	assert.True(t, e.ServerDiskUsage) // unchanged
}

func TestEventAllowedFields_AllFourteen(t *testing.T) {
	t.Parallel()
	allowed := notificationcommon.EventAllowedFields("email")
	require.Len(t, allowed, 14)
	for _, name := range notificationcommon.EventAttributeNames() {
		key := notificationcommon.EventJSONKey("email", name)
		assert.True(t, allowed[key], "missing %s", key)
	}
}

// TestEventStore_AlignsWithEventAttributeNames guards fieldPtrs attrs against
// eventNames drift (EventAllowedFields uses eventNames; PutSnapshot uses fieldPtrs).
func TestEventStore_AlignsWithEventAttributeNames(t *testing.T) {
	t.Parallel()
	names := notificationcommon.EventAttributeNames()
	require.Len(t, names, 14)

	var e notificationcommon.EventStore
	// Flip every field so the snapshot is not all-false noise.
	e.DeploymentSuccess = true
	e.DeploymentFailure = true
	e.StatusChange = true
	e.BackupSuccess = true
	e.BackupFailure = true
	e.ScheduledTaskSuccess = true
	e.ScheduledTaskFailure = true
	e.DockerCleanupSuccess = true
	e.DockerCleanupFailure = true
	e.ServerDiskUsage = true
	e.ServerReachable = true
	e.ServerUnreachable = true
	e.ServerPatch = true
	e.TraefikOutdated = true

	out := map[string]interface{}{}
	e.PutSnapshot(out, "discord")
	require.Len(t, out, len(names))
	for _, name := range names {
		key := notificationcommon.EventJSONKey("discord", name)
		v, ok := out[key]
		require.True(t, ok, "PutSnapshot missing key for attr %q", name)
		assert.Equal(t, true, v, "PutSnapshot value for %q", name)
	}

	// ApplyBody must accept every EventJSONKey from EventAttributeNames.
	body := make(map[string]interface{}, len(names))
	for _, name := range names {
		body[notificationcommon.EventJSONKey("discord", name)] = false
	}
	e.ApplyBody("discord", body)
	out2 := map[string]interface{}{}
	e.PutSnapshot(out2, "discord")
	for _, name := range names {
		key := notificationcommon.EventJSONKey("discord", name)
		assert.Equal(t, false, out2[key], "ApplyBody did not clear %q", name)
	}
}

func TestThreadJSONKey(t *testing.T) {
	t.Parallel()
	assert.Equal(t, "telegram_notifications_deployment_failure_thread_id",
		notificationcommon.ThreadJSONKey("telegram", "deployment_failure"))
	assert.Equal(t, "telegram_notifications_traefik_outdated_thread_id",
		notificationcommon.ThreadJSONKey("telegram", "traefik_outdated"))
}

func TestThreadAllowedFields_AllFourteen(t *testing.T) {
	t.Parallel()
	allowed := notificationcommon.ThreadAllowedFields("telegram")
	require.Len(t, allowed, 14)
	for _, name := range notificationcommon.EventAttributeNames() {
		key := notificationcommon.ThreadJSONKey("telegram", name)
		assert.True(t, allowed[key], "missing %s", key)
	}
}

func TestMergeAllowedMaps(t *testing.T) {
	t.Parallel()
	merged := notificationcommon.MergeAllowedMaps(
		notificationcommon.EventAllowedFields("telegram"),
		notificationcommon.ThreadAllowedFields("telegram"),
	)
	require.Len(t, merged, 28)
	assert.True(t, merged[notificationcommon.EventJSONKey("telegram", "backup_failure")])
	assert.True(t, merged[notificationcommon.ThreadJSONKey("telegram", "backup_failure")])
}

func TestMergeAllowed(t *testing.T) {
	t.Parallel()
	base := notificationcommon.EventAllowedFields("slack")
	merged := notificationcommon.MergeAllowed(base, "slack_enabled", "slack_webhook_url")
	assert.True(t, merged["slack_enabled"])
	assert.True(t, merged["slack_webhook_url"])
	assert.True(t, merged[notificationcommon.EventJSONKey("slack", "backup_failure")])
	// base not mutated
	assert.False(t, base["slack_enabled"])
}

func TestRejectUnknownFields(t *testing.T) {
	t.Parallel()
	allowed := notificationcommon.MergeAllowed(
		notificationcommon.EventAllowedFields("discord"),
		"discord_enabled",
	)
	rec := httptest.NewRecorder()
	okBody := map[string]interface{}{"discord_enabled": true}
	assert.False(t, notificationcommon.RejectUnknownFields(rec, okBody, allowed))
	assert.Equal(t, 200, rec.Code) // unchanged (no write on accept)

	rec = httptest.NewRecorder()
	badBody := map[string]interface{}{"not_a_field": true}
	assert.True(t, notificationcommon.RejectUnknownFields(rec, badBody, allowed))
	assert.Equal(t, http.StatusUnprocessableEntity, rec.Code)
}

func TestDecodeJSONBodyAndWriteJSON(t *testing.T) {
	t.Parallel()
	req := httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{"a":true}`))
	rec := httptest.NewRecorder()
	body, ok := notificationcommon.DecodeJSONBody(rec, req)
	require.True(t, ok)
	assert.Equal(t, true, body["a"])

	req = httptest.NewRequest(http.MethodPatch, "/", bytes.NewBufferString(`{`))
	rec = httptest.NewRecorder()
	_, ok = notificationcommon.DecodeJSONBody(rec, req)
	assert.False(t, ok)
	assert.Equal(t, http.StatusBadRequest, rec.Code)

	rec = httptest.NewRecorder()
	notificationcommon.WriteJSON(rec, map[string]interface{}{"ok": true})
	assert.Equal(t, "application/json", rec.Header().Get("Content-Type"))
	var got map[string]interface{}
	require.NoError(t, json.NewDecoder(rec.Body).Decode(&got))
	assert.Equal(t, true, got["ok"])
}

func TestBoolAndStringFromBody(t *testing.T) {
	t.Parallel()
	body := map[string]interface{}{"b": true, "s": "hi", "n": 1.0}
	var b bool
	var s string
	notificationcommon.BoolFromBody(body, "b", &b)
	notificationcommon.StringFromBody(body, "s", &s)
	assert.True(t, b)
	assert.Equal(t, "hi", s)
	// wrong types no-op
	notificationcommon.BoolFromBody(body, "s", &b)
	assert.True(t, b)
	notificationcommon.StringFromBody(body, "n", &s)
	assert.Equal(t, "hi", s)
}
