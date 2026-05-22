package flex

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestStringPtrNonDefault(t *testing.T) {
	t.Parallel()
	v := types.StringValue("custom")
	dflt := types.StringValue("default")
	null := types.StringNull()
	unk := types.StringUnknown()
	tests := []struct {
		name string
		v    *types.String
		dflt string
		want bool
	}{
		{"nil", nil, "", false},
		{"null", &null, "", false},
		{"unknown", &unk, "", false},
		{"matches default", &dflt, "default", false},
		{"differs from default", &v, "default", true},
		{"empty vs empty", &null, "", false},
		{"value vs empty", &v, "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StringPtrNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("StringPtrNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringPtrConfigured(t *testing.T) {
	t.Parallel()
	v := types.StringValue("")
	set := types.StringValue("x")
	null := types.StringNull()
	unk := types.StringUnknown()
	tests := []struct {
		name string
		v    *types.String
		want bool
	}{
		{"nil", nil, false},
		{"null", &null, false},
		{"unknown", &unk, false},
		{"empty string", &v, true},
		{"non-empty", &set, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StringPtrConfigured(tt.v); got != tt.want {
				t.Errorf("StringPtrConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64PtrNonDefault(t *testing.T) {
	t.Parallel()
	v := types.Int64Value(10)
	dflt := types.Int64Value(60)
	null := types.Int64Null()
	tests := []struct {
		name string
		v    *types.Int64
		dflt int64
		want bool
	}{
		{"nil", nil, 0, false},
		{"null", &null, 0, false},
		{"matches default", &dflt, 60, false},
		{"differs", &v, 60, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Int64PtrNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("Int64PtrNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoolPtrNonDefault(t *testing.T) {
	t.Parallel()
	tr := types.BoolValue(true)
	fa := types.BoolValue(false)
	null := types.BoolNull()
	tests := []struct {
		name string
		v    *types.Bool
		dflt bool
		want bool
	}{
		{"nil", nil, false, false},
		{"null", &null, false, false},
		{"true vs false default", &tr, false, true},
		{"false vs false default", &fa, false, false},
		{"false vs true default", &fa, true, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BoolPtrNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("BoolPtrNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64PtrConfigured(t *testing.T) {
	t.Parallel()
	v := types.Int64Value(10)
	zero := types.Int64Value(0)
	null := types.Int64Null()
	unk := types.Int64Unknown()
	tests := []struct {
		name string
		v    *types.Int64
		want bool
	}{
		{"nil", nil, false},
		{"null", &null, false},
		{"unknown", &unk, false},
		{"zero", &zero, true},
		{"non-zero", &v, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Int64PtrConfigured(tt.v); got != tt.want {
				t.Errorf("Int64PtrConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringValueNonDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    types.String
		dflt string
		want bool
	}{
		{"null", types.StringNull(), "", false},
		{"unknown", types.StringUnknown(), "", false},
		{"matches default", types.StringValue("default"), "default", false},
		{"differs from default", types.StringValue("custom"), "default", true},
		{"empty vs non-empty default", types.StringValue(""), "default", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StringValueNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("StringValueNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestStringValueConfigured(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    types.String
		want bool
	}{
		{"null", types.StringNull(), false},
		{"unknown", types.StringUnknown(), false},
		{"empty", types.StringValue(""), true},
		{"set", types.StringValue("x"), true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := StringValueConfigured(tt.v); got != tt.want {
				t.Errorf("StringValueConfigured() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestInt64ValueNonDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    types.Int64
		dflt int64
		want bool
	}{
		{"null", types.Int64Null(), 0, false},
		{"matches", types.Int64Value(60), 60, false},
		{"differs", types.Int64Value(10), 60, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := Int64ValueNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("Int64ValueNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestBoolValueNonDefault(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		v    types.Bool
		dflt bool
		want bool
	}{
		{"null", types.BoolNull(), false, false},
		{"true vs false", types.BoolValue(true), false, true},
		{"false vs false", types.BoolValue(false), false, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := BoolValueNonDefault(tt.v, tt.dflt); got != tt.want {
				t.Errorf("BoolValueNonDefault() = %v, want %v", got, tt.want)
			}
		})
	}
}
