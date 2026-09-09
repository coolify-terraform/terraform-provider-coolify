package backup

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullUnknownBackupComputed_SetsComputedUnknownToNull(t *testing.T) {
	plan := databaseBackupResourceModel{
		Description:        types.StringUnknown(),
		DisableLocalBackup: types.BoolUnknown(),
		DatabasesToBackup:  types.StringUnknown(),
		Timeout:            types.Int64Unknown(),
	}

	nullUnknownBackupComputed(&plan)

	if !plan.Description.IsNull() {
		t.Errorf("Description = %v, want Null", plan.Description)
	}
	if !plan.DisableLocalBackup.IsNull() {
		t.Errorf("DisableLocalBackup = %v, want Null", plan.DisableLocalBackup)
	}
	if !plan.DatabasesToBackup.IsNull() {
		t.Errorf("DatabasesToBackup = %v, want Null", plan.DatabasesToBackup)
	}
	if !plan.Timeout.IsNull() {
		t.Errorf("Timeout = %v, want Null", plan.Timeout)
	}
}
