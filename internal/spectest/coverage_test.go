package spectest

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	v3high "github.com/pb33f/libopenapi/datamodel/high/v3"
)

// coverageStatus tracks a single API endpoint's provider coverage.
type coverageStatus struct {
	category string // "covered", "planned", "skipped"
	resource string // Terraform resource name or skip reason
	since    string // provider version that added support (covered only)
	priority int    // 1=high, 2=medium, 3=low (planned only)
	notes    string // human-readable context
}

// coveredEndpoints returns the full API endpoint registry.
// This is the single source of truth for API coverage. The
// TestSpecCoverage_Completeness test fails if the OpenAPI spec has
// endpoints not listed here. The TestSpecCoverage_GenerateDoc test
// generates API_COVERAGE.md from this data.
func coveredEndpoints() map[string]coverageStatus {
	covered := func(resource, since string) coverageStatus {
		return coverageStatus{category: "covered", resource: resource, since: since}
	}
	planned := func(priority int, notes string) coverageStatus {
		return coverageStatus{category: "planned", priority: priority, notes: notes}
	}
	skipped := func(reason string) coverageStatus {
		return coverageStatus{category: "skipped", resource: reason}
	}

	return map[string]coverageStatus{
		// ── Projects ──
		"GET /projects":        covered("data.coolify_projects", "v0.1.0"),
		"POST /projects":       covered("coolify_project", "v0.1.0"),
		"GET /projects/{uuid}": covered("data.coolify_project", "v0.1.0"),
		"PATCH /projects/{uuid}":  covered("coolify_project", "v0.1.0"),
		"DELETE /projects/{uuid}": covered("coolify_project", "v0.1.0"),
		"GET /projects/{uuid}/environments":                               planned(1, "Environment CRUD; core for multi-env Terraform workflows"),
		"POST /projects/{uuid}/environments":                              planned(1, "Environment CRUD"),
		"DELETE /projects/{uuid}/environments/{environment_name_or_uuid}": planned(1, "Environment CRUD"),
		"GET /projects/{uuid}/{environment_name_or_uuid}":                 planned(1, "Environment details data source"),

		// ── Servers ──
		"GET /servers":              covered("data.coolify_servers", "v0.1.0"),
		"POST /servers":             covered("coolify_server", "v0.1.0"),
		"GET /servers/{uuid}":       covered("data.coolify_server", "v0.1.0"),
		"PATCH /servers/{uuid}":     covered("coolify_server", "v0.1.0"),
		"DELETE /servers/{uuid}":    covered("coolify_server", "v0.1.0"),
		"GET /servers/{uuid}/domains":   covered("data.coolify_server_domains", "v0.1.0"),
		"GET /servers/{uuid}/resources": covered("data.coolify_server_resources", "v0.1.0"),
		"POST /servers/hetzner":        planned(2, "Hetzner cloud server provisioning"),
		"GET /servers/{uuid}/validate": skipped("Operational validation, not a Terraform resource"),

		// ── Applications ──
		"GET /applications":                    covered("data.coolify_applications", "v0.1.0"),
		"POST /applications/public":            covered("coolify_application", "v0.1.0"),
		"POST /applications/dockercompose":     covered("coolify_docker_compose_application", "v0.1.0"),
		"POST /applications/dockerimage":       covered("coolify_docker_image_application", "v0.1.0"),
		"POST /applications/private-deploy-key": covered("coolify_private_git_application", "v0.1.0"),
		"GET /applications/{uuid}":             covered("data.coolify_application", "v0.1.0"),
		"PATCH /applications/{uuid}":           covered("coolify_application + variants", "v0.1.0"),
		"DELETE /applications/{uuid}":          covered("coolify_application + variants", "v0.1.0"),
		"GET /applications/{uuid}/envs":        covered("data.coolify_environment_variables", "v0.1.0"),
		"POST /applications/{uuid}/envs":       covered("coolify_environment_variable", "v0.1.0"),
		"PATCH /applications/{uuid}/envs":      covered("coolify_environment_variable", "v0.1.0"),
		"DELETE /applications/{uuid}/envs/{env_uuid}": covered("coolify_environment_variable", "v0.1.0"),
		"GET /applications/{uuid}/restart":     covered("coolify_deployment", "v0.1.0"),
		"POST /applications/dockerfile":        planned(1, "Dockerfile app without git; common deployment pattern"),
		"POST /applications/private-github-app": planned(1, "GitHub App deployment; common in organizations"),
		"GET /applications/{uuid}/start":       planned(2, "Application lifecycle; client method exists"),
		"GET /applications/{uuid}/stop":        planned(2, "Application lifecycle; client method exists"),
		"GET /applications/{uuid}/scheduled-tasks":                        planned(2, "Scheduled tasks CRUD"),
		"POST /applications/{uuid}/scheduled-tasks":                       planned(2, "Scheduled tasks CRUD"),
		"PATCH /applications/{uuid}/scheduled-tasks/{task_uuid}":          planned(2, "Scheduled tasks CRUD"),
		"DELETE /applications/{uuid}/scheduled-tasks/{task_uuid}":         planned(2, "Scheduled tasks CRUD"),
		"GET /applications/{uuid}/scheduled-tasks/{task_uuid}/executions": planned(3, "Read-only task execution history"),
		"GET /applications/{uuid}/storages":                   planned(2, "Persistent storage CRUD"),
		"POST /applications/{uuid}/storages":                  planned(2, "Persistent storage CRUD"),
		"PATCH /applications/{uuid}/storages":                 planned(2, "Persistent storage CRUD"),
		"DELETE /applications/{uuid}/storages/{storage_uuid}": planned(2, "Persistent storage CRUD"),
		"PATCH /applications/{uuid}/envs/bulk":                planned(3, "Bulk env var update; convenience"),
		"GET /applications/{uuid}/logs":                       skipped("Streaming logs, not a Terraform resource"),
		"DELETE /applications/{uuid}/previews/{pull_request_id}": skipped("Preview deployment management, niche"),

		// ── Databases ──
		"GET /databases":             covered("data.coolify_databases", "v0.1.0"),
		"POST /databases/postgresql": covered("coolify_postgresql_database", "v0.1.0"),
		"POST /databases/mysql":      covered("coolify_mysql_database", "v0.1.0"),
		"POST /databases/mariadb":    covered("coolify_mariadb_database", "v0.1.0"),
		"POST /databases/mongodb":    covered("coolify_mongodb_database", "v0.1.0"),
		"POST /databases/redis":      covered("coolify_redis_database", "v0.1.0"),
		"POST /databases/clickhouse": covered("coolify_clickhouse_database", "v0.1.0"),
		"POST /databases/keydb":      covered("coolify_keydb_database", "v0.1.0"),
		"POST /databases/dragonfly":  covered("coolify_dragonfly_database", "v0.1.0"),
		"GET /databases/{uuid}":      covered("data.coolify_database", "v0.1.0"),
		"PATCH /databases/{uuid}":    covered("coolify_*_database", "v0.1.0"),
		"DELETE /databases/{uuid}":   covered("coolify_*_database", "v0.1.0"),
		"GET /databases/{uuid}/backups":                                    covered("coolify_database_backup", "v0.1.0"),
		"POST /databases/{uuid}/backups":                                   covered("coolify_database_backup", "v0.1.0"),
		"PATCH /databases/{uuid}/backups/{scheduled_backup_uuid}":          covered("coolify_database_backup", "v0.1.0"),
		"DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}":         covered("coolify_database_backup", "v0.1.0"),
		"GET /databases/{uuid}/envs":               planned(1, "Database env vars; parity with application env vars"),
		"POST /databases/{uuid}/envs":              planned(1, "Database env vars CRUD"),
		"PATCH /databases/{uuid}/envs":             planned(1, "Database env vars CRUD"),
		"DELETE /databases/{uuid}/envs/{env_uuid}": planned(1, "Database env vars CRUD"),
		"GET /databases/{uuid}/storages":                   planned(2, "Database persistent storage CRUD"),
		"POST /databases/{uuid}/storages":                  planned(2, "Database persistent storage CRUD"),
		"PATCH /databases/{uuid}/storages":                 planned(2, "Database persistent storage CRUD"),
		"DELETE /databases/{uuid}/storages/{storage_uuid}": planned(2, "Database persistent storage CRUD"),
		"GET /databases/{uuid}/backups/{scheduled_backup_uuid}/executions":                     planned(3, "Read-only backup execution history"),
		"DELETE /databases/{uuid}/backups/{scheduled_backup_uuid}/executions/{execution_uuid}": planned(3, "Delete individual backup execution"),
		"PATCH /databases/{uuid}/envs/bulk": planned(3, "Bulk env var update; convenience"),
		"GET /databases/{uuid}/restart":     skipped("Operational action, not a Terraform resource"),
		"GET /databases/{uuid}/start":       skipped("Operational action, not a Terraform resource"),
		"GET /databases/{uuid}/stop":        skipped("Operational action, not a Terraform resource"),

		// ── Services ──
		"GET /services":           covered("data.coolify_services", "v0.1.0"),
		"POST /services":          covered("coolify_service", "v0.1.0"),
		"GET /services/{uuid}":    covered("data.coolify_service", "v0.1.0"),
		"PATCH /services/{uuid}":  covered("coolify_service", "v0.1.0"),
		"DELETE /services/{uuid}": covered("coolify_service", "v0.1.0"),
		"POST /services/{uuid}/envs":              covered("coolify_environment_variable", "v0.1.0"),
		"PATCH /services/{uuid}/envs":             covered("coolify_environment_variable", "v0.1.0"),
		"DELETE /services/{uuid}/envs/{env_uuid}": covered("coolify_environment_variable", "v0.1.0"),
		"GET /services/{uuid}/envs":               planned(1, "Service env var listing; data source completeness"),
		"GET /services/{uuid}/scheduled-tasks":                        planned(2, "Service scheduled tasks CRUD"),
		"POST /services/{uuid}/scheduled-tasks":                       planned(2, "Service scheduled tasks CRUD"),
		"PATCH /services/{uuid}/scheduled-tasks/{task_uuid}":          planned(2, "Service scheduled tasks CRUD"),
		"DELETE /services/{uuid}/scheduled-tasks/{task_uuid}":         planned(2, "Service scheduled tasks CRUD"),
		"GET /services/{uuid}/scheduled-tasks/{task_uuid}/executions": planned(3, "Read-only task execution history"),
		"GET /services/{uuid}/storages":                   planned(2, "Service persistent storage CRUD"),
		"POST /services/{uuid}/storages":                  planned(2, "Service persistent storage CRUD"),
		"PATCH /services/{uuid}/storages":                 planned(2, "Service persistent storage CRUD"),
		"DELETE /services/{uuid}/storages/{storage_uuid}": planned(2, "Service persistent storage CRUD"),
		"PATCH /services/{uuid}/envs/bulk":  planned(3, "Bulk env var update; convenience"),
		"GET /services/{uuid}/restart":      skipped("Operational action, not a Terraform resource"),
		"GET /services/{uuid}/start":        skipped("Operational action, not a Terraform resource"),
		"GET /services/{uuid}/stop":         skipped("Operational action, not a Terraform resource"),

		// ── Security Keys ──
		"GET /security/keys":           covered("data.coolify_private_keys", "v0.1.0"),
		"POST /security/keys":          covered("coolify_private_key", "v0.1.0"),
		"PATCH /security/keys":         covered("coolify_private_key", "v0.1.0"),
		"GET /security/keys/{uuid}":    covered("data.coolify_private_key", "v0.1.0"),
		"DELETE /security/keys/{uuid}": covered("coolify_private_key", "v0.1.0"),

		// ── Deployments ──
		"GET /deployments/{uuid}":              covered("coolify_deployment", "v0.1.0"),
		"GET /deployments":                     planned(1, "Deployment list data source"),
		"GET /deployments/applications/{uuid}": planned(1, "Application deployment history"),
		"POST /deployments/{uuid}/cancel":      planned(2, "Cancel running deployment"),
		"GET /deploy":                          skipped("Generic deploy trigger; use coolify_deployment resource"),

		// ── Teams ──
		"GET /teams/{id}":            covered("data.coolify_team", "v0.1.0"),
		"GET /teams":                 planned(2, "Teams list data source"),
		"GET /teams/{id}/members":    planned(2, "Team members data source"),
		"GET /teams/current":         planned(2, "Current team data source"),
		"GET /teams/current/members": planned(2, "Current team members data source"),

		// ── Cloud Tokens ──
		"GET /cloud-tokens":                  planned(3, "Cloud provider token CRUD"),
		"POST /cloud-tokens":                 planned(3, "Cloud provider token CRUD"),
		"GET /cloud-tokens/{uuid}":           planned(3, "Cloud provider token CRUD"),
		"PATCH /cloud-tokens/{uuid}":         planned(3, "Cloud provider token CRUD"),
		"DELETE /cloud-tokens/{uuid}":        planned(3, "Cloud provider token CRUD"),
		"POST /cloud-tokens/{uuid}/validate": planned(3, "Cloud provider token validation"),

		// ── GitHub Apps ──
		"GET /github-apps":                planned(3, "GitHub App integration CRUD"),
		"POST /github-apps":               planned(3, "GitHub App integration CRUD"),
		"PATCH /github-apps/{github_app_id}":  planned(3, "GitHub App integration CRUD"),
		"DELETE /github-apps/{github_app_id}": planned(3, "GitHub App integration CRUD"),
		"GET /github-apps/{github_app_id}/repositories":                         planned(3, "Read-only GitHub repository lookup"),
		"GET /github-apps/{github_app_id}/repositories/{owner}/{repo}/branches": planned(3, "Read-only GitHub branch lookup"),

		// ── Hetzner ──
		"GET /hetzner/images":       planned(3, "Hetzner cloud metadata lookup"),
		"GET /hetzner/locations":    planned(3, "Hetzner cloud metadata lookup"),
		"GET /hetzner/server-types": planned(3, "Hetzner cloud metadata lookup"),
		"GET /hetzner/ssh-keys":     planned(3, "Hetzner cloud metadata lookup"),

		// ── Operational / Meta ──
		"GET /version":   covered("data.coolify_version", "v0.1.0"),
		"GET /resources": planned(2, "Global resource listing data source"),
		"GET /health":    skipped("Operational healthcheck, not a Terraform resource"),
		"GET /enable":    skipped("API lifecycle management, not a Terraform resource"),
		"GET /disable":   skipped("API lifecycle management, not a Terraform resource"),
	}
}

