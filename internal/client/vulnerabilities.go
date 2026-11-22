package client

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
)

// Vulnerability status values accepted by GET /vulnerabilities.
// Order matches TeamVulnerabilities::STATUS_ORDER (worst first).
var VulnerabilityStatuses = []string{
	"update_now", "update_available", "no_fix", "check_needed", "ok", "not_checked",
}

// VulnerabilityFindingStates are the finding states accepted by the
// per-resource vulnerability routes.
var VulnerabilityFindingStates = []string{"fixed", "no_fix", "ignored"}

// VulnerabilitySeverities are the finding severities accepted by the
// per-resource vulnerability routes.
var VulnerabilitySeverities = []string{"critical", "high", "medium", "low", "unknown"}

// VulnerabilityImageCounts is the team-list image summary.
// update is how many scanned images need an update. total is how many
// images the status counts.
type VulnerabilityImageCounts struct {
	Update int64 `json:"update"`
	Total  int64 `json:"total"`
}

// VulnerabilityEOL is the end-of-life distribution on a security status.
// Coolify sends null when the scanned image is not a known end-of-life distro.
type VulnerabilityEOL struct {
	Name    string `json:"name"`
	Version string `json:"version"`
	Fixed   bool   `json:"fixed"`
}

// VulnerabilityAction is one suggested fix on a security status.
type VulnerabilityAction struct {
	Type     string  `json:"type"`
	Sentence string  `json:"sentence"`
	Target   *string `json:"target"`
	Copy     *string `json:"copy"`
	Image    *string `json:"image"`
	Ref      *string `json:"ref"`
}

// VulnerabilitySecurity is the stored security status Coolify shows on
// the Security pages. The per-resource route sends null when nothing
// has been scanned.
type VulnerabilitySecurity struct {
	Status         string                `json:"status"`
	Label          string                `json:"label"`
	Action         *string               `json:"action"`
	CheckReason    *string               `json:"check_reason"`
	ProblemCount   int64                 `json:"problem_count"`
	ExploitedCount int64                 `json:"exploited_count"`
	AnyExploited   bool                  `json:"any_exploited"`
	CriticalCount  int64                 `json:"critical_count"`
	HighCount      int64                 `json:"high_count"`
	MediumCount    int64                 `json:"medium_count"`
	EOL            *VulnerabilityEOL     `json:"eol"`
	CheckedAt      *string               `json:"checked_at"`
	Actions        []VulnerabilityAction `json:"actions"`
}

// TeamVulnerability is one row of GET /vulnerabilities.
type TeamVulnerability struct {
	UUID            string                   `json:"uuid"`
	Name            string                   `json:"name"`
	Type            string                   `json:"type"`
	ProjectUUID     *string                  `json:"project_uuid"`
	EnvironmentUUID *string                  `json:"environment_uuid"`
	ServerUUID      *string                  `json:"server_uuid"`
	Images          VulnerabilityImageCounts `json:"images"`
	Security        VulnerabilitySecurity    `json:"security"`
}

// VulnerabilityIgnore is a team or resource ignore on one finding.
type VulnerabilityIgnore struct {
	Reason    *string `json:"reason"`
	ExpiresAt *string `json:"expires_at"`
	TeamWide  bool    `json:"team_wide"`
}

// VulnerabilityFinding is one vulnerability on a resource scan.
type VulnerabilityFinding struct {
	ID          string               `json:"id"`
	DisplayID   string               `json:"display_id"`
	Aliases     []string             `json:"aliases"`
	Title       string               `json:"title"`
	Severity    string               `json:"severity"`
	Package     string               `json:"package"`
	Version     string               `json:"version"`
	PackageType string               `json:"package_type"`
	FixedIn     *string              `json:"fixed_in"`
	State       string               `json:"state"`
	Ignore      *VulnerabilityIgnore `json:"ignore"`
	KEV         bool                 `json:"kev"`
	EPSS        *float64             `json:"epss"`
	Risk        *float64             `json:"risk"`
	CVSSScore   *float64             `json:"cvss_score"`
	AdvisoryURL *string              `json:"advisory_url"`
	ImageRef    string               `json:"image_ref"`
	OwnerUUID   string               `json:"owner_uuid"`
}

// VulnerabilityScanImage is one scanned image on a resource.
type VulnerabilityScanImage struct {
	ScanUUID       string  `json:"scan_uuid"`
	OwnerUUID      string  `json:"owner_uuid"`
	ImageRef       string  `json:"image_ref"`
	ScanStatus     *string `json:"scan_status"`
	ErrorCode      *string `json:"error_code"`
	ScannedAt      *string `json:"scanned_at"`
	Truncated      bool    `json:"truncated"`
	SecurityStatus string  `json:"security_status"`
}

// VulnerabilityPagination is the page Coolify applied after filtering.
type VulnerabilityPagination struct {
	Page    int64 `json:"page"`
	PerPage int64 `json:"per_page"`
	Total   int64 `json:"total"`
}

// ResourceVulnerabilities is GET /{applications|services|databases}/{uuid}/vulnerabilities.
type ResourceVulnerabilities struct {
	UUID       string                   `json:"uuid"`
	Type       string                   `json:"type"`
	OffReason  *string                  `json:"off_reason"`
	Security   *VulnerabilitySecurity   `json:"security"`
	Images     []VulnerabilityScanImage `json:"images"`
	Findings   []VulnerabilityFinding   `json:"findings"`
	Pagination VulnerabilityPagination  `json:"pagination"`
}

