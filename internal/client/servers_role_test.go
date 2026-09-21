package client

import (
	"testing"
)

func TestApplyServerRoleWrite(t *testing.T) {
	t.Parallel()

	trueVal := true
	falseVal := false
	build := "build"
	deploy := "deployment"

	tests := []struct {
		name    string
		version string
		isBuild *bool
		role    *string
		wantB   *bool
		wantR   string // empty means nil
	}{
		{name: "4.3 keeps is_build_server true", version: "4.3.23", isBuild: &trueVal, wantB: &trueVal},
		{name: "4.3 keeps is_build_server false", version: "4.3.23", isBuild: &falseVal, wantB: &falseVal},
		{name: "4.4-rc.1 keeps is_build_server", version: "4.4-rc.1", isBuild: &trueVal, wantB: &trueVal},
		{name: "4.4 maps true to build", version: "4.4.0", isBuild: &trueVal, wantR: "build"},
		{name: "4.4 omits false default", version: "4.4.0", isBuild: &falseVal},
		{name: "4.4 explicit role wins", version: "4.4.0", isBuild: &trueVal, role: &deploy, wantR: "deployment"},
		{name: "4.4 role only", version: "4.4.0", role: &build, wantR: "build"},
		{name: "empty version maps like 4.4", isBuild: &trueVal, wantR: "build"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			c := &Client{CoolifyVersion: tt.version}
			gotB, gotR := applyServerRoleWrite(c, tt.isBuild, tt.role)
			if (gotB == nil) != (tt.wantB == nil) {
				t.Fatalf("is_build_server nil=%v want nil=%v", gotB == nil, tt.wantB == nil)
			}
			if gotB != nil && *gotB != *tt.wantB {
				t.Fatalf("is_build_server=%v want %v", *gotB, *tt.wantB)
			}
			if tt.wantR == "" {
				if gotR != nil {
					t.Fatalf("server_role=%q want nil", *gotR)
				}
				return
			}
			if gotR == nil || *gotR != tt.wantR {
				got := "<nil>"
				if gotR != nil {
					got = *gotR
				}
				t.Fatalf("server_role=%s want %s", got, tt.wantR)
			}
		})
	}
}
