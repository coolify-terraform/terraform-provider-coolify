#!/usr/bin/env python3
"""Unit tests for scripts/ci-unit-packages.sh."""

from __future__ import annotations

import os
import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "ci-unit-packages.sh"


def run_shard(shard: int, count: int) -> list[str]:
    env = os.environ.copy()
    # Ensure go is on PATH in minimal CI python-test environments.
    proc = subprocess.run(
        ["bash", str(SCRIPT), str(shard), str(count)],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
        env=env,
    )
    lines = [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]
    return lines


class TestCIUnitPackages(unittest.TestCase):
    def test_three_shards_partition_all_packages(self) -> None:
        shards = [run_shard(i, 3) for i in range(3)]
        union: list[str] = []
        for s in shards:
            self.assertGreater(len(s), 0, "shard must not be empty")
            union.extend(s)

        all_pkgs = subprocess.run(
            ["go", "list", "./..."],
            cwd=ROOT,
            check=True,
            capture_output=True,
            text=True,
        ).stdout.splitlines()
        all_pkgs = sorted(p for p in all_pkgs if "/tools" not in p)

        self.assertEqual(sorted(union), all_pkgs)
        self.assertEqual(len(union), len(set(union)), "packages must not overlap shards")

    def test_heavy_packages_land_on_distinct_shards(self) -> None:
        heavy = {
            "github.com/coolify-terraform/terraform-provider-coolify/internal/service/application": 0,
            "github.com/coolify-terraform/terraform-provider-coolify/internal/service/service": 1,
            "github.com/coolify-terraform/terraform-provider-coolify/internal/service/database/redis": 2,
        }
        for pkg, want_shard in heavy.items():
            got = run_shard(want_shard, 3)
            self.assertIn(pkg, got, f"{pkg} should be on shard {want_shard}")
            for other in range(3):
                if other == want_shard:
                    continue
                self.assertNotIn(pkg, run_shard(other, 3))

    def test_bad_args(self) -> None:
        bad = [
            [],
            ["0"],
            ["0", "0"],
            ["3", "3"],
            ["x", "3"],
        ]
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
