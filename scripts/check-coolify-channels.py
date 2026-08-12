#!/usr/bin/env python3
"""Watch Coolify release channels for pre-stable and stable signals.

Coolify does not always use classic RC tags. For v4.2 the early "almost
stable" signal was the nightly channel version (CDN versions.json), with
GitHub prereleases as a later freeze signal. For v4.3 the CDN version
stayed at 4.2.0 until the day of stable release, so version numbers alone
were too late.

This script compares:
  - stable:  coolify.v4.version from CDN versions.json
  - nightly: coolify.nightly.version from the same file
  - pin:     version from testdata/contracts/coolify-v4.json
  - prereleases: GitHub releases with prerelease=true (optional)
  - latest_release: newest non-draft GitHub release tag (stable or pre)
  - tip_version: version string from Coolify source config/constants.php
    on the default branch (earliest "next line" signal)
  - tip_contract: optional extracted contract from tip vs pin (API drift
    even when version strings still match)

It prints a machine-readable decision and can create/update a GitHub
issue when the pin lags any of those signals.

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

  # Proactive: also compare tip contract extract to pin
  python3 scripts/check-coolify-channels.py --fetch \\
    --tip-contract testdata/contracts/coolify-v4-latest.json \\
    --apply

Environment (for --apply / GitHub fetches):
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
COOLIFY_REPO = "coollabsio/coolify"
CONSTANTS_RAW_URL = (
    "https://raw.githubusercontent.com/coollabsio/coolify/main/config/constants.php"
)
STATE_MARKER_PREFIX = "<!-- coolify-channel-state:"
STATE_MARKER_RE = re.compile(
    r"<!-- coolify-channel-state:\s*(.*?)\s*-->",
    re.DOTALL,
)
ISSUE_LABEL = "coolify-channel"
LABELS = ["coolify-channel", "ready"]
# Early-signal label when tip/API drifts before stable CDN moves.
PRIORITY_LABEL = "coolify-channel-early"
# Matches: 'version' => env('COOLIFY_VERSION') ?: '4.3.0',
VERSION_IN_CONSTANTS_RE = re.compile(
    r"'version'\s*=>\s*env\('COOLIFY_VERSION'\)\s*\?:\s*'([^']+)'"
)


@dataclass
class ChannelSnapshot:
    stable: str
    nightly: str
    pin: str
    prereleases: list[str] = field(default_factory=list)
    latest_release: str = ""
    tip_version: str = ""
    tip_contract_diff_count: int = 0
    tip_contract_summary: str = ""


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
    pin_behind_latest_release: bool
    pin_behind_tip_version: bool
    pin_behind_tip_api: bool
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
    return normalize_version(ver)


def newest_prerelease(tags: list[str]) -> Optional[str]:
    if not tags:
        return None
    return max(tags, key=parse_semver)


def contract_field_signature(contract: dict[str, Any]) -> dict[str, Any]:
    """Build a stable signature of model fields + endpoint allow-lists."""
    models: dict[str, list[str]] = {}
    for name, model in (contract.get("models") or {}).items():
        fields = model.get("fields") or {}
        if isinstance(fields, dict):
            models[name] = sorted(fields.keys())
        else:
            models[name] = sorted(model.get("fillable") or [])
    endpoints: dict[str, list[str]] = {}
    for name, ep in (contract.get("endpoints") or {}).items():
        endpoints[name] = sorted(ep.get("allowed_fields") or [])
    return {"models": models, "endpoints": endpoints}


def diff_contract_signatures(
    pin: dict[str, Any], tip: dict[str, Any]
) -> tuple[int, str]:
    """Return (change_count, short summary) of tip vs pin API surface."""
    a = contract_field_signature(pin)
    b = contract_field_signature(tip)
    lines: list[str] = []
    changes = 0

    for name in sorted(set(a["models"]) | set(b["models"])):
        old = set(a["models"].get(name, []))
        new = set(b["models"].get(name, []))
        added = sorted(new - old)
        removed = sorted(old - new)
        if added:
            changes += len(added)
            lines.append(f"+ {name} fields: {', '.join(added[:12])}" + ("…" if len(added) > 12 else ""))
        if removed:
            changes += len(removed)
            lines.append(f"- {name} fields: {', '.join(removed[:12])}" + ("…" if len(removed) > 12 else ""))

    for name in sorted(set(a["endpoints"]) | set(b["endpoints"])):
        old = set(a["endpoints"].get(name, []))
        new = set(b["endpoints"].get(name, []))
        added = sorted(new - old)
        removed = sorted(old - new)
        if name not in a["endpoints"]:
            changes += 1
            lines.append(f"+ endpoint {name}")
            continue
        if name not in b["endpoints"]:
            changes += 1
            lines.append(f"- endpoint {name}")
            continue
        if added or removed:
            changes += len(added) + len(removed)
            detail = []
            if added:
                detail.append(f"+{len(added)} allow fields")
            if removed:
                detail.append(f"-{len(removed)} allow fields")
            sample = (added or removed)[:6]
            lines.append(f"~ {name}: {', '.join(detail)} ({', '.join(sample)})")

    summary = "\n".join(f"  {ln}" for ln in lines[:40])
    if len(lines) > 40:
        summary += f"\n  … and {len(lines) - 40} more lines"
    return changes, summary


def encode_state(snap: ChannelSnapshot) -> str:
    payload = {
        "stable": snap.stable,
        "nightly": snap.nightly,
        "pin": snap.pin,
        "prereleases": snap.prereleases,
        "latest_release": snap.latest_release,
        "tip_version": snap.tip_version,
        "tip_contract_diff_count": snap.tip_contract_diff_count,
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
    latest = snap.latest_release or "_none_"
    tip_v = snap.tip_version or "_none_"
    tip_diff = snap.tip_contract_diff_count

    extract_nightly = f"make contract-extract VERSION=v{nightly_v}"
    extract_stable = f"make contract-extract VERSION=v{stable_v}"
    extract_tip = "make contract-extract VERSION=latest"
    extract_pre = ""
    if pre:
        tag = pre if pre.startswith("v") else f"v{pre}"
        extract_pre = f"make contract-extract VERSION={tag}"

    reasons_md = "\n".join(f"- {r}" for r in reasons) if reasons else "- (no lag)"

    tip_block = ""
    if tip_diff > 0 and snap.tip_contract_summary:
        tip_block = f"""
