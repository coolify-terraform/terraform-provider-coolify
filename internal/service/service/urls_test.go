package service

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func urlEntry(name, url string) serviceURLModel {
	return serviceURLModel{Name: types.StringValue(name), URL: types.StringValue(url)}
}

func TestServiceURLsForUpdate_UnchangedOmits(t *testing.T) {
	t.Parallel()
	plan := []serviceURLModel{urlEntry("web", "https://b.example,https://a.example")}
	state := []serviceURLModel{urlEntry("web", "https://a.example,https://b.example")}
	if got := serviceURLsForUpdate(plan, state); got != nil {
		t.Fatalf("got %#v, want nil for equivalent URLs", got)
	}
}

func TestServiceURLsForUpdate_ClearRemovedName(t *testing.T) {
	t.Parallel()
	plan := []serviceURLModel{urlEntry("web", "https://web.example")}
	state := []serviceURLModel{
		urlEntry("web", "https://web.example"),
		urlEntry("api", "https://api.example"),
	}
	got := serviceURLsForUpdate(plan, state)
	if len(got) != 2 {
		t.Fatalf("len = %d, want web plus cleared api", len(got))
	}
	if got[0] != (client.ServiceURL{Name: "web", URL: "https://web.example"}) {
		t.Fatalf("kept = %#v", got[0])
	}
	if got[1] != (client.ServiceURL{Name: "api", URL: ""}) {
		t.Fatalf("cleared = %#v", got[1])
	}
}

func TestServiceURLsForUpdate_EmptyPlanClearsState(t *testing.T) {
	t.Parallel()
	state := []serviceURLModel{urlEntry("web", "https://web.example")}
	got := serviceURLsForUpdate(nil, state)
	if len(got) != 1 || got[0].Name != "web" || got[0].URL != "" {
		t.Fatalf("got %#v, want web cleared", got)
	}
	if got := serviceURLsForUpdate(nil, nil); got != nil {
		t.Fatalf("both empty = %#v, want nil", got)
	}
}
