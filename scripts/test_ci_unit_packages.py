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

    def test_four_shards_partition_all_packages(self) -> None:
        app = "github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
        shards = [run_shard(i, 4) for i in range(4)]
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
        # Application is intentionally on shards 0 and 1 (complementary
        # ci_app_a / ci_app_b test-file slices). Other packages stay unique.
        self.assertIn(app, shards[0])
        self.assertIn(app, shards[1])
        self.assertNotIn(app, shards[2])
        self.assertNotIn(app, shards[3])
        unique = [p for p in union if p != app]
        self.assertEqual(len(unique), len(set(unique)), "non-application packages must not overlap shards")
        self.assertEqual(sorted(set(union)), all_pkgs)

    def test_application_not_stacked_with_next_heavies(self) -> None:
        app = "github.com/coolify-terraform/terraform-provider-coolify/internal/service/application"
        stacked = {
            # Previously forced onto shard 0 by pinning every 3rd heavy.
            "github.com/coolify-terraform/terraform-provider-coolify/internal/service/scheduledtask",
            "github.com/coolify-terraform/terraform-provider-coolify/internal/service/storage",
        }
        shard0 = set(run_shard(0, 3))
        self.assertIn(app, shard0)
        for pkg in stacked:
            self.assertNotIn(pkg, shard0, f"{pkg} must not share shard 0 with application")

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

    def test_application_test_files_have_complementary_build_tags(self) -> None:
        app_dir = ROOT / "internal" / "service" / "application"
        tagged_a: list[str] = []
        tagged_b: list[str] = []
        untagged: list[str] = []
        for path in sorted(app_dir.glob("*_test.go")):
            first = path.read_text(encoding="utf-8").splitlines()[0]
            if first == "//go:build !ci_app_b":
                tagged_a.append(path.name)
            elif first == "//go:build !ci_app_a":
                tagged_b.append(path.name)
            elif first.startswith("//go:build"):
                self.fail(f"{path.name} has unexpected build tag: {first}")
            else:
                untagged.append(path.name)

        self.assertIn("resource_test.go", tagged_a)
        self.assertIn("resource_github_app_test.go", tagged_a)
        self.assertIn("resource_docker_test.go", tagged_b)
        self.assertIn("resource_dockerfile_test.go", tagged_b)
        self.assertIn("resource_private_git_test.go", tagged_b)
        self.assertIn("testing_helpers_test.go", untagged)
        helper = (app_dir / "testing_helpers_test.go").read_text(encoding="utf-8")
        self.assertIn("func decodeRequestBodyMap(", helper)
        self.assertGreater(len(tagged_a), 0)
        self.assertGreater(len(tagged_b), 0)

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
