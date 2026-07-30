package spectest

import (
	"fmt"
	"sort"
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

// skipTableEndpointScope pairs a skip table with the contract endpoint name
// prefixes it is allowed to justify. Shared field names (e.g. is_public) must
// only be checked against the endpoints for that surface.
type skipTableEndpointScope struct {
	name     string
	skips    map[string]FieldSkip
	prefixes []string
}

// TestSkipTaxonomy_NoInternalOnAllowedFields fails if skipInternal is used for
// a field that appears on allowed_fields of a scoped endpoint (the #619 class).
func TestSkipTaxonomy_NoInternalOnAllowedFields(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	scopes := []skipTableEndpointScope{
		{"applicationFieldSkips", applicationFieldSkips, []string{"ApplicationsController::"}},
		{"appEnvWriteSkips", appEnvWriteSkips, []string{
			"ApplicationsController::create_env",
			"ApplicationsController::update_env_by_uuid",
		}},
		{"environmentVariableCoverageSkips", environmentVariableCoverageSkips, []string{
			"ApplicationsController::create_env",
			"ApplicationsController::update_env_by_uuid",
		}},
		{"githubAppCoverageSkips", githubAppCoverageSkips, []string{"GithubController::"}},
		{"scheduledTaskCoverageSkips", scheduledTaskCoverageSkips, []string{"ScheduledTasksController::"}},
		{"serverCoverageSkips", serverCoverageSkips, []string{
			"ServersController::",
			"HetznerController::",
			"DigitalOceanController::",
			"VultrController::",
		}},
		{"databaseModelSkips", databaseModelSkips, []string{"DatabasesController::"}},
		{"databaseBackupCoverageSkips", databaseBackupCoverageSkips, []string{"DatabasesController::"}},
		{"serviceCoverageSkips", serviceCoverageSkips, []string{"ServicesController::", "ServiceApplicationsController::", "ServiceDatabasesController::"}},
		{"storageCoverageSkips", storageCoverageSkips, []string{"ApplicationsController::", "ServicesController::"}},
		{"cloudTokenCoverageSkips", cloudTokenCoverageSkips, []string{"CloudTokensController::", "CloudProviderTokensController::"}},
		{"projectCoverageSkips", projectCoverageSkips, []string{"ProjectController::", "ProjectsController::"}},
		{"environmentCoverageSkips", environmentCoverageSkips, []string{"ProjectController::", "ProjectsController::"}},
		{"privateKeyCoverageSkips", privateKeyCoverageSkips, []string{"SecurityController::", "PrivateKeysController::"}},
		{"serverSettingCoverageSkips", serverSettingCoverageSkips, []string{"ServersController::"}},
	}

	var violations []string
	for _, sc := range scopes {
		for field, skip := range sc.skips {
			if skip.Status != SkipInternal {
				continue
			}
			var hits []string
			for epName, ep := range c.Endpoints {
				if !endpointMatchesPrefixes(epName, sc.prefixes) {
					continue
				}
				for _, af := range ep.AllowedFields {
					if af == field {
						hits = append(hits, epName)
						break
					}
				}
			}
			if len(hits) > 0 {
				sort.Strings(hits)
				violations = append(violations, fmt.Sprintf(
					"%s: %s marked internal but is on allowed_fields of: %s",
					sc.name, field, strings.Join(hits, ", ")))
			}
		}
	}
	sort.Strings(violations)
	if len(violations) > 0 {
		t.Errorf("public allowed_fields must not use skipInternal:\n  %s",
			strings.Join(violations, "\n  "))
	}
}

func endpointMatchesPrefixes(name string, prefixes []string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(name, p) || name == p {
			return true
		}
	}
	return false
}
