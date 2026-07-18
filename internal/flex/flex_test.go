package flex_test

import (
	encoding_base64 "encoding/base64"
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
// SetStringSeedOrClear
// ---------------------------------------------------------------------------

func TestSetStringSeedOrClear(t *testing.T) {
	t.Parallel()
	t.Run("seeds null from non-empty API", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringSeedOrClear(&dst, "from-api")
		if dst.ValueString() != "from-api" {
			t.Fatalf("expected seeded value, got %q", dst.ValueString())
		}
	})
	t.Run("leaves null when API empty", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringSeedOrClear(&dst, "")
		if !dst.IsNull() {
			t.Fatalf("expected null, got %q", dst.ValueString())
		}
	})
	t.Run("clears configured when API empty", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringSeedOrClear(&dst, "")
		if !dst.IsNull() {
			t.Fatalf("expected null, got %q", dst.ValueString())
		}
	})
	t.Run("handles nil dst", func(t *testing.T) {
		flex.SetStringSeedOrClear(nil, "x")
	})
}

// ---------------------------------------------------------------------------
// SetStringSeedIfConfigured
// ---------------------------------------------------------------------------

func TestSetStringSeedIfConfigured(t *testing.T) {
	t.Parallel()
	t.Run("seeds null from non-default API value", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringSeedIfConfigured(&dst, "/mockup", "/")
		if dst.ValueString() != "/mockup" {
			t.Fatalf("expected /mockup, got %q", dst.ValueString())
		}
	})
	t.Run("does not seed API default into null", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringSeedIfConfigured(&dst, "/", "/")
		if !dst.IsNull() {
			t.Fatalf("expected null for default API value, got %q", dst.ValueString())
		}
	})
	t.Run("updates configured from non-empty API", func(t *testing.T) {
		dst := types.StringValue("/old")
		flex.SetStringSeedIfConfigured(&dst, "/new", "/")
		if dst.ValueString() != "/new" {
			t.Fatalf("expected /new, got %q", dst.ValueString())
		}
	})
	t.Run("preserves configured when API empty", func(t *testing.T) {
		dst := types.StringValue("/keep")
		flex.SetStringSeedIfConfigured(&dst, "", "/")
		if dst.ValueString() != "/keep" {
			t.Fatalf("expected /keep, got %q", dst.ValueString())
		}
	})
}

// ---------------------------------------------------------------------------
// SetStringPreserveEmpty
// ---------------------------------------------------------------------------

