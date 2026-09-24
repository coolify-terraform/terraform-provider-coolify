#!/usr/bin/env python3
"""Guards for scripts/setup-coolify-test.sh (no Coolify required)."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "setup-coolify-test.sh"
ACTION = ROOT / ".github" / "actions" / "setup-coolify" / "action.yml"


class TestSetupCoolifyTestTimeouts(unittest.TestCase):
    def test_register_has_alarm_and_no_networkidle(self) -> None:
        src = SCRIPT.read_text()
        self.assertIn("signal.alarm(180)", src)
        self.assertIn("set_default_timeout(30000)", src)
        self.assertNotIn('wait_for_load_state("networkidle")', src)
        self.assertNotIn("install chromium 2>/dev/null", src)
        self.assertIn("timeout --foreground 180", src)

    def test_ci_bootstrap_step_has_hard_cap(self) -> None:
        src = ACTION.read_text()
        self.assertIn("timeout --foreground 300", src)
        self.assertIn("setup-coolify-test.sh", src)

    def test_minio_does_not_bind_host_ports(self) -> None:
        src = SCRIPT.read_text()
        self.assertNotIn("-p 9000:9000", src)
        self.assertNotIn("-p 9001:9001", src)
        self.assertIn("docker start coolify-minio", src)
        self.assertIn("chainguard/minio@sha256:", src)
        self.assertNotIn("quay.io/minio/minio:", src)
        start = src.index("Starting MinIO for S3 backup tests")
        run = src[start : src.index("sleep 3", start)]
        self.assertIn("server /tmp/data", run)
        self.assertNotIn(">/dev/null", run)
        self.assertNotIn("2>&1", run)

    def test_minio_bucket_must_exist_before_register(self) -> None:
        src = SCRIPT.read_text()
        register = src.index("Registering MinIO S3 storage in Coolify")
        prelude = src[:register]
        self.assertIn("coolify-backups does not exist", prelude)
        self.assertIn("chainguard/minio-client@sha256:", prelude)
        self.assertIn("MC_HOST_local=", prelude)
        self.assertNotIn("quay.io/minio/mc:", prelude)
        self.assertIn("exit 1", prelude[prelude.rfind("Starting MinIO for S3 backup tests") :])


if __name__ == "__main__":
    unittest.main()
