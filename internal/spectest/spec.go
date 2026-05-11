package spectest

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sync"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
)

var (
	specCache      sync.Map
	validatorCache sync.Map
)

// LoadSpec loads an OpenAPI spec from testdata/specs/ by version name.
// Example: LoadSpec("coolify-v4") loads testdata/specs/coolify-v4.json.
// The result is cached for the lifetime of the test process.
func LoadSpec(version string) (*libopenapi.Document, error) {
	if cached, ok := specCache.Load(version); ok {
		return cached.(*libopenapi.Document), nil
	}

	specPath := filepath.Join(testdataDir(), "specs", version+".json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}

	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, err
	}

	specCache.Store(version, &doc)
	return &doc, nil
}

// loadValidator returns a cached validator for the given spec version.
// The validator is created once and reused across all concurrent tests,
// avoiding a data race in libopenapi's BuildV3Model which mutates the
// document during NewValidator.
func loadValidator(version string) (validator.Validator, error) {
	if cached, ok := validatorCache.Load(version); ok {
		return cached.(validator.Validator), nil
	}
	doc, err := LoadSpec(version)
	if err != nil {
		return nil, err
	}
	v, errs := validator.NewValidator(*doc)
	if len(errs) > 0 {
		return nil, fmt.Errorf("creating validator: %v", errs)
	}
	validatorCache.Store(version, v)
	return v, nil
}

// SpecVersions returns the available spec version names from testdata/specs/.
func SpecVersions() ([]string, error) {
	dir := filepath.Join(testdataDir(), "specs")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var versions []string
	for _, e := range entries {
		if !e.IsDir() && filepath.Ext(e.Name()) == ".json" {
			name := e.Name()[:len(e.Name())-5] // strip .json
			versions = append(versions, name)
		}
	}
	return versions, nil
}

// testdataDir returns the absolute path to the project's testdata/ directory.
func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "..", "testdata")
}
