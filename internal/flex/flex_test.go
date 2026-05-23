package flex_test

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/flex"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// ---------------------------------------------------------------------------
// StringValueOrNull
// ---------------------------------------------------------------------------

func TestStringValueOrNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  types.String
		wantNl bool
		want   string
	}{
		{"non-null value", types.StringValue("abc"), false, "abc"},
		{"null value", types.StringNull(), true, ""},
		{"unknown value", types.StringUnknown(), true, ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.StringValueOrNull(tc.input)
			if tc.wantNl {
				if got != nil {
					t.Fatalf("expected nil, got %q", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil pointer, got nil")
				return
			}
			if *got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, *got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// IntValueOrNull
// ---------------------------------------------------------------------------

func TestIntValueOrNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  types.Int64
		wantNl bool
		want   int
	}{
		{"positive value", types.Int64Value(42), false, 42},
		{"zero value", types.Int64Value(0), false, 0},
		{"negative value", types.Int64Value(-1), false, -1},
		{"null", types.Int64Null(), true, 0},
		{"unknown", types.Int64Unknown(), true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.IntValueOrNull(tc.input)
			if tc.wantNl {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil pointer, got nil")
				return
			}
			if *got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, *got)
			}
		})
	}
}

// BoolValue / BoolValueOrNull
// ---------------------------------------------------------------------------

func TestBoolValueOrNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  types.Bool
		wantNl bool
		want   bool
	}{
		{"true value", types.BoolValue(true), false, true},
		{"false value", types.BoolValue(false), false, false},
		{"null", types.BoolNull(), true, false},
		{"unknown", types.BoolUnknown(), true, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.BoolValueOrNull(tc.input)
			if tc.wantNl {
				if got != nil {
					t.Fatalf("expected nil, got %v", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil pointer, got nil")
				return
			}
			if *got != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, *got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// StringToFramework
// ---------------------------------------------------------------------------

func TestStringToFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		input    string
		wantNull bool
		want     string
	}{
		{"empty becomes null", "", true, ""},
		{"non-empty becomes value", "foo", false, "foo"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.StringToFramework(tc.input)
			if tc.wantNull {
				if !got.IsNull() {
					t.Fatalf("expected null, got %q", got.ValueString())
				}
				return
			}
			if got.IsNull() {
				t.Fatal("expected non-null, got null")
			}
			if got.ValueString() != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got.ValueString())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Int64PtrToFramework
// ---------------------------------------------------------------------------

func TestInt64PtrToFramework(t *testing.T) {
	t.Parallel()
	v := int64(42)
	tests := []struct {
		name     string
		input    *int64
		wantNull bool
		want     int64
	}{
		{"nil becomes null", nil, true, 0},
		{"non-nil becomes value", &v, false, 42},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.Int64PtrToFramework(tc.input)
			if tc.wantNull {
				if !got.IsNull() {
					t.Fatalf("expected null, got %d", got.ValueInt64())
				}
				return
			}
			if got.IsNull() {
				t.Fatal("expected non-null, got null")
			}
			if got.ValueInt64() != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got.ValueInt64())
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Int64PtrFromFramework
// ---------------------------------------------------------------------------

func TestInt64PtrFromFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  types.Int64
		wantNl bool
		want   int64
	}{
		{"non-null", types.Int64Value(55), false, 55},
		{"null", types.Int64Null(), true, 0},
		{"unknown", types.Int64Unknown(), true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.Int64PtrFromFramework(tc.input)
			if tc.wantNl {
				if got != nil {
					t.Fatalf("expected nil, got %d", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil pointer, got nil")
				return
			}
			if *got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, *got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Setter helpers (SetIfKnown, SetStrPtr, SetInt64Ptr, SetBoolPtr)
// ---------------------------------------------------------------------------

func TestSetIfKnown(t *testing.T) {
	t.Parallel()
	t.Run("known value sets destination", func(t *testing.T) {
		var dst string
		flex.SetIfKnown(&dst, types.StringValue("hello"))
		if dst != "hello" {
			t.Fatalf("expected 'hello', got %q", dst)
		}
	})
	t.Run("null leaves unchanged", func(t *testing.T) {
		dst := "original"
		flex.SetIfKnown(&dst, types.StringNull())
		if dst != "original" {
			t.Fatalf("expected 'original', got %q", dst)
		}
	})
	t.Run("unknown leaves unchanged", func(t *testing.T) {
		dst := "original"
		flex.SetIfKnown(&dst, types.StringUnknown())
		if dst != "original" {
			t.Fatalf("expected 'original', got %q", dst)
		}
	})
}

func TestSetStrPtr(t *testing.T) {
	t.Parallel()
	t.Run("known value sets pointer", func(t *testing.T) {
		var dst *string
		flex.SetStrPtr(&dst, types.StringValue("hello"))
		if dst == nil || *dst != "hello" {
			t.Fatal("expected pointer to 'hello'")
		}
	})
	t.Run("null leaves nil", func(t *testing.T) {
		var dst *string
		flex.SetStrPtr(&dst, types.StringNull())
		if dst != nil {
			t.Fatal("expected nil")
		}
	})
}

func TestSetInt64Ptr(t *testing.T) {
	t.Parallel()
	t.Run("known value sets pointer", func(t *testing.T) {
		var dst *int64
		flex.SetInt64Ptr(&dst, types.Int64Value(42))
		if dst == nil || *dst != 42 {
			t.Fatal("expected pointer to 42")
		}
	})
	t.Run("null leaves nil", func(t *testing.T) {
		var dst *int64
		flex.SetInt64Ptr(&dst, types.Int64Null())
		if dst != nil {
			t.Fatal("expected nil")
		}
	})
}

func TestSetBoolPtr(t *testing.T) {
	t.Parallel()
	t.Run("known value sets pointer", func(t *testing.T) {
		var dst *bool
		flex.SetBoolPtr(&dst, types.BoolValue(true))
		if dst == nil || !*dst {
			t.Fatal("expected pointer to true")
		}
	})
	t.Run("null leaves nil", func(t *testing.T) {
		var dst *bool
		flex.SetBoolPtr(&dst, types.BoolNull())
		if dst != nil {
			t.Fatal("expected nil")
		}
	})
}

func TestNormalizeUnknownString(t *testing.T) {
	t.Parallel()
	t.Run("nil pointer is safe", func(t *testing.T) {
		flex.NormalizeUnknownString(nil)
	})
	t.Run("unknown becomes null", func(t *testing.T) {
		v := types.StringUnknown()
		flex.NormalizeUnknownString(&v)
		if !v.IsNull() {
			t.Fatal("expected null")
		}
	})
	t.Run("null stays null", func(t *testing.T) {
		v := types.StringNull()
		flex.NormalizeUnknownString(&v)
		if !v.IsNull() {
			t.Fatal("expected null")
		}
	})
	t.Run("known value unchanged", func(t *testing.T) {
		v := types.StringValue("keep")
		flex.NormalizeUnknownString(&v)
		if v.ValueString() != "keep" {
			t.Fatal("value changed")
		}
	})
}

func TestNormalizeUnknownBool(t *testing.T) {
	t.Parallel()
	t.Run("nil pointer is safe", func(t *testing.T) {
		flex.NormalizeUnknownBool(nil)
	})
	t.Run("unknown becomes null", func(t *testing.T) {
		v := types.BoolUnknown()
		flex.NormalizeUnknownBool(&v)
		if !v.IsNull() {
			t.Fatal("expected null")
		}
	})
	t.Run("known value unchanged", func(t *testing.T) {
		v := types.BoolValue(true)
		flex.NormalizeUnknownBool(&v)
		if !v.ValueBool() {
			t.Fatal("value changed")
		}
	})
}

func TestNormalizeUnknownInt64(t *testing.T) {
	t.Parallel()
	t.Run("nil pointer is safe", func(t *testing.T) {
		flex.NormalizeUnknownInt64(nil)
	})
	t.Run("unknown becomes null", func(t *testing.T) {
		v := types.Int64Unknown()
		flex.NormalizeUnknownInt64(&v)
		if !v.IsNull() {
			t.Fatal("expected null")
		}
	})
	t.Run("known value unchanged", func(t *testing.T) {
		v := types.Int64Value(42)
		flex.NormalizeUnknownInt64(&v)
		if v.ValueInt64() != 42 {
			t.Fatal("value changed")
		}
	})
}

// ---------------------------------------------------------------------------
// StringIfChanged / BoolIfChanged / Int64IfChanged
// ---------------------------------------------------------------------------

func TestStringIfChanged(t *testing.T) {
	t.Parallel()

	t.Run("different values returns plan", func(t *testing.T) {
		result := flex.StringIfChanged(types.StringValue("new"), types.StringValue("old"))
		if result == nil || *result != "new" {
			t.Fatalf("expected 'new', got %v", result)
		}
	})

	t.Run("same values returns nil", func(t *testing.T) {
		result := flex.StringIfChanged(types.StringValue("same"), types.StringValue("same"))
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})

	t.Run("both null returns nil", func(t *testing.T) {
		result := flex.StringIfChanged(types.StringNull(), types.StringNull())
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})

	t.Run("plan null state has value returns nil for plan", func(t *testing.T) {
		result := flex.StringIfChanged(types.StringNull(), types.StringValue("old"))
		if result != nil {
			t.Fatalf("expected nil (plan is null), got %v", *result)
		}
	})

	t.Run("plan has value state null returns plan", func(t *testing.T) {
		result := flex.StringIfChanged(types.StringValue("new"), types.StringNull())
		if result == nil || *result != "new" {
			t.Fatalf("expected 'new', got %v", result)
		}
	})
}

func TestBoolIfChanged(t *testing.T) {
	t.Parallel()

	t.Run("different values returns plan", func(t *testing.T) {
		result := flex.BoolIfChanged(types.BoolValue(true), types.BoolValue(false))
		if result == nil || !*result {
			t.Fatalf("expected true, got %v", result)
		}
	})

	t.Run("same values returns nil", func(t *testing.T) {
		result := flex.BoolIfChanged(types.BoolValue(true), types.BoolValue(true))
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})

	t.Run("both null returns nil", func(t *testing.T) {
		result := flex.BoolIfChanged(types.BoolNull(), types.BoolNull())
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})
}

func TestInt64IfChanged(t *testing.T) {
	t.Parallel()

	t.Run("different values returns plan", func(t *testing.T) {
		result := flex.Int64IfChanged(types.Int64Value(42), types.Int64Value(10))
		if result == nil || *result != 42 {
			t.Fatalf("expected 42, got %v", result)
		}
	})

	t.Run("same values returns nil", func(t *testing.T) {
		result := flex.Int64IfChanged(types.Int64Value(42), types.Int64Value(42))
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})

	t.Run("both null returns nil", func(t *testing.T) {
		result := flex.Int64IfChanged(types.Int64Null(), types.Int64Null())
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})
}

// ---------------------------------------------------------------------------
// StringPtrForUpdate / Int64PtrForUpdate
// ---------------------------------------------------------------------------

func TestStringPtrForUpdate(t *testing.T) {
	t.Parallel()

	t.Run("plan has value", func(t *testing.T) {
		result := flex.StringPtrForUpdate(types.StringValue("new"), types.StringValue("old"))
		if result == nil || *result != "new" {
			t.Fatalf("expected 'new', got %v", result)
		}
	})

	t.Run("plan null state had value clears", func(t *testing.T) {
		result := flex.StringPtrForUpdate(types.StringNull(), types.StringValue("old"))
		if result == nil || *result != "" {
			t.Fatalf("expected empty string (clear), got %v", result)
		}
	})

	t.Run("both null returns nil", func(t *testing.T) {
		result := flex.StringPtrForUpdate(types.StringNull(), types.StringNull())
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})

	t.Run("plan unknown state had value clears", func(t *testing.T) {
		result := flex.StringPtrForUpdate(types.StringUnknown(), types.StringValue("old"))
		if result == nil || *result != "" {
			t.Fatalf("expected empty string (clear), got %v", result)
		}
	})

	t.Run("plan unknown state null returns nil", func(t *testing.T) {
		result := flex.StringPtrForUpdate(types.StringUnknown(), types.StringNull())
		if result != nil {
			t.Fatalf("expected nil, got %v", *result)
		}
	})
}

// ---------------------------------------------------------------------------
// StringValueOrDefault
// ---------------------------------------------------------------------------

func TestStringValueOrDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		s    string
		def  string
		want string
	}{
		{"non-empty returns value", "hello", "fallback", "hello"},
		{"empty returns default", "", "fallback", "fallback"},
		{"empty with empty default", "", "", ""},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.StringValueOrDefault(tc.s, tc.def)
			if got.ValueString() != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, got.ValueString())
			}
			if got.IsNull() {
				t.Fatal("expected non-null, got null")
			}
		})
	}
}

// ---------------------------------------------------------------------------
// SetStringIfConfigured
// ---------------------------------------------------------------------------

func TestSetStringIfConfigured(t *testing.T) {
	t.Parallel()
	t.Run("sets value when configured and non-empty", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringIfConfigured(&dst, "new")
		if dst.ValueString() != "new" {
			t.Fatalf("expected 'new', got %q", dst.ValueString())
		}
	})
	t.Run("skips empty value when configured", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringIfConfigured(&dst, "")
		if dst.ValueString() != "old" {
			t.Fatalf("expected 'old' unchanged, got %q", dst.ValueString())
		}
	})
	t.Run("skips when null", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringIfConfigured(&dst, "new")
		if !dst.IsNull() {
			t.Fatalf("expected null, got %q", dst.ValueString())
		}
	})
	t.Run("skips when unknown", func(t *testing.T) {
		dst := types.StringUnknown()
		flex.SetStringIfConfigured(&dst, "new")
		if !dst.IsUnknown() {
			t.Fatal("expected unknown, got non-unknown")
		}
	})
	t.Run("handles nil dst", func(t *testing.T) {
		flex.SetStringIfConfigured(nil, "new") // should not panic
	})
}

