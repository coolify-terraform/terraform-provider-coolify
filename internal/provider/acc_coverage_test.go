package provider

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/resource"
)

// TestAccCoverage_AllResourcesHaveAccTests verifies that every resource
// registered in the provider has at least one acceptance test in an
// _acc_test.go file. This catches the case where a new resource is added
// to provider.go but no acceptance test is written.
func TestAccCoverage_AllResourcesHaveAccTests(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := &coolifyProvider{version: "test"}

	accContent := loadAccTestContent(t)

	for _, factory := range p.Resources(ctx) {
		r := factory()
		resp := resource.MetadataResponse{}
		r.Metadata(ctx, resource.MetadataRequest{ProviderTypeName: "coolify"}, &resp)
		typeName := resp.TypeName
		if !strings.Contains(accContent, `"`+typeName+`"`) {
			t.Errorf("resource %s has no acceptance test (type name not found in any *_acc_test.go)", typeName)
		}
	}
}

// TestAccCoverage_AllDataSourcesHaveAccTests verifies that every data source
// registered in the provider has at least one acceptance test.
func TestAccCoverage_AllDataSourcesHaveAccTests(t *testing.T) {
	t.Parallel()
	ctx := context.Background()
	p := &coolifyProvider{version: "test"}

	accContent := loadAccTestContent(t)

	for _, factory := range p.DataSources(ctx) {
		ds := factory()
		resp := datasource.MetadataResponse{}
		ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "coolify"}, &resp)
		typeName := resp.TypeName
		if !strings.Contains(accContent, `"`+typeName+`"`) {
			t.Errorf("data source %s has no acceptance test (type name not found in any *_acc_test.go)", typeName)
		}
	}
}

func loadAccTestContent(t *testing.T) string {
	t.Helper()
	var b strings.Builder
	err := filepath.Walk(filepath.Join("..", "service"), func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if strings.HasSuffix(path, "_acc_test.go") {
			data, readErr := os.ReadFile(path)
			if readErr != nil {
				return readErr
			}
			b.Write(data)
		}
		return nil
	})
	if err != nil {
		t.Fatalf("failed to scan acc test files: %v", err)
	}
	if b.Len() == 0 {
		t.Fatal("no *_acc_test.go files found under internal/service/")
	}
	return b.String()
}
