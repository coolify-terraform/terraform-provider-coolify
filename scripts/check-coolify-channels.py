#!/usr/bin/env python3
"""Watch Coolify release channels for pre-stable and stable signals.

Coolify does not always use classic RC tags. For v4.2 the early "almost
stable" signal was the nightly channel version (CDN versions.json), with
GitHub prereleases as a later freeze signal.

This script compares:
  - stable:  coolify.v4.version from CDN versions.json
  - nightly: coolify.nightly.version from the same file
  - pin:     version from testdata/contracts/coolify-v4.json
  - prereleases: GitHub releases with prerelease=true (optional input)

It prints a machine-readable decision and can create/update a GitHub
issue when the pin lags nightly/stable/prerelease.

Usage:
  # Decision only (no network if inputs provided)
  python3 scripts/check-coolify-channels.py \\
    --versions-json /tmp/versions.json \\
    --pinned-version v4.1.2 \\
    --prerelease-tags v4.2.0

  # Fetch CDN + local pin, print decision
  python3 scripts/check-coolify-channels.py --fetch

  # Create/update GitHub issue when action is needed
  python3 scripts/check-coolify-channels.py --fetch --apply

Environment (for --apply / --fetch-prereleases):
  GH_TOKEN or GITHUB_TOKEN, GITHUB_REPOSITORY (owner/repo)
"""

from __future__ import annotations

import argparse
import json
import os
import re
import subprocess
import sys
import urllib.error
import urllib.request
from dataclasses import asdict, dataclass, field
from pathlib import Path
from typing import Any, Optional

CDN_VERSIONS_URL = "https://cdn.coollabs.io/coolify/versions.json"
STATE_MARKER_PREFIX = "<!-- coolify-channel-state:"
STATE_MARKER_RE = re.compile(
    r"<!-- coolify-channel-state:\s*(.*?)\s*-->",
    re.DOTALL,
)
ISSUE_LABEL = "coolify-channel"
LABELS = ["coolify-channel", "ready"]


@dataclass
class ChannelSnapshot:
    stable: str
    nightly: str
    pin: str
    prereleases: list[str] = field(default_factory=list)


@dataclass
class Decision:
    action: str  # none | open | update | comment
    reasons: list[str]
    snapshot: ChannelSnapshot
    title: str
    body: str
    pin_behind_nightly: bool
    pin_behind_stable: bool
    pin_behind_prerelease: bool
    nightly_ahead_of_stable: bool


def normalize_version(v: str) -> str:
    """Strip leading v and whitespace."""
    return (v or "").strip().lstrip("vV")


def parse_semver(v: str) -> tuple[int, ...]:
    """Parse a Coolify-ish version into a comparable tuple.

    Supports 4.2.0, v4.2.0, 4.0.0-beta.474 (beta suffix sorts below release).
    """
    raw = normalize_version(v)
    if not raw:
        return (0,)
    # Split pre-release
    main, _, pre = raw.partition("-")
    parts: list[int] = []
    for p in main.split("."):
        if p.isdigit():
            parts.append(int(p))
        else:
            m = re.match(r"(\d+)", p)
            parts.append(int(m.group(1)) if m else 0)
    if not parts:
        return (0,)
    # No pre => treat as final (.999999 sentinel so 4.2.0 > 4.2.0-beta.1)
    if not pre:
        return tuple(parts + [999999])
    pre_nums = [int(x) for x in re.findall(r"\d+", pre)]
    return tuple(parts + [0] + pre_nums)


def version_lt(a: str, b: str) -> bool:
    return parse_semver(a) < parse_semver(b)


def version_eq(a: str, b: str) -> bool:
    return normalize_version(a) == normalize_version(b)


def load_versions_json(data: dict[str, Any]) -> tuple[str, str]:
    coolify = data.get("coolify") or {}
    stable = (coolify.get("v4") or {}).get("version") or ""
    nightly = (coolify.get("nightly") or {}).get("version") or ""
    if not stable or not nightly:
        raise ValueError(
            "versions.json missing coolify.v4.version or coolify.nightly.version"
        )
    return normalize_version(stable), normalize_version(nightly)


