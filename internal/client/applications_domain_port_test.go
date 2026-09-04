package client_test

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestApplication_DomainPortOverridesJSON(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"uuid": "app-1",
		"name": "web",
		"domain_port_overrides": {
			"https://app.example.com": 3000,
			"https://api.example.com": 8080
		}
	}`)
	var app client.Application
	require.NoError(t, json.Unmarshal(raw, &app))
	require.NotNil(t, app.DomainPortOverrides)
	assert.Equal(t, int64(3000), (*app.DomainPortOverrides)["https://app.example.com"])
	assert.Equal(t, int64(8080), (*app.DomainPortOverrides)["https://api.example.com"])
}

func TestApplication_DomainPortOverridesJSONAbsent(t *testing.T) {
	t.Parallel()
	var app client.Application
	require.NoError(t, json.Unmarshal([]byte(`{"uuid":"app-1","name":"web"}`), &app))
	assert.Nil(t, app.DomainPortOverrides)
}

func TestApplication_DomainPortOverridesJSONNull(t *testing.T) {
	t.Parallel()
	var app client.Application
	require.NoError(t, json.Unmarshal([]byte(`{"uuid":"app-1","name":"web","domain_port_overrides":null}`), &app))
	assert.Nil(t, app.DomainPortOverrides)
}
