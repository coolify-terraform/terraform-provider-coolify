package spectest

import (
	"strings"
	"testing"
)

func TestValidateFieldSkip_DeferredRequiresIssue(t *testing.T) {
	t.Parallel()
	err := validateFieldSkip(FieldSkip{
		Field:  "is_shown_once",
		Status: SkipDeferred,
		Issue:  0,
		Reason: "UI flag",
	})
	if err == nil || !strings.Contains(err.Error(), "Issue") {
		t.Fatalf("expected deferred without issue to fail, got %v", err)
	}
}

func TestValidateFieldSkip_InternalFlagBannedOnDeferred(t *testing.T) {
	t.Parallel()
	err := validateFieldSkip(FieldSkip{
		Field:  "is_runtime",
		Status: SkipDeferred,
		Issue:  626,
		Reason: "internal flag we might expose later",
	})
	if err == nil || !strings.Contains(err.Error(), "internal flag") {
		t.Fatalf("expected banned phrase error, got %v", err)
	}
}

func TestSkipMap_ValidDeferred(t *testing.T) {
	t.Parallel()
	m := skipMap(
		skipInternal("team_id", "FK"),
		skipDeferred("order", 626, "UI ordering not managed in Terraform"),
		skipNA("service_type", "mapped to type in client"),
	)
	if !isSkipped(m, "order") {
		t.Fatal("expected order skipped")
	}
	if isSkipped(m, "key") {
		t.Fatal("key should not be skipped")
	}
	issues := deferredIssueNumbers(m)
	if len(issues) != 1 || issues[0] != 626 {
		t.Fatalf("deferred issues = %v, want [626]", issues)
	}
}

func TestSkipMap_PanicsOnDuplicate(t *testing.T) {
	t.Parallel()
	defer func() {
		if recover() == nil {
			t.Fatal("expected panic on duplicate")
		}
	}()
	_ = skipMap(
		skipInternal("team_id", "FK"),
		skipInternal("team_id", "FK again"),
	)
}
