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
	specOnce  sync.Map // version -> *sync.Once
	specCache sync.Map // version -> *libopenapi.Document
	specErr   sync.Map // version -> error

)

// LoadSpec loads an OpenAPI spec from testdata/specs/ by version name.
// Example: LoadSpec("coolify-v4") loads testdata/specs/coolify-v4.json.
// The result is cached for the lifetime of the test process.
func LoadSpec(version string) (*libopenapi.Document, error) {
	once, _ := specOnce.LoadOrStore(version, &sync.Once{})
	once.(*sync.Once).Do(func() {
		specPath := filepath.Join(testdataDir(), "specs", version+".json")
		data, err := os.ReadFile(specPath)
		if err != nil {
			specErr.Store(version, err)
			return
		}
		doc, err := libopenapi.NewDocument(data)
		if err != nil {
			specErr.Store(version, err)
			return
		}
		specCache.Store(version, &doc)
	})
	if e, ok := specErr.Load(version); ok {
		return nil, e.(error)
	}
	cached, _ := specCache.Load(version)
	return cached.(*libopenapi.Document), nil
}

// newValidator creates a fresh validator for the given spec version.
// Each caller gets its own instance because the libopenapi-validator
// is not safe for concurrent ValidateHttpRequest/ValidateHttpResponse
// calls from multiple goroutines.
func newValidator(version string) (validator.Validator, error) {
	specPath := filepath.Join(testdataDir(), "specs", version+".json")
	data, err := os.ReadFile(specPath)
	if err != nil {
		return nil, err
	}
	doc, err := libopenapi.NewDocument(data)
	if err != nil {
		return nil, err
	}
	v, errs := validator.NewValidator(doc)
	if len(errs) > 0 {
		return nil, fmt.Errorf("creating validator: %v", errs)
	}
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
