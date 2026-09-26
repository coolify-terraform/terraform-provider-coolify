package spectest

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
)

// requestGateWriteTags maps a contract endpoint to the client structs that
// encode its body. A struct that sends a gate's companion field must carry
// the gate on that same struct. A later PATCH does not count.
func requestGateWriteTags(endpoint string) (map[string]map[string]struct{}, bool) {
	one := func(name string, v any) map[string]map[string]struct{} {
		return map[string]map[string]struct{}{name: jsonTagSet(reflect.TypeOf(v))}
	}
	switch endpoint {
	case "ApplicationsController::create_application":
		return client.CreateAppInputJSONTagsByType(), true
	case "ApplicationsController::update_by_uuid":
		return one("UpdateApplicationInput", client.UpdateApplicationInput{}), true
	case "ServicesController::create_service":
		return one("CreateServiceInput", client.CreateServiceInput{}), true
	case "ServicesController::update_by_uuid":
		return one("UpdateServiceInput", client.UpdateServiceInput{}), true
	default:
		return nil, false
	}
}

func jsonTagSet(t reflect.Type) map[string]struct{} {
	tags := map[string]struct{}{}
	for i := 0; i < t.NumField(); i++ {
		tag := t.Field(i).Tag.Get("json")
		if tag == "" || tag == "-" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name != "" {
			tags[name] = struct{}{}
		}
	}
	return tags
}

// TestWriteCoverage_RequestGateSameBody fails when a create or update struct
// sends domains, docker_compose_domains, or urls without the request-only
// boolean Coolify reads on that same body.
func TestWriteCoverage_RequestGateSameBody(t *testing.T) {
	t.Parallel()
	c := loadContract(t)
	if len(c.RequestGates) == 0 {
		t.Fatal("contract request_gates is empty; re-extract the pinned contract")
	}
	var missing []string
	for _, gate := range c.RequestGates {
		if gate.Flag == "" || gate.Endpoint == "" || len(gate.SameRequestFields) == 0 {
			t.Errorf("incomplete request gate: %+v", gate)
			continue
		}
		structs, ok := requestGateWriteTags(gate.Endpoint)
		if !ok {
			t.Errorf("request gate %s has no client write struct mapping", gate.Endpoint)
			continue
		}
		for typeName, tags := range structs {
			sendsCompanion := false
			for _, field := range gate.SameRequestFields {
				if _, ok := tags[field]; ok {
					sendsCompanion = true
					break
				}
			}
			if !sendsCompanion {
				continue
			}
			if _, ok := tags[gate.Flag]; !ok {
				missing = append(missing, typeName+" missing "+gate.Flag+" required with "+strings.Join(gate.SameRequestFields, ","))
			}
		}
	}
	sort.Strings(missing)
	if len(missing) > 0 {
		t.Errorf("request gate missing from the same write struct:\n  %s", strings.Join(missing, "\n  "))
	}
}
