// Package dbtest provides shared test helpers for database resource tests.
package dbtest

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/coolify-terraform/terraform-provider-coolify/internal/acctest"
)

// MockState holds the mutable state of a mock database server.
type MockState struct {
	mu          sync.Mutex
	UUID        string
	Name        string
	Description string
	Image       string
	ExtraFields map[string]interface{}
	Deleted     bool
	LastCreate  map[string]interface{}
	LastPatch   map[string]interface{}
}

// databaseUpdateAllowed is Coolify DatabasesController::update_by_uuid
// $allowedFields plus health_check_* (merged on Coolify >= v4.1.2).
var databaseUpdateAllowed = map[string]struct{}{
	"name": {}, "description": {}, "image": {},
	"public_port": {}, "public_port_timeout": {}, "is_public": {},
	"instant_deploy": {},
	"limits_memory":  {}, "limits_memory_swap": {}, "limits_memory_swappiness": {},
	"limits_memory_reservation": {}, "limits_cpus": {}, "limits_cpuset": {},
	"limits_cpu_shares": {},
	"postgres_user":     {}, "postgres_password": {}, "postgres_db": {},
	"postgres_initdb_args": {}, "postgres_host_auth_method": {}, "postgres_conf": {},
	"clickhouse_admin_user": {}, "clickhouse_admin_password": {},
	"dragonfly_password": {},
	"redis_password":     {}, "redis_conf": {},
	"keydb_password": {}, "keydb_conf": {},
	"mariadb_conf": {}, "mariadb_root_password": {}, "mariadb_user": {},
	"mariadb_password": {}, "mariadb_database": {},
	"mongo_conf": {}, "mongo_initdb_root_username": {}, "mongo_initdb_root_password": {},
	"mongo_initdb_database": {},
	"mysql_root_password":   {}, "mysql_password": {}, "mysql_user": {},
	"mysql_database": {}, "mysql_conf": {},
	"health_check_enabled": {}, "health_check_interval": {},
	"health_check_timeout": {}, "health_check_retries": {},
	"health_check_start_period": {},
}

func rejectUnallowedUpdate(w http.ResponseWriter, body map[string]interface{}) bool {
	errors := map[string]string{}
	for k := range body {
		if _, ok := databaseUpdateAllowed[k]; !ok {
			errors[k] = "This field is not allowed."
		}
	}
	if len(errors) == 0 {
		return false
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusUnprocessableEntity)
	_ = json.NewEncoder(w).Encode(map[string]interface{}{
		"message": "Validation failed.",
		"errors":  errors,
	})
	return true
}

// buildResponse returns the JSON-serializable map for a GET response.
func (s *MockState) buildResponse() map[string]interface{} {
	resp := map[string]interface{}{
		"uuid":                      s.UUID,
		"name":                      s.Name,
		"description":               s.Description,
		"project_uuid":              "aaaa0001-0001-4000-8000-000000000001",
		"server_uuid":               "bbbb0001-0001-4000-8000-000000000001",
		"environment_name":          "production",
		"image":                     s.Image,
		"is_public":                 false,
		"public_port":               nil,
		"limits_memory":             "0",
		"limits_memory_swap":        "0",
		"limits_memory_swappiness":  60,
		"limits_memory_reservation": "0",
		"limits_cpus":               "0",
		"limits_cpuset":             "0",
		"limits_cpu_shares":         1024,
		"status":                    "running",
		"internal_db_url":           "",
	}
	for k, v := range s.ExtraFields {
		resp[k] = v
	}
	return resp
}

// applyPatch updates the state from a PATCH request body.
func (s *MockState) applyPatch(body map[string]interface{}) {
	if v, ok := body["name"].(string); ok {
		s.Name = v
	}
	if v, ok := body["description"].(string); ok {
		s.Description = v
	}
	if v, ok := body["image"].(string); ok {
		s.Image = v
	}
	for k, v := range body {
		switch k {
		case "name", "description", "image", "project_uuid", "server_uuid", "environment_name", "destination_uuid", "instant_deploy":
			continue
		default:
			s.ExtraFields[k] = v
		}
	}
}

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func (s *MockState) handlePatch(w http.ResponseWriter, r *http.Request) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}
	s.LastPatch = body
	if rejectUnallowedUpdate(w, body) {
		return
	}
	s.applyPatch(body)
	writeJSON(w, http.StatusOK, map[string]string{"message": "updated"})
}