// TestSpecCoverage_Completeness verifies that every endpoint in the OpenAPI
// spec is tracked in coveredEndpoints(). Fails if the spec has new endpoints
// not yet classified, or if the registry has stale entries removed from the spec.
func TestSpecCoverage_Completeness(t *testing.T) {
	t.Parallel()

	doc, err := LoadSpec("coolify-v4")
	if err != nil {
		t.Fatalf("loading spec: %v", err)
	}

	model, errs := (*doc).BuildV3Model()
	if errs != nil {
		t.Fatalf("building model: %v", errs)
	}

	coverage := coveredEndpoints()
	specOps := extractSpecOperations(model.Model)

	var missing []string
	for _, op := range specOps {
		if _, ok := coverage[op]; !ok {
			missing = append(missing, op)
		}
	}
	if len(missing) > 0 {
		sort.Strings(missing)
		t.Errorf("spec has %d endpoints not tracked in coveredEndpoints():\n  %s",
			len(missing), strings.Join(missing, "\n  "))
	}

	specSet := make(map[string]bool, len(specOps))
	for _, op := range specOps {
		specSet[op] = true
	}
	var stale []string
	for op := range coverage {
		if !specSet[op] {
			stale = append(stale, op)
		}
	}
	if len(stale) > 0 {
		sort.Strings(stale)
		t.Errorf("coveredEndpoints() has %d entries not in the spec (stale?):\n  %s",
			len(stale), strings.Join(stale, "\n  "))
	}
}

