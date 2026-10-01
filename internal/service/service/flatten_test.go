package service

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenServiceURLs_PreservesConfigOrder(t *testing.T) {
	t.Parallel()

	current := []serviceURLModel{
		{Name: types.StringValue("zebra"), URL: types.StringValue("https://zebra.example.com")},
		{Name: types.StringValue("alpha"), URL: types.StringValue("https://alpha.example.com")},
	}
	apps := []client.ServiceApplication{
		{Name: "alpha", FQDN: "https://alpha.example.com"},
		{Name: "zebra", FQDN: "https://zebra.example.com"},
	}

	got := flattenServiceURLs(apps, current)
	if len(got) != 2 {
		t.Fatalf("len = %d, want 2", len(got))
	}
	if got[0].Name.ValueString() != "zebra" || got[1].Name.ValueString() != "alpha" {
		t.Fatalf("order = [%s, %s], want [zebra, alpha]",
			got[0].Name.ValueString(), got[1].Name.ValueString())
	}
	if got[0].URL.ValueString() != "https://zebra.example.com" {
		t.Fatalf("zebra url = %q", got[0].URL.ValueString())
	}
}

func TestFlattenServiceURLs_NilCurrentStaysNil(t *testing.T) {
	t.Parallel()
	apps := []client.ServiceApplication{{Name: "web", FQDN: "https://auto.example.com"}}
	if got := flattenServiceURLs(apps, nil); got != nil {
		t.Fatalf("got %#v, want nil", got)
	}
}

func TestFlattenServiceURLs_EmptyAppsKeepsCurrent(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://app.example.com")},
	}
	got := flattenServiceURLs(nil, current)
	if len(got) != 1 || got[0].Name.ValueString() != "web" {
		t.Fatalf("got %#v, want current", got)
	}
}

func TestFlattenServiceURLs_DropsAPIExtras(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://web.example.com")},
	}
	apps := []client.ServiceApplication{
		{Name: "db", FQDN: "https://db.example.com"},
		{Name: "web", FQDN: "https://web.example.com"},
	}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].Name.ValueString() != "web" {
		t.Fatalf("got %#v, want only web", got)
	}
}

func TestFlattenServiceURLs_KeepsConfiguredWhenAPIMissing(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://web.example.com")},
		{Name: types.StringValue("api"), URL: types.StringValue("https://api.example.com")},
	}
	apps := []client.ServiceApplication{
		{Name: "web", FQDN: "https://web.example.com"},
	}
	got := flattenServiceURLs(apps, current)
	if len(got) != 2 || got[1].Name.ValueString() != "api" {
		t.Fatalf("got %#v, want web then api", got)
	}
}

func TestFlattenServiceURLs_PreservesEquivalentCommaOrder(t *testing.T) {
	t.Parallel()
	configured := "https://qbt.example.com:8080,https://prowlarr.example.com:9696"
	current := []serviceURLModel{
		{Name: types.StringValue("gluetun"), URL: types.StringValue(configured)},
	}
	apps := []client.ServiceApplication{
		{Name: "gluetun", FQDN: "https://prowlarr.example.com:9696,https://qbt.example.com:8080"},
	}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 {
		t.Fatalf("len = %d", len(got))
	}
	if got[0].URL.ValueString() != configured {
		t.Fatalf("url = %q, want configured order", got[0].URL.ValueString())
	}
}

func TestFlattenServiceURLs_HostCaseKeepsConfigured(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://Example.com/API")},
	}
	apps := []client.ServiceApplication{
		{Name: "web", FQDN: "https://example.com/API"},
	}
	got := flattenServiceURLs(apps, current)
	if got[0].URL.ValueString() != "https://Example.com/API" {
		t.Fatalf("url = %q, want configured host case", got[0].URL.ValueString())
	}
}

func TestFlattenServiceURLs_PathCaseUsesAPI(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://example.com/API")},
	}
	apps := []client.ServiceApplication{
		{Name: "web", FQDN: "https://example.com/api"},
	}
	got := flattenServiceURLs(apps, current)
	if got[0].URL.ValueString() != "https://example.com/api" {
		t.Fatalf("url = %q, want API path case", got[0].URL.ValueString())
	}
}

func TestFlattenServiceURLs_PreservesMatchingPort(t *testing.T) {
	t.Parallel()
	configured := "HTTPS://App.Example.com:8443/Path"
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue(configured)},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://app.example.com/Path",
		URL:  "https://app.example.com:8443/Path",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != configured {
		t.Fatalf("url = %#v, want configured port and host case", got)
	}
}

func TestFlattenServiceURLs_AdoptsAPIWhenPortDiffers(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://app.example.com:8443")},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://app.example.com",
		URL:  "https://app.example.com:9000",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != "https://app.example.com:9000" {
		t.Fatalf("url = %#v, want API url with Coolify's port", got)
	}
}

func TestFlattenServiceURLs_AdoptsPortlessURLWhenNoPortReturned(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://app.example.com:8443")},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://app.example.com",
		URL:  "https://app.example.com",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != "https://app.example.com" {
		t.Fatalf("url = %#v, want portless API URL", got)
	}
}

func TestFlattenServiceURLs_PreservesCommaOrderWhenReturnedPortsMatch(t *testing.T) {
	t.Parallel()
	configured := "https://Qbt.Example.com:8080,https://prowlarr.example.com:9696"
	current := []serviceURLModel{
		{Name: types.StringValue("gluetun"), URL: types.StringValue(configured)},
	}
	apps := []client.ServiceApplication{{
		Name: "gluetun",
		FQDN: "https://prowlarr.example.com,https://qbt.example.com",
		URL:  "https://prowlarr.example.com:9696,https://qbt.example.com:8080",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != configured {
		t.Fatalf("url = %#v, want configured order", got)
	}
}

func TestFlattenServiceURLs_PathCaseUsesReturnedURL(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://example.com:8443/API")},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://example.com/api",
		URL:  "https://example.com:8443/api",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != "https://example.com:8443/api" {
		t.Fatalf("url = %#v, want API path case", got)
	}
}

func TestFlattenServiceURLs_OmitsPortWhenConfigHasNone(t *testing.T) {
	t.Parallel()
	configured := "https://app.example.com/Path"
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue(configured)},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://app.example.com/Path",
		URL:  "https://app.example.com:8443/Path",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != configured {
		t.Fatalf("url = %#v, want configured URL without a port", got)
	}
}

func TestFlattenServiceURLs_FallsBackToFQDNWhenURLMissing(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://Example.com:8080")},
	}
	apps := []client.ServiceApplication{{
		Name: "web",
		FQDN: "https://example.com:8080",
	}}
	got := flattenServiceURLs(apps, current)
	if len(got) != 1 || got[0].URL.ValueString() != "https://Example.com:8080" {
		t.Fatalf("url = %#v, want configured when fqdn still has the port", got)
	}
}

func TestFlattenServiceURLs_UsesAPIWhenURLChanges(t *testing.T) {
	t.Parallel()
	current := []serviceURLModel{
		{Name: types.StringValue("web"), URL: types.StringValue("https://old.example.com")},
	}
	apps := []client.ServiceApplication{
		{Name: "web", FQDN: "https://new.example.com"},
	}
	got := flattenServiceURLs(apps, current)
	if got[0].URL.ValueString() != "https://new.example.com" {
		t.Fatalf("url = %q, want API value", got[0].URL.ValueString())
	}
}
