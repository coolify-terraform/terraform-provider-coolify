package client

import (
	"context"
	"encoding/json"
	"testing"
)

func TestUpdateApplication_OmitsGitHubAppUUIDAndGatesDockerfile(t *testing.T) {
	t.Parallel()
	gh := "cccc0001-0001-4000-8000-000000000001"
	docker := "RlJPTSBuZ2lueA=="
	input := UpdateApplicationInput{GitHubAppUUID: &gh, Dockerfile: &docker}
	for _, tc := range []struct {
		version    string
		wantDocker bool
	}{
		{version: "4.4.2", wantDocker: false},
		{version: "4.5.0", wantDocker: true},
		{version: "", wantDocker: true},
	} {
		t.Run(tc.version, func(t *testing.T) {
			var body map[string]json.RawMessage
			srv := capturingServer(t, &body)
			defer srv.Close()
			c := New(srv.URL, "token")
			c.CoolifyVersion = tc.version
			if _, err := c.UpdateApplication(context.Background(), "app-1", input); err != nil {
				t.Fatalf("UpdateApplication: %v", err)
			}
			if _, ok := body["github_app_uuid"]; ok {
				t.Fatalf("Coolify %q sent github_app_uuid", tc.version)
			}
			_, has := body["dockerfile"]
			if has != tc.wantDocker {
				t.Fatalf("Coolify %q dockerfile present=%v, want %v", tc.version, has, tc.wantDocker)
			}
		})
	}
}
