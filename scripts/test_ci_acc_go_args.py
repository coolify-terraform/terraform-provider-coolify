#!/usr/bin/env python3
"""Unit tests for scripts/ci-acc-go-args.sh."""

from __future__ import annotations

import subprocess
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "ci-acc-go-args.sh"


def go_args(image: str, label: str = "") -> list[str]:
    proc = subprocess.run(
        ["bash", str(SCRIPT), image, label],
        cwd=ROOT,
        check=True,
        capture_output=True,
        text=True,
    )
    return [ln.strip() for ln in proc.stdout.splitlines() if ln.strip()]


class TestAccGoArgs(unittest.TestCase):
    def test_latest_serializes_packages(self):
        args = go_args("latest", "stable-latest")
        self.assertEqual(args[:2], ["-p", "1"])
        self.assertIn("-timeout=55m", args)
        self.assertIn("-parallel=1", args)
        self.assertIn("-count=1", args)

    def test_custom_latest_image_serializes_packages(self):
        args = go_args("latest", "custom-latest")
        self.assertEqual(args[:2], ["-p", "1"])
        self.assertIn("-timeout=55m", args)

    def test_stable_label_serializes_even_if_image_changes(self):
        args = go_args("4.4.2", "stable-latest")
        self.assertIn("-p", args)
        self.assertIn("1", args[:2])

    def test_edge_keeps_package_parallelism(self):
        args = go_args("edge", "tip-edge")
        self.assertNotIn("-p", args)
        self.assertIn("-timeout=40m", args)
        self.assertIn("-parallel=1", args)

    def test_floor_keeps_package_parallelism(self):
        args = go_args("4.1.2", "floor-4.1.2")
        self.assertNotIn("-p", args)
        self.assertIn("-timeout=40m", args)


if __name__ == "__main__":
    unittest.main()
