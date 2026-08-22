#!/usr/bin/env python3
"""Unit tests for scripts/ci-scenario-suite-need.sh."""

from __future__ import annotations

import os
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "ci-scenario-suite-need.sh"


def run_need(suite: str, env_updates: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    env.pop("GITHUB_APP_ID", None)
    env.pop("HETZNER_TOKEN", None)
    env.pop("COOLIFY_GITHUB_APP_APP_ID", None)
    env.pop("COOLIFY_HETZNER_TOKEN", None)
    env.pop("TF_VAR_github_app_id", None)
    env.pop("TF_VAR_hetzner_api_token", None)
    if env_updates:
        env.update(env_updates)
    return subprocess.run(
        ["bash", str(SCRIPT), suite],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=env,
    )


class TestCIScenarioSuiteNeed(unittest.TestCase):
    def test_missing_suite_exits_2(self) -> None:
        env = os.environ.copy()
        env.pop("GITHUB_APP_ID", None)
        env.pop("HETZNER_TOKEN", None)
        env.pop("COOLIFY_GITHUB_APP_APP_ID", None)
        env.pop("COOLIFY_HETZNER_TOKEN", None)
        env.pop("TF_VAR_github_app_id", None)
        env.pop("TF_VAR_hetzner_api_token", None)
        proc = subprocess.run(
            ["bash", str(SCRIPT)],
            cwd=ROOT,
            capture_output=True,
            text=True,
            env=env,
        )
        self.assertEqual(proc.returncode, 2)
        self.assertIn("usage", proc.stderr)

    def test_unknown_suite_exits_1(self) -> None:
        proc = run_need("nope")
        self.assertEqual(proc.returncode, 1)
        self.assertIn("unknown suite", proc.stderr)

    def test_core_always_runs(self) -> None:
        proc = run_need("core")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")

    def test_all_always_runs(self) -> None:
        proc = run_need("all")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")

    def test_github_cicd_skips_without_secret(self) -> None:
        proc = run_need("github-cicd")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=false")

    def test_github_cicd_runs_with_secret(self) -> None:
        proc = run_need("github-cicd", {"GITHUB_APP_ID": "123"})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")

    def test_hetzner_skips_without_secret(self) -> None:
        proc = run_need("hetzner")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=false")

    def test_hetzner_runs_with_secret(self) -> None:
        proc = run_need("hetzner", {"HETZNER_TOKEN": "tok"})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")

    def test_github_cicd_accepts_coolify_env_name(self) -> None:
        proc = run_need("github-cicd", {"COOLIFY_GITHUB_APP_APP_ID": "123"})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")

    def test_hetzner_accepts_tf_var_name(self) -> None:
        proc = run_need("hetzner", {"TF_VAR_hetzner_api_token": "tok"})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "run=true")


if __name__ == "__main__":
    unittest.main()
