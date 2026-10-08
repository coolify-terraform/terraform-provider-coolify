package backup

import (
	"testing"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/client"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestFlattenDatabaseBackup_ClearsMissingSentAtOn44Tip(t *testing.T) {
	t.Parallel()
	prior := types.StringValue("2020-01-01T00:00:00Z")

	tip := databaseBackupResourceModel{MissingBackupNotificationSentAt: prior}
	flattenDatabaseBackup(&client.Client{CoolifyVersion: "4.4.0"}, &client.DatabaseBackup{}, &tip)
	if !tip.MissingBackupNotificationSentAt.IsNull() {
		t.Fatalf("4.4.0 kept %s", tip.MissingBackupNotificationSentAt)
	}

	stable := databaseBackupResourceModel{MissingBackupNotificationSentAt: prior}
	flattenDatabaseBackup(&client.Client{CoolifyVersion: "4.3.23"}, &client.DatabaseBackup{}, &stable)
	if stable.MissingBackupNotificationSentAt.ValueString() != prior.ValueString() {
		t.Fatalf("4.3.23 stored %s", stable.MissingBackupNotificationSentAt)
	}
}

func TestFlattenDatabaseBackup_EmptyDescription(t *testing.T) {
	t.Parallel()
	clientStub := &client.Client{CoolifyVersion: "4.4.2"}

	kept := databaseBackupResourceModel{Description: types.StringValue("")}
	flattenDatabaseBackup(clientStub, &client.DatabaseBackup{Description: ""}, &kept)
	if kept.Description.IsNull() || kept.Description.ValueString() != "" {
		t.Fatalf("configured empty description = %#v, want empty string", kept.Description)
	}

	imported := databaseBackupResourceModel{Description: types.StringNull()}
	flattenDatabaseBackup(clientStub, &client.DatabaseBackup{Description: ""}, &imported)
	if !imported.Description.IsNull() {
		t.Fatalf("null prior with empty API description = %#v, want null", imported.Description)
	}
}
