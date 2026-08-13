package client

import (
	"encoding/json"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDeployment_FormatLogs_Array(t *testing.T) {
	t.Parallel()
	logs, err := json.Marshal([]map[string]interface{}{
		{"output": "cloning repo", "type": "stdout", "hidden": false},
		{"output": "secret", "type": "stdout", "hidden": true},
		{"output": "nixpacks failed", "type": "stderr", "hidden": false},
	})
	require.NoError(t, err)
	d := Deployment{Logs: logs}
	got := d.FormatLogs(40)
	assert.Contains(t, got, "cloning repo")
	assert.Contains(t, got, "nixpacks failed")
	assert.NotContains(t, got, "secret")
}

func TestDeployment_FormatLogs_JSONString(t *testing.T) {
	t.Parallel()
	inner := `[{"output":"git clone failed","type":"stderr","hidden":false}]`
	quoted, err := json.Marshal(inner)
	require.NoError(t, err)
	d := Deployment{Logs: quoted}
	assert.Equal(t, "git clone failed", d.FormatLogs(10))
}

func TestDeployment_FormatLogs_Tail(t *testing.T) {
	t.Parallel()
	entries := make([]map[string]interface{}, 0, 5)
	for _, s := range []string{"a", "b", "c", "d", "e"} {
		entries = append(entries, map[string]interface{}{"output": s, "hidden": false})
	}
	logs, err := json.Marshal(entries)
	require.NoError(t, err)
	d := Deployment{Logs: logs}
	assert.Equal(t, "d\ne", d.FormatLogs(2))
}

func TestDeployment_FormatLogs_Empty(t *testing.T) {
	t.Parallel()
	assert.Empty(t, Deployment{}.FormatLogs(10))
	assert.Empty(t, Deployment{Logs: json.RawMessage("null")}.FormatLogs(10))
}