// ---------------------------------------------------------------------------
// SetStringOrClear
// ---------------------------------------------------------------------------

func TestSetStringOrClear(t *testing.T) {
	t.Parallel()
	t.Run("sets non-empty value", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringOrClear(&dst, "new")
		if dst.ValueString() != "new" {
			t.Fatalf("expected 'new', got %q", dst.ValueString())
		}
	})
	t.Run("clears to null on empty", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringOrClear(&dst, "")
		if !dst.IsNull() {
			t.Fatalf("expected null, got %q", dst.ValueString())
		}
	})
	t.Run("skips when null", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringOrClear(&dst, "new")
		if !dst.IsNull() {
			t.Fatalf("expected null unchanged, got %q", dst.ValueString())
		}
	})
	t.Run("skips when unknown", func(t *testing.T) {
		dst := types.StringUnknown()
		flex.SetStringOrClear(&dst, "new")
		if !dst.IsUnknown() {
			t.Fatal("expected unknown unchanged")
		}
	})
	t.Run("handles nil dst", func(t *testing.T) {
		flex.SetStringOrClear(nil, "new") // should not panic
	})
}

// ---------------------------------------------------------------------------
// SetInt64IfConfigured
// ---------------------------------------------------------------------------

func TestSetInt64IfConfigured(t *testing.T) {
	t.Parallel()
	t.Run("sets value when configured and non-nil", func(t *testing.T) {
		dst := types.Int64Value(10)
		v := int64(42)
		flex.SetInt64IfConfigured(&dst, &v)
		if dst.ValueInt64() != 42 {
			t.Fatalf("expected 42, got %d", dst.ValueInt64())
		}
	})
	t.Run("skips nil value when configured", func(t *testing.T) {
		dst := types.Int64Value(10)
		flex.SetInt64IfConfigured(&dst, nil)
		if dst.ValueInt64() != 10 {
			t.Fatalf("expected 10 unchanged, got %d", dst.ValueInt64())
		}
	})
	t.Run("skips when null", func(t *testing.T) {
		dst := types.Int64Null()
		v := int64(42)
		flex.SetInt64IfConfigured(&dst, &v)
		if !dst.IsNull() {
			t.Fatalf("expected null, got %d", dst.ValueInt64())
		}
	})
	t.Run("skips when unknown", func(t *testing.T) {
		dst := types.Int64Unknown()
		v := int64(42)
		flex.SetInt64IfConfigured(&dst, &v)
		if !dst.IsUnknown() {
			t.Fatal("expected unknown, got non-unknown")
		}
	})
	t.Run("handles nil dst", func(t *testing.T) {
		v := int64(42)
		flex.SetInt64IfConfigured(nil, &v) // should not panic
	})
}

