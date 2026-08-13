package spectest

import (
	"testing"
)

func TestCoveredEndpoints_HasS3AndDestinations(t *testing.T) {
	t.Parallel()
	cov := coveredEndpoints()
	for _, op := range []string{
		"GET /s3-storages",
		"POST /s3-storages/{uuid}/validate",
		"GET /destinations",
		"POST /servers/{server_uuid}/destinations",
		"GET /digitalocean/regions",
		"GET /vultr/plans",
	} {
		s, ok := cov[op]
		if !ok {
			t.Errorf("missing registry entry %s", op)
			continue
		}
		if s.category != "covered" {
			t.Errorf("%s: want covered, got %s (%s)", op, s.category, s.resource)
		}
	}
}

func TestCoveredEndpoints_NotificationsCoverage(t *testing.T) {
	t.Parallel()
	cov := coveredEndpoints()
	for _, op := range []string{
		"GET /notifications/discord",
		"PATCH /notifications/discord",
		"GET /notifications/slack",
		"PATCH /notifications/slack",
		"GET /notifications/webhook",
		"PATCH /notifications/webhook",
		"GET /notifications/pushover",
		"PATCH /notifications/pushover",
	} {
		s, ok := cov[op]
		if !ok {
			t.Errorf("missing registry entry %s", op)
			continue
		}
		if s.category != "covered" {
			t.Errorf("%s: want covered, got %s (%s)", op, s.category, s.resource)
		}
	}
	// Remaining channels stay planned until their resources land (#394 follow-up).
	for _, op := range []string{
		"GET /notifications/email",
		"GET /notifications/telegram",
	} {
		s, ok := cov[op]
		if !ok {
			t.Errorf("missing registry entry %s", op)
			continue
		}
		if s.category != "planned" {
			t.Errorf("%s: want planned, got %s", op, s.category)
		}
	}
}

func TestCoveredEndpoints_FalseCoveredGuards(t *testing.T) {
	t.Parallel()
	cov := coveredEndpoints()
	// These Coolify routes exist but the provider does not call them today.
	for _, op := range []string{
		"PATCH /destinations/{uuid}",
		"PATCH /projects/{uuid}/environments/{environment_name_or_uuid}",
	} {
		s, ok := cov[op]
		if !ok {
			t.Errorf("missing registry entry %s", op)
			continue
		}
		if s.category == "covered" {
			t.Errorf("%s must not be covered (no client call path)", op)
		}
	}
}
