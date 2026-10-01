package service

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullUnknownServiceCreate_SetsComputedUnknownToNull(t *testing.T) {
	t.Parallel()

	plan := serviceResourceModel{
		Name:          types.StringValue("kept-name"),
		Status:        types.StringUnknown(),
		DockerCompose: types.StringUnknown(),
		ConfigHash:    types.StringUnknown(),
	}

	nullUnknownServiceCreate(&plan)

	if !plan.Status.IsNull() {
		t.Errorf("Status = %v, want Null", plan.Status)
	}
	if !plan.DockerCompose.IsNull() {
		t.Errorf("DockerCompose = %v, want Null", plan.DockerCompose)
	}
	if !plan.ConfigHash.IsNull() {
		t.Errorf("ConfigHash = %v, want Null", plan.ConfigHash)
	}
	if plan.Name.IsNull() || plan.Name.IsUnknown() || plan.Name.ValueString() != "kept-name" {
		t.Errorf("Name = %v, want known %q", plan.Name, "kept-name")
	}
}
