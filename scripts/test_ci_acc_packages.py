#!/usr/bin/env python3
"""Unit tests for scripts/ci-acc-packages.sh."""

from __future__ import annotations

import os
import stat
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "ci-acc-packages.sh"
MODULE = "github.com/coolify-terraform/terraform-provider-coolify"


def run_shard(shard: int, count: int) -> list[str]:
    proc = subprocess.run(
        ["bash", str(SCRIPT), str(shard), str(count)],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        env=os.environ.copy(),
    )
    return [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]


def acc_packages() -> list[str]:
    listed = subprocess.run(
        ["go", "list", "./internal/service/..."],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    ).stdout.splitlines()
    out: list[str] = []
    for pkg in listed:
        rel = Path(pkg.removeprefix(MODULE + "/"))
        if any(
            "func TestAcc" in line
            for test in (ROOT / rel).glob("*_test.go")
            for line in test.read_text(encoding="utf-8").splitlines()
        ):
            out.append(pkg)
    return sorted(out)


class TestCIAccPackages(unittest.TestCase):
    def test_script_is_executable(self) -> None:
        self.assertTrue(SCRIPT.stat().st_mode & stat.S_IXUSR)

    def test_two_shards_partition_all_acc_packages(self) -> None:
        shards = [run_shard(i, 2) for i in range(2)]
        union: list[str] = []
        for s in shards:
            self.assertGreater(len(s), 0, "shard must not be empty")
            union.extend(s)
        want = acc_packages()
        self.assertGreater(len(want), 10)
        self.assertEqual(sorted(union), want)
        self.assertEqual(len(union), len(set(union)), "packages must not overlap shards")

    def test_application_and_service_land_on_distinct_shards(self) -> None:
        app = f"{MODULE}/internal/service/application"
        svc = f"{MODULE}/internal/service/service"
        shard0 = set(run_shard(0, 2))
        shard1 = set(run_shard(1, 2))
        self.assertIn(app, shard0)
        self.assertNotIn(app, shard1)
        self.assertIn(svc, shard1)
        self.assertNotIn(svc, shard0)

    def test_only_service_packages_with_testacc(self) -> None:
        for pkg in run_shard(0, 2) + run_shard(1, 2):
            self.assertTrue(
                pkg.startswith(f"{MODULE}/internal/service/"),
                pkg,
            )
            self.assertNotIn("/tools", pkg)

    def test_bad_args(self) -> None:
        bad = [[], ["0"], ["0", "0"], ["2", "2"], ["x", "2"]]
        for args in bad:
            with self.subTest(args=args):
                proc = subprocess.run(
                    ["bash", str(SCRIPT), *args],
                    cwd=ROOT,
                    capture_output=True,
                    text=True,
                )
                self.assertNotEqual(proc.returncode, 0)


if __name__ == "__main__":
    unittest.main()
