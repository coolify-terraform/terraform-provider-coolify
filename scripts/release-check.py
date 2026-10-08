#!/usr/bin/env python3
"""Report whether current main is ready to release.

Ready means, on that commit:

- the CI workflow succeeded (unit tests, edge acceptance, scenarios)
- nightly acceptance succeeded for edge, stable (latest), and 4.1.2

A failed acceptance test is a flake only when every failure is HTTP 429
or a `/api/v1/version` deadline. A Monday `ci.yml` schedule run does not
count: it skips Test, acceptance, and scenarios. Pass --rerun-once to
dispatch one fresh stable run and then stop. This command does not merge.

Usage:
    python3 scripts/release-check.py
    python3 scripts/release-check.py --sha <commit>
    python3 scripts/release-check.py --rerun-once
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from pathlib import Path

REPO = "coolify-terraform/terraform-provider-coolify"
TIP_JOBS = {"Acc (tip-edge)"}
STABLE_JOBS = {"Acc (stable-latest)", "Acc (custom-latest)"}
FLOOR_JOBS = {"Acc (floor-4.1.2)"}
SLOTS = (
    ("edge", TIP_JOBS),
    ("stable", STABLE_JOBS),
    ("floor", FLOOR_JOBS),
)
# Matrix job names from ci.yml. A Monday schedule run skips these.
PRODUCT_CI_PREFIXES = ("Test (", "Acceptance Tests", "Scenario Tests")
_PRODUCT_MARKERS = (
    "Difference is shown",
    "ImportStateVerify",
    "attributes not equivalent",
    "TestCheckResourceAttr",
)
_OPEN_RUN = ("queued", "in_progress", "waiting", "pending")


def git_sha(ref: str) -> str:
    return subprocess.check_output(["git", "rev-parse", ref], text=True).strip()


def gh_json(args: list[str]) -> object:
    out = subprocess.check_output(["gh", *args], text=True)
    return json.loads(out) if out.strip() else []


def runs(workflow: str, sha: str) -> list[dict]:
    data = gh_json(
        [
            "run",
            "list",
            "--repo",
            REPO,
            "--commit",
            sha,
            "--workflow",
            workflow,
            "--limit",
            "20",
            "--json",
            "databaseId,conclusion,status,event,headSha,url,createdAt",
        ]
    )
    return list(data) if isinstance(data, list) else []


def jobs_for(run_id: int) -> list[dict]:
    data = gh_json(["run", "view", str(run_id), "--repo", REPO, "--json", "jobs"])
    if isinstance(data, dict):
        return list(data.get("jobs") or [])
    return []


def slot_jobs(all_jobs: list[dict], names: set[str]) -> list[dict]:
    matched = [job for job in all_jobs if job.get("name") in names]
    matched.sort(key=lambda job: job.get("startedAt") or "")
    return matched


def _fail_blocks(log: str) -> list[str]:
    if "--- FAIL:" not in log:
        return []
    return ["--- FAIL:" + part for part in log.split("--- FAIL:")[1:]]


def _block_is_flake(block: str) -> bool:
    """True when this failed test is only Coolify 429 or a version-endpoint deadline."""
    if any(marker in block for marker in _PRODUCT_MARKERS):
        return False
    has_429 = "429" in block or "Too Many Attempts" in block
    has_deadline = "deadline exceeded" in block
    version_deadline = has_deadline and "/api/v1/version" in block
    if has_deadline and not version_deadline:
        return False
    return has_429 or version_deadline


def classify_log(log: str) -> str:
    """Return product, flake, or unknown for a failed acceptance log.

    go test prints `--- FAIL:`. A 429 or `/api/v1/version` deadline is a
    flake only when every failed test is that. Any other failure, including
    a resource deadline, stays product.
    """
    if any(marker in log for marker in _PRODUCT_MARKERS):
        return "product"
    blocks = _fail_blocks(log)
    if blocks:
        if all(_block_is_flake(block) for block in blocks):
            return "flake"
        return "product"
    if "429" in log or "Too Many Attempts" in log:
        return "flake"
    if "deadline exceeded" in log and "/api/v1/version" in log:
        return "flake"
    if "deadline exceeded" in log:
        return "product"
    return "unknown"


def latest_conclusion(jobs: list[dict]) -> str:
    if not jobs:
        return "missing"
    last = jobs[-1]
    conclusion = last.get("conclusion") or ""
    if conclusion == "success":
        return "pass"
    if last.get("status") not in ("completed", "", None) and conclusion in ("", None):
        return "running"
    if conclusion in ("", None) and last.get("status") in ("queued", "in_progress", "waiting", "pending"):
        return "running"
    return "fail"


def product_ci_jobs_ok(jobs: list[dict]) -> bool:
    """True when Test, acceptance, and scenarios each have a successful job."""
    found = {prefix: [] for prefix in PRODUCT_CI_PREFIXES}
    for job in jobs:
        name = job.get("name") or ""
        for prefix in PRODUCT_CI_PREFIXES:
            if name.startswith(prefix):
                found[prefix].append(job)
                break
    for group in found.values():
        if not group or any(job.get("conclusion") != "success" for job in group):
            return False
    return True


def classify_ci(runs_list: list[dict]) -> tuple[str, dict | None]:
    """Return (pass|running|fail|missing, latest product run).

    The newest non-schedule run wins. A Monday schedule success skips
    Test, acceptance, and scenarios, so it is not a product pass.
    """
    product_runs = [run for run in runs_list if run.get("event") != "schedule"]
    if not product_runs:
        return "missing", None
    latest = max(product_runs, key=lambda run: run.get("createdAt") or "")
    if latest.get("status") in _OPEN_RUN:
        return "running", latest
    if latest.get("conclusion") == "success":
        return "pass", latest
    return "fail", latest


def nightly_slot_states(
    runs_list: list[dict], jobs_by_run: dict[int, list[dict]]
) -> dict[str, tuple[str, list[dict]]]:
    """Return slot name to (state, jobs).

    The newest nightly run wins while it is still open. A completed run
    that has a slot's job owns that slot. A failure before any Acc job
    (prepare died) does not fall back to an older matrix. A custom run
    that only includes stable leaves edge and floor to older runs.
    """
    ordered = sorted(runs_list, key=lambda run: run.get("createdAt") or "", reverse=True)
    if not ordered:
        return {name: ("missing", []) for name, _names in SLOTS}
    newest = ordered[0]
    if newest.get("status") in _OPEN_RUN:
        return {name: ("running", []) for name, _names in SLOTS}
    newest_jobs = jobs_by_run.get(int(newest["databaseId"]), [])
    newest_has_acc = any((job.get("name") or "").startswith("Acc (") for job in newest_jobs)
    if newest.get("conclusion") not in ("success", "") and not newest_has_acc:
        return {name: ("fail", []) for name, _names in SLOTS}

    states: dict[str, tuple[str, list[dict]]] = {}
    for name, names in SLOTS:
        state = "missing"
        chosen: list[dict] = []
        for run in ordered:
            if run.get("status") in _OPEN_RUN:
                continue
            matched = slot_jobs(jobs_by_run.get(int(run["databaseId"]), []), names)
            if not matched:
                continue
            chosen = matched
            state = latest_conclusion(matched)
            break
        states[name] = (state, chosen)
    return states


def collect(sha: str) -> tuple[str, dict[str, str], dict[str, list[dict]], list[str]]:
    """Return CI status, nightly state by slot, jobs by slot, and human lines."""
    lines: list[str] = []
    ci_runs = runs("ci.yml", sha)
    ci_status, latest_ci = classify_ci(ci_runs)
    if ci_status == "pass" and latest_ci is not None:
        if not product_ci_jobs_ok(jobs_for(int(latest_ci["databaseId"]))):
            ci_status = "missing"
            lines.append(
                f"CI: {sha[:7]} succeeded without Test, acceptance, and scenarios"
            )
        else:
            lines.append(f"CI: pass on {sha[:7]}")
    elif ci_status == "running":
        lines.append(f"CI: still running on {sha[:7]}")
    elif ci_status == "fail":
        url = (latest_ci or {}).get("url", "")
        lines.append(f"CI: failed on {sha[:7]} ({url})")
    else:
        lines.append(f"CI: not run on {sha[:7]} (a Monday schedule run does not count)")

    nightly = runs("coolify-nightly.yml", sha)
    jobs_by_run = {int(run["databaseId"]): jobs_for(int(run["databaseId"])) for run in nightly}
    slot_states = nightly_slot_states(nightly, jobs_by_run)
    slot_state = {name: state for name, (state, _jobs) in slot_states.items()}
    slotted = {name: jobs for name, (_state, jobs) in slot_states.items()}
    for name, _names in SLOTS:
        lines.append(f"nightly {name}: {slot_state[name]}")
    return ci_status, slot_state, slotted, lines


def failing_run_id(jobs: list[dict]) -> int | None:
    if not jobs:
        return None
    last = jobs[-1]
    if last.get("conclusion") == "success":
        return None
    return last.get("databaseId")


def job_log(job_id: int) -> str:
    try:
        return subprocess.check_output(
            [
                "gh",
                "api",
                "--allow-escape-sequences",
                f"repos/{REPO}/actions/jobs/{job_id}/logs",
            ],
            text=True,
            stderr=subprocess.DEVNULL,
        )
    except subprocess.CalledProcessError:
        return ""


def rerun_stable(sha: str) -> str:
    marker = Path(f"/tmp/release-check-rerun-{sha}")
    if marker.exists():
        return f"stable flake already rerun once ({marker.read_text().strip()})"
    subprocess.check_call(
        [
            "gh",
            "workflow",
            "run",
            "coolify-nightly.yml",
            "--repo",
            REPO,
            "--ref",
            "main",
            "-f",
            "profile=custom",
            "-f",
            "custom_image=latest",
            "-f",
            "run_scenarios=false",
        ]
    )
    marker.write_text("dispatched\n")
    return "dispatched one stable acceptance rerun (profile=custom, image=latest)"


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--sha", default="", help="Commit to check. Default: origin/main")
    parser.add_argument(
        "--rerun-once",
        action="store_true",
        help="If the latest stable failure is only 429 or a version deadline, dispatch one rerun and stop",
    )
    args = parser.parse_args()
    sha = args.sha or git_sha("origin/main")

    try:
        ci_status, slot_state, slotted, lines = collect(sha)
    except (subprocess.CalledProcessError, OSError) as exc:
        print(f"release-check: {exc}", file=sys.stderr)
        return 1

    flake = False
    product = False
    for name, jobs in slotted.items():
        if slot_state.get(name) != "fail":
            continue
        job_id = failing_run_id(jobs)
        kind = "product"
        if job_id:
            kind = classify_log(job_log(int(job_id)))
        lines.append(f"nightly {name} failure class: {kind}")
        if kind == "flake":
            flake = True
        else:
            product = True

    ready = (
        ci_status == "pass"
        and all(slot_state.get(name) == "pass" for name, _names in SLOTS)
        and not product
    )
    print(f"commit: {sha}")
    for line in lines:
        print(line)
    if ready:
        print("release-check: ready")
        return 0

    if args.rerun_once and flake and not product:
        print(rerun_stable(sha))
        print("release-check: not ready (flake rerun dispatched once)")
        return 2

    print("release-check: not ready")
    if ci_status == "missing":
        print("  gh workflow run ci.yml --repo coolify-terraform/terraform-provider-coolify --ref main")
    if any(slot_state.get(name) == "missing" for name, _names in SLOTS):
        print(
            "  gh workflow run coolify-nightly.yml --repo "
            "coolify-terraform/terraform-provider-coolify --ref main -f profile=all -f run_scenarios=true"
        )
    return 1


if __name__ == "__main__":
    sys.exit(main())