def load_pinned_version(contract_path: Path) -> str:
    with contract_path.open() as f:
        contract = json.load(f)
    ver = contract.get("version") or ""
    # contract version is like "v4.2.0"
    return normalize_version(ver)


def newest_prerelease(tags: list[str]) -> Optional[str]:
    if not tags:
        return None
    return max(tags, key=parse_semver)


def encode_state(snap: ChannelSnapshot) -> str:
    payload = {
        "stable": snap.stable,
        "nightly": snap.nightly,
        "pin": snap.pin,
        "prereleases": snap.prereleases,
    }
    return f"{STATE_MARKER_PREFIX} {json.dumps(payload, separators=(',', ':'))} -->"


def decode_state(body: str) -> Optional[dict[str, Any]]:
    m = STATE_MARKER_RE.search(body or "")
    if not m:
        return None
    try:
        return json.loads(m.group(1))
    except json.JSONDecodeError:
        return None


def build_body(snap: ChannelSnapshot, reasons: list[str]) -> str:
    pin_v = snap.pin or "(unknown)"
    stable_v = snap.stable
    nightly_v = snap.nightly
    pre = newest_prerelease(snap.prereleases)
    pre_line = f"`{pre}`" if pre else "_none_"

    extract_nightly = f"make contract-extract VERSION=v{nightly_v}"
    extract_stable = f"make contract-extract VERSION=v{stable_v}"
    extract_pre = f"make contract-extract VERSION={pre if pre and pre.startswith('v') else 'v' + (pre or '')}" if pre else ""

    reasons_md = "\n".join(f"- {r}" for r in reasons) if reasons else "- (no lag)"

    steps = [
        f"1. Extract the next-line contract: `{extract_nightly}`",
        "2. Diff against the pin: `scripts/diff-contracts.sh testdata/contracts/coolify-v4.json testdata/contracts/coolify-v"
        + nightly_v
        + ".json` (or the extract output path)",
        "3. Run `make contract-check` and implement provider changes for new fields/routes",
        "4. Prefer living on a feature branch until GitHub unmarks the prerelease / stable CDN catches up",
        "5. When stable CDN `v4` matches the pin (or pin is intentionally current), close this issue",
    ]
    if pre:
        steps.insert(
            1,
            f"1b. If a prerelease tag exists, also extract it: `{extract_pre}`",
        )

    return f"""{encode_state(snap)}

## Coolify channel watch

Automated daily check of Coolify's stable vs nightly CDN channels and GitHub prereleases.

| Channel | Version |
|---------|---------|
| **Pinned contract** (`testdata/contracts/coolify-v4.json`) | `{pin_v}` |
| **Stable CDN** (`coolify.v4`) | `{stable_v}` |
| **Nightly CDN** (`coolify.nightly`) | `{nightly_v}` |
| **GitHub prerelease (newest)** | {pre_line} |

### Why this fired

{reasons_md}

### What this means

Coolify's early "almost stable" signal is usually the **nightly** channel
version (not a classic `rc.N` tag). For v4.2, nightly moved to 4.2.0 weeks
before the GitHub prerelease tag. Use nightly as the start-support signal;
treat GitHub prerelease as a freeze candidate; promote the pin when stable CDN catches up.

### Next commands

```bash
# Extract next-line contract
{extract_nightly}

# Extract current stable (for dual-pin / compat work)
{extract_stable}
```

{chr(10).join(steps)}

### Sources

- CDN: {CDN_VERSIONS_URL}
- Nightly install CDN: `https://cdn.coollabs.io/coolify-nightly`
- Coolify default branch for tip extract: `v4.x`
"""


