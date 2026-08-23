#!/usr/bin/env python3
"""Keep a single social-preview upload reminder issue.

The Update Social Preview workflow used to close issues labeled
social-preview and then create a new one. gh issue create did not apply
that label (issues landed as needs-triage only), so the close query
matched nothing and every release opened a duplicate.

This script lists open issues by title and by the social-preview label,
keeps the oldest as canonical, updates its body, and closes extras.
"""

from __future__ import annotations

import argparse
import json
import subprocess
import sys
from typing import Any, Callable

ISSUE_TITLE = "Upload social preview image"
LABEL = "social-preview"
READY_LABEL = "ready"
TRIAGE_LABEL = "needs-triage"

GhFn = Callable[[list[str]], str]


def run_gh(args: list[str]) -> str:
    proc = subprocess.run(
        ["gh", *args],
        check=True,
        capture_output=True,
        text=True,
    )
    return proc.stdout


def parse_issues(raw: str) -> list[dict[str, Any]]:
    data = json.loads(raw or "[]")
    if not isinstance(data, list):
        raise ValueError("expected a JSON array of issues")
    return data


def collect_candidates(
    by_title: list[dict[str, Any]],
    by_label: list[dict[str, Any]],
) -> list[dict[str, Any]]:
    seen: dict[int, dict[str, Any]] = {}
    for issue in by_title + by_label:
        num = int(issue["number"])
        seen[num] = issue
    return sorted(seen.values(), key=lambda i: int(i["number"]))


def plan_upsert(candidates: list[dict[str, Any]]) -> dict[str, Any]:
    ordered = sorted(candidates, key=lambda i: int(i["number"]))
    if not ordered:
        return {"action": "create", "canonical": None, "close": []}
    canonical = int(ordered[0]["number"])
    extras = [int(i["number"]) for i in ordered[1:]]
    return {"action": "update", "canonical": canonical, "close": extras}


def upsert(
    body: str,
    *,
    gh: GhFn = run_gh,
    title: str = ISSUE_TITLE,
    label: str = LABEL,
) -> dict[str, Any]:
    by_title = parse_issues(
        gh(
            [
                "issue",
                "list",
                "--state",
                "open",
                "--search",
                f'is:open in:title "{title}"',
                "--json",
                "number,title",
                "--limit",
                "50",
            ]
        )
    )
    by_title = [i for i in by_title if i.get("title") == title]
    by_label = parse_issues(
        gh(
            [
                "issue",
                "list",
                "--state",
                "open",
                "--label",
                label,
                "--json",
                "number,title",
                "--limit",
                "50",
            ]
        )
    )
    plan = plan_upsert(collect_candidates(by_title, by_label))

    if plan["action"] == "create":
        out = gh(
            [
                "issue",
                "create",
                "--title",
                title,
                "--label",
                f"{label},{READY_LABEL}",
                "--body",
                body,
            ]
        )
        print(f"created {out.strip()}")
        return plan

    canonical = plan["canonical"]
    gh(["issue", "edit", str(canonical), "--body", body])
    gh(
        [
            "issue",
            "edit",
            str(canonical),
            "--add-label",
            f"{label},{READY_LABEL}",
            "--remove-label",
            TRIAGE_LABEL,
        ]
    )
    print(f"updated #{canonical}")
    for num in plan["close"]:
        gh(
            [
                "issue",
                "close",
                str(num),
                "--reason",
                "completed",
                "--comment",
                f"Duplicate of #{canonical}. One social-preview reminder issue only.",
            ]
        )
        print(f"closed #{num} as duplicate of #{canonical}")
    return plan


def main(argv: list[str] | None = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--body-file", required=True, help="Markdown body to write")
    args = parser.parse_args(argv)
    with open(args.body_file, encoding="utf-8") as fh:
        body = fh.read()
    upsert(body)
    return 0


if __name__ == "__main__":
    sys.exit(main())
