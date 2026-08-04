package client

import (
	"encoding/json"
	"testing"
)

func TestFlexibleJSONString_UnmarshalString(t *testing.T) {
	t.Parallel()
	// Coolify string column: JSON string of object map.
	in := []byte(`"{\"grafana\":{\"domain\":\"http://x\"}}"`)
	var s FlexibleJSONString
	if err := json.Unmarshal(in, &s); err != nil {
		t.Fatal(err)
	}
	if s.String() != `{"grafana":{"domain":"http://x"}}` {
		t.Fatalf("got %q", s.String())
	}
}

func TestFlexibleJSONString_UnmarshalObject(t *testing.T) {
	t.Parallel()
	in := []byte(`{"grafana":{"domain":"http://x"}}`)
	var s FlexibleJSONString
	if err := json.Unmarshal(in, &s); err != nil {
		t.Fatal(err)
	}
	if s.String() != `{"grafana":{"domain":"http://x"}}` {
		t.Fatalf("got %q", s.String())
	}
}

func TestFlexibleJSONString_UnmarshalArray(t *testing.T) {
	t.Parallel()
	in := []byte(`[{"name":"grafana","domain":"http://x"}]`)
	var s FlexibleJSONString
	if err := json.Unmarshal(in, &s); err != nil {
		t.Fatal(err)
	}
	if s.String() != `[{"name":"grafana","domain":"http://x"}]` {
		t.Fatalf("got %q", s.String())
	}
}

func TestFlexibleJSONString_UnmarshalNull(t *testing.T) {
	t.Parallel()
	var s FlexibleJSONString
	if err := json.Unmarshal([]byte(`null`), &s); err != nil {
		t.Fatal(err)
	}
	if s.String() != "" {
		t.Fatalf("got %q", s.String())
	}
}

func TestUpdateApplicationInput_DockerComposeDomainsIsArray(t *testing.T) {
	t.Parallel()
	// json.RawMessage must encode as a JSON array, not a string of JSON.
	raw := json.RawMessage(`[{"name":"web","domain":"https://app.example.com"}]`)
	b, err := json.Marshal(UpdateApplicationInput{DockerComposeDomains: raw})
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]json.RawMessage
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	field := m["docker_compose_domains"]
	if len(field) == 0 || field[0] != '[' {
		t.Fatalf("docker_compose_domains must be JSON array on wire, got %s", b)
	}
}
