package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestClient_ListVulnerabilitiesRequires47(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1", "test-token")
	c.CoolifyVersion = "4.6.0"
	_, err := c.ListVulnerabilities(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "Coolify >= v4.7.0") {
		t.Fatalf("got %v", err)
	}
}

func TestClient_ListVulnerabilitiesRejectsUnknownStatus(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1", "test-token")
	c.CoolifyVersion = "4.7.0"
	_, err := c.ListVulnerabilities(context.Background(), []string{"urgent"})
	if err == nil || !strings.Contains(err.Error(), "urgent") {
		t.Fatalf("got %v", err)
	}
}

func TestClient_ListVulnerabilities(t *testing.T) {
	t.Parallel()
	body := `{"resources":[{
		"uuid":"550e8400-e29b-41d4-a716-446655440000",
		"name":"web",
		"type":"application",
		"project_uuid":null,
		"environment_uuid":"env-1",
		"server_uuid":"srv-1",
		"images":{"update":1,"total":2},
		"security":{
			"status":"update_now","label":"Update now","action":null,"check_reason":null,
			"problem_count":1,"exploited_count":0,"any_exploited":false,
			"critical_count":1,"high_count":0,"medium_count":0,
			"eol":null,"checked_at":null,"actions":[]
		}
	}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/vulnerabilities" || r.URL.Query().Get("status") != "update_now,ok" {
			http.Error(w, `{"message":"bad request"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.0"
	got, err := c.ListVulnerabilities(context.Background(), []string{"update_now", "ok"})
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].UUID == "" || got[0].Name != "web" || got[0].Type != "application" {
		t.Fatalf("row: %+v", got)
	}
	if got[0].ProjectUUID != nil || got[0].EnvironmentUUID == nil || *got[0].EnvironmentUUID != "env-1" {
		t.Fatalf("uuids: %+v", got[0])
	}
	if got[0].Images.Update != 1 || got[0].Images.Total != 2 || got[0].Security.Status != "update_now" {
		t.Fatalf("images/security: %+v %+v", got[0].Images, got[0].Security)
	}
}

func TestClient_ListVulnerabilitiesEmptyUUID(t *testing.T) {
	t.Parallel()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"resources":[{"uuid":"","name":"web","type":"application","images":{"update":0,"total":0},"security":{"status":"ok","label":"No action needed","problem_count":0,"exploited_count":0,"any_exploited":false,"critical_count":0,"high_count":0,"medium_count":0,"actions":[]}}]}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.6"
	_, err := c.ListVulnerabilities(context.Background(), nil)
	if err == nil || !strings.Contains(err.Error(), "empty uuid") {
		t.Fatalf("got %v", err)
	}
}

