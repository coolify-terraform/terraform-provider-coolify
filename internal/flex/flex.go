package flex

import (
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// StringValueOrNull extracts the underlying Go string as a pointer.
// Returns nil if the Terraform value is null or unknown.
func StringValueOrNull(v types.String) *string {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	s := v.ValueString()
	return &s
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

// StringValueOrDefault converts a Go string to a Terraform String value.
// If the string is empty, returns the default value instead of null.
func StringValueOrDefault(s, def string) types.String {
	if s == "" {
		return types.StringValue(def)
	}
	return types.StringValue(s)
}

// StringPtrToFramework converts a Go string pointer to a Terraform String.
func StringPtrToFramework(v *string) types.String {
	if v == nil {
		return types.StringNull()
	}
	return types.StringValue(*v)
}

// Int64PtrToFramework converts a *int64 to a framework types.Int64.
func Int64PtrToFramework(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
}

// BoolPtrToFramework converts a Go bool pointer to a Terraform Bool.
func BoolPtrToFramework(v *bool) types.Bool {
	if v == nil {
		return types.BoolNull()
	}
	return types.BoolValue(*v)
}

// SetIfKnown sets dst to the string value if v is known and non-null.
func SetIfKnown(dst *string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		*dst = v.ValueString()
	}
}

// SetStrPtr sets dst to a pointer to the string value if v is known and non-null.
func SetStrPtr(dst **string, v types.String) {
	if !v.IsNull() && !v.IsUnknown() {
		s := v.ValueString()
		*dst = &s
	}
}

// SetInt64Ptr sets dst to a pointer to the int64 value if v is known and non-null.
func SetInt64Ptr(dst **int64, v types.Int64) {
	if !v.IsNull() && !v.IsUnknown() {
		i := v.ValueInt64()
		*dst = &i
	}
}

// SetBoolPtr sets dst to a pointer to the bool value if v is known and non-null.
func SetBoolPtr(dst **bool, v types.Bool) {
	if !v.IsNull() && !v.IsUnknown() {
		b := v.ValueBool()
		*dst = &b
	}
}

// Int64PtrFromFramework converts a framework types.Int64 to a *int64.
func Int64PtrFromFramework(v types.Int64) *int64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	val := v.ValueInt64()
	return &val
}

// StringPtrForUpdate returns a *string suitable for a JSON PATCH body.
// If the plan has a value, returns a pointer to it. If the plan is null
// but the prior state had a value, returns a pointer to "" (explicit clear).
// If both are null, returns nil (omit from body).
func StringPtrForUpdate(plan, state types.String) *string {
	if !plan.IsNull() && !plan.IsUnknown() {
		s := plan.ValueString()
		return &s
	}
	if !state.IsNull() {
		empty := ""
		return &empty
	}
	return nil
}

// SetStringIfConfigured sets dst to the string value only if dst was
// configured by the user (non-null, non-unknown) and v is non-empty.
// Prevents "inconsistent result after apply" errors when the API
// returns empty/default values for fields the user didn't set.
func SetStringIfConfigured(dst *types.String, v string) {
	if dst == nil || dst.IsNull() || dst.IsUnknown() {
		return
	}
	if v != "" {
		*dst = types.StringValue(v)
	}
}

// SetInt64IfConfigured sets dst to the int64 value only if dst was
// configured by the user (non-null, non-unknown) and v is non-nil.
func SetInt64IfConfigured(dst *types.Int64, v *int64) {
	if dst == nil || dst.IsNull() || dst.IsUnknown() {
		return
	}
	if v != nil {
		*dst = types.Int64Value(*v)
	}
}

// Int64PtrForUpdate returns a *int64 suitable for a JSON PATCH body.
// If the plan has a value, returns a pointer to it. If the plan is null
// but the prior state had a value, returns a pointer to 0 (explicit clear).
// If both are null, returns nil (omit from body).
func Int64PtrForUpdate(plan, state types.Int64) *int64 {
	if !plan.IsNull() && !plan.IsUnknown() {
		i := plan.ValueInt64()
		return &i
	}
	if !state.IsNull() {
		zero := int64(0)
		return &zero
	}
	return nil
}
