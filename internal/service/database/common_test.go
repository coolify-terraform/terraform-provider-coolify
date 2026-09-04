package database

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework-timeouts/resource/timeouts"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func strPtr(v string) *types.String { s := types.StringValue(v); return &s }

func int64Ptr(v int64) *types.Int64 { i := types.Int64Value(v); return &i }

func boolPtr(v bool) *types.Bool { b := types.BoolValue(v); return &b }

func TestSetUpdateExtended_OmitsDisallowedKeys(t *testing.T) {
	t.Parallel()
	input := client.UpdateDatabaseInput{}
	f := DatabaseExtendedPtrs{
		LimitsMemory:            strPtr("512M"),
		LimitsMemorySwap:        strPtr("0"),
		LimitsMemoryReservation: strPtr("0"),
		LimitsCPUs:              strPtr("0"),
		LimitsCPUSet:            strPtr(""),
		LimitsMemorySwappiness:  int64Ptr(60),
		LimitsCPUShares:         int64Ptr(1024),
		PortsMappings:           strPtr("5432:5432"),
		CustomDockerRunOptions:  strPtr("--shm-size=1g"),
		PublicPortTimeout:       int64Ptr(30),
		IsLogDrainEnabled:       boolPtr(false),
		IsIncludeTimestamps:     boolPtr(false),
		HealthCheckEnabled:      boolPtr(true),
		HealthCheckInterval:     int64Ptr(15),
		HealthCheckTimeout:      int64Ptr(5),
		HealthCheckRetries:      int64Ptr(5),
		HealthCheckStartPeriod:  int64Ptr(5),
		EnableSSL:               boolPtr(false),
		SSLMode:                 strPtr("require"),
	}
	SetUpdateExtended(&input, f)
	raw, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var body map[string]interface{}
	if err := json.Unmarshal(raw, &body); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if body["limits_memory"] != "512M" {
		t.Fatalf("limits_memory = %v, want 512M", body["limits_memory"])
	}
	for _, k := range client.DatabaseUpdateDisallowedJSONKeys {
		if _, ok := body[k]; ok {
			t.Errorf("PATCH body contains disallowed key %s", k)
		}
	}
}

func TestPopulateBaseCreateInput_LimitsMemory(t *testing.T) {
	t.Parallel()
	m := CommonModel{
		ServerUUID:      types.StringValue("bbbb0001-0001-4000-8000-000000000001"),
		ProjectUUID:     types.StringValue("aaaa0001-0001-4000-8000-000000000001"),
		EnvironmentName: types.StringValue("production"),
		LimitsMemory:    types.StringValue("512M"),
	}
	var base client.CreateDatabaseBaseInput
	PopulateBaseCreateInput(&base, &m)
	if base.LimitsMemory != "512M" {
		t.Fatalf("LimitsMemory = %q, want 512M", base.LimitsMemory)
	}

	m.LimitsMemory = types.StringValue("0")
	base = client.CreateDatabaseBaseInput{}
	PopulateBaseCreateInput(&base, &m)
	if base.LimitsMemory != "" {
		t.Fatalf("default LimitsMemory = %q, want empty", base.LimitsMemory)
	}
}

func TestHasExtendedFields_AllDefaults(t *testing.T) {
	t.Parallel()
	f := DatabaseExtendedPtrs{}
	if HasExtendedFields(f) {
		t.Error("expected false for zero-value struct (all nils)")
	}
}

// TestPopulateBaseCreateInput_DestinationUUID locks create-only destination
// wiring for all database types (via PopulateBaseCreateInput). Coolify accepts
// destination_uuid on create across supported contracts (v4.1.0+).
func TestPopulateBaseCreateInput_DestinationUUID(t *testing.T) {
	t.Parallel()
	const dest = "dddd0001-0001-4000-8000-000000000001"
	m := CommonModel{
		ServerUUID:      types.StringValue("bbbb0001-0001-4000-8000-000000000001"),
		ProjectUUID:     types.StringValue("aaaa0001-0001-4000-8000-000000000001"),
		EnvironmentName: types.StringValue("production"),
		DestinationUUID: types.StringValue(dest),
	}
	var base client.CreateDatabaseBaseInput
	PopulateBaseCreateInput(&base, &m)
	if base.DestinationUUID != dest {
		t.Fatalf("DestinationUUID = %q, want %q", base.DestinationUUID, dest)
	}

	// Omitted destination must stay empty so omitempty drops it from JSON
	// (single-destination servers and older installs remain compatible).
	m.DestinationUUID = types.StringNull()
	base = client.CreateDatabaseBaseInput{}
	PopulateBaseCreateInput(&base, &m)
	if base.DestinationUUID != "" {
		t.Fatalf("DestinationUUID = %q, want empty when null in plan", base.DestinationUUID)
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
		{"PublicPortTimeout", func(f *DatabaseExtendedPtrs) { f.PublicPortTimeout = int64Ptr(30) }},
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

	err := DeleteDatabase(ctx, c, "coolify_database_postgresql", uuid, timeouts.Value{}, resp)
	if err != nil {
		t.Fatalf("unexpected delete error: %v", err)
	}
	if !resp.Diagnostics.HasError() {
		t.Fatal("expected delete error when resource is still present after polling")
	}
	if resp.Diagnostics.WarningsCount() != 0 {
		t.Fatalf("expected no warnings, got %d", resp.Diagnostics.WarningsCount())
	}
	got := resp.Diagnostics.Errors()[0]
	if got.Summary() != deletePollingErrorSummary {
		t.Fatalf("error summary = %q, want %q", got.Summary(), deletePollingErrorSummary)
	}
	if !strings.Contains(got.Detail(), uuid) {
		t.Fatalf("error detail %q does not mention uuid %s", got.Detail(), uuid)
	}
	if !strings.Contains(got.Detail(), "keep it in state") {
		t.Fatalf("error detail %q does not say Terraform will keep state", got.Detail())
	}
}