// ResourceVulnerabilityQuery is the per-resource filter query.
// Zero values are omitted. Coolify then uses all states, all severities,
// page 1, and 50 findings per page.
type ResourceVulnerabilityQuery struct {
	State    []string
	Severity []string
	Image    string
	Search   string
	Page     *int64
	PerPage  *int64
}

func (c *Client) requireCoolify47(action string) error {
	if c != nil && !c.SupportsCoolify47() {
		return fmt.Errorf("%s requires Coolify >= v4.7.0", action)
	}
	return nil
}

// ListVulnerabilities calls GET /api/v1/vulnerabilities.
// status empty returns every stored status. Unknown status values error
// before the request.
func (c *Client) ListVulnerabilities(ctx context.Context, status []string) ([]TeamVulnerability, error) {
	if err := c.requireCoolify47("listing vulnerabilities"); err != nil {
		return nil, err
	}
	path := "/api/v1/vulnerabilities"
	q, err := csvQuery(url.Values{}, "status", status, VulnerabilityStatuses)
	if err != nil {
		return nil, fmt.Errorf("listing vulnerabilities: %w", err)
	}
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	var wrap struct {
		Resources []TeamVulnerability `json:"resources"`
	}
	if err := c.do(ctx, http.MethodGet, path, nil, &wrap); err != nil {
		return nil, fmt.Errorf("listing vulnerabilities: %w", err)
	}
	if wrap.Resources == nil {
		wrap.Resources = []TeamVulnerability{}
	}
	for i := range wrap.Resources {
		if wrap.Resources[i].UUID == "" {
			return nil, fmt.Errorf("listing vulnerabilities: resource at index %d has an empty uuid", i)
		}
	}
	return wrap.Resources, nil
}

// GetResourceVulnerabilities calls GET /api/v1/{collection}/{uuid}/vulnerabilities.
// collection is applications, services, or databases.
func (c *Client) GetResourceVulnerabilities(ctx context.Context, collection, uuid string, query ResourceVulnerabilityQuery) (*ResourceVulnerabilities, error) {
	if err := c.requireCoolify47("reading vulnerabilities"); err != nil {
		return nil, err
	}
	path, err := resourceVulnerabilityPath(collection, uuid, query)
	if err != nil {
		return nil, err
	}
	var out ResourceVulnerabilities
	if err := c.do(ctx, http.MethodGet, path, nil, &out); err != nil {
		return nil, fmt.Errorf("reading vulnerabilities for %s %s: %w", collection, uuid, err)
	}
	if out.UUID == "" {
		return nil, fmt.Errorf("reading vulnerabilities for %s %s: response uuid is empty", collection, uuid)
	}
	return &out, nil
}

func resourceVulnerabilityPath(collection, uuid string, query ResourceVulnerabilityQuery) (string, error) {
	switch collection {
	case "applications", "services", "databases":
	default:
		return "", fmt.Errorf("reading vulnerabilities: unsupported collection %q", collection)
	}
	if uuid == "" || strings.ContainsAny(uuid, "/?#%") {
		return "", fmt.Errorf("reading vulnerabilities: invalid uuid %q", uuid)
	}
	q := url.Values{}
	var err error
	if q, err = csvQuery(q, "state", query.State, VulnerabilityFindingStates); err != nil {
		return "", fmt.Errorf("reading vulnerabilities: %w", err)
	}
	if q, err = csvQuery(q, "severity", query.Severity, VulnerabilitySeverities); err != nil {
		return "", fmt.Errorf("reading vulnerabilities: %w", err)
	}
	if query.Image != "" {
		if len(query.Image) > 1000 {
			return "", fmt.Errorf("reading vulnerabilities: image filter is longer than 1000 characters")
		}
		q.Set("image", query.Image)
	}
	if query.Search != "" {
		if len(query.Search) > 200 {
			return "", fmt.Errorf("reading vulnerabilities: search filter is longer than 200 characters")
		}
		q.Set("search", query.Search)
	}
	if query.Page != nil {
		if *query.Page < 1 {
			return "", fmt.Errorf("reading vulnerabilities: page must be at least 1")
		}
		q.Set("page", fmt.Sprintf("%d", *query.Page))
	}
	if query.PerPage != nil {
		if *query.PerPage < 1 || *query.PerPage > 200 {
			return "", fmt.Errorf("reading vulnerabilities: per_page must be from 1 to 200")
		}
		q.Set("per_page", fmt.Sprintf("%d", *query.PerPage))
	}
	path := "/api/v1/" + collection + "/" + url.PathEscape(uuid) + "/vulnerabilities"
	if enc := q.Encode(); enc != "" {
		path += "?" + enc
	}
	return path, nil
}

func csvQuery(q url.Values, key string, values, allowed []string) (url.Values, error) {
	if len(values) == 0 {
		return q, nil
	}
	allow := make(map[string]struct{}, len(allowed))
	for _, item := range allowed {
		allow[item] = struct{}{}
	}
	for _, item := range values {
		if _, ok := allow[item]; !ok {
			return q, fmt.Errorf("%s %q is not one of %s", key, item, strings.Join(allowed, ", "))
		}
	}
	q.Set(key, strings.Join(values, ","))
	return q, nil
}
