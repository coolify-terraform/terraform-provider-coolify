#!/usr/bin/env python3
"""Open or replace a GitHub issue when scheduled Nightly Acc is red.

A red nightly that only fails in Actions does not notify anyone. GitHub
notifies on new-issue assignment. This script:

  - first red: create an issue and assign the maintainer
  - still red on a later UTC day, or a new failing-job signature:
    close the old issue and open a new assigned one (fresh notification)
    with the consecutive-day count
  - same UTC day and same signature: update the existing issue body
  - same run id: no-op (idempotent if the report step retries)
  - green: close the open issue
  - cancelled with no failure: do nothing (do not open, do not close)

Do not use `gh issue list --search`. List by label through the Issues API.

Environment:
  GH_TOKEN or GITHUB_TOKEN, GITHUB_REPOSITORY
  NIGHTLY_FAILURE_ASSIGNEE (optional GitHub login)
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
from dataclasses import dataclass
from datetime import date, datetime, timezone
from typing import Any, Callable, Optional

ISSUE_LABEL = "nightly-failure"
READY_LABEL = "ready"
LABELS = [ISSUE_LABEL, READY_LABEL]
STATE_MARKER_PREFIX = "<!-- nightly-failure-state:"
STATE_MARKER_RE = re.compile(
    r"<!-- nightly-failure-state:\s*(.*?)\s*-->",
    re.DOTALL,
)
TITLE_PREFIX = "Nightly Acc red:"
IGNORE_JOB_NAMES = frozenset({"Nightly gate", "Prepare matrix"})
GhFn = Callable[[list[str]], str]


@dataclass
class Snapshot:
    today: date
    run_id: str
    head_sha: str
    run_url: str
    prepare: str
    acceptance: str
    scenarios: str
    failed_jobs: list[str]


@dataclass
class Decision:
    action: str  # none | open | update | replace | close
    title: str
    body: str
    consecutive_days: int
    signature: str
    first_failed_on: str
    close_comment: str = ""


def utc_today() -> date:
    return datetime.now(timezone.utc).date()


def parse_date(value: str) -> Optional[date]:
    try:
        return date.fromisoformat((value or "").strip()[:10])
    except ValueError:
        return None


def consecutive_days(first_failed_on: date, today: date) -> int:
    delta = (today - first_failed_on).days
    if delta < 0:
        return 1
    return delta + 1


def is_failure(result: str) -> bool:
    return (result or "").strip().lower() == "failure"


def is_cancelled(result: str) -> bool:
    return (result or "").strip().lower() == "cancelled"


def classify_run(prepare: str, acceptance: str, scenarios: str) -> str:
    """Return failure, cancelled, or success."""
    results = (prepare, acceptance, scenarios)
    if any(is_failure(r) for r in results):
        return "failure"
    if any(is_cancelled(r) for r in results):
        return "cancelled"
    return "success"


def signature_from_jobs(failed_jobs: list[str]) -> str:
    names = sorted({n.strip() for n in failed_jobs if n and n.strip()})
    return ", ".join(names)


def fallback_failed_jobs(prepare: str, acceptance: str, scenarios: str) -> list[str]:
    jobs: list[str] = []
    if is_failure(prepare):
        jobs.append("Prepare matrix")
    if is_failure(acceptance):
        jobs.append("Acceptance matrix")
    if is_failure(scenarios):
        jobs.append("Scenarios (tip-edge)")
    return jobs


def failed_jobs_from_run(jobs: list[dict[str, Any]]) -> list[str]:
    names: list[str] = []
    for job in jobs:
        if (job.get("conclusion") or "").lower() != "failure":
            continue
        name = (job.get("name") or "").strip()
        if not name or name in IGNORE_JOB_NAMES:
            continue
        names.append(name)
    return sorted(set(names))


def decode_state(body: str) -> dict[str, Any]:
    match = STATE_MARKER_RE.search(body or "")
    if not match:
        return {}
    try:
        data = json.loads(match.group(1))
    except json.JSONDecodeError:
        return {}
    return data if isinstance(data, dict) else {}


def encode_state(state: dict[str, Any]) -> str:
    return f"{STATE_MARKER_PREFIX} {json.dumps(state, separators=(',', ':'))} -->"


def format_job_summary(jobs: list[str]) -> str:
    if not jobs:
        return "unknown jobs"
    if len(jobs) <= 2:
        return ", ".join(jobs)
    return f"{jobs[0]} + {len(jobs) - 1} more"


def issue_title(failed_jobs: list[str], days: int) -> str:
    summary = format_job_summary(failed_jobs)
    if days <= 1:
        return f"{TITLE_PREFIX} {summary}"
    return f"{TITLE_PREFIX} {summary} ({days} consecutive days)"


def issue_body(snapshot: Snapshot, *, days: int, first_failed_on: str, previous_issue: Optional[int]) -> str:
    jobs = snapshot.failed_jobs or ["unknown jobs"]
    job_lines = "\n".join(f"- `{name}`" for name in jobs)
    sha = snapshot.head_sha
    short = sha[:12] if sha else "unknown"
    prev = ""
    if previous_issue:
        prev = f"\nPrevious issue: #{previous_issue}\n"
    state = encode_state(
        {
            "first_failed_on": first_failed_on,
            "last_failed_on": snapshot.today.isoformat(),
            "consecutive_days": days,
            "signature": signature_from_jobs(snapshot.failed_jobs),
            "run_id": snapshot.run_id,
            "sha": sha,
            "previous_issue": previous_issue,
        }
    )
    return f"""{state}

