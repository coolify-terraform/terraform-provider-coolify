#!/usr/bin/env python3
"""Unit tests for scripts/ci-coolify-boot.sh (no Docker required)."""

from __future__ import annotations

import os
import stat
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "ci-coolify-boot.sh"


def run_boot(*args: str, env: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    merged = os.environ.copy()
    if env:
        merged.update(env)
    return subprocess.run(
        ["bash", str(SCRIPT), *args],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=merged,
    )


class TestCICoolifyBoot(unittest.TestCase):
    def test_script_is_executable(self) -> None:
        mode = SCRIPT.stat().st_mode
        self.assertTrue(mode & stat.S_IXUSR, "ci-coolify-boot.sh must be executable")

    def test_usage_without_step(self) -> None:
        proc = run_boot()
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("usage:", proc.stderr)

    def test_unknown_step(self) -> None:
        proc = run_boot("not-a-step")
        self.assertEqual(proc.returncode, 2)
        self.assertIn("unknown step", proc.stderr)

    def test_wait_pull_without_marker_is_ok(self) -> None:
        proc = run_boot("wait-pull", env={"COOLIFY_PULL_DIR": "/tmp/coolify-pull-missing-for-test"})
        self.assertEqual(proc.returncode, 0, proc.stdout + proc.stderr)
        self.assertIn("no background pull", proc.stdout)

    def test_wait_ready_fails_fast_when_nothing_listens(self) -> None:
        proc = run_boot("wait-ready", env={"COOLIFY_READY_TRIES": "2"})
        self.assertNotEqual(proc.returncode, 0)
        self.assertIn("FAIL: Coolify not ready", proc.stdout)

    def test_compose_up_has_a_timeout_cap(self) -> None:
        src = SCRIPT.read_text()
        self.assertIn("run_limited 180", src)
        self.assertIn("timeout --foreground", src)


if __name__ == "__main__":
    unittest.main()
