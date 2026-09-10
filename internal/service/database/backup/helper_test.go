package backup

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestNullUnknownBackupComputed_SetsComputedUnknownToNull(t *testing.T) {
	plan := databaseBackupResourceModel{
		ID:                              types.Int64Unknown(),
		Enabled:                         types.BoolUnknown(),
		SaveS3:                          types.BoolUnknown(),
		DumpAll:                         types.BoolUnknown(),
		RetainAmountLocally:             types.Int64Unknown(),
		RetainDaysLocally:               types.Int64Unknown(),
		RetainMaxStorageLocal:           types.Int64Unknown(),
		RetainAmountS3:                  types.Int64Unknown(),
		RetainDaysS3:                    types.Int64Unknown(),
		RetainMaxStorageS3:              types.Int64Unknown(),
		Timeout:                         types.Int64Unknown(),
		MissingBackupNotificationDays:   types.Int64Unknown(),
		LastExecutionAt:                 types.StringUnknown(),
		MissingBackupNotificationSentAt: types.StringUnknown(),
		Description:                     types.StringUnknown(),
		DisableLocalBackup:              types.BoolUnknown(),
		DatabasesToBackup:               types.StringUnknown(),
	}

	nullUnknownBackupComputed(&plan)

	if !plan.ID.IsNull() {
		t.Errorf("ID = %v, want Null", plan.ID)
	}
	if !plan.Enabled.IsNull() {
		t.Errorf("Enabled = %v, want Null", plan.Enabled)
	}
	if !plan.SaveS3.IsNull() {
		t.Errorf("SaveS3 = %v, want Null", plan.SaveS3)
	}
	if !plan.DumpAll.IsNull() {
		t.Errorf("DumpAll = %v, want Null", plan.DumpAll)
	}
	if !plan.RetainAmountLocally.IsNull() {
		t.Errorf("RetainAmountLocally = %v, want Null", plan.RetainAmountLocally)
	}
	if !plan.RetainDaysLocally.IsNull() {
		t.Errorf("RetainDaysLocally = %v, want Null", plan.RetainDaysLocally)
	}
	if !plan.RetainMaxStorageLocal.IsNull() {
		t.Errorf("RetainMaxStorageLocal = %v, want Null", plan.RetainMaxStorageLocal)
	}
	if !plan.RetainAmountS3.IsNull() {
		t.Errorf("RetainAmountS3 = %v, want Null", plan.RetainAmountS3)
	}
	if !plan.RetainDaysS3.IsNull() {
		t.Errorf("RetainDaysS3 = %v, want Null", plan.RetainDaysS3)
	}
	if !plan.RetainMaxStorageS3.IsNull() {
		t.Errorf("RetainMaxStorageS3 = %v, want Null", plan.RetainMaxStorageS3)
	}
	if !plan.Timeout.IsNull() {
		t.Errorf("Timeout = %v, want Null", plan.Timeout)
	}
	if !plan.MissingBackupNotificationDays.IsNull() {
		t.Errorf("MissingBackupNotificationDays = %v, want Null", plan.MissingBackupNotificationDays)
	}
	if !plan.LastExecutionAt.IsNull() {
		t.Errorf("LastExecutionAt = %v, want Null", plan.LastExecutionAt)
	}
	if !plan.MissingBackupNotificationSentAt.IsNull() {
		t.Errorf("MissingBackupNotificationSentAt = %v, want Null", plan.MissingBackupNotificationSentAt)
	}
	if !plan.Description.IsNull() {
		t.Errorf("Description = %v, want Null", plan.Description)
	}
	if !plan.DisableLocalBackup.IsNull() {
		t.Errorf("DisableLocalBackup = %v, want Null", plan.DisableLocalBackup)
	}
	if !plan.DatabasesToBackup.IsNull() {
		t.Errorf("DatabasesToBackup = %v, want Null", plan.DatabasesToBackup)
	}
}