Coolify Nightly Acc is red.

| Field | Value |
| --- | --- |
| Consecutive days | {days} |
| First failed (UTC) | {first_failed_on} |
| Latest run | [{snapshot.run_id}]({snapshot.run_url}) |
| SHA | `{short}` |
| Prepare | `{snapshot.prepare}` |
| Acceptance | `{snapshot.acceptance}` |
| Scenarios | `{snapshot.scenarios}` |

Failed jobs:

{job_lines}
{prev}
Treat this as a provider write/import mismatch against current Coolify until a flake is proven. Do not wait for another scheduled run before investigating.
"""


def decide(
    snapshot: Snapshot,
    *,
    existing: Optional[dict[str, Any]] = None,
) -> Decision:
    status = classify_run(snapshot.prepare, snapshot.acceptance, snapshot.scenarios)
    jobs = snapshot.failed_jobs or fallback_failed_jobs(
        snapshot.prepare, snapshot.acceptance, snapshot.scenarios
    )
    snapshot.failed_jobs = jobs
    sig = signature_from_jobs(jobs)
    today = snapshot.today

    if status == "cancelled":
        return Decision(
            action="none",
            title="",
            body="",
            consecutive_days=0,
            signature=sig,
            first_failed_on="",
        )

    if status == "success":
        if existing is None:
            return Decision(
                action="none",
                title="",
                body="",
                consecutive_days=0,
                signature="",
                first_failed_on="",
            )
        days = int(decode_state(existing.get("body") or "").get("consecutive_days") or 0)
        return Decision(
            action="close",
            title="",
            body="",
            consecutive_days=days,
            signature="",
            first_failed_on="",
            close_comment=(
                f"Nightly Acc is green on [{snapshot.run_id}]({snapshot.run_url}) "
                f"(`{(snapshot.head_sha or '')[:12] or 'unknown'}`)."
                + (
                    f" It was red for {days} consecutive day"
                    f"{'s' if days != 1 else ''}."
                    if days
                    else ""
                )
            ),
        )

    prev = decode_state((existing or {}).get("body") or "")
    first = parse_date(str(prev.get("first_failed_on") or "")) or today
    days = consecutive_days(first, today)
    previous_issue = None
    if existing is not None:
        try:
            previous_issue = int(existing["number"])
        except (KeyError, TypeError, ValueError):
            previous_issue = None
    prev_sig = str(prev.get("signature") or "")
    prev_run = str(prev.get("run_id") or "")
    last_failed = parse_date(str(prev.get("last_failed_on") or ""))
    same_day = last_failed == today
    same_sig = prev_sig == sig
    # Link the prior issue only when this decision will replace it.
    link_previous = None
    if existing is not None and not (same_day and same_sig):
        link_previous = previous_issue
    title = issue_title(jobs, days)
    body = issue_body(
        snapshot,
        days=days,
        first_failed_on=first.isoformat(),
        previous_issue=link_previous,
    )

    if existing is None:
        return Decision(
            action="open",
            title=title,
            body=body,
            consecutive_days=days,
            signature=sig,
            first_failed_on=first.isoformat(),
        )

    if prev_run and prev_run == snapshot.run_id:
        return Decision(
            action="none",
            title=title,
            body=body,
            consecutive_days=days,
            signature=sig,
            first_failed_on=first.isoformat(),
        )
    if same_day and same_sig:
        return Decision(
            action="update",
            title=title,
            body=body,
            consecutive_days=days,
            signature=sig,
            first_failed_on=first.isoformat(),
        )
    return Decision(
        action="replace",
        title=title,
        body=body,
        consecutive_days=days,
        signature=sig,
        first_failed_on=first.isoformat(),
    )


def run_gh(args: list[str]) -> str:
    proc = subprocess.run(
        ["gh", *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return proc.stdout


def parse_created_issue_number(output: str) -> str:
    match = re.search(r"/issues/(\d+)", (output or "").strip())
    if not match:
        raise ValueError(f"could not parse issue number from {output!r}")
    return match.group(1)


def find_open_issue(gh: GhFn) -> Optional[dict[str, Any]]:
    raw = gh(
        [
            "issue",
            "list",
            "--label",
            ISSUE_LABEL,
            "--state",
            "open",
            "--json",
            "number,title,body,url",
            "--limit",
            "20",
        ]
    )
    issues = json.loads(raw or "[]")
    if not isinstance(issues, list) or not issues:
        return None
    for issue in issues:
        if STATE_MARKER_PREFIX in (issue.get("body") or ""):
            return issue
    return issues[0]


def fetch_failed_jobs(gh: GhFn, run_id: str) -> list[str]:
    if not run_id:
        return []
    try:
        raw = gh(["run", "view", run_id, "--json", "jobs"])
    except (subprocess.CalledProcessError, OSError):
        return []
    data = json.loads(raw or "{}")
    jobs = data.get("jobs") if isinstance(data, dict) else None
    if not isinstance(jobs, list):
        return []
    return failed_jobs_from_run(jobs)


def ensure_labels(gh: GhFn) -> None:
    for name, desc, color in (
        (ISSUE_LABEL, "Scheduled Nightly Acc is red", "b60205"),
        (READY_LABEL, "Ready to implement", "0e8a16"),
    ):
        try:
            gh(
                [
                    "label",
                    "create",
                    name,
                    "--description",
                    desc,
                    "--color",
                    color,
                    "--force",
                ]
            )
        except (subprocess.CalledProcessError, OSError):
            continue


def apply_issue_labels(gh: GhFn, number: str) -> None:
    for lab in LABELS:
        try:
            gh(["issue", "edit", number, "--add-label", lab])
        except (subprocess.CalledProcessError, OSError):
            continue


def create_issue(gh: GhFn, decision: Decision, assignee: str) -> str:
    cmd = [
        "issue",
        "create",
        "--title",
        decision.title,
        "--body",
        decision.body,
    ]
    for lab in LABELS:
        cmd.extend(["--label", lab])
    if assignee:
        cmd.extend(["--assignee", assignee])
    created = gh(cmd)
    return parse_created_issue_number(created)


def apply_decision(
    decision: Decision,
    *,
    existing: Optional[dict[str, Any]],
    gh: GhFn,
    assignee: str,
) -> int:
    if decision.action == "none":
        print("No nightly-failure issue action.")
        return 0

    ensure_labels(gh)

    if decision.action == "close":
        if existing is None:
            return 0
        number = str(existing["number"])
        print(f"Closing issue #{number}: nightly green.")
        gh(
            [
                "issue",
                "close",
                number,
                "--reason",
                "completed",
                "--comment",
                decision.close_comment or "Nightly Acc is green.",
            ]
        )
        return 0

    if decision.action == "open":
        print("Creating issue:", decision.title)
        created = create_issue(gh, decision, assignee)
        apply_issue_labels(gh, created)
        print(f"Opened #{created}")
        return 0

    if existing is None:
        print("Creating issue:", decision.title)
        created = create_issue(gh, decision, assignee)
        apply_issue_labels(gh, created)
        print(f"Opened #{created}")
        return 0

    number = str(existing["number"])
    if decision.action == "update":
        print(f"Updating issue #{number}: {decision.title}")
        gh(["issue", "edit", number, "--title", decision.title, "--body", decision.body])
        apply_issue_labels(gh, number)
        return 0

    print(f"Replacing issue #{number}: {existing.get('title')} -> {decision.title}")
    created = create_issue(gh, decision, assignee)
    apply_issue_labels(gh, created)
    print(f"Opened #{created}")
    gh(
        [
            "issue",
            "close",
            number,
            "--reason",
            "completed",
            "--comment",
            (
                f"Superseded by #{created}. Still red after "
                f"{decision.consecutive_days} consecutive day"
                f"{'s' if decision.consecutive_days != 1 else ''}."
            ),
        ]
    )
    return 0


def build_snapshot(args: argparse.Namespace, gh: GhFn) -> Snapshot:
    jobs = list(args.failed_job or [])
    if not jobs:
        jobs = fetch_failed_jobs(gh, args.run_id)
    if not jobs:
        jobs = fallback_failed_jobs(args.prepare, args.acceptance, args.scenarios)
    return Snapshot(
        today=parse_date(args.today) or utc_today(),
        run_id=args.run_id,
        head_sha=args.head_sha,
        run_url=args.run_url,
        prepare=args.prepare,
        acceptance=args.acceptance,
        scenarios=args.scenarios,
        failed_jobs=jobs,
    )


def main(argv: Optional[list[str]] = None, *, gh: GhFn = run_gh) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--run-id", default=os.environ.get("GITHUB_RUN_ID", ""))
    parser.add_argument("--head-sha", default=os.environ.get("GITHUB_SHA", ""))
    parser.add_argument(
        "--run-url",
        default="",
        help="HTML URL of the workflow run",
    )
    parser.add_argument("--prepare", default="success")
    parser.add_argument("--acceptance", default="success")
    parser.add_argument("--scenarios", default="success")
    parser.add_argument(
        "--failed-job",
        action="append",
        default=None,
        help="Failed job name (repeatable). Default: query the run via gh.",
    )
    parser.add_argument("--today", default="", help="UTC date YYYY-MM-DD (tests)")
    parser.add_argument(
        "--assignee",
        default=os.environ.get("NIGHTLY_FAILURE_ASSIGNEE", ""),
        help="GitHub login to assign. Empty skips assignment.",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Create/update/replace/close the GitHub issue",
    )
    args = parser.parse_args(argv)
    if not args.run_url and args.run_id:
        repo = os.environ.get("GITHUB_REPOSITORY", "")
        if repo:
            args.run_url = f"https://github.com/{repo}/actions/runs/{args.run_id}"

    snapshot = build_snapshot(args, gh)
    existing = None
    if args.apply:
        existing = find_open_issue(gh)
    decision = decide(snapshot, existing=existing)
    print(
        json.dumps(
            {
                "action": decision.action,
                "title": decision.title,
                "consecutive_days": decision.consecutive_days,
                "signature": decision.signature,
                "failed_jobs": snapshot.failed_jobs,
            }
        )
    )
    if not args.apply:
        return 0
    try:
        return apply_decision(
            decision,
            existing=existing,
            gh=gh,
            assignee=(args.assignee or "").strip(),
        )
    except (subprocess.CalledProcessError, OSError, ValueError) as err:
        print(f"::warning::nightly-failure issue update failed: {err}", file=sys.stderr)
        return 0


if __name__ == "__main__":
    sys.exit(main())
