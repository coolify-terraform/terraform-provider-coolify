package application_test

import (
	"encoding/json"
	"net/http"
	"testing"
)

// decodeRequestBodyMap is shared by application resource unit tests that
// assert on Create/Update JSON. It lives in an untagged file so CI can
// compile complementary application test slices (ci_app_a / ci_app_b)
// without dropping the helper.
func decodeRequestBodyMap(t *testing.T, w http.ResponseWriter, r *http.Request) (map[string]interface{}, bool) {
	t.Helper()

	var requestBody map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&requestBody); err != nil {
		t.Errorf("decoding %s %s request body: %v", r.Method, r.URL.Path, err)
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return nil, false
	}
	return requestBody, true
}