### Tip API drift vs pin ({tip_diff} change(s))

```
{snap.tip_contract_summary}
```
"""

    steps = [
        f"1. Extract tip (default branch): `{extract_tip}` and diff against pin",
        f"2. Extract next-line contract: `{extract_nightly}` (and stable `{extract_stable}`)",
        "3. Diff: `scripts/diff-contracts.sh testdata/contracts/coolify-v4.json testdata/contracts/coolify-v"
        + nightly_v
        + ".json`",
        "4. Run `make contract-check` and implement provider changes for new fields/routes",
        "5. Promote pin (`testdata/contracts/coolify-v4.json`) when support is ready; do not wait for stable CDN if nightly/tip already expose the API",
        "6. When stable, nightly, tip version, and pin align (and tip API drift is zero), the channel job closes this issue automatically",
    ]
    if extract_pre:
        steps.insert(2, f"2b. Prerelease extract: `{extract_pre}`")

    return f"""{encode_state(snap)}

## Coolify channel watch

Automated check of Coolify CDN channels, GitHub releases, **source tip version**,
and optional **tip contract API drift**. Goal: start provider work *before*
stable ships.

| Channel | Version |
|---------|---------|
| **Pinned contract** (`testdata/contracts/coolify-v4.json`) | `{pin_v}` |
| **Stable CDN** (`coolify.v4`) | `{stable_v}` |
| **Nightly CDN** (`coolify.nightly`) | `{nightly_v}` |
| **GitHub latest release** (any non-draft) | `{latest}` |
| **GitHub prerelease (newest)** | {pre_line} |
| **Source tip version** (`config/constants.php` on main) | `{tip_v}` |
| **Tip contract API drift vs pin** | `{tip_diff}` change(s) |

### Why this fired

{reasons_md}

### What this means

