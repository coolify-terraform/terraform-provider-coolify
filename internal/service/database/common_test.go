package database

import (
	"testing"

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