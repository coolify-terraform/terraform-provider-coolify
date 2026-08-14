package spectest

import (
	"fmt"
	"sort"
	"strings"
)

// coverageStatus tracks a single API endpoint's provider coverage.
type coverageStatus struct {
	category string // "covered", "planned", "skipped"
	resource string // Terraform resource name, or skip kind id when skipped
	since    string // provider version that added support (covered only)
	priority int    // 1=high, 2=medium, 3=low (planned only)
	notes    string // human-readable context
}

// Skip class IDs used by skipped() in coveredEndpoints().
const (
	skipCloneMove     = "clone-move"
	skipRunNow        = "run-now"
	skipRollback      = "rollback"
	skipLogs          = "logs"
	skipNestedService = "nested-service"
	skipControlPlane  = "control-plane"
	skipEnableDisable = "enable-disable"
	skipAlias         = "alias"
	skipDeprecated    = "deprecated"
	skipHetznerExtra  = "hetzner-extra"
	skipFeedback      = "not-infra"
)

type skipKind struct {
	id      string
	title   string
	why     string
	instead string
	order   int
}

// skipKindCatalog is the user-facing skip taxonomy. Every skipped() kind
// must appear here or generateCoverageMarkdown fails closed.
var skipKindCatalog = []skipKind{
	{
		id:    skipCloneMove,
		title: "Clone, migrate, and move",
		why:   "Coolify can copy or relocate an existing app, database, service, or server in one API call. Terraform then either owns two objects or loses the original.",
		instead: "`coolify_application` (and variants), `coolify_database_*`, `coolify_service`, or `coolify_server` " +
			"to create the destination. For a one-time move, use the Coolify UI.",
		order: 1,
	},
	{
		id:    skipRunNow,
		title: "Run now (backup or scheduled task)",
		why:   "These POSTs fire a single backup or task execution. They are not a schedule you can keep in state.",
		instead: "`coolify_storage_backup` or `coolify_database_backup` for the schedule. " +
			"`coolify_scheduled_task` for the task definition. Trigger a run from the Coolify UI when you need one now.",
		order: 2,
	},
	{
		id:      skipRollback,
		title:   "Rollback and prior images",
		why:     "Listing rollback images and posting a rollback is an operational recovery step, not desired state.",
		instead: "The Coolify UI. There is no `coolify_rollback` resource.",
		order:   3,
	},
	{
		id:    skipLogs,
		title: "Log streams",
		why:   "Log endpoints stream runtime output. That is not durable Terraform state.",
		instead: "`data.coolify_application_logs` for application logs. " +
			"Database and service log streams stay in the Coolify UI.",
		order: 4,
	},
	{
		id:    skipNestedService,
		title: "Nested compose service apps and databases",
		why:   "A catalog `coolify_service` owns the whole compose stack. Coolify also exposes each inner app and database as its own API object.",
		instead: "`coolify_service` for the stack. Do not manage inner " +
			"`/services/{uuid}/applications/...` or `/services/{uuid}/databases/...` as separate resources.",
		order: 5,
	},
	{
		id:    skipControlPlane,
		title: "Server control-plane actions",
		why:   "Export, import, claim, transfer, Sentinel push, proxy restart, and one-shot docker cleanup are operator actions, not server settings.",
		instead: "`coolify_server_proxy`, `coolify_server_sentinel`, `coolify_server_docker_cleanup`, " +
			"and `coolify_server_cloudflare_tunnel` for settings. Use the Coolify UI for export, claim, transfer, and run-now cleanup.",
		order: 6,
	},
	{
		id:      skipEnableDisable,
		title:   "Cloudflare tunnel enable and disable POSTs",
		why:     "Coolify also has POST enable/disable routes. The provider writes tunnel settings with PATCH.",
		instead: "`coolify_server_cloudflare_tunnel`.",
		order:   7,
	},
	{
		id:      skipAlias,
		title:   "Team URL aliases",
		why:     "`GET /team` and `GET /team/members` are aliases of `/teams/current` and `/teams/current/members`.",
		instead: "`data.coolify_team` and `data.coolify_team_members`.",
		order:   8,
	},
	{
		id:      skipDeprecated,
		title:   "Deprecated docker-compose create",
		why:     "`POST /applications/dockercompose` creates a Service, not an Application. Coolify kept the path as a deprecated alias.",
		instead: "`coolify_service` (`POST /services`).",
		order:   9,
	},
	{
		id:      skipHetznerExtra,
		title:   "Hetzner firewalls and networks lists",
		why:     "Coolify can list Hetzner firewalls and networks when provisioning. Those lists are not required to manage `coolify_server_hetzner`.",
		instead: "`coolify_server_hetzner` for the server. Manage Hetzner firewalls and networks outside this provider if you need them.",
		order:   10,
	},
	{
		id:      skipFeedback,
		title:   "Product feedback",
		why:     "`POST /feedback` is a Coolify product endpoint, not infrastructure.",
		instead: "Nothing in Terraform. Send feedback through Coolify.",
		order:   11,
	},
}