Early signals (in rough order of lead time):

1. **Source tip version** and **tip contract extract** (API fields on default branch)
2. **Nightly CDN** version bump
3. **GitHub prerelease / latest release** tags
4. **Stable CDN** promotion (users on default installs)

Do not wait for stable CDN to start `contract-extract` and provider work.
v4.3.0 taught us that CDN nightly can stay on the previous line until release day
while tip already carries the next-line API.
{tip_block}
### Next commands

```bash
# Earliest: tip of default branch
{extract_tip}

# Next-line / stable tags
{extract_nightly}
{extract_stable}
```

{chr(10).join(steps)}

### Sources

- CDN: {CDN_VERSIONS_URL}
- Nightly install CDN: `https://cdn.coollabs.io/coolify-nightly`
- Source tip version: `{CONSTANTS_RAW_URL}`
- Coolify default branch for tip extract: `main`
"""


def decide(snap: ChannelSnapshot, previous: Optional[dict[str, Any]] = None) -> Decision:
    reasons: list[str] = []
    pin = snap.pin
    stable = snap.stable
    nightly = snap.nightly
    pre = newest_prerelease(snap.prereleases)
    latest = snap.latest_release
    tip_v = snap.tip_version

    pin_behind_nightly = bool(pin and nightly and version_lt(pin, nightly))
    pin_behind_stable = bool(pin and stable and version_lt(pin, stable))
    pin_behind_prerelease = bool(pin and pre and version_lt(pin, pre))
    pin_behind_latest = bool(pin and latest and version_lt(pin, latest))
    pin_behind_tip_ver = bool(pin and tip_v and version_lt(pin, tip_v))
    pin_behind_tip_api = snap.tip_contract_diff_count > 0
    nightly_ahead = bool(stable and nightly and version_lt(stable, nightly))

    if pin_behind_stable:
        reasons.append(
            f"Pinned contract `{pin}` is behind **stable** CDN `{stable}` "
            "(users on stable may already have this; upgrade pin promptly)."
        )
    if pin_behind_nightly:
        reasons.append(
            f"Pinned contract `{pin}` is behind **nightly** CDN `{nightly}` "
            "(start supporting this line; historically an almost-stable signal)."
        )
    if pin_behind_prerelease and pre:
        reasons.append(
            f"Pinned contract `{pin}` is behind GitHub **prerelease** `{pre}` "
            "(freeze candidate; harden support against this tag)."
        )
    if pin_behind_latest and latest:
        reasons.append(
            f"Pinned contract `{pin}` is behind GitHub **latest release** `{latest}` "
            "(full or pre; do not wait for CDN alone)."
        )
    if pin_behind_tip_ver and tip_v:
        reasons.append(
            f"Pinned contract `{pin}` is behind **source tip version** `{tip_v}` "
            "(config/constants.php on main; earliest next-line version signal)."
        )
    if pin_behind_tip_api:
        reasons.append(
            f"**Tip contract API drift**: {snap.tip_contract_diff_count} field/endpoint "
            "change(s) vs pin even when CDN version strings match. Extract tip and implement "
            "before stable ships."
        )
    if nightly_ahead and not (
        pin_behind_nightly
        or pin_behind_stable
        or pin_behind_tip_ver
        or pin_behind_tip_api
        or pin_behind_latest
    ):
        reasons.append(
            f"Nightly `{nightly}` is ahead of stable `{stable}`; pin `{pin}` "
            "already tracks the next line. Keep this issue for stable promotion watch."
        )

    channels_changed = False
    if previous:
        for key, label in (
            ("stable", "Stable CDN"),
            ("nightly", "Nightly CDN"),
            ("latest_release", "GitHub latest release"),
            ("tip_version", "Source tip version"),
        ):
            prev_v = normalize_version(str(previous.get(key, "") or ""))
            cur_v = normalize_version(str(getattr(snap, key, "") or ""))
            if prev_v != cur_v and (prev_v or cur_v):
                channels_changed = True
                reasons.append(f"{label} changed: `{previous.get(key)}` → `{cur_v or getattr(snap, key)}`.")
        prev_pre = previous.get("prereleases") or []
        if sorted(map(normalize_version, prev_pre)) != sorted(
            map(normalize_version, snap.prereleases)
        ):
            channels_changed = True
            reasons.append(
                f"GitHub prerelease set changed: {prev_pre} → {snap.prereleases}."
            )
        prev_diff = int(previous.get("tip_contract_diff_count") or 0)
        if prev_diff != snap.tip_contract_diff_count:
            channels_changed = True
            reasons.append(
                f"Tip API drift count changed: {prev_diff} → {snap.tip_contract_diff_count}."
            )

    needs_support_work = (
        pin_behind_nightly
        or pin_behind_stable
        or pin_behind_prerelease
        or pin_behind_latest
        or pin_behind_tip_ver
        or pin_behind_tip_api
    )
    watch_next_line = nightly_ahead

    # Title prefers the highest-urgency target version.
    target = pin
    for cand in (tip_v, nightly, latest, stable, pre or ""):
        if cand and (not target or version_lt(target, cand)):
            target = cand

    if needs_support_work:
        title = f"coolify-channel: support Coolify {target} (pin is {pin})"
        action = "open"
    elif watch_next_line:
        title = f"coolify-channel: nightly {nightly} ahead of stable {stable} (pin {pin})"
        action = "open"
    elif channels_changed:
        title = f"coolify-channel: channels updated (stable {stable}, nightly {nightly})"
        action = "comment"
    else:
        title = f"coolify-channel: stable {stable}, nightly {nightly}, pin {pin}"
        action = "none"

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
        pin_behind_latest_release=pin_behind_latest,
        pin_behind_tip_version=pin_behind_tip_ver,
        pin_behind_tip_api=pin_behind_tip_api,
        nightly_ahead_of_stable=nightly_ahead,
    )


def fetch_json(url: str, timeout: int = 30) -> dict[str, Any]:
    req = urllib.request.Request(
        url, headers={"User-Agent": "terraform-provider-coolify-channel-watch"}
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return json.loads(resp.read().decode())


def fetch_text(url: str, timeout: int = 30) -> str:
    req = urllib.request.Request(
        url, headers={"User-Agent": "terraform-provider-coolify-channel-watch"}
    )
    with urllib.request.urlopen(req, timeout=timeout) as resp:
        return resp.read().decode()


def parse_tip_version_from_constants(php: str) -> str:
    m = VERSION_IN_CONSTANTS_RE.search(php or "")
    if not m:
        return ""
    return normalize_version(m.group(1))


def fetch_tip_version() -> str:
    """Read Coolify source tip version from config/constants.php on main."""
    try:
        out = subprocess.check_output(
            [
                "gh",
                "api",
                f"repos/{COOLIFY_REPO}/contents/config/constants.php?ref=main",
                "--jq",
                ".content",
            ],
            text=True,
            stderr=subprocess.DEVNULL,
        )
        import base64

        php = base64.b64decode(out.strip().replace("\n", "")).decode()
        return parse_tip_version_from_constants(php)
    except (subprocess.CalledProcessError, FileNotFoundError, Exception):
        pass
    try:
        return parse_tip_version_from_constants(fetch_text(CONSTANTS_RAW_URL))
    except (urllib.error.URLError, TimeoutError, UnicodeDecodeError):
        return ""


def fetch_prerelease_tags(repo: str = COOLIFY_REPO) -> list[str]:
    """Return prerelease tag names via gh if available, else GitHub API."""
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN") or ""
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
    headers = {
        "User-Agent": "terraform-provider-coolify-channel-watch",
        "Accept": "application/vnd.github+json",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=30) as resp:
        releases = json.loads(resp.read().decode())
    return [normalize_version(r["tag_name"]) for r in releases if r.get("prerelease")]


def fetch_latest_release_tag(repo: str = COOLIFY_REPO) -> str:
    """Newest non-draft release tag (prerelease or full)."""
    try:
        out = subprocess.check_output(
            [
                "gh",
                "api",
                f"repos/{repo}/releases",
                "--jq",
                "[.[] | select(.draft==false) | .tag_name] | .[0]",
            ],
            text=True,
            stderr=subprocess.DEVNULL,
        )
        tag = out.strip().strip('"')
        return normalize_version(tag)
    except (subprocess.CalledProcessError, FileNotFoundError):
        pass
    token = os.environ.get("GH_TOKEN") or os.environ.get("GITHUB_TOKEN") or ""
    url = f"https://api.github.com/repos/{repo}/releases?per_page=10"
    headers = {
        "User-Agent": "terraform-provider-coolify-channel-watch",
        "Accept": "application/vnd.github+json",
    }
    if token:
        headers["Authorization"] = f"Bearer {token}"
    req = urllib.request.Request(url, headers=headers)
    with urllib.request.urlopen(req, timeout=30) as resp:
        releases = json.loads(resp.read().decode())
    for r in releases:
        if not r.get("draft"):
            return normalize_version(r.get("tag_name") or "")
    return ""


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
    for issue in issues:
        if STATE_MARKER_PREFIX in (issue.get("body") or ""):
            return issue
    return issues[0]


def ensure_labels() -> None:
    for name, desc, color in (
        (ISSUE_LABEL, "Coolify CDN/GitHub channel watch (stable vs nightly vs pin)", "1d76db"),
        (PRIORITY_LABEL, "Early Coolify tip/API drift before stable CDN", "b60205"),
        ("ready", "Ready to implement", "0e8a16"),
    ):
        subprocess.run(
            [
                "gh",
                "label",
                "create",
                name,
                "--description",
                desc,
                "--color",
                color,
                "--force",
            ],
            check=False,
            stdout=subprocess.DEVNULL,
            stderr=subprocess.DEVNULL,
        )


def apply_decision(decision: Decision) -> int:
    """Create, update, or close GitHub issue. Returns 0 unless gh fails hard."""
    ensure_labels()
    existing = find_open_channel_issue()
    label_args: list[str] = []
    for lab in LABELS:
        label_args.extend(["--label", lab])
    early = (
        decision.pin_behind_tip_api
        or decision.pin_behind_tip_version
        or decision.pin_behind_latest_release
    ) and not decision.pin_behind_stable
    if early:
        label_args.extend(["--label", PRIORITY_LABEL])

    if decision.action == "none":
        if existing is not None and STATE_MARKER_PREFIX in (existing.get("body") or ""):
            number = str(existing["number"])
            print(f"Closing issue #{number}: channels aligned (stable/nightly/pin/tip).")
            subprocess.check_call(
                [
                    "gh",
                    "issue",
                    "close",
                    number,
                    "--comment",
                    (
                        "Closing automatically: channels aligned at pin "
                        f"`{decision.snapshot.pin}` "
                        f"(stable=`{decision.snapshot.stable}`, "
                        f"nightly=`{decision.snapshot.nightly}`, "
                        f"tip=`{decision.snapshot.tip_version or 'n/a'}`, "
                        f"tip API drift=`{decision.snapshot.tip_contract_diff_count}`)."
                    ),
                ]
            )
            return 0
        print("No channel action needed.")
        return 0

    if existing is None:
        if decision.action in ("open", "update", "comment"):
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
    if prev and decision.action == "open":
        decision = decide(decision.snapshot, previous=prev)
        if decision.action == "none" and not (
            decision.pin_behind_nightly
            or decision.pin_behind_stable
            or decision.pin_behind_prerelease
            or decision.pin_behind_latest_release
            or decision.pin_behind_tip_version
            or decision.pin_behind_tip_api
            or decision.nightly_ahead_of_stable
        ):
            print(f"Open issue #{number} already current; no update.")
            return 0

    if decision.action in ("open", "update") or (
        decision.nightly_ahead_of_stable
        or decision.pin_behind_nightly
        or decision.pin_behind_stable
        or decision.pin_behind_prerelease
        or decision.pin_behind_latest_release
        or decision.pin_behind_tip_version
        or decision.pin_behind_tip_api
    ):
        print(f"Updating issue #{number}: {decision.title}")
        subprocess.check_call(
            ["gh", "issue", "edit", number, "--title", decision.title, "--body", decision.body]
        )
        labels_to_add = list(LABELS)
        if early:
            labels_to_add.append(PRIORITY_LABEL)
        for lab in labels_to_add:
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
        "--latest-release",
        default=None,
        help="Latest GitHub release tag override (without fetching)",
    )
    parser.add_argument(
        "--tip-version",
        default=None,
        help="Source tip version override (without fetching constants.php)",
    )
    parser.add_argument(
        "--tip-contract",
        type=Path,
        help="Tip contract JSON to diff against --contract for API drift",
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
        "--fetch-latest-release",
        action="store_true",
        help="Fetch newest non-draft GitHub release tag",
    )
    parser.add_argument(
        "--fetch-tip-version",
        action="store_true",
        help="Fetch source tip version from config/constants.php on main",
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

    if args.pinned_version is not None:
        pin = normalize_version(args.pinned_version)
    elif args.contract.exists():
        pin = load_pinned_version(args.contract)
    else:
        print(f"ERROR: contract not found: {args.contract}", file=sys.stderr)
        return 2

    if not pin:
        print(
            "ERROR: pinned contract version is empty "
            f"(check --pinned-version or {args.contract} 'version' field)",
            file=sys.stderr,
        )
        return 2

    if args.prerelease_tags is not None:
        prereleases = [normalize_version(t) for t in args.prerelease_tags]
    elif args.fetch_prereleases or args.fetch:
        try:
            prereleases = fetch_prerelease_tags()
        except Exception as e:  # noqa: BLE001
            print(f"WARNING: could not fetch prereleases: {e}", file=sys.stderr)
            prereleases = []
    else:
        prereleases = []

    if args.latest_release is not None:
        latest_release = normalize_version(args.latest_release)
    elif args.fetch_latest_release or args.fetch:
        try:
            latest_release = fetch_latest_release_tag()
        except Exception as e:  # noqa: BLE001
            print(f"WARNING: could not fetch latest release: {e}", file=sys.stderr)
            latest_release = ""
    else:
        latest_release = ""

    if args.tip_version is not None:
        tip_version = normalize_version(args.tip_version)
    elif args.fetch_tip_version or args.fetch:
        try:
            tip_version = fetch_tip_version()
        except Exception as e:  # noqa: BLE001
            print(f"WARNING: could not fetch tip version: {e}", file=sys.stderr)
            tip_version = ""
    else:
        tip_version = ""

    tip_diff = 0
    tip_summary = ""
    if args.tip_contract and args.tip_contract.exists() and args.contract.exists():
        try:
            pin_c = json.loads(args.contract.read_text())
            tip_c = json.loads(args.tip_contract.read_text())
            tip_diff, tip_summary = diff_contract_signatures(pin_c, tip_c)
        except (OSError, json.JSONDecodeError) as e:
            print(f"WARNING: could not diff tip contract: {e}", file=sys.stderr)

    snap = ChannelSnapshot(
        stable=stable,
        nightly=nightly,
        pin=pin,
        prereleases=prereleases,
        latest_release=latest_release,
        tip_version=tip_version,
        tip_contract_diff_count=tip_diff,
        tip_contract_summary=tip_summary,
    )

    previous = None
    if args.previous_state_json and args.previous_state_json.exists():
        previous = json.loads(args.previous_state_json.read_text())

    if args.apply and previous is None:
        try:
            issue = find_open_channel_issue()
            if issue:
                previous = decode_state(issue.get("body") or "")
        except (subprocess.CalledProcessError, FileNotFoundError):
            previous = None

    decision = decide(snap, previous=previous)

    print(
        f"stable={snap.stable} nightly={snap.nightly} pin={snap.pin} "
        f"latest={snap.latest_release or '-'} tip={snap.tip_version or '-'} "
        f"tip_api_diff={snap.tip_contract_diff_count} "
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

    if (
        decision.pin_behind_nightly
        or decision.pin_behind_stable
        or decision.pin_behind_prerelease
        or decision.pin_behind_latest_release
        or decision.pin_behind_tip_version
        or decision.pin_behind_tip_api
    ):
        return 1
    return 0


if __name__ == "__main__":
    sys.exit(main())
