#!/usr/bin/env python3
"""Unit tests for scripts/run-scenario-tests.sh suite selection."""

from __future__ import annotations

import os
import stat
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "run-scenario-tests.sh"


def run_script(*args: str, env_updates: dict[str, str] | None = None) -> subprocess.CompletedProcess[str]:
    env = os.environ.copy()
    if env_updates:
        env.update(env_updates)
    return subprocess.run(
        ["bash", str(SCRIPT), *args],
        cwd=ROOT,
        capture_output=True,
        text=True,
        env=env,
    )


def names_from_list(proc: subprocess.CompletedProcess[str]) -> list[str]:
    return [Path(ln.strip()).name for ln in proc.stdout.splitlines() if ln.strip()]


class TestRunScenarioTests(unittest.TestCase):
    def test_script_is_executable(self) -> None:
        mode = SCRIPT.stat().st_mode
        self.assertTrue(mode & stat.S_IXUSR, "run-scenario-tests.sh must be executable")

    def test_unknown_suite_exits_2(self) -> None:
        proc = run_script("--list", "--suite", "nope")
        self.assertEqual(proc.returncode, 2)
        self.assertIn("unknown suite", proc.stderr)

    def test_unknown_flag_exits_2(self) -> None:
        proc = run_script("--wat")
        self.assertEqual(proc.returncode, 2)
        self.assertIn("unknown argument", proc.stderr)

    def test_list_all_is_eighteen_acme_dirs(self) -> None:
        proc = run_script("--list", "--suite", "all")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        names = names_from_list(proc)
        self.assertEqual(len(names), 18)
        self.assertIn("acme-github-cicd", names)
        self.assertIn("acme-hetzner-infra", names)
        self.assertIn("acme-website", names)

    def test_list_core_excludes_slow_suites(self) -> None:
        proc = run_script("--list", "--suite=core")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        names = names_from_list(proc)
        self.assertEqual(len(names), 16)
        self.assertNotIn("acme-github-cicd", names)
        self.assertNotIn("acme-hetzner-infra", names)
        self.assertIn("acme-website", names)

    def test_list_github_cicd_is_one_dir(self) -> None:
        proc = run_script("--list", "--suite", "github-cicd")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(names_from_list(proc), ["acme-github-cicd"])

    def test_list_hetzner_is_one_dir(self) -> None:
        proc = run_script("--list", "--suite", "hetzner")
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(names_from_list(proc), ["acme-hetzner-infra"])

    def test_core_plus_slow_covers_all(self) -> None:
        all_names = set(names_from_list(run_script("--list", "--suite", "all")))
        core = set(names_from_list(run_script("--list", "--suite", "core")))
        gh = set(names_from_list(run_script("--list", "--suite", "github-cicd")))
        hz = set(names_from_list(run_script("--list", "--suite", "hetzner")))
        self.assertEqual(core | gh | hz, all_names)
        self.assertEqual(len(core & gh), 0)
        self.assertEqual(len(core & hz), 0)

    def test_list_default_is_all(self) -> None:
        proc = run_script("--list", env_updates={"SCENARIO_SUITE": ""})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(len(names_from_list(proc)), 18)

    def test_list_suite_from_env(self) -> None:
        proc = run_script("--list", env_updates={"SCENARIO_SUITE": "core"})
        self.assertEqual(proc.returncode, 0, proc.stderr)
        names = names_from_list(proc)
        self.assertEqual(len(names), 16)
        self.assertNotIn("acme-github-cicd", names)
        self.assertNotIn("acme-hetzner-infra", names)

    def test_flag_overrides_env_suite(self) -> None:
        proc = run_script(
            "--list",
            "--suite",
            "hetzner",
            env_updates={"SCENARIO_SUITE": "core"},
        )
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(names_from_list(proc), ["acme-hetzner-infra"])
