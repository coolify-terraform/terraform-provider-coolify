package service

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullUnknownServiceCreate_SetsComputedUnknownToNull(t *testing.T) {
	t.Parallel()

	plan := serviceResourceModel{
		Name:                          types.StringValue("kept-name"),
		Status:                        types.StringUnknown(),
		DockerCompose:                 types.StringUnknown(),
		ConfigHash:                    types.StringUnknown(),
		ConnectToNetwork:              types.BoolUnknown(),
		IsContainerLabelEscapeEnabled: types.BoolUnknown(),
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
	if !plan.ConnectToNetwork.IsNull() {
		t.Errorf("ConnectToNetwork = %v, want Null", plan.ConnectToNetwork)
	}
	if !plan.IsContainerLabelEscapeEnabled.IsNull() {
		t.Errorf("IsContainerLabelEscapeEnabled = %v, want Null", plan.IsContainerLabelEscapeEnabled)
	}
	if plan.Name.IsNull() || plan.Name.IsUnknown() || plan.Name.ValueString() != "kept-name" {
		t.Errorf("Name = %v, want known %q", plan.Name, "kept-name")
	}

	kept := serviceResourceModel{
		ConnectToNetwork:              types.BoolValue(true),
		IsContainerLabelEscapeEnabled: types.BoolValue(false),
	}
	nullUnknownServiceCreate(&kept)
	if !kept.ConnectToNetwork.Equal(types.BoolValue(true)) {
		t.Errorf("ConnectToNetwork = %v, want true", kept.ConnectToNetwork)
	}
	if !kept.IsContainerLabelEscapeEnabled.Equal(types.BoolValue(false)) {
		t.Errorf("IsContainerLabelEscapeEnabled = %v, want false", kept.IsContainerLabelEscapeEnabled)
	}
}
