#!/usr/bin/env python3
"""Tests for scripts/dump-coolify-deployments.py."""

from __future__ import annotations

import json
import tempfile
import threading
import unittest
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer
from pathlib import Path

import importlib.util

_SCRIPT = Path(__file__).resolve().parent / "dump-coolify-deployments.py"
_spec = importlib.util.spec_from_file_location("dump_coolify_deployments", _SCRIPT)
_mod = importlib.util.module_from_spec(_spec)
assert _spec.loader is not None
_spec.loader.exec_module(_mod)
application_uuids = _mod.application_uuids
deployment_uuids = _mod.deployment_uuids
dump = _mod.dump


class _Handler(BaseHTTPRequestHandler):
    def log_message(self, fmt: str, *args: object) -> None:
        return

    def do_GET(self) -> None:  # noqa: N802
        routes = {
            "/api/v1/deployments": [{"deployment_uuid": "dep-run", "status": "in_progress"}],
            "/api/v1/applications": [{"uuid": "app-1"}],
            "/api/v1/deployments/applications/app-1": {
                "deployments": [
                    {"deployment_uuid": "dep-fail", "status": "failed"},
                ]
            },
            "/api/v1/deployments/dep-run": {"deployment_uuid": "dep-run", "status": "in_progress"},
            "/api/v1/deployments/dep-fail": {
                "deployment_uuid": "dep-fail",
                "status": "failed",
                "logs": [{"output": "nixpacks failed", "hidden": False}],
            },
        }
        body = routes.get(self.path)
        if body is None:
            self.send_error(404)
            return
        raw = json.dumps(body).encode()
        self.send_response(200)
        self.send_header("Content-Type", "application/json")
        self.send_header("Content-Length", str(len(raw)))
        self.end_headers()
        self.wfile.write(raw)


class TestDumpCoolifyDeployments(unittest.TestCase):
    def test_uuid_helpers(self) -> None:
        self.assertEqual(deployment_uuids([{"deployment_uuid": "a"}]), ["a"])
        self.assertEqual(
            deployment_uuids({"deployments": [{"uuid": "b"}]}),
            ["b"],
        )
        self.assertEqual(application_uuids([{"uuid": "app"}]), ["app"])

    def test_dump_writes_failed_deployment_logs(self) -> None:
        httpd = ThreadingHTTPServer(("127.0.0.1", 0), _Handler)
        thread = threading.Thread(target=httpd.serve_forever, daemon=True)
        thread.start()
        try:
            host, port = httpd.server_address
            with tempfile.TemporaryDirectory() as tmp:
                out = Path(tmp)
                rc = dump(f"http://{host}:{port}", "tok", out)
                self.assertEqual(rc, 0)
                failed = json.loads((out / "deployment-dep-fail.json").read_text())
                self.assertEqual(failed["status"], "failed")
                self.assertEqual(failed["logs"][0]["output"], "nixpacks failed")
        finally:
            httpd.shutdown()


if __name__ == "__main__":
    unittest.main()
