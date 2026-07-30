package spectest

import (
	"context"
	"fmt"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/service/environmentvariable"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// SchemaCoverageEntry maps a Coolify contract field to a Terraform schema
// attribute (or an explicit skip). Phase A: EnvironmentVariable (#621). Phase B: Application settings + ScheduledTask. Status is StatusCovered or a SkipStatus value.
type SchemaCoverageEntry struct {
	ContractField   string
	SchemaAttribute string // tfsdk name; empty when not covered
	Status          SkipStatus
	Issue           int
	Notes           string
}

func (e SchemaCoverageEntry) validate() error {
	if e.ContractField == "" {
		return fmt.Errorf("ContractField required")
	}
	switch e.Status {
	case StatusCovered:
		if e.SchemaAttribute == "" {
			return fmt.Errorf("covered requires SchemaAttribute")
		}
		if e.Issue != 0 {
			return fmt.Errorf("covered must not set Issue")
		}
	case SkipInternal, SkipNA:
		if e.SchemaAttribute != "" {
			return fmt.Errorf("%s must not set SchemaAttribute", e.Status)
		}
		if e.Issue != 0 {
			return fmt.Errorf("%s must not set Issue", e.Status)
		}
		if e.Notes == "" {
			return fmt.Errorf("%s requires Notes", e.Status)
		}
	case SkipDeferred:
		if e.Issue <= 0 {
			return fmt.Errorf("deferred requires Issue")
		}
		if e.Notes == "" {
			return fmt.Errorf("deferred requires Notes")
		}
	default:
		return fmt.Errorf("unknown status %q", e.Status)
	}
	return nil
}

// environmentVariableSchemaRegistry is Phase A of the schema coverage
// registry: every fillable EnvironmentVariable contract field has a row.
var environmentVariableSchemaRegistry = []SchemaCoverageEntry{
	{ContractField: "key", SchemaAttribute: "key", Status: StatusCovered},
	{ContractField: "value", SchemaAttribute: "value", Status: StatusCovered},
	{ContractField: "is_preview", SchemaAttribute: "is_preview", Status: StatusCovered},
	{ContractField: "is_buildtime", SchemaAttribute: "is_build", Status: StatusCovered, Notes: "Coolify is_buildtime"},
	{ContractField: "is_runtime", SchemaAttribute: "is_runtime", Status: StatusCovered},
	{ContractField: "is_literal", SchemaAttribute: "is_literal", Status: StatusCovered},
	{ContractField: "is_multiline", SchemaAttribute: "is_multiline", Status: StatusCovered},
	{ContractField: "comment", SchemaAttribute: "comment", Status: StatusCovered},
	{ContractField: "is_shown_once", Status: SkipDeferred, Issue: 626, Notes: "UI reveal-once"},
	{ContractField: "is_required", Status: SkipDeferred, Issue: 626, Notes: "product surface not yet managed"},
	{ContractField: "is_shared", Status: SkipDeferred, Issue: 626, Notes: "shared-var surface"},
	{ContractField: "order", Status: SkipDeferred, Issue: 626, Notes: "UI ordering"},
	{ContractField: "resourceable_id", Status: SkipInternal, Notes: "polymorphic FK"},
	{ContractField: "resourceable_type", Status: SkipInternal, Notes: "polymorphic FK"},
	{ContractField: "version", Status: SkipInternal, Notes: "Coolify internal version stamp"},
}

// resourceSchemaAttributeNames returns top-level attribute names from a resource Schema().
func resourceSchemaAttributeNames(r resource.Resource) (map[string]struct{}, error) {
	resp := &resource.SchemaResponse{}
	r.Schema(context.Background(), resource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		return nil, fmt.Errorf("schema diagnostics: %v", resp.Diagnostics)
	}
	out := make(map[string]struct{}, len(resp.Schema.Attributes))
	for name := range resp.Schema.Attributes {
		out[name] = struct{}{}
	}
	return out, nil
}

// coolifyEnvironmentVariableResource constructs the env var resource for schema walk.
func coolifyEnvironmentVariableResource() resource.Resource {
	return environmentvariable.NewResource()
}
