package flex

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// --- Pointer-accepting predicates (for field-pointer structs) ---

// StringPtrNonDefault returns true if v is non-nil, known (non-null,
// non-unknown), and has a value different from dflt.
func StringPtrNonDefault(v *types.String, dflt string) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueString() != dflt
}

// StringPtrConfigured returns true if v is non-nil and known (non-null,
// non-unknown). Does not check the string value itself.
func StringPtrConfigured(v *types.String) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown()
}

// Int64PtrNonDefault returns true if v is non-nil, known, and has a
// value different from dflt.
func Int64PtrNonDefault(v *types.Int64, dflt int64) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != dflt
}

// Int64PtrConfigured returns true if v is non-nil and known.
func Int64PtrConfigured(v *types.Int64) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown()
}

// BoolPtrNonDefault returns true if v is non-nil, known, and has a
// value different from dflt.
func BoolPtrNonDefault(v *types.Bool, dflt bool) bool {
	return v != nil && !v.IsNull() && !v.IsUnknown() && v.ValueBool() != dflt
}

// --- Value-accepting predicates (for model struct fields) ---

// StringValueConfigured returns true if the value is known (non-null,
// non-unknown). Use this to check whether the user explicitly set a
// field in their HCL configuration.
func StringValueConfigured(v types.String) bool {
	return !v.IsNull() && !v.IsUnknown()
}

// StringValueNonDefault returns true if the value is known and differs
// from dflt.
func StringValueNonDefault(v types.String, dflt string) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueString() != dflt
}

// Int64ValueNonDefault returns true if the value is known and differs
// from dflt.
func Int64ValueNonDefault(v types.Int64, dflt int64) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueInt64() != dflt
}

// BoolValueNonDefault returns true if the value is known and differs
// from dflt.
func BoolValueNonDefault(v types.Bool, dflt bool) bool {
	return !v.IsNull() && !v.IsUnknown() && v.ValueBool() != dflt
}