// TestSpecCoverage_Report prints a coverage summary. Run with -v to see it.
func TestSpecCoverage_Report(t *testing.T) {
	t.Parallel()

	coverage := coveredEndpoints()
	var coveredN, plannedN, skippedN int
	for _, s := range coverage {
		switch s.category {
		case "covered":
			coveredN++
		case "planned":
			plannedN++
		case "skipped":
			skippedN++
		}
	}

	total := coveredN + plannedN + skippedN
	pct := float64(coveredN) / float64(total) * 100

	t.Logf("\n=== API Coverage Report ===")
	t.Logf("Total endpoints: %d", total)
	t.Logf("Covered:         %d (%.1f%%)", coveredN, pct)
	t.Logf("Planned:         %d", plannedN)
	t.Logf("Skipped:         %d", skippedN)
}

// TestSpecCoverage_GenerateDoc generates API_COVERAGE.md at the repo root.
// Run: go test ./internal/spectest/ -run TestSpecCoverage_GenerateDoc -v
func TestSpecCoverage_GenerateDoc(t *testing.T) {
	if os.Getenv("GENERATE_COVERAGE_DOC") == "" {
		t.Skip("set GENERATE_COVERAGE_DOC=1 to regenerate API_COVERAGE.md")
	}

	coverage := coveredEndpoints()

	type entry struct {
		endpoint string
		status   coverageStatus
	}

	var coveredList, plannedList, skippedList []entry
	for ep, s := range coverage {
		e := entry{endpoint: ep, status: s}
		switch s.category {
		case "covered":
			coveredList = append(coveredList, e)
		case "planned":
			plannedList = append(plannedList, e)
		case "skipped":
			skippedList = append(skippedList, e)
		}
	}

	sort.Slice(coveredList, func(i, j int) bool {
		return coveredList[i].endpoint < coveredList[j].endpoint
	})
	sort.Slice(plannedList, func(i, j int) bool {
		if plannedList[i].status.priority != plannedList[j].status.priority {
			return plannedList[i].status.priority < plannedList[j].status.priority
		}
		return plannedList[i].endpoint < plannedList[j].endpoint
	})
	sort.Slice(skippedList, func(i, j int) bool {
		return skippedList[i].endpoint < skippedList[j].endpoint
	})

	total := len(coveredList) + len(plannedList) + len(skippedList)
	pct := float64(len(coveredList)) / float64(total) * 100

	var b strings.Builder
	b.WriteString("# API Coverage\n\n")
	b.WriteString("<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->\n")
	b.WriteString("<!-- Run: make api-coverage -->\n\n")
	b.WriteString(fmt.Sprintf("**Spec**: Coolify v4 (pinned in `testdata/specs/coolify-v4.json`)  \n"))
	b.WriteString(fmt.Sprintf("**Coverage**: %d / %d endpoints (%.1f%%)  \n", len(coveredList), total, pct))
	b.WriteString(fmt.Sprintf("**Planned**: %d | **Skipped**: %d\n", len(plannedList), len(skippedList)))

	// Covered
	b.WriteString("\n## Covered\n\n")
	b.WriteString("| Endpoint | Terraform Resource / Data Source | Since |\n")
	b.WriteString("|----------|----------------------------------|-------|\n")
	for _, e := range coveredList {
		b.WriteString(fmt.Sprintf("| `%s` | `%s` | %s |\n", e.endpoint, e.status.resource, e.status.since))
	}

	// Planned
	b.WriteString("\n## Planned\n\n")
	b.WriteString("Ordered by priority (1 = most needed by users).\n\n")
	b.WriteString("| Priority | Endpoint | Notes |\n")
	b.WriteString("|----------|----------|-------|\n")
	for _, e := range plannedList {
		b.WriteString(fmt.Sprintf("| %d | `%s` | %s |\n", e.status.priority, e.endpoint, e.status.notes))
	}

	// Skipped
	b.WriteString("\n## Intentionally Skipped\n\n")
	b.WriteString("These endpoints are not appropriate for Terraform resource management.\n\n")
	b.WriteString("| Endpoint | Reason |\n")
	b.WriteString("|----------|--------|\n")
	for _, e := range skippedList {
		b.WriteString(fmt.Sprintf("| `%s` | %s |\n", e.endpoint, e.status.resource))
	}

	b.WriteString("\n## New in Spec (Unclassified)\n\n")
	b.WriteString("_None. All spec endpoints are classified._\n\n")
	b.WriteString("This section appears when the pinned spec is updated with new endpoints\n")
	b.WriteString("that haven't been added to the coverage registry yet. The\n")
	b.WriteString("`TestSpecCoverage_Completeness` test also fails in this case.\n")

	outPath := filepath.Join(testdataDir(), "..", "API_COVERAGE.md")
	if err := os.WriteFile(outPath, []byte(b.String()), 0644); err != nil {
		t.Fatalf("writing API_COVERAGE.md: %v", err)
	}
	t.Logf("Generated %s (%d bytes)", outPath, len(b.String()))
}

// extractSpecOperations returns all "METHOD /path" strings from the spec.
func extractSpecOperations(model v3high.Document) []string {
	var ops []string
	if model.Paths == nil {
		return ops
	}
	for pair := model.Paths.PathItems.Oldest(); pair != nil; pair = pair.Next() {
		path := pair.Key
		item := pair.Value

		for method, op := range map[string]*v3high.Operation{
			"GET":    item.Get,
			"POST":   item.Post,
			"PUT":    item.Put,
			"PATCH":  item.Patch,
			"DELETE": item.Delete,
		} {
			if op != nil {
				ops = append(ops, fmt.Sprintf("%s %s", method, path))
			}
		}
	}
	sort.Strings(ops)
	return ops
}
