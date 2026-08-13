package client_test

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestServerSettings_VersionProbeJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"concurrent_builds": 1,
		"dynamic_timeout": 10,
		"deployment_queue_limit": 1,
		"server_disk_usage_notification_threshold": 80,
		"server_disk_usage_check_frequency": "0 * * * *",
		"connection_timeout": 10,
		"compose_version": "2.29.1",
		"compose_version_checked_at": "2026-08-13T12:00:00.000000Z",
		"docker_version": "27.0.3",
		"docker_version_checked_at": "2026-08-13T12:00:01.000000Z"
	}`)
	var s client.ServerSettings
	require.NoError(t, json.Unmarshal(raw, &s))
	assert.Equal(t, "2.29.1", s.ComposeVersion)
	assert.Equal(t, "27.0.3", s.DockerVersion)
	assert.Contains(t, s.ComposeVersionCheckedAt, "2026-08-13")

	out, err := json.Marshal(s)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, "2.29.1", m["compose_version"])
	assert.Equal(t, "27.0.3", m["docker_version"])
}
