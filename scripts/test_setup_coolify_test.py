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


if __name__ == "__main__":
    unittest.main()