func skipKindByID() map[string]skipKind {
	out := make(map[string]skipKind, len(skipKindCatalog))
	for _, k := range skipKindCatalog {
		out[k.id] = k
	}
	return out
}

type coverageEntry struct {
	endpoint string
	status   coverageStatus
}

func generateCoverageMarkdown(coverage map[string]coverageStatus) (string, error) {
	kinds := skipKindByID()

	var coveredList, plannedList, skippedList []coverageEntry
	for ep, s := range coverage {
		e := coverageEntry{endpoint: ep, status: s}
		switch s.category {
		case "covered":
			coveredList = append(coveredList, e)
		case "planned":
			plannedList = append(plannedList, e)
		case "skipped":
			if _, ok := kinds[s.resource]; !ok {
				return "", fmt.Errorf("skipped route %s has unknown kind %q", ep, s.resource)
			}
			skippedList = append(skippedList, e)
		default:
			return "", fmt.Errorf("route %s has unknown category %q", ep, s.category)
		}
	}

	sort.Slice(coveredList, func(i, j int) bool { return coveredList[i].endpoint < coveredList[j].endpoint })
	sort.Slice(plannedList, func(i, j int) bool {
		if plannedList[i].status.priority != plannedList[j].status.priority {
			return plannedList[i].status.priority < plannedList[j].status.priority
		}
		return plannedList[i].endpoint < plannedList[j].endpoint
	})
	sort.Slice(skippedList, func(i, j int) bool { return skippedList[i].endpoint < skippedList[j].endpoint })

	total := len(coveredList) + len(plannedList) + len(skippedList)
	pct := float64(len(coveredList)) / float64(total) * 100

	var b strings.Builder
	b.WriteString("# API Coverage\n\n")
	b.WriteString("<!-- Auto-generated from internal/spectest/coverage_test.go. Do not edit manually. -->\n")
	b.WriteString("<!-- Run: make api-coverage -->\n\n")

	b.WriteString("This page answers: **which Coolify HTTP routes does the provider wrap, and what do I use when it does not?**\n\n")
	b.WriteString("It is a **route inventory** against Coolify source (`testdata/contracts/coolify-v4.json`). ")
	b.WriteString("It is not a field catalog, and it does not say which Coolify **version** you need.\n\n")
	b.WriteString("- **Which Coolify version for each resource and field:** ")
	b.WriteString("[Coolify Version Support](docs/guides/coolify-version-support.md)\n")
	b.WriteString("- **Resource and attribute docs:** [docs/](docs/) (also on the Terraform Registry)\n")
	b.WriteString("- **Field-level gaps** (numeric FKs, UI-only columns on an existing GET) live in ")
	b.WriteString("`internal/spectest/contract_skips.go`, not in this route list.\n\n")

	fmt.Fprintf(&b, "**Coverage**: %d covered / %d registry entries (%.1f%%)  \n", len(coveredList), total, pct)
	fmt.Fprintf(&b, "**Planned**: %d | **Skipped**: %d  \n", len(plannedList), len(skippedList))
	fmt.Fprintf(&b, "**Registry size**: %d (contract routes + allowlisted extras)\n", total)

	writeSkipSections(&b, skippedList)
	writeCoveredByResource(&b, coveredList)

	if len(plannedList) > 0 {
		b.WriteString("\n## Planned\n\n")
		b.WriteString("Ordered by priority (1 = most needed by users).\n\n")
		b.WriteString("| Priority | Endpoint | Notes |\n")
		b.WriteString("|----------|----------|-------|\n")
		for _, e := range plannedList {
			fmt.Fprintf(&b, "| %d | `%s` | %s |\n", e.status.priority, e.endpoint, e.status.notes)
		}
	}

	writeAppendix(&b, coveredList, skippedList, plannedList)

	b.WriteString("\n## Unclassified contract routes\n\n")
	b.WriteString("_None. All pin contract routes are classified in `coveredEndpoints()`._\n\n")
	b.WriteString("When `make contract-extract` adds routes, classify them in\n")
	b.WriteString("`internal/spectest/coverage_test.go` or `TestSpecCoverage_Completeness` fails.\n")

	return b.String(), nil
}