// ---------------------------------------------------------------------------
// IntIfChanged
// ---------------------------------------------------------------------------

func TestIntIfChanged(t *testing.T) {
	t.Parallel()
	t.Run("different values returns plan", func(t *testing.T) {
		got := flex.IntIfChanged(types.Int64Value(42), types.Int64Value(10))
		if got == nil || *got != 42 {
			t.Fatalf("expected 42, got %v", got)
		}
	})
	t.Run("same values returns nil", func(t *testing.T) {
		got := flex.IntIfChanged(types.Int64Value(42), types.Int64Value(42))
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("both null returns nil", func(t *testing.T) {
		got := flex.IntIfChanged(types.Int64Null(), types.Int64Null())
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("plan null returns nil", func(t *testing.T) {
		got := flex.IntIfChanged(types.Int64Null(), types.Int64Value(10))
		if got != nil {
			t.Fatalf("expected nil (plan is null), got %v", *got)
		}
	})
	t.Run("plan unknown returns nil", func(t *testing.T) {
		got := flex.IntIfChanged(types.Int64Unknown(), types.Int64Value(10))
		if got != nil {
			t.Fatalf("expected nil (plan is unknown), got %v", *got)
		}
	})
}

// ---------------------------------------------------------------------------
// Float64PtrToInt64Framework
// ---------------------------------------------------------------------------

func TestFloat64PtrToInt64Framework(t *testing.T) {
	t.Parallel()
	t.Run("nil returns null", func(t *testing.T) {
		got := flex.Float64PtrToInt64Framework(nil)
		if !got.IsNull() {
			t.Fatalf("expected null, got %d", got.ValueInt64())
		}
	})
	t.Run("non-nil returns truncated value", func(t *testing.T) {
		f := 42.9
		got := flex.Float64PtrToInt64Framework(&f)
		if got.IsNull() {
			t.Fatal("expected non-null")
		}
		if got.ValueInt64() != 42 {
			t.Fatalf("expected 42 (truncated), got %d", got.ValueInt64())
		}
	})
	t.Run("exact integer", func(t *testing.T) {
		f := 100.0
		got := flex.Float64PtrToInt64Framework(&f)
		if got.ValueInt64() != 100 {
			t.Fatalf("expected 100, got %d", got.ValueInt64())
		}
	})
}

// ---------------------------------------------------------------------------
// Float64PtrFromInt64Framework
// ---------------------------------------------------------------------------

func TestFloat64PtrFromInt64Framework(t *testing.T) {
	t.Parallel()
	t.Run("null returns nil", func(t *testing.T) {
		got := flex.Float64PtrFromInt64Framework(types.Int64Null())
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("unknown returns nil", func(t *testing.T) {
		got := flex.Float64PtrFromInt64Framework(types.Int64Unknown())
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("value returns float64 pointer", func(t *testing.T) {
		got := flex.Float64PtrFromInt64Framework(types.Int64Value(42))
		if got == nil {
			t.Fatal("expected non-nil")
			return
		}
		if *got != 42.0 {
			t.Fatalf("expected 42.0, got %v", *got)
		}
	})
}

// ---------------------------------------------------------------------------
// Float64IfChangedFromInt64
// ---------------------------------------------------------------------------

func TestFloat64IfChangedFromInt64(t *testing.T) {
	t.Parallel()
	t.Run("different values returns plan as float64", func(t *testing.T) {
		got := flex.Float64IfChangedFromInt64(types.Int64Value(42), types.Int64Value(10))
		if got == nil || *got != 42.0 {
			t.Fatalf("expected 42.0, got %v", got)
		}
	})
	t.Run("same values returns nil", func(t *testing.T) {
		got := flex.Float64IfChangedFromInt64(types.Int64Value(42), types.Int64Value(42))
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("both null returns nil", func(t *testing.T) {
		got := flex.Float64IfChangedFromInt64(types.Int64Null(), types.Int64Null())
		if got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
}

// ---------------------------------------------------------------------------
// IntIfNonDefault
// ---------------------------------------------------------------------------

func TestIntIfNonDefault(t *testing.T) {
	t.Run("null returns nil", func(t *testing.T) {
		if got := flex.IntIfNonDefault(types.Int64Null(), 2); got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("unknown returns nil", func(t *testing.T) {
		if got := flex.IntIfNonDefault(types.Int64Unknown(), 2); got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("matches default returns nil", func(t *testing.T) {
		if got := flex.IntIfNonDefault(types.Int64Value(2), 2); got != nil {
			t.Fatalf("expected nil, got %v", *got)
		}
	})
	t.Run("differs from default returns pointer", func(t *testing.T) {
		got := flex.IntIfNonDefault(types.Int64Value(8), 2)
		if got == nil {
			t.Fatal("expected non-nil")
			return
		}
		if *got != 8 {
			t.Fatalf("expected 8, got %d", *got)
		}
	})
}
