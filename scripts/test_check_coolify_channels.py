"""Tests for check-coolify-channels.py.

Run with:
    python3 -m unittest scripts.test_check_coolify_channels -v
"""

import importlib.util
import json
import sys
import tempfile
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "check-coolify-channels.py"
_spec = importlib.util.spec_from_file_location("check_coolify_channels", _SCRIPT)
cc = importlib.util.module_from_spec(_spec)
# Required for @dataclass under Python 3.12+ when loading via importlib.
sys.modules["check_coolify_channels"] = cc
_spec.loader.exec_module(cc)


class TestSemver(unittest.TestCase):
    def test_normalize(self):
        self.assertEqual(cc.normalize_version("v4.2.0"), "4.2.0")
        self.assertEqual(cc.normalize_version("4.1.2"), "4.1.2")

    def test_order_release_vs_beta(self):
        self.assertTrue(cc.version_lt("4.0.0-beta.474", "4.0.0"))
        self.assertTrue(cc.version_lt("4.1.2", "4.2.0"))
        self.assertFalse(cc.version_lt("4.2.0", "4.2.0"))

    def test_eq(self):
        self.assertTrue(cc.version_eq("v4.2.0", "4.2.0"))


class TestLoadVersions(unittest.TestCase):
    def test_load(self):
        data = {
            "coolify": {
                "v4": {"version": "4.1.2"},
                "nightly": {"version": "4.2.0"},
            }
        }
        stable, nightly = cc.load_versions_json(data)
        self.assertEqual(stable, "4.1.2")
        self.assertEqual(nightly, "4.2.0")

    def test_missing(self):
        with self.assertRaises(ValueError):
            cc.load_versions_json({"coolify": {"v4": {"version": "4.1.2"}}})


class TestDecide(unittest.TestCase):
    def test_pin_behind_nightly(self):
        snap = cc.ChannelSnapshot(
            stable="4.1.2", nightly="4.2.0", pin="4.1.2", prereleases=["4.2.0"]
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_nightly)
        self.assertTrue(d.pin_behind_prerelease)
        self.assertFalse(d.pin_behind_stable)
        self.assertIn(d.action, ("open", "update"))
        self.assertTrue(any("nightly" in r for r in d.reasons))
        self.assertIn("coolify-channel-state:", d.body)
        self.assertIn("make contract-extract VERSION=v4.2.0", d.body)

    def test_pin_current_with_nightly_watch_stable(self):
        # Our situation after supporting 4.2 early: pin==nightly, stable lags
        snap = cc.ChannelSnapshot(
            stable="4.1.2", nightly="4.2.0", pin="4.2.0", prereleases=["4.2.0"]
        )
        d = cc.decide(snap)
        self.assertFalse(d.pin_behind_nightly)
        self.assertTrue(d.nightly_ahead_of_stable)
        self.assertEqual(d.action, "open")
        self.assertTrue(any("ahead of stable" in r for r in d.reasons))

    def test_all_aligned(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.2.0", prereleases=[]
        )
        d = cc.decide(snap)
        self.assertEqual(d.action, "none")
        self.assertFalse(d.pin_behind_nightly)
        self.assertFalse(d.nightly_ahead_of_stable)

    def test_pin_behind_stable(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.1.2", prereleases=[]
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_stable)
        self.assertIn(d.action, ("open", "update"))

    def test_channel_change_comment(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.2.0", prereleases=[]
        )
        prev = {"stable": "4.1.2", "nightly": "4.2.0", "pin": "4.2.0", "prereleases": []}
        d = cc.decide(snap, previous=prev)
        self.assertEqual(d.action, "comment")
        self.assertTrue(any("Stable CDN changed" in r for r in d.reasons))

    def test_state_roundtrip(self):
        snap = cc.ChannelSnapshot(
            stable="4.1.2", nightly="4.2.0", pin="4.1.2", prereleases=["4.2.0"]
        )
        body = cc.build_body(snap, ["reason"])
        state = cc.decode_state(body)
        self.assertIsNotNone(state)
        self.assertEqual(state["stable"], "4.1.2")
        self.assertEqual(state["nightly"], "4.2.0")
        self.assertEqual(state["prereleases"], ["4.2.0"])


class TestCLI(unittest.TestCase):
    def test_cli_json_exit_code(self):
        data = {
            "coolify": {
                "v4": {"version": "4.1.2"},
                "nightly": {"version": "4.2.0"},
            }
        }
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "versions.json"
            path.write_text(json.dumps(data))
            # pin behind nightly => exit 1
            rc = cc.main(
                [
                    "--versions-json",
                    str(path),
                    "--pinned-version",
                    "4.1.2",
                    "--prerelease-tags",
                    "v4.2.0",
                    "--json",
                ]
            )
            self.assertEqual(rc, 1)
            # pin current => exit 0 (may still open watch when apply, but no apply)
            rc2 = cc.main(
                [
                    "--versions-json",
                    str(path),
                    "--pinned-version",
                    "4.2.0",
                    "--prerelease-tags",
                    "4.2.0",
                    "--json",
                ]
            )
            # action is open for nightly ahead of stable but exit 0 without pin lag
            self.assertEqual(rc2, 0)


if __name__ == "__main__":
    unittest.main()
