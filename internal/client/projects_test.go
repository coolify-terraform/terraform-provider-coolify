package client_test

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProject_IconJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"uuid": "proj-1",
		"name": "acme",
		"description": "demo",
		"icon_path": "project-icons/acme.png",
		"icon_storage_type": "local"
	}`)
	var p client.Project
	require.NoError(t, json.Unmarshal(raw, &p))
	assert.Equal(t, "project-icons/acme.png", p.IconPath)
	assert.Equal(t, "local", p.IconStorageType)

	out, err := json.Marshal(p)
	require.NoError(t, err)
	var m map[string]interface{}
	require.NoError(t, json.Unmarshal(out, &m))
	assert.Equal(t, "project-icons/acme.png", m["icon_path"])
	assert.Equal(t, "local", m["icon_storage_type"])
	_, hasFK := m["icon_s3_storage_id"]
	assert.False(t, hasFK)
}

func TestProject_IconJSONAbsent(t *testing.T) {
	t.Parallel()
	var p client.Project
	require.NoError(t, json.Unmarshal([]byte(`{"uuid":"p","name":"n"}`), &p))
	assert.Empty(t, p.IconPath)
	assert.Empty(t, p.IconStorageType)
}
