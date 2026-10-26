package client

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestS3StorageUUIDsEmptyListEncodes(t *testing.T) {
	t.Parallel()
	empty := []string{}
	cases := []any{
		UpdateDatabaseBackupInput{S3StorageUUIDs: &empty},
		CreateDatabaseBackupInput{Frequency: "daily", S3StorageUUIDs: &empty},
		UpsertVolumeBackupInput{Frequency: "daily", S3StorageUUIDs: &empty},
	}
	for _, in := range cases {
		raw, err := json.Marshal(in)
		if err != nil {
			t.Fatal(err)
		}
		if !strings.Contains(string(raw), `"s3_storage_uuids":[]`) {
			t.Fatalf("%T encoded %s", in, raw)
		}
	}
	omitted, err := json.Marshal(UpdateDatabaseBackupInput{})
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(omitted), "s3_storage_uuids") {
		t.Fatalf("nil list was sent: %s", omitted)
	}
}