func writeSkipSections(b *strings.Builder, skipped []coverageEntry) {
	b.WriteString("\n## What Terraform does not wrap\n\n")
	b.WriteString("Terraform models desired durable state. Coolify also exposes one-shot buttons, ")
	b.WriteString("log streams, URL aliases, and control-plane actions. Those routes are skipped on purpose.\n")

	byKind := make(map[string][]string)
	for _, e := range skipped {
		byKind[e.status.resource] = append(byKind[e.status.resource], e.endpoint)
	}
	for _, eps := range byKind {
		sort.Strings(eps)
	}

	catalog := append([]skipKind(nil), skipKindCatalog...)
	sort.Slice(catalog, func(i, j int) bool { return catalog[i].order < catalog[j].order })

	for _, k := range catalog {
		eps := byKind[k.id]
		if len(eps) == 0 {
			continue
		}
		fmt.Fprintf(b, "\n### %s\n\n", k.title)
		fmt.Fprintf(b, "%s\n\n", k.why)
		fmt.Fprintf(b, "**Use this instead:** %s\n\n", k.instead)
		b.WriteString("| Route |\n")
		b.WriteString("|-------|\n")
		for _, ep := range eps {
			fmt.Fprintf(b, "| `%s` |\n", ep)
		}
	}
}

func writeCoveredByResource(b *strings.Builder, covered []coverageEntry) {
	b.WriteString("\n## Routes by Terraform resource\n\n")
	b.WriteString("A row here means the provider calls that Coolify route. ")
	b.WriteString("`client.*` helpers are used from resources or are not a standalone resource.\n")

	type row struct {
		endpoint, since string
	}
	groups := map[string][]row{}
	var names []string
	seen := map[string]bool{}
	for _, e := range covered {
		name := e.status.resource
		if name == "" {
			name = "(unlabeled)"
		}
		groups[name] = append(groups[name], row{endpoint: e.endpoint, since: e.status.since})
		if !seen[name] {
			names = append(names, name)
			seen[name] = true
		}
	}
	sort.Strings(names)
	for _, name := range names {
		rows := groups[name]
		sort.Slice(rows, func(i, j int) bool { return rows[i].endpoint < rows[j].endpoint })
		fmt.Fprintf(b, "\n### `%s`\n\n", name)
		b.WriteString("| Route | Since |\n")
		b.WriteString("|-------|-------|\n")
		for _, r := range rows {
			fmt.Fprintf(b, "| `%s` | %s |\n", r.endpoint, r.since)
		}
	}
}

func writeAppendix(b *strings.Builder, covered, skipped, planned []coverageEntry) {
	b.WriteString("\n## Appendix: all classified routes\n\n")
	b.WriteString("Completeness tests use this list. Sorted by `METHOD /path`.\n\n")
	b.WriteString("| Route | Status | Resource or skip class |\n")
	b.WriteString("|-------|--------|------------------------|\n")

	all := make([]coverageEntry, 0, len(covered)+len(skipped)+len(planned))
	all = append(all, covered...)
	all = append(all, skipped...)
	all = append(all, planned...)
	sort.Slice(all, func(i, j int) bool { return all[i].endpoint < all[j].endpoint })
	for _, e := range all {
		label := e.status.resource
		if e.status.category == "planned" && e.status.notes != "" {
			label = e.status.notes
		}
		fmt.Fprintf(b, "| `%s` | %s | `%s` |\n", e.endpoint, e.status.category, label)
	}
}
