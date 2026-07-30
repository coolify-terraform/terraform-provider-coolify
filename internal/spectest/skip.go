package spectest

import (
	"fmt"
	"sort"
	"strings"
	"testing"
)

// SkipStatus classifies why a contract field is not required on the client
// and/or Terraform schema. Public Coolify API fields must never use
// SkipInternal; use SkipDeferred with an open issue number instead.
//
// See #622 (taxonomy), #623 (allowed_fields vs write input), #621 (schema
// registry), and #624 (freshness issue checklist).
type SkipStatus string

const (
	// SkipInternal is for non-user-facing fields: polymorphic FKs, team_id,
	// computed accessors, ephemeral status, true non-API columns.
	SkipInternal SkipStatus = "internal"
	// SkipDeferred is a public product field not yet on the client/schema.
	// Issue must be a non-zero GitHub issue number tracking the gap.
	SkipDeferred SkipStatus = "deferred"
	// SkipNA is wrong surface / never ship / owned by another resource type /
	// renamed contract field superseded by another name.
	SkipNA SkipStatus = "n/a"
	// StatusCovered means the field is present (used by schema registry only;
	// not a skip).
	StatusCovered SkipStatus = "covered"
)

// FieldSkip is one intentional skip of a contract field.
type FieldSkip struct {
	Field  string
	Status SkipStatus
	Issue  int    // required when Status == SkipDeferred
	Reason string // short why (required non-empty)
}

// skipInternal marks a true non-user-facing field.
func skipInternal(field, reason string) FieldSkip {
	return FieldSkip{Field: field, Status: SkipInternal, Reason: reason}
}

// skipDeferred marks a public product field not yet implemented. issue must
// be non-zero (open GitHub issue number).
func skipDeferred(field string, issue int, reason string) FieldSkip {
	return FieldSkip{Field: field, Status: SkipDeferred, Issue: issue, Reason: reason}
}

// skipNA marks a field that will never map on this surface.
func skipNA(field, reason string) FieldSkip {
	return FieldSkip{Field: field, Status: SkipNA, Reason: reason}
}

// skipMap builds a field-name → skip table and validates taxonomy rules.
// Panics on invalid entries so misuse fails at test init, not silently.
func skipMap(skips ...FieldSkip) map[string]FieldSkip {
	out := make(map[string]FieldSkip, len(skips))
	for _, s := range skips {
		if err := validateFieldSkip(s); err != nil {
			panic(fmt.Sprintf("invalid FieldSkip for %q: %v", s.Field, err))
		}
		if _, dup := out[s.Field]; dup {
			panic(fmt.Sprintf("duplicate FieldSkip for %q", s.Field))
		}
		out[s.Field] = s
	}
	return out
}

func validateFieldSkip(s FieldSkip) error {
	if s.Field == "" {
		return fmt.Errorf("empty field name")
	}
	if strings.TrimSpace(s.Reason) == "" {
		return fmt.Errorf("reason required")
	}
	switch s.Status {
	case SkipInternal, SkipNA:
		if s.Issue != 0 {
			return fmt.Errorf("%s must not set Issue (got %d)", s.Status, s.Issue)
		}
	case SkipDeferred:
		if s.Issue <= 0 {
			return fmt.Errorf("deferred requires positive Issue number")
		}
	default:
		return fmt.Errorf("unknown status %q", s.Status)
	}
	// Banned phrasing that historically hid public API as "internal".
	low := strings.ToLower(s.Reason)
	if s.Status != SkipInternal && strings.Contains(low, "internal flag") {
		return fmt.Errorf("reason must not use %q for non-internal skip", "internal flag")
	}
	return nil
}

// isSkipped reports whether field has a skip entry.
func isSkipped(skips map[string]FieldSkip, field string) bool {
	_, ok := skips[field]
	return ok
}

// requireValidSkips fails the test if any skip violates taxonomy rules.
// Use in tests that build skip maps dynamically.
func requireValidSkips(t *testing.T, skips map[string]FieldSkip) {
	t.Helper()
	for name, s := range skips {
		if s.Field == "" {
			s.Field = name
		}
		if err := validateFieldSkip(s); err != nil {
			t.Errorf("skip %q: %v", name, err)
		}
	}
}

// deferredIssueNumbers returns sorted unique issue numbers used by deferred skips.
func deferredIssueNumbers(skips map[string]FieldSkip) []int {
	seen := map[int]struct{}{}
	var out []int
	for _, s := range skips {
		if s.Status == SkipDeferred && s.Issue > 0 {
			if _, ok := seen[s.Issue]; !ok {
				seen[s.Issue] = struct{}{}
				out = append(out, s.Issue)
			}
		}
	}
	sort.Ints(out)
	return out
}