// validateCreateBody decodes the POST body and checks that all required fields
// are present. Returns the decoded body and true on success, or writes an error
// response and returns nil, false. Returning the body allows the caller to apply
// user-supplied fields (name, image, etc.) to the mock state so that the
// subsequent GET reflects what was actually sent on Create.
func validateCreateBody(w http.ResponseWriter, r *http.Request) (map[string]interface{}, bool) {
	var body map[string]interface{}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return nil, false
	}
	for _, field := range []string{"project_uuid", "server_uuid", "environment_name"} {
		if _, ok := body[field]; !ok {
			http.Error(w, fmt.Sprintf(`{"error":"missing required field: %s"}`, field), http.StatusUnprocessableEntity)
			return nil, false
		}
	}
	return body, true
}

// NewMockServer creates an httptest.Server that simulates the Coolify database
// API for the given database type. extraFields are db-specific fields included
// in GET responses and updatable via PATCH (e.g., {"redis_password": "pass"}).
func NewMockServer(dbType, name, image string, extraFields map[string]interface{}) (*httptest.Server, *MockState) {
	// Seed common fields as defaults so applyPatch can update them.
	merged := map[string]interface{}{
		"is_log_drain_enabled":      false,
		"is_include_timestamps":     false,
		"health_check_enabled":      true,
		"health_check_interval":     15,
		"health_check_timeout":      5,
		"health_check_retries":      5,
		"health_check_start_period": 5,
		"enable_ssl":                false,
		"ssl_mode":                  "",
	}
	for k, v := range extraFields {
		merged[k] = v
	}
	state := &MockState{
		UUID:        "aaaa0001-0001-4000-8000-000000000001",
		Name:        name,
		Image:       image,
		ExtraFields: merged,
	}

	dbPath := fmt.Sprintf("/api/v1/databases/%s", state.UUID)
	srv := httptest.NewServer(acctest.WithVersionEndpoint(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		state.mu.Lock()
		defer state.mu.Unlock()

		switch {
		case r.Method == http.MethodPost && r.URL.Path == "/api/v1/databases/"+dbType:
			body, ok := validateCreateBody(w, r)
			if !ok {
				return
			}
			state.LastCreate = body
			state.applyPatch(body)
			writeJSON(w, http.StatusCreated, map[string]string{"uuid": state.UUID})

		case r.Method == http.MethodGet && r.URL.Path == dbPath:
			if state.Deleted {
				http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
				return
			}
			writeJSON(w, http.StatusOK, state.buildResponse())

		case r.Method == http.MethodPatch && r.URL.Path == dbPath:
			state.handlePatch(w, r)

		case r.Method == http.MethodDelete && r.URL.Path == dbPath:
			state.Deleted = true
			writeJSON(w, http.StatusOK, map[string]string{"message": "deleted"})

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/start"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodPost && strings.HasSuffix(r.URL.Path, "/stop"):
			w.WriteHeader(http.StatusOK)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/v1/servers/") && strings.HasSuffix(r.URL.Path, "/resources"):
			writeServerResources(w, r.URL.Path, state)

		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})))
	return srv, state
}

// writeServerResources answers GET /servers/{uuid}/resources for compound import.
// Only the default test server UUID hosts the mock database.
func writeServerResources(w http.ResponseWriter, path string, state *MockState) {
	const defaultServerUUID = "bbbb0001-0001-4000-8000-000000000001"
	parts := strings.Split(strings.Trim(path, "/"), "/")
	serverUUID := ""
	if len(parts) >= 4 {
		serverUUID = parts[3]
	}
	if serverUUID != defaultServerUUID {
		writeJSON(w, http.StatusOK, []map[string]string{})
		return
	}
	writeJSON(w, http.StatusOK, []map[string]string{
		{"uuid": state.UUID, "name": state.Name, "type": "database"},
	})
}
