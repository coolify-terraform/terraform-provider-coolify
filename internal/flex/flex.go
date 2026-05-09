package flex

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringValue extracts the underlying Go string from a Terraform String value.
func StringValue(v types.String) string {
	return v.ValueString()
}

// StringValueOrNull extracts the underlying Go string as a pointer.
// Returns nil if the Terraform value is null or unknown.
func StringValueOrNull(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
}

// Int64Value extracts the underlying Go int64 from a Terraform Int64 value.
func Int64Value(v types.Int64) int64 {
	return v.ValueInt64()
}

// Int64ValueOrNull extracts the underlying Go int64 as a pointer.
func Int64ValueOrNull(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	i := v.ValueInt64()
	return &i
}

// BoolValue extracts the underlying Go bool from a Terraform Bool value.
func BoolValue(v types.Bool) bool {
	return v.ValueBool()
}

// BoolValueOrNull extracts the underlying Go bool as a pointer.
func BoolValueOrNull(v types.Bool) *bool {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	b := v.ValueBool()
	return &b
}

// StringToFramework converts a Go string to a Terraform String value.
// Empty strings become null.
func StringToFramework(s string) types.String {
	if s == "" {
		return types.StringNull()
	}
	return types.StringValue(s)
}

// StringValueToFramework converts a Go string to a framework types.String.
// Empty strings are preserved as empty (not null).
func StringValueToFramework(s string) types.String {
	return types.StringValue(s)
}

// StringPtrToFramework converts a Go string pointer to a Terraform String.
func StringPtrToFramework(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// Int64ToFramework converts a Go int64 to a Terraform Int64 value.
func Int64ToFramework(v int64) types.Int64 {
	return types.Int64Value(v)
}

// Int64PtrToFramework converts a *int64 to a framework types.Int64.
func Int64PtrToFramework(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

// BoolToFramework converts a Go bool to a Terraform Bool value.
func BoolToFramework(v bool) types.Bool {
	return types.BoolValue(v)
}

// BoolPtrToFramework converts a Go bool pointer to a Terraform Bool.
func BoolPtrToFramework(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// StringFromFramework converts a framework types.String to a Go string.
func StringFromFramework(s types.String) string {
	if s.IsNull() || s.IsUnknown() {
		return ""
	}
	return s.ValueString()
}

// BoolFromFramework converts a framework types.Bool to a Go bool.
func BoolFromFramework(b types.Bool) bool {
	if b.IsNull() || b.IsUnknown() {
		return false
	}
	return b.ValueBool()
}

// Int64PtrFromFramework converts a framework types.Int64 to a *int64.
func Int64PtrFromFramework(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	val := v.ValueInt64()
	return &val
}
