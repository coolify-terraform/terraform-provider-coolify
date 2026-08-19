package database

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestDatabaseCRUDFiles_SetLimitsMemory fails if a sibling engine acc
// CRUD file stops creating/updating limits_memory. Description-only Update
// missed #789 on every engine that shares SetUpdateExtended.
func TestDatabaseCRUDFiles_SetLimitsMemory(t *testing.T) {
	t.Parallel()
	engines := []string{
		"postgresql", "mysql", "mariadb", "mongodb",
		"redis", "keydb", "dragonfly", "clickhouse",
	}
	for _, engine := range engines {
		engine := engine
		t.Run(engine, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(engine, "resource_acc_test.go")
			b, err := os.ReadFile(path) //nolint:gosec // test fixture path from the engines list
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			body := string(b)
			if !strings.Contains(body, `limits_memory = "256M"`) {
				t.Errorf("%s acc CRUD must create with limits_memory", path)
			}
			if !strings.Contains(body, `limits_memory = "512M"`) {
				t.Errorf("%s acc CRUD must update limits_memory", path)
			}
		})
	}
}