func TestSetStringPreserveEmpty(t *testing.T) {
	t.Parallel()
	t.Run("sets non-empty API value", func(t *testing.T) {
		dst := types.StringValue("old")
		flex.SetStringPreserveEmpty(&dst, "api-secret")
		if dst.ValueString() != "api-secret" {
			t.Fatalf("expected api-secret, got %q", dst.ValueString())
		}
	})
	t.Run("preserves prior when API empty", func(t *testing.T) {
		dst := types.StringValue("user-secret")
		flex.SetStringPreserveEmpty(&dst, "")
		if dst.ValueString() != "user-secret" {
			t.Fatalf("expected user-secret preserved, got %q", dst.ValueString())
		}
	})
	t.Run("null stays null when API empty", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringPreserveEmpty(&dst, "")
		if !dst.IsNull() {
			t.Fatalf("expected null, got %q", dst.ValueString())
		}
	})
	t.Run("seeds null when API non-empty", func(t *testing.T) {
		dst := types.StringNull()
		flex.SetStringPreserveEmpty(&dst, "generated")
		if dst.ValueString() != "generated" {
			t.Fatalf("expected generated, got %q", dst.ValueString())
		}
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

func TestEnsureBase64(t *testing.T) {
	t.Parallel()
	t.Run("raw YAML is encoded", func(t *testing.T) {
		raw := "version: '3'\nservices:\n  web:\n    image: nginx\n"
		got := flex.EnsureBase64(raw)
		if got == raw {
			t.Fatal("expected base64 encoding, got raw input back")
		}
		decoded, err := base64Decode(got)
		if err != nil {
			t.Fatalf("result is not valid base64: %v", err)
		}
		if decoded != raw {
			t.Fatalf("decoded = %q, want %q", decoded, raw)
		}
	})
	t.Run("already-encoded input is returned unchanged", func(t *testing.T) {
		raw := "version: '3'\nservices:\n  web:\n    image: nginx\n"
		encoded := base64Encode(raw)
		got := flex.EnsureBase64(encoded)
		if got != encoded {
			t.Fatalf("expected already-encoded input to be returned unchanged, got %q", got)
		}
	})
	t.Run("empty string returns empty", func(t *testing.T) {
		if got := flex.EnsureBase64(""); got != "" {
			t.Fatalf("expected empty, got %q", got)
		}
	})
}

func TestResolveBase64Field(t *testing.T) {
	t.Parallel()
	t.Run("user raw preserved when content matches API base64", func(t *testing.T) {
		raw := "traefik.enable=true\ntraefik.port=80"
		apiBase64 := base64Encode(raw)
		got := flex.ResolveBase64Field(types.StringValue(raw), apiBase64)
		if got.ValueString() != raw {
			t.Fatalf("expected user's raw value %q, got %q", raw, got.ValueString())
		}
	})
	t.Run("user pre-encoded preserved when matches API", func(t *testing.T) {
		raw := "traefik.enable=true"
		encoded := base64Encode(raw)
		got := flex.ResolveBase64Field(types.StringValue(encoded), encoded)
		if got.ValueString() != encoded {
			t.Fatalf("expected user's encoded value %q, got %q", encoded, got.ValueString())
		}
	})
	t.Run("external change returns decoded API value", func(t *testing.T) {
		userRaw := "traefik.enable=true"
		externalRaw := "traefik.enable=false"
		apiBase64 := base64Encode(externalRaw)
		got := flex.ResolveBase64Field(types.StringValue(userRaw), apiBase64)
		if got.ValueString() != externalRaw {
			t.Fatalf("expected decoded external value %q, got %q", externalRaw, got.ValueString())
		}
	})
	t.Run("null user value preserved", func(t *testing.T) {
		got := flex.ResolveBase64Field(types.StringNull(), base64Encode("anything"))
		if !got.IsNull() {
			t.Fatal("expected null to be preserved")
		}
	})
	t.Run("empty API value preserves user value", func(t *testing.T) {
		got := flex.ResolveBase64Field(types.StringValue("labels"), "")
		if got.ValueString() != "labels" {
			t.Fatalf("expected user value preserved, got %q", got.ValueString())
		}
	})
	t.Run("non-UTF8 API value returned as raw base64", func(t *testing.T) {
		// Simulate an API returning base64 of binary (non-UTF8) data.
		binaryBase64 := encoding_base64.StdEncoding.EncodeToString([]byte{0xff, 0xfe, 0x00, 0x01})
		got := flex.ResolveBase64Field(types.StringValue("something-else"), binaryBase64)
		// Since content differs from user value AND decoded bytes are not valid UTF-8,
		// the function returns the raw base64 string.
		if got.ValueString() != binaryBase64 {
			t.Fatalf("expected raw base64 %q, got %q", binaryBase64, got.ValueString())
		}
	})
	t.Run("unknown user value preserved", func(t *testing.T) {
		got := flex.ResolveBase64Field(types.StringUnknown(), base64Encode("anything"))
		if !got.IsUnknown() {
			t.Fatal("expected unknown to be preserved")
		}
	})
}

func TestEncodeBase64Ptr(t *testing.T) {
	t.Parallel()
	t.Run("encodes raw string", func(t *testing.T) {
		original := "server { listen 80; }"
		s := new(string)
		*s = original
		flex.EncodeBase64Ptr(&s)
		decoded, err := base64Decode(*s)
		if err != nil {
			t.Fatalf("result is not valid base64: %v", err)
		}
		if decoded != original {
			t.Fatalf("decoded = %q, want %q", decoded, original)
		}
	})
	t.Run("nil pointer is no-op", func(t *testing.T) {
		var s *string
		flex.EncodeBase64Ptr(&s)
		if s != nil {
			t.Fatal("expected nil to remain nil")
		}
	})
}

func base64Encode(s string) string {
	return encoding_base64.StdEncoding.EncodeToString([]byte(s))
}
func base64Decode(s string) (string, error) {
	b, err := encoding_base64.StdEncoding.DecodeString(s)
	return string(b), err
}
