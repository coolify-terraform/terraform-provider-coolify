package spectest

import (
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// TestWriteCoverage_ApplicationEnv ensures application env create/update
// allowed_fields appear on the client write payload type (#623 Phase A).
func TestWriteCoverage_ApplicationEnv(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	endpoints := []string{
		"ApplicationsController::create_env",
		"ApplicationsController::update_env_by_uuid",
	}
	tags := client.EnvVarWriteJSONTags("applications")
	if len(tags) == 0 {
		t.Fatal("EnvVarWriteJSONTags(applications) returned empty set")
	}
	requireValidSkips(t, appEnvWriteSkips)

	for _, name := range endpoints {
		ep, ok := c.Endpoints[name]
		if !ok {
			t.Fatalf("endpoint %s not found in contract", name)
		}
		if len(ep.AllowedFields) == 0 {
			t.Fatalf("endpoint %s has empty allowed_fields", name)
		}
		var missing []string
		for _, field := range ep.AllowedFields {
			if isSkipped(appEnvWriteSkips, field) {
				continue
			}
			if _, ok := tags[field]; !ok {
				missing = append(missing, field)
			}
		}
		sort.Strings(missing)
		if len(missing) > 0 {
			t.Errorf("%s allowed_fields missing from application env write input:\n  %s",
				name, strings.Join(missing, "\n  "))
		}
	}
}

// TestWriteCoverage_ServiceEnvSanity ensures service write payloads never
// claim is_runtime / is_buildtime (parent-scoped allow lists).
func TestWriteCoverage_ServiceEnvSanity(t *testing.T) {
	t.Parallel()
	tags := client.EnvVarWriteJSONTags("services")
	for _, forbidden := range []string{"is_runtime", "is_buildtime"} {
		if _, ok := tags[forbidden]; ok {
			t.Errorf("service env write input must not include %s", forbidden)
		}
	}
	for _, required := range []string{"key", "value", "is_preview", "is_literal", "is_multiline", "comment"} {
		if _, ok := tags[required]; !ok {
			t.Errorf("service env write input missing %s", required)
		}
	}
}

// applicationCreateSilentDefaultFields are create allow-list fields that Coolify
// applies with a non-empty/true generative default when omitted. Missing these
// from Create*Input + schema is a product footgun (see #642 autogenerate_domain).
// Post-create PATCH fields are intentionally excluded (different pattern).
var applicationCreateSilentDefaultFields = []string{
	"autogenerate_domain", // boolean default true; blank domains => generateUrl/sslip
}

// TestWriteCoverage_ApplicationCreateSilentDefaults fails when a known
// create-only silent-default field is absent from any Create*AppInput JSON
// tag set and not listed in applicationCreateWriteSkips (#643).
func TestWriteCoverage_ApplicationCreateSilentDefaults(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	ep, ok := c.Endpoints["ApplicationsController::create_application"]
	if !ok {
		t.Fatal("ApplicationsController::create_application not in contract")
	}
	allow := map[string]struct{}{}
	for _, f := range ep.AllowedFields {
		allow[f] = struct{}{}
	}
	byType := client.CreateAppInputJSONTagsByType()
	requireValidSkips(t, applicationCreateWriteSkips)

	var missing []string
	for _, field := range applicationCreateSilentDefaultFields {
		if _, ok := allow[field]; !ok {
			t.Errorf("silent-default field %q not in create allow-list (update list or Coolify source)", field)
			continue
		}
		if isSkipped(applicationCreateWriteSkips, field) {
			continue
		}
		for typeName, tags := range byType {
			if _, ok := tags[field]; !ok {
				missing = append(missing, typeName+"."+field)
			}
		}
	}
	if len(missing) > 0 {
		t.Errorf("create-only silent-default fields missing from Create*AppInput JSON tags:\n  %s\nAdd to every client create input + schema, or skipDeferred with an issue", strings.Join(missing, "\n  "))
	}
}

// TestWriteCoverage_ApplicationCreateSilentDefaultsInSchema ensures each
// silent-default field is either a Terraform schema attribute or an explicit skip.
func TestWriteCoverage_ApplicationCreateSilentDefaultsInSchema(t *testing.T) {
	t.Parallel()
	attrs, err := resourceSchemaAttributeNames(coolifyApplicationResource())
	if err != nil {
		t.Fatal(err)
	}
	for _, field := range applicationCreateSilentDefaultFields {
		if isSkipped(applicationCreateWriteSkips, field) {
			continue
		}
		if _, ok := attrs[field]; !ok {
			t.Errorf("create silent-default field %q missing from coolify_application schema (and not skipped)", field)
		}
	}
}

// notificationWriteChannels maps Coolify channel names to contract endpoint keys.
// Rules live in NotificationsController::channelConfig (not $allowedFields).
var notificationWriteChannels = []struct {
	channel  string
	endpoint string
}{
	{"email", "NotificationsController::update_email"},
	{"discord", "NotificationsController::update_discord"},
	{"slack", "NotificationsController::update_slack"},
	{"telegram", "NotificationsController::update_telegram"},
	{"pushover", "NotificationsController::update_pushover"},
	{"webhook", "NotificationsController::update_webhook"},
}

// TestWriteCoverage_ServiceUpdateNoExtraKeys fails if UpdateServiceInput
// would send a JSON key Coolify PATCH rejects. Extra-key 422 is the #789
// class (database UI bools). destination_uuid is create-only and must stay
// off this struct.
func TestWriteCoverage_ServiceUpdateNoExtraKeys(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	ep, ok := c.Endpoints["ServicesController::update_by_uuid"]
	if !ok {
		t.Fatal("ServicesController::update_by_uuid not in contract")
	}
	if len(ep.AllowedFields) == 0 {
		t.Fatal("ServicesController::update_by_uuid has empty allowed_fields")
	}
	allow := map[string]struct{}{}
	for _, f := range ep.AllowedFields {
		allow[f] = struct{}{}
	}
	tags := client.UpdateServiceJSONTags()
	if len(tags) == 0 {
		t.Fatal("UpdateServiceJSONTags returned empty set")
	}
	var extra []string
	for field := range tags {
		if _, ok := allow[field]; !ok {
			extra = append(extra, field)
		}
	}
	sort.Strings(extra)
	if len(extra) > 0 {
		t.Errorf("UpdateServiceInput JSON keys not on ServicesController::update_by_uuid allowed_fields:\n  %s",
			strings.Join(extra, "\n  "))
	}
}

// TestWriteCoverage_DatabaseDisallowedNotOnAllowList fails if a key we
// strip before PATCH appears on the extracted update allow list. That
// would mean Coolify started accepting it, or the strip list is stale.
func TestWriteCoverage_DatabaseDisallowedNotOnAllowList(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	ep, ok := c.Endpoints["DatabasesController::update_by_uuid"]
	if !ok {
		t.Fatal("DatabasesController::update_by_uuid not in contract")
	}
	allow := map[string]struct{}{}
	for _, f := range ep.AllowedFields {
		allow[f] = struct{}{}
	}
	var leaked []string
	for _, field := range client.DatabaseUpdateDisallowedJSONKeys {
		if _, ok := allow[field]; ok {
			leaked = append(leaked, field)
		}
	}
	sort.Strings(leaked)
	if len(leaked) > 0 {
		t.Errorf("DatabaseUpdateDisallowedJSONKeys still listed on update allowed_fields (stop stripping or fix extract):\n  %s",
			strings.Join(leaked, "\n  "))
	}
}

// TestWriteCoverage_NotificationUpdates ensures channelConfig write fields
// appear on client Update*NotificationInput JSON tags.
func TestWriteCoverage_NotificationUpdates(t *testing.T) {
	t.Parallel()
	c := loadContract(t)

	for _, ch := range notificationWriteChannels {
		ch := ch
		t.Run(ch.channel, func(t *testing.T) {
			t.Parallel()
			ep, ok := c.Endpoints[ch.endpoint]
			if !ok {
				t.Fatalf("endpoint %s not found in contract", ch.endpoint)
			}
			if len(ep.AllowedFields) == 0 {
				t.Fatalf("endpoint %s has empty allowed_fields (channelConfig extract missing?)", ch.endpoint)
			}
			tags := client.NotificationUpdateJSONTags(ch.channel)
			if len(tags) == 0 {
				t.Fatalf("NotificationUpdateJSONTags(%q) returned empty set", ch.channel)
			}
			var missing []string
			for _, field := range ep.AllowedFields {
				if _, ok := tags[field]; !ok {
					missing = append(missing, field)
				}
			}
			sort.Strings(missing)
			if len(missing) > 0 {
				t.Errorf("%s allowed_fields missing from Update*NotificationInput:\n  %s",
					ch.endpoint, strings.Join(missing, "\n  "))
			}
		})
	}
}

// TestWriteCoverage_InstanceEmailUpdate keeps PATCH /settings/email keys
// aligned with UpdateInstanceEmailInput (Coolify v4.3.10+).
func TestWriteCoverage_InstanceEmailUpdate(t *testing.T) {
	t.Parallel()
	want := []string{
		"smtp_enabled", "smtp_from_address", "smtp_from_name", "smtp_host",
		"smtp_port", "smtp_encryption", "smtp_username", "smtp_password",
		"smtp_timeout", "smtp_ehlo_domain", "resend_enabled", "resend_api_key",
	}
	tags := client.InstanceEmailUpdateJSONTags()
	var missing []string
	for _, field := range want {
		if _, ok := tags[field]; !ok {
			missing = append(missing, field)
		}
	}
	if len(missing) > 0 {
		t.Errorf("UpdateInstanceEmailInput missing JSON tags:\n  %s", strings.Join(missing, "\n  "))
	}
}
