package client_test

import (
	"encoding/json"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestService_ApplicationURLKeepsReturnedPort(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"uuid": "svc-1",
		"name": "stack",
		"type": "custom",
		"applications": [{
			"name": "web",
			"fqdn": "https://app.example.com/Path",
			"url": "https://app.example.com:8443/Path",
			"domain_port_overrides": {"https://app.example.com/Path": 8443}
		}]
	}`)
	var svc client.Service
	require.NoError(t, json.Unmarshal(raw, &svc))
	require.Len(t, svc.Applications, 1)
	assert.Equal(t, "https://app.example.com/Path", svc.Applications[0].FQDN)
	assert.Equal(t, "https://app.example.com:8443/Path", svc.Applications[0].URL)
}

func TestService_ApplicationURLOmitted(t *testing.T) {
	t.Parallel()
	raw := []byte(`{
		"uuid": "svc-1",
		"name": "stack",
		"type": "custom",
		"applications": [{
			"name": "web",
			"fqdn": "https://app.example.com:8080"
		}]
	}`)
	var svc client.Service
	require.NoError(t, json.Unmarshal(raw, &svc))
	require.Len(t, svc.Applications, 1)
	assert.Empty(t, svc.Applications[0].URL)
	assert.Equal(t, "https://app.example.com:8080", svc.Applications[0].FQDN)
}