def decide(snap: ChannelSnapshot, previous: Optional[dict[str, Any]] = None) -> Decision:
    reasons: list[str] = []
    pin = snap.pin
    stable = snap.stable
    nightly = snap.nightly
    pre = newest_prerelease(snap.prereleases)

    pin_behind_nightly = bool(pin and nightly and version_lt(pin, nightly))
    pin_behind_stable = bool(pin and stable and version_lt(pin, stable))
    pin_behind_prerelease = bool(pin and pre and version_lt(pin, pre))
    nightly_ahead = bool(stable and nightly and version_lt(stable, nightly))

    if pin_behind_stable:
        reasons.append(
            f"Pinned contract `{pin}` is behind **stable** CDN `{stable}` "
            "(users on stable may already have this; upgrade pin promptly)."
        )
    if pin_behind_nightly:
        reasons.append(
            f"Pinned contract `{pin}` is behind **nightly** CDN `{nightly}` "
            "(start supporting this line; historically the earliest almost-stable signal)."
        )
    if pin_behind_prerelease and pre:
        reasons.append(
            f"Pinned contract `{pin}` is behind GitHub **prerelease** `{pre}` "
            "(freeze candidate; harden support against this tag)."
        )
    if nightly_ahead and not pin_behind_nightly and not pin_behind_stable:
        # Pin already matches nightly but stable has not caught up
        reasons.append(
            f"Nightly `{nightly}` is ahead of stable `{stable}`; pin `{pin}` "
            "already tracks the next line. Keep this issue for stable promotion watch."
        )

    channels_changed = False
    if previous:
        if normalize_version(str(previous.get("stable", ""))) != stable:
            channels_changed = True
            reasons.append(
                f"Stable CDN changed: `{previous.get('stable')}` → `{stable}`."
            )
        if normalize_version(str(previous.get("nightly", ""))) != nightly:
            channels_changed = True
            reasons.append(
                f"Nightly CDN changed: `{previous.get('nightly')}` → `{nightly}`."
            )
        prev_pre = previous.get("prereleases") or []
        if sorted(map(normalize_version, prev_pre)) != sorted(
            map(normalize_version, snap.prereleases)
        ):
            channels_changed = True
            reasons.append(
                f"GitHub prerelease set changed: {prev_pre} → {snap.prereleases}."
            )

    needs_support_work = pin_behind_nightly or pin_behind_stable or pin_behind_prerelease
    # Keep a single open watch issue while next line is ahead of stable even if pin matches nightly
    watch_next_line = nightly_ahead

    if needs_support_work:
        title = f"coolify-channel: support Coolify {nightly if pin_behind_nightly or pin_behind_prerelease else stable} (pin is {pin})"
        action = "open"  # caller upgrades to update if issue exists
    elif watch_next_line:
        title = f"coolify-channel: nightly {nightly} ahead of stable {stable} (pin {pin})"
        action = "open"
    elif channels_changed:
        title = f"coolify-channel: channels updated (stable {stable}, nightly {nightly})"
        action = "comment"
    else:
        title = f"coolify-channel: stable {stable}, nightly {nightly}, pin {pin}"
        action = "none"

    # Deduplicate reasons while preserving order
    seen: set[str] = set()
    uniq_reasons: list[str] = []
    for r in reasons:
        if r not in seen:
            seen.add(r)
            uniq_reasons.append(r)

    body = build_body(snap, uniq_reasons)
    return Decision(
        action=action,
        reasons=uniq_reasons,
        snapshot=snap,
        title=title,
        body=body,
        pin_behind_nightly=pin_behind_nightly,
        pin_behind_stable=pin_behind_stable,
        pin_behind_prerelease=pin_behind_prerelease,
        nightly_ahead_of_stable=nightly_ahead,
    )


def fetch_json(url: str, timeout: int = 30) -> dict[str, Any]:
    req = urllib.request.Request(url, headers={"User-Agent": "terraform-provider-coolify-channel-watch"})
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode())


