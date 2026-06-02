package database

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(v string) *types.String { s := types.StringValue(v); return &s }

func int64Ptr(v int64) *types.Int64 { i := types.Int64Value(v); return &i }

func boolPtr(v bool) *types.Bool { b := types.BoolValue(v); return &b }

func TestHasExtendedFields_AllDefaults(t *testing.T) {
	t.Parallel()
	f := DatabaseExtendedPtrs{}
	if HasExtendedFields(f) {
		t.Error("expected false for zero-value struct (all nils)")
	}
}

func TestHasExtendedFields_EachField(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(*DatabaseExtendedPtrs)
	}{
		{"LimitsMemory", func(f *DatabaseExtendedPtrs) { f.LimitsMemory = strPtr("512M") }},
		{"LimitsMemorySwap", func(f *DatabaseExtendedPtrs) { f.LimitsMemorySwap = strPtr("1G") }},
		{"LimitsMemoryReservation", func(f *DatabaseExtendedPtrs) { f.LimitsMemoryReservation = strPtr("256M") }},
		{"LimitsCPUs", func(f *DatabaseExtendedPtrs) { f.LimitsCPUs = strPtr("2") }},
		{"LimitsCPUSet", func(f *DatabaseExtendedPtrs) { f.LimitsCPUSet = strPtr("0-3") }},
		{"LimitsMemorySwappiness", func(f *DatabaseExtendedPtrs) { f.LimitsMemorySwappiness = int64Ptr(10) }},
		{"LimitsCPUShares", func(f *DatabaseExtendedPtrs) { f.LimitsCPUShares = int64Ptr(512) }},
		{"PortsMappings", func(f *DatabaseExtendedPtrs) { f.PortsMappings = strPtr("5432:5432") }},
		{"CustomDockerRunOptions", func(f *DatabaseExtendedPtrs) { f.CustomDockerRunOptions = strPtr("--shm-size=1g") }},
		{"PublicPortTimeout", func(f *DatabaseExtendedPtrs) { f.PublicPortTimeout = int64Ptr(30) }},
		{"IsLogDrainEnabled", func(f *DatabaseExtendedPtrs) { f.IsLogDrainEnabled = boolPtr(true) }},
		{"IsIncludeTimestamps", func(f *DatabaseExtendedPtrs) { f.IsIncludeTimestamps = boolPtr(true) }},
		{"EnableSSL", func(f *DatabaseExtendedPtrs) { f.EnableSSL = boolPtr(true) }},
		{"SSLMode", func(f *DatabaseExtendedPtrs) { f.SSLMode = strPtr("require") }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := DatabaseExtendedPtrs{}
			tt.setup(&f)
			if !HasExtendedFields(f) {
				t.Errorf("expected true when %s is non-default", tt.name)
			}
		})
	}
}

func TestHasExtendedFields_DefaultValues(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		setup func(*DatabaseExtendedPtrs)
	}{
		{"LimitsMemory=0", func(f *DatabaseExtendedPtrs) { f.LimitsMemory = strPtr("0") }},
		{"LimitsMemorySwap=0", func(f *DatabaseExtendedPtrs) { f.LimitsMemorySwap = strPtr("0") }},
		{"LimitsMemoryReservation=0", func(f *DatabaseExtendedPtrs) { f.LimitsMemoryReservation = strPtr("0") }},
		{"LimitsCPUs=0", func(f *DatabaseExtendedPtrs) { f.LimitsCPUs = strPtr("0") }},
		{"LimitsMemorySwappiness=60", func(f *DatabaseExtendedPtrs) { f.LimitsMemorySwappiness = int64Ptr(60) }},
		{"LimitsCPUShares=1024", func(f *DatabaseExtendedPtrs) { f.LimitsCPUShares = int64Ptr(1024) }},
		{"IsLogDrainEnabled=false", func(f *DatabaseExtendedPtrs) { f.IsLogDrainEnabled = boolPtr(false) }},
		{"IsIncludeTimestamps=false", func(f *DatabaseExtendedPtrs) { f.IsIncludeTimestamps = boolPtr(false) }},
		{"EnableSSL=false", func(f *DatabaseExtendedPtrs) { f.EnableSSL = boolPtr(false) }},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			f := DatabaseExtendedPtrs{}
			tt.setup(&f)
			if HasExtendedFields(f) {
				t.Errorf("expected false when %s is set to its default", tt.name)
			}
		})
	}
}

func TestDeleteDatabase_AddsWarningWhenPollingTimesOut(t *testing.T) {
	t.Parallel()

	const uuid = "db-delete-timeout-uuid"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodDelete && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", uuid):
			w.WriteHeader(http.StatusNoContent)
		case r.Method == http.MethodGet && r.URL.Path == fmt.Sprintf("/api/v1/databases/%s", uuid):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"uuid":"` + uuid + `"}`))
		case r.Method == http.MethodGet && r.URL.Path == "/api/v1/version":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"version":"test"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := client.New(srv.URL, "test-token")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()
	resp := &resource.DeleteResponse{}

	err := DeleteDatabase(ctx, c, "coolify_database_postgresql", uuid, resp)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if resp.Diagnostics.HasError() {
		t.Fatalf("unexpected errors: %v", resp.Diagnostics.Errors())
	}
	if resp.Diagnostics.WarningsCount() != 1 {
		t.Fatalf("expected 1 warning, got %d", resp.Diagnostics.WarningsCount())
	}
	warning := resp.Diagnostics.Warnings()[0]
	if warning.Summary() != deletePollingTimeoutWarningSummary {
		t.Fatalf("warning summary = %q, want %q", warning.Summary(), deletePollingTimeoutWarningSummary)
	}
	if !strings.Contains(warning.Detail(), uuid) {
		t.Fatalf("warning detail %q does not mention uuid %s", warning.Detail(), uuid)
	}
	if !strings.Contains(warning.Detail(), "may still exist temporarily") {
		t.Fatalf("warning detail %q does not explain the temporary remote state", warning.Detail())
	}
}
