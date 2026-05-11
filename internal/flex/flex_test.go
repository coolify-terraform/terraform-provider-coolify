package flex_test

import (
	"testing"

	"github.com/SebTardif/terraform-provider-coolify/internal/flex"
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
			}
			if *got != tc.want {
				t.Fatalf("expected %q, got %q", tc.want, *got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
// Int64Value / Int64ValueOrNull
// ---------------------------------------------------------------------------

func TestInt64ValueOrNull(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		input  types.Int64
		wantNl bool
		want   int64
	}{
		{"non-null", types.Int64Value(99), false, 99},
		{"null", types.Int64Null(), true, 0},
		{"unknown", types.Int64Unknown(), true, 0},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.Int64ValueOrNull(tc.input)
			if tc.wantNl {
				if got != nil {
					t.Fatalf("expected nil, got %d", *got)
				}
				return
			}
			if got == nil {
				t.Fatal("expected non-nil pointer, got nil")
			}
			if *got != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, *got)
			}
		})
	}
}

// ---------------------------------------------------------------------------
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
// StringValueToFramework
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// StringPtrToFramework
// ---------------------------------------------------------------------------

func TestStringPtrToFramework(t *testing.T) {
	t.Parallel()
	s := "hello"
	tests := []struct {
		name     string
		input    *string
		wantNull bool
		want     string
	}{
		{"nil becomes null", nil, true, ""},
		{"non-nil becomes value", &s, false, "hello"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.StringPtrToFramework(tc.input)
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
// Int64ToFramework / Int64PtrToFramework
// ---------------------------------------------------------------------------

func TestInt64ToFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input int64
		want  int64
	}{
		{"positive", 10, 10},
		{"zero", 0, 0},
		{"negative", -3, -3},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.Int64ToFramework(tc.input)
			if got.ValueInt64() != tc.want {
				t.Fatalf("expected %d, got %d", tc.want, got.ValueInt64())
			}
		})
	}
}

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
// BoolToFramework / BoolPtrToFramework
// ---------------------------------------------------------------------------

func TestBoolToFramework(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		input bool
		want  bool
	}{
		{"true", true, true},
		{"false", false, false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.BoolToFramework(tc.input)
			if got.ValueBool() != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got.ValueBool())
			}
		})
	}
}

func TestBoolPtrToFramework(t *testing.T) {
	t.Parallel()
	b := true
	tests := []struct {
		name     string
		input    *bool
		wantNull bool
		want     bool
	}{
		{"nil becomes null", nil, true, false},
		{"non-nil becomes value", &b, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := flex.BoolPtrToFramework(tc.input)
			if tc.wantNull {
				if !got.IsNull() {
					t.Fatalf("expected null, got %v", got.ValueBool())
				}
				return
			}
			if got.IsNull() {
				t.Fatal("expected non-null, got null")
			}
			if got.ValueBool() != tc.want {
				t.Fatalf("expected %v, got %v", tc.want, got.ValueBool())
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
