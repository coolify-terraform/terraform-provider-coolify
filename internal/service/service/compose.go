package service

import (
	"encoding/base64"
	"strings"
	"unicode/utf8"

	"github.com/hashicorp/terraform-plugin-framework/types"
	"go.yaml.in/yaml/v3"
)

// preserveDockerComposeRaw keeps the user's string when it is the same YAML
// document Coolify stored. Coolify re-dumps YAML, and the user may have sent
// raw text or base64, so a byte compare would drift forever. A different
// document replaces state so a UI edit shows up on plan.
func preserveDockerComposeRaw(current types.String, api string) types.String {
	if api == "" {
		if current.IsUnknown() {
			return types.StringNull()
		}
		return current
	}
	if current.IsNull() || current.IsUnknown() {
		return types.StringValue(api)
	}
	if composeEquivalent(current.ValueString(), api) {
		return current
	}
	return types.StringValue(api)
}

func composeEquivalent(configured, api string) bool {
	if configured == api {
		return true
	}
	left := composeText(configured)
	right := composeText(api)
	if left == right {
		return true
	}
	canonLeft, okLeft := canonicalYAML(left)
	canonRight, okRight := canonicalYAML(right)
	if okLeft && okRight {
		return canonLeft == canonRight
	}
	return false
}

func composeText(s string) string {
	s = strings.TrimSpace(s)
	decoded, err := base64.StdEncoding.DecodeString(s)
	if err != nil || !utf8.Valid(decoded) {
		return s
	}
	text := strings.TrimSpace(string(decoded))
	if strings.Contains(text, ":") || strings.Contains(text, "\n") {
		return text
	}
	return s
}

func canonicalYAML(s string) (string, bool) {
	var v any
	if err := yaml.Unmarshal([]byte(s), &v); err != nil {
		return "", false
	}
	out, err := yaml.Marshal(v)
	if err != nil {
		return "", false
	}
	return string(out), true
}
