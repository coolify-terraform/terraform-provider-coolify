#!/usr/bin/env python3
"""Write Coolify deployment payloads (including logs) to a directory.

GET /deployments only lists in-progress jobs. After a failed scenario
teardown those are usually gone, so this also walks applications and
GET /deployments/applications/{uuid}.
"""

from __future__ import annotations

import argparse
import json
import sys
import urllib.error
import urllib.request
from pathlib import Path
from typing import Any


def fetch_json(endpoint: str, token: str, path: str) -> Any:
    url = endpoint.rstrip("/") + path
    req = urllib.request.Request(
        url,
        headers={
            "Authorization": f"Bearer {token}",
            "Accept": "application/json",
        },
    )
    with urllib.request.urlopen(req, timeout=20) as resp:
        return json.load(resp)


def deployment_uuids(payload: Any) -> list[str]:
    items: list[Any]
    if isinstance(payload, list):
        items = payload
    elif isinstance(payload, dict):
        for key in ("deployments", "data"):
            if isinstance(payload.get(key), list):
                items = payload[key]
                break
        else:
            items = [payload]
    else:
        return []
    out: list[str] = []
    for item in items:
        if not isinstance(item, dict):
            continue
        uid = item.get("deployment_uuid") or item.get("uuid")
        if uid:
            out.append(str(uid))
    return out


def application_uuids(payload: Any) -> list[str]:
    items = payload if isinstance(payload, list) else []
    if isinstance(payload, dict):
        items = payload.get("data") or payload.get("applications") or []
    out: list[str] = []
    for item in items:
        if isinstance(item, dict) and item.get("uuid"):
            out.append(str(item["uuid"]))
    return out


def write_json(path: Path, data: Any) -> None:
    path.write_text(json.dumps(data, indent=2) + "\n", encoding="utf-8")


def dump(endpoint: str, token: str, out_dir: Path) -> int:
    out_dir.mkdir(parents=True, exist_ok=True)
    written = 0
    try:
        running = fetch_json(endpoint, token, "/api/v1/deployments")
        write_json(out_dir / "deployments-running.json", running)
        written += 1
    except (urllib.error.URLError, urllib.error.HTTPError, json.JSONDecodeError, TimeoutError) as err:
        print(f"FAIL: list running deployments: {err}", file=sys.stderr)
        running = []

    uuids = set(deployment_uuids(running))
    try:
        apps = fetch_json(endpoint, token, "/api/v1/applications")
        write_json(out_dir / "applications.json", apps)
        written += 1
    except (urllib.error.URLError, urllib.error.HTTPError, json.JSONDecodeError, TimeoutError) as err:
        print(f"FAIL: list applications: {err}", file=sys.stderr)
        apps = []

    for app_uuid in application_uuids(apps):
        rel = f"/api/v1/deployments/applications/{app_uuid}"
        try:
            hist = fetch_json(endpoint, token, rel)
            write_json(out_dir / f"app-{app_uuid}-deployments.json", hist)
            written += 1
            uuids.update(deployment_uuids(hist))
        except (urllib.error.URLError, urllib.error.HTTPError, json.JSONDecodeError, TimeoutError) as err:
            print(f"FAIL: list deployments for {app_uuid}: {err}", file=sys.stderr)

    for uid in sorted(uuids):
        try:
            detail = fetch_json(endpoint, token, f"/api/v1/deployments/{uid}")
            write_json(out_dir / f"deployment-{uid}.json", detail)
            written += 1
        except (urllib.error.URLError, urllib.error.HTTPError, json.JSONDecodeError, TimeoutError) as err:
            print(f"FAIL: get deployment {uid}: {err}", file=sys.stderr)

    print(f"OK: wrote {written} file(s) under {out_dir}")
    return 0 if written else 1


def main() -> int:
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--endpoint", required=True)
    parser.add_argument("--token", required=True)
    parser.add_argument("--out-dir", required=True, type=Path)
    args = parser.parse_args()
    print("PLAN: dump Coolify deployment JSON (including logs when the token can read them)")
    print(f"DO: fetch deployments from {args.endpoint}")
    return dump(args.endpoint, args.token, args.out_dir)


if __name__ == "__main__":
    sys.exit(main())
