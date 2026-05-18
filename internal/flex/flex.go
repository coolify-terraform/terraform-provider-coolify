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
func IntValueOrNull(v types.Int64) *int {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

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

// Int64PtrToFramework converts a *int64 to a framework types.Int64.
func Int64PtrToFramework(v *int64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(*v)
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

// SetStringOrClear sets dst to the string value when v is non-empty, or
// clears it to null when v is empty (allowing drift detection for nullable
// API fields). Skips if dst was never configured (null/unknown).
func SetStringOrClear(dst *types.String, v string) {
	if dst == nil || dst.IsNull() || dst.IsUnknown() {
		return
	}
	if v != "" {
		*dst = types.StringValue(v)
	} else {
		*dst = types.StringNull()
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

// StringIfChanged returns a pointer to the plan value only if it differs
// from state. Returns nil when unchanged (field omitted via omitempty).
func StringIfChanged(plan, state types.String) *string {
	if plan.Equal(state) {
		return nil
	}
	return StringValueOrNull(plan)
}

// BoolIfChanged returns a pointer to the plan value only if it differs from state.
func BoolIfChanged(plan, state types.Bool) *bool {
	if plan.Equal(state) {
		return nil
	}
	return BoolValueOrNull(plan)
}

// Int64IfChanged returns a pointer to the plan value only if it differs from state.
func Int64IfChanged(plan, state types.Int64) *int64 {
	if plan.Equal(state) {
		return nil
	}
	return Int64PtrFromFramework(plan)
}

// IntIfNonDefault returns a pointer to the value only if it differs from the
// given default. Used to skip sending fields that already match the API's
// create-time default.
func IntIfNonDefault(v types.Int64, dflt int64) *int {
	if v.IsNull() || v.IsUnknown() || v.ValueInt64() == dflt {
		return nil
	}
	n := int(v.ValueInt64())
	return &n
}

// IntIfChanged returns a pointer to the plan value (as int) only if it differs from state.
func IntIfChanged(plan, state types.Int64) *int {
	if plan.Equal(state) {
		return nil
	}
	if plan.IsNull() || plan.IsUnknown() {
		return nil
	}
	v := int(plan.ValueInt64())
	return &v
}

// Float64PtrToInt64Framework converts a *float64 from the API to a
// Terraform Int64 value, truncating to integer. Used when the API
// contract specifies float but the schema uses Int64Attribute.
func Float64PtrToInt64Framework(v *float64) types.Int64 {
	if v == nil {
		return types.Int64Null()
	}
	return types.Int64Value(int64(*v))
}

// Float64PtrFromInt64Framework converts a Terraform Int64 to a *float64
// for sending to the API.
func Float64PtrFromInt64Framework(v types.Int64) *float64 {
	if v.IsNull() || v.IsUnknown() {
		return nil
	}
	f := float64(v.ValueInt64())
	return &f
}

// Float64IfChangedFromInt64 returns a *float64 only if the Int64 plan
// value differs from state.
func Float64IfChangedFromInt64(plan, state types.Int64) *float64 {
	if plan.Equal(state) {
		return nil
	}
	return Float64PtrFromInt64Framework(plan)
}

// NormalizeUnknownString converts an unknown String to null. Used before
// saving state so plan values that the API doesn't return are stored as
// null instead of unknown, which would cause "inconsistent result" errors.
func NormalizeUnknownString(v *types.String) {
	if v != nil && v.IsUnknown() {
		*v = types.StringNull()
	}
}

// NormalizeUnknownBool converts an unknown Bool to null.
func NormalizeUnknownBool(v *types.Bool) {
	if v != nil && v.IsUnknown() {
		*v = types.BoolNull()
	}
}

// NormalizeUnknownInt64 converts an unknown Int64 to null.
func NormalizeUnknownInt64(v *types.Int64) {
	if v != nil && v.IsUnknown() {
		*v = types.Int64Null()
	}
}