func TestClient_GetResourceVulnerabilities(t *testing.T) {
	t.Parallel()
	const uuid = "550e8400-e29b-41d4-a716-446655440000"
	body := `{
		"uuid":"` + uuid + `",
		"type":"application",
		"off_reason":null,
		"security":{
			"status":"update_now","label":"Update now","action":"Upgrade nginx","check_reason":null,
			"problem_count":2,"exploited_count":1,"any_exploited":true,
			"critical_count":1,"high_count":1,"medium_count":0,
			"eol":{"name":"Alpine","version":"3.18","fixed":false},
			"checked_at":"2026-11-21T00:00:00Z",
			"actions":[{"type":"update","sentence":"Upgrade nginx","target":"nginx:1.27","copy":null,"image":"nginx:1.25","ref":"nginx"}]
		},
		"images":[{
			"scan_uuid":"scan-1","owner_uuid":"` + uuid + `","image_ref":"nginx:1.25",
			"scan_status":"success","error_code":null,"scanned_at":"2026-11-21T00:00:00Z",
			"truncated":false,"security_status":"update_now"
		}],
		"findings":[{
			"id":"CVE-2026-0001","display_id":"CVE-2026-0001","aliases":["GHSA-xxxx-xxxx-xxxx"],
			"title":"Example","severity":"critical","package":"nginx","version":"1.25.0",
			"package_type":"deb","fixed_in":"1.27.0","state":"no_fix","ignore":null,
			"kev":true,"epss":0.5,"risk":9,"cvss_score":null,
			"advisory_url":"https://nvd.nist.gov/vuln/detail/CVE-2026-0001",
			"image_ref":"nginx:1.25","owner_uuid":"` + uuid + `"
		}],
		"pagination":{"page":2,"per_page":10,"total":3}
	}`
	page := int64(2)
	perPage := int64(10)
	query := ResourceVulnerabilityQuery{
		State:    []string{"no_fix"},
		Severity: []string{"critical"},
		Image:    "nginx:1.25",
		Search:   "nginx",
		Page:     &page,
		PerPage:  &perPage,
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/applications/"+uuid+"/vulnerabilities" {
			http.NotFound(w, r)
			return
		}
		q := r.URL.Query()
		if q.Get("state") != "no_fix" || q.Get("severity") != "critical" || q.Get("image") != "nginx:1.25" || q.Get("search") != "nginx" || q.Get("page") != "2" || q.Get("per_page") != "10" {
			http.Error(w, `{"message":"bad query"}`, http.StatusBadRequest)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(body))
	}))
	t.Cleanup(srv.Close)

	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.0"
	got, err := c.GetResourceVulnerabilities(context.Background(), "applications", uuid, query)
	if err != nil {
		t.Fatal(err)
	}
	if got.UUID != uuid || got.Type != "application" || got.OffReason != nil || got.Security == nil {
		t.Fatalf("header: %+v", got)
	}
	if got.Security.EOL == nil || got.Security.EOL.Name != "Alpine" || got.Security.EOL.Fixed {
		t.Fatalf("eol: %+v", got.Security.EOL)
	}
	if len(got.Findings) != 1 || !got.Findings[0].KEV || got.Findings[0].EPSS == nil || *got.Findings[0].EPSS != 0.5 {
		t.Fatalf("finding: %+v", got.Findings)
	}
	if got.Findings[0].CVSSScore != nil || got.Findings[0].Risk == nil || *got.Findings[0].Risk != 9 {
		t.Fatalf("scores: %+v", got.Findings[0])
	}
	if got.Pagination.Page != 2 || got.Pagination.Total != 3 || len(got.Images) != 1 {
		t.Fatalf("page/images: %+v %d", got.Pagination, len(got.Images))
	}
}

func TestClient_GetResourceVulnerabilitiesNullSecurity(t *testing.T) {
	t.Parallel()
	const uuid = "550e8400-e29b-41d4-a716-446655440001"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/services/"+uuid+"/vulnerabilities" || r.URL.RawQuery != "" {
			http.NotFound(w, r)
			return
		}
		_, _ = w.Write([]byte(`{"uuid":"` + uuid + `","type":"service","off_reason":"server","security":null,"images":[],"findings":[],"pagination":{"page":1,"per_page":50,"total":0}}`))
	}))
	t.Cleanup(srv.Close)
	c := New(srv.URL, "test-token")
	c.CoolifyVersion = "4.7.0"
	got, err := c.GetResourceVulnerabilities(context.Background(), "services", uuid, ResourceVulnerabilityQuery{})
	if err != nil {
		t.Fatal(err)
	}
	if got.Security != nil || got.OffReason == nil || *got.OffReason != "server" || got.Pagination.PerPage != 50 {
		t.Fatalf("%+v", got)
	}
}

func TestClient_GetResourceVulnerabilitiesRejectsPath(t *testing.T) {
	t.Parallel()
	c := New("http://127.0.0.1:1", "test-token")
	c.CoolifyVersion = "4.7.0"
	if _, err := c.GetResourceVulnerabilities(context.Background(), "projects", "550e8400-e29b-41d4-a716-446655440000", ResourceVulnerabilityQuery{}); err == nil {
		t.Fatal("expected unsupported collection")
	}
	if _, err := c.GetResourceVulnerabilities(context.Background(), "databases", "../x", ResourceVulnerabilityQuery{}); err == nil || !strings.Contains(err.Error(), "invalid uuid") {
		t.Fatalf("got %v", err)
	}
}