def fetch_prerelease_tags(repo: str = "coollabsio/coolify") -> list[str]:
    """Return prerelease tag names via gh if available, else GitHub API."""
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN") or ""
    # Prefer gh for auth/rate limits in Actions
    try:
        out = subprocess.check_output(
            [
                "gh",
                "api",
                f"repos/{repo}/releases",
                "--jq",
                "[.[] | select(.prerelease==true) | .tag_name]",
            ],
            text=True,
            stderr=subprocess.DEVNULL,
        )
        tags = json.loads(out)
        return [normalize_version(t) for t in tags]
    except (subprocess.CalledProcessError, FileNotFoundError, json.JSONDecodeError):
        pass
    url = f"https://api.github.com/repos/{repo}/releases?per_page=30"
    headers = {"User-Agent": "terraform-provider-coolify-channel-watch", "Accept": "application/vnd.github+json"}
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=30) as resp:
        releases = json.loads(resp.read().decode())
    return [normalize_version(r["tag_name"]) for r in releases if r.get("prerelease")]


def gh_json(args: list[str]) -> Any:
    out = subprocess.check_output(["gh", *args], text=True)
    return json.loads(out) if out.strip() else None


def find_open_channel_issue() -> Optional[dict[str, Any]]:
    issues = gh_json(
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
    if not issues:
        return None
    # Prefer issues that carry our state marker
    for issue in issues:
        if STATE_MARKER_PREFIX in (issue.get("body") or ""):
            return issue
    return issues[0]


def ensure_labels() -> None:
    subprocess.run(
        [
            "gh",
            "label",
            "create",
            ISSUE_LABEL,
            "--description",
            "Coolify CDN/GitHub channel watch (stable vs nightly vs pin)",
            "--color",
            "1d76db",
            "--force",
        ],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )
    subprocess.run(
        [
            "gh",
            "label",
            "create",
            "ready",
            "--description",
            "Ready to implement",
            "--color",
            "0e8a16",
            "--force",
        ],
        check=False,
        stdout=subprocess.DEVNULL,
        stderr=subprocess.DEVNULL,
    )


def apply_decision(decision: Decision) -> int:
    """Create or update GitHub issue. Returns 0 always unless gh fails hard."""
    if decision.action == "none":
        print("No channel action needed.")
        return 0

    ensure_labels()
    existing = find_open_channel_issue()
    label_args: list[str] = []
    for lab in LABELS:
        label_args.extend(["--label", lab])

    if existing is None:
        if decision.action in ("open", "update", "comment"):
            # comment with no issue => open
            cmd = [
                "gh",
                "issue",
                "create",
                "--title",
                decision.title,
                "--body",
                decision.body,
                *label_args,
            ]
            print("Creating issue:", decision.title)
            subprocess.check_call(cmd)
            return 0
        return 0

    number = str(existing["number"])
    prev = decode_state(existing.get("body") or "")
    # Re-decide with previous state for better reasons if caller did not
    if prev and decision.action == "open":
        # Issue exists: update body/title when support work or channels changed
        decision = decide(decision.snapshot, previous=prev)
        if decision.action == "none" and not (
            decision.pin_behind_nightly
            or decision.pin_behind_stable
            or decision.pin_behind_prerelease
            or decision.nightly_ahead_of_stable
        ):
            print(f"Open issue #{number} already current; no update.")
            return 0

    # Always refresh body on open issue when we have something to say
    if decision.action in ("open", "update") or (
        decision.nightly_ahead_of_stable
        or decision.pin_behind_nightly
        or decision.pin_behind_stable
        or decision.pin_behind_prerelease
    ):
        print(f"Updating issue #{number}: {decision.title}")
        subprocess.check_call(
            ["gh", "issue", "edit", number, "--title", decision.title, "--body", decision.body]
        )
        # Ensure labels
        for lab in LABELS:
            subprocess.run(
                ["gh", "issue", "edit", number, "--add-label", lab],
                check=False,
                stdout=subprocess.DEVNULL,
                stderr=subprocess.DEVNULL,
            )
        return 0

    if decision.action == "comment":
        print(f"Commenting on issue #{number}")
        subprocess.check_call(
            [
                "gh",
                "issue",
                "comment",
                number,
                "--body",
                decision.body,
            ]
        )
        # Keep body state fresh
        subprocess.check_call(
            ["gh", "issue", "edit", number, "--title", decision.title, "--body", decision.body]
        )
        return 0

    return 0


def main(argv: Optional[list[str]] = None) -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument(
        "--versions-json",
        type=Path,
        help="Path to CDN versions.json (skips network fetch for that file)",
    )
    parser.add_argument(
        "--pinned-version",
        help="Pinned contract version override (e.g. 4.1.2 or v4.1.2)",
    )
    parser.add_argument(
        "--contract",
        type=Path,
        default=Path("testdata/contracts/coolify-v4.json"),
        help="Pinned contract JSON (default: testdata/contracts/coolify-v4.json)",
    )
    parser.add_argument(
        "--prerelease-tags",
        nargs="*",
        default=None,
        help="Prerelease tags (without fetching). Empty list means none.",
    )
    parser.add_argument(
        "--fetch",
        action="store_true",
        help="Fetch CDN versions.json from the network",
    )
    parser.add_argument(
        "--fetch-prereleases",
        action="store_true",
        help="Fetch GitHub prerelease tags for coollabsio/coolify",
    )
    parser.add_argument(
        "--previous-state-json",
        type=Path,
        help="Optional previous state JSON for change detection tests",
    )
    parser.add_argument(
        "--apply",
        action="store_true",
        help="Create/update GitHub issue via gh CLI when action is needed",
    )
    parser.add_argument(
        "--json",
        action="store_true",
        help="Print decision as JSON",
    )
    args = parser.parse_args(argv)

    # Load versions
    if args.versions_json:
        data = json.loads(args.versions_json.read_text())
        stable, nightly = load_versions_json(data)
    elif args.fetch:
        try:
            data = fetch_json(CDN_VERSIONS_URL)
        except (urllib.error.URLError, TimeoutError, json.JSONDecodeError) as e:
            print(f"ERROR: failed to fetch {CDN_VERSIONS_URL}: {e}", file=sys.stderr)
            return 2
        stable, nightly = load_versions_json(data)
    else:
        print("ERROR: provide --versions-json or --fetch", file=sys.stderr)
        return 2

    if args.pinned_version:
        pin = normalize_version(args.pinned_version)
    elif args.contract.exists():
        pin = load_pinned_version(args.contract)
    else:
        print(f"ERROR: contract not found: {args.contract}", file=sys.stderr)
        return 2

    if args.prerelease_tags is not None:
        prereleases = [normalize_version(t) for t in args.prerelease_tags]
    elif args.fetch_prereleases or args.fetch:
        try:
            prereleases = fetch_prerelease_tags()
        except Exception as e:  # noqa: BLE001 - best-effort enrichment
            print(f"WARNING: could not fetch prereleases: {e}", file=sys.stderr)
            prereleases = []
    else:
        prereleases = []

    snap = ChannelSnapshot(
        stable=stable,
        nightly=nightly,
        pin=pin,
        prereleases=prereleases,
    )

    previous = None
    if args.previous_state_json and args.previous_state_json.exists():
        previous = json.loads(args.previous_state_json.read_text())

    # When applying, merge previous state from open issue for change reasons
    if args.apply and previous is None:
        try:
            issue = find_open_channel_issue()
            if issue:
                previous = decode_state(issue.get("body") or "")
        except (subprocess.CalledProcessError, FileNotFoundError):
            previous = None

    decision = decide(snap, previous=previous)

    # Always emit a concise human line on stderr for Actions logs.
    print(
        f"stable={snap.stable} nightly={snap.nightly} pin={snap.pin} "
        f"prereleases={snap.prereleases} action={decision.action}",
        file=sys.stderr,
    )
    for r in decision.reasons:
        print(f"  - {r}", file=sys.stderr)
    if args.json:
        print(json.dumps(asdict(decision), indent=2))
    else:
        print(f"title={decision.title}")

    if args.apply:
        return apply_decision(decision)

    # Non-apply: exit 1 when support work is needed (useful in CI summaries)
    if decision.pin_behind_nightly or decision.pin_behind_stable or decision.pin_behind_prerelease:
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
