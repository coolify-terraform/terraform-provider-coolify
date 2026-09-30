package service

import (
	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

// serviceURLsForUpdate builds the urls PATCH list.
//
// Coolify only rewrites containers named in the list. A removed name has to
// be sent with an empty url, which sets that container FQDN to null. An
// unchanged list is omitted so a description update does not re-check domains.
func serviceURLsForUpdate(plan, state []serviceURLModel) []client.ServiceURL {
	if serviceURLListEqual(plan, state) {
		return nil
	}
	out := make([]client.ServiceURL, 0, len(plan)+len(state))
	planned := make(map[string]struct{}, len(plan))
	for _, u := range plan {
		name := u.Name.ValueString()
		planned[name] = struct{}{}
		out = append(out, client.ServiceURL{Name: name, URL: serviceURLString(u.URL)})
	}
	for _, u := range state {
		name := u.Name.ValueString()
		if _, ok := planned[name]; ok {
			continue
		}
		out = append(out, client.ServiceURL{Name: name, URL: ""})
	}
	return out
}

func serviceURLString(v types.String) string {
	if v.IsNull() || v.IsUnknown() {
		return ""
	}
	return v.ValueString()
}

func serviceURLListEqual(plan, state []serviceURLModel) bool {
	if len(plan) != len(state) {
		return false
	}
	stateByName := make(map[string]string, len(state))
	for _, u := range state {
		name := u.Name.ValueString()
		if _, dup := stateByName[name]; dup {
			return false
		}
		stateByName[name] = serviceURLTokenKey(serviceURLString(u.URL))
	}
	seen := make(map[string]struct{}, len(plan))
	for _, u := range plan {
		name := u.Name.ValueString()
		if _, dup := seen[name]; dup {
			return false
		}
		seen[name] = struct{}{}
		got, ok := stateByName[name]
		if !ok || got != serviceURLTokenKey(serviceURLString(u.URL)) {
			return false
		}
	}
	return true
}
