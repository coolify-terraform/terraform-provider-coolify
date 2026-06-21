package acctest

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestDeleteOnceFailGate_Wrap(t *testing.T) {
	t.Parallel()
	var gate DeleteOnceFailGate
	gate.Armed.Store(true)

	handler := gate.Wrap(http.StatusOK, http.StatusInternalServerError, `{"message":"fail"}`)

	req := httptest.NewRequest(http.MethodDelete, "/api/v1/test", nil)

	rec := httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusInternalServerError {
		t.Fatalf("first call: expected 500, got %d", rec.Code)
	}

	rec = httptest.NewRecorder()
	handler(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("second call: expected 200, got %d", rec.Code)
	}
}
