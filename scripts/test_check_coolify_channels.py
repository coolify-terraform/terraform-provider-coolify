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
from unittest import mock

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


class TestTipVersionParse(unittest.TestCase):
    def test_parse_constants_php(self):
        php = """
        return [
            'coolify' => [
                'version' => env('COOLIFY_VERSION') ?: '4.3.0',
                'helper_version' => '1.0.14',
            ],
        ];
        """
        self.assertEqual(cc.parse_tip_version_from_constants(php), "4.3.0")

    def test_parse_missing(self):
        self.assertEqual(cc.parse_tip_version_from_constants("no version here"), "")


class TestContractDiff(unittest.TestCase):
    def test_detects_new_fields(self):
        pin = {
            "models": {"Application": {"fields": {"name": {}, "domains": {}}}},
            "endpoints": {
                "ApplicationsController::update_by_uuid": {
                    "allowed_fields": ["name", "domains"]
                }
            },
        }
        tip = {
            "models": {
                "Application": {
                    "fields": {"name": {}, "domains": {}, "noindex_domains": {}}
                }
            },
            "endpoints": {
                "ApplicationsController::update_by_uuid": {
                    "allowed_fields": ["name", "domains", "noindex_domains"]
                }
            },
        }
        count, summary = cc.diff_contract_signatures(pin, tip)
        self.assertGreater(count, 0)
        self.assertIn("noindex_domains", summary)

    def test_identical_zero(self):
        c = {
            "models": {"Application": {"fields": {"name": {}}}},
            "endpoints": {},
        }
        count, _ = cc.diff_contract_signatures(c, c)
        self.assertEqual(count, 0)


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
        snap = cc.ChannelSnapshot(
            stable="4.1.2", nightly="4.2.0", pin="4.2.0", prereleases=["4.2.0"]
        )
        d = cc.decide(snap)
        self.assertFalse(d.pin_behind_nightly)
        self.assertTrue(d.nightly_ahead_of_stable)
        self.assertEqual(d.action, "open")
        self.assertTrue(any("ahead of stable" in r for r in d.reasons))

    def test_fetch_errors_block_aligned_and_channel_change(self):
        self.assertTrue(cc.fetch_errors_block_apply(["tip_version"]))
        self.assertFalse(cc.fetch_errors_block_apply([]))

    def test_all_aligned(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0",
            nightly="4.2.0",
            pin="4.2.0",
            prereleases=[],
            latest_release="4.2.0",
            tip_version="4.2.0",
            tip_contract_diff_count=0,
        )
        d = cc.decide(snap)
        self.assertEqual(d.action, "none")
        self.assertFalse(d.pin_behind_nightly)
        self.assertFalse(d.nightly_ahead_of_stable)
        self.assertFalse(d.pin_behind_tip_api)

    def test_old_prerelease_ignored_when_aligned(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.2.0", prereleases=["4.1.0"]
        )
        d = cc.decide(snap)
        self.assertEqual(d.action, "none")
        self.assertFalse(d.pin_behind_prerelease)

    def test_pin_behind_prerelease_beta(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0",
            nightly="4.3.0",
            pin="4.2.0",
            prereleases=["4.3.0-beta.1"],
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_nightly)
        self.assertTrue(d.pin_behind_prerelease)
        self.assertEqual(d.action, "open")

    def test_pin_behind_stable(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.1.2", prereleases=[]
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_stable)
        self.assertIn(d.action, ("open", "update"))

    def test_pin_behind_tip_version_before_cdn(self):
        # The v4.3 failure mode: CDN still 4.2.0, tip already 4.3.0.
        snap = cc.ChannelSnapshot(
            stable="4.1.2",
            nightly="4.2.0",
            pin="4.2.0",
            prereleases=["4.2.0"],
            tip_version="4.3.0",
            latest_release="4.2.0",
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_tip_version)
        self.assertFalse(d.pin_behind_nightly)
        self.assertEqual(d.action, "open")
        self.assertTrue(any("tip version" in r.lower() for r in d.reasons))
        self.assertIn("4.3.0", d.title)

    def test_pin_behind_tip_api_without_version_bump(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0",
            nightly="4.2.0",
            pin="4.2.0",
            tip_version="4.2.0",
            tip_contract_diff_count=3,
            tip_contract_summary="+ Application fields: noindex_domains",
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_tip_api)
        self.assertEqual(d.action, "open")
        self.assertTrue(any("API drift" in r for r in d.reasons))
        self.assertIn("noindex_domains", d.body)

    def test_pin_behind_latest_release(self):
        snap = cc.ChannelSnapshot(
            stable="4.1.2",
            nightly="4.1.2",
            pin="4.1.2",
            latest_release="4.3.0",
        )
        d = cc.decide(snap)
        self.assertTrue(d.pin_behind_latest_release)
        self.assertEqual(d.action, "open")

    def test_channel_change_updates_not_comment(self):
        snap = cc.ChannelSnapshot(
            stable="4.2.0", nightly="4.2.0", pin="4.2.0", prereleases=[]
        )
        prev = {"stable": "4.1.2", "nightly": "4.2.0", "pin": "4.2.0", "prereleases": []}
        d = cc.decide(snap, previous=prev)
        self.assertEqual(d.action, "update")
        self.assertTrue(any("Stable CDN changed" in r for r in d.reasons))

    def test_same_title_updates_existing_issue(self):
        snap = cc.ChannelSnapshot(
            stable="4.1.2", nightly="4.2.0", pin="4.1.2", prereleases=["4.2.0"]
        )
        first = cc.decide(snap)
        self.assertEqual(first.action, "open")
        again = cc.decide(snap, existing_title=first.title)
        self.assertEqual(again.action, "update")
        self.assertEqual(again.title, first.title)

    def test_pin_or_target_change_replaces_issue(self):
        snap = cc.ChannelSnapshot(
            stable="4.3.19",
            nightly="4.4-rc.1",
            pin="4.3.19",
            prereleases=["4.4-rc.1"],
            latest_release="4.3.19",
            tip_version="4.3.19",
        )
        d = cc.decide(
            snap,
            existing_title="coolify-channel: support Coolify 4.4-rc.1 (pin is 4.3.18)",
        )
        self.assertEqual(d.action, "replace")
        self.assertIn("pin is 4.3.19", d.title)
        self.assertIn("opens a new one", d.body)

    def test_replace_path_reapplies_labels(self):
        src = Path(cc.__file__).read_text()
        replace = src[src.index("if decision.action == \"replace\":") :]
        replace = replace[: replace.index("print(f\"Updating issue")]
        self.assertIn("apply_issue_labels(new_number, early)", replace)
        self.assertIn("close_duplicate_freshness_issues(new_number)", replace)
        self.assertIn("def apply_issue_labels(", src)
        # The close-aligned path also assigns number = str(existing[...])
        # before first-create. Search that marker only after if existing is None.
        none_at = src.index("if existing is None:")
        first_create = src[none_at : src.index("number = str(existing", none_at)]
        self.assertTrue(first_create.strip(), "first-create slice must be non-empty")
        self.assertIn("new_number = parse_created_issue_number(created)", first_create)
        self.assertIn("apply_issue_labels(new_number, early)", first_create)
        self.assertIn("close_duplicate_freshness_issues(new_number)", first_create)

    def test_parse_created_issue_number(self):
        url = (
            "https://github.com/coolify-terraform/terraform-provider-coolify"
            "/issues/860\n"
        )
        self.assertEqual(cc.parse_created_issue_number(url), "860")

    def test_state_roundtrip(self):
        snap = cc.ChannelSnapshot(
            stable="4.1.2",
            nightly="4.2.0",
            pin="4.1.2",
            prereleases=["4.2.0"],
            latest_release="4.2.0",
            tip_version="4.2.0",
            tip_contract_diff_count=1,
        )
        body = cc.build_body(snap, ["reason"])
        state = cc.decode_state(body)
        self.assertIsNotNone(state)
        self.assertEqual(state["stable"], "4.1.2")
        self.assertEqual(state["nightly"], "4.2.0")
        self.assertEqual(state["prereleases"], ["4.2.0"])
        self.assertEqual(state["tip_version"], "4.2.0")
        self.assertEqual(state["tip_contract_diff_count"], 1)


class TestCLI(unittest.TestCase):
    def test_empty_pin_exits_error(self):
        data = {
            "coolify": {
                "v4": {"version": "4.1.2"},
                "nightly": {"version": "4.2.0"},
            }
        }
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "versions.json"
            path.write_text(json.dumps(data))
            rc = cc.main(
                [
                    "--versions-json",
                    str(path),
                    "--pinned-version",
                    "",
                    "--json",
                ]
            )
            self.assertEqual(rc, 2)

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
            rc2 = cc.main(
                [
                    "--versions-json",
                    str(path),
                    "--pinned-version",
                    "4.2.0",
                    "--prerelease-tags",
                    "4.2.0",
                    "--tip-version",
                    "4.2.0",
                    "--latest-release",
                    "4.2.0",
                    "--json",
                ]
            )
            self.assertEqual(rc2, 0)

    def test_cli_tip_api_diff_exit_one(self):
        data = {
            "coolify": {
                "v4": {"version": "4.2.0"},
                "nightly": {"version": "4.2.0"},
            }
        }
        pin = {
            "version": "v4.2.0",
            "models": {"Application": {"fields": {"name": {}}}},
            "endpoints": {},
        }
        tip = {
            "version": "v4-latest",
            "models": {"Application": {"fields": {"name": {}, "noindex_domains": {}}}},
            "endpoints": {},
        }
        with tempfile.TemporaryDirectory() as td:
            root = Path(td)
            vpath = root / "versions.json"
            vpath.write_text(json.dumps(data))
            pin_path = root / "pin.json"
            tip_path = root / "tip.json"
            pin_path.write_text(json.dumps(pin))
            tip_path.write_text(json.dumps(tip))
            rc = cc.main(
                [
                    "--versions-json",
                    str(vpath),
                    "--contract",
                    str(pin_path),
                    "--tip-contract",
                    str(tip_path),
                    "--tip-version",
                    "4.2.0",
                    "--latest-release",
                    "4.2.0",
                    "--prerelease-tags",
                    "--json",
                ]
            )
            self.assertEqual(rc, 1)

    def _aligned_versions(self, td: str) -> str:
        path = Path(td) / "versions.json"
        path.write_text(
            json.dumps(
                {
                    "coolify": {
                        "v4": {"version": "4.2.0"},
                        "nightly": {"version": "4.2.0"},
                    }
                }
            )
        )
        return str(path)

    def _open_watch_issue(self) -> dict:
        return {
            "number": 868,
            "title": "coolify-channel: support Coolify 4.4-rc.1 (pin is 4.3.21)",
            "body": (
                "<!-- coolify-channel-state: "
                '{"stable":"4.3.21","nightly":"4.4-rc.1","pin":"4.3.21",'
                '"prereleases":["4.4-rc.1"],"latest_release":"4.3.21",'
                '"tip_version":"4.3.21","tip_contract_diff_count":0} -->\n'
                "open watch"
            ),
        }

    def test_tip_fetch_failure_does_not_apply_aligned_close(self):
        with tempfile.TemporaryDirectory() as td:
            path = self._aligned_versions(td)
            with mock.patch.object(
                cc, "fetch_tip_version", side_effect=RuntimeError("gh down")
            ), mock.patch.object(
                cc, "find_open_channel_issue", return_value=self._open_watch_issue()
            ), mock.patch.object(cc, "apply_decision") as apply:
                rc = cc.main(
                    [
                        "--versions-json",
                        path,
                        "--pinned-version",
                        "4.2.0",
                        "--latest-release",
                        "4.2.0",
                        "--prerelease-tags",
                        "--fetch-tip-version",
                        "--apply",
                    ]
                )
        apply.assert_not_called()
        self.assertEqual(rc, 2)

    def test_latest_release_fetch_failure_does_not_apply_aligned_close(self):
        with tempfile.TemporaryDirectory() as td:
            path = self._aligned_versions(td)
            with mock.patch.object(
                cc,
                "fetch_latest_release_tag",
                side_effect=RuntimeError("releases api down"),
            ), mock.patch.object(
                cc, "find_open_channel_issue", return_value=self._open_watch_issue()
            ), mock.patch.object(cc, "apply_decision") as apply:
                rc = cc.main(
                    [
                        "--versions-json",
                        path,
                        "--pinned-version",
                        "4.2.0",
                        "--tip-version",
                        "4.2.0",
                        "--prerelease-tags",
                        "--fetch-latest-release",
                        "--apply",
                    ]
                )
        apply.assert_not_called()
        self.assertEqual(rc, 2)

    def test_prerelease_fetch_failure_does_not_apply_aligned_close(self):
        with tempfile.TemporaryDirectory() as td:
            path = self._aligned_versions(td)
            with mock.patch.object(
                cc, "fetch_prerelease_tags", side_effect=RuntimeError("tags down")
            ), mock.patch.object(
                cc, "find_open_channel_issue", return_value=self._open_watch_issue()
            ), mock.patch.object(cc, "apply_decision") as apply:
                rc = cc.main(
                    [
                        "--versions-json",
                        path,
                        "--pinned-version",
                        "4.2.0",
                        "--tip-version",
                        "4.2.0",
                        "--latest-release",
                        "4.2.0",
                        "--fetch-prereleases",
                        "--apply",
                    ]
                )
        apply.assert_not_called()
        self.assertEqual(rc, 2)

    def test_tip_fetch_failure_does_not_replace_open_watch(self):
        data = {
            "coolify": {
                "v4": {"version": "4.1.2"},
                "nightly": {"version": "4.2.0"},
            }
        }
        existing = {
            "number": 868,
            "title": "coolify-channel: support Coolify 4.3.0 (pin is 4.1.2)",
            "body": (
                "<!-- coolify-channel-state: "
                '{"stable":"4.1.2","nightly":"4.2.0","pin":"4.1.2",'
                '"prereleases":[],"latest_release":"4.1.2",'
                '"tip_version":"4.3.0","tip_contract_diff_count":0} -->\n'
                "open watch"
            ),
        }
        with tempfile.TemporaryDirectory() as td:
            path = Path(td) / "versions.json"
            path.write_text(json.dumps(data))
            with mock.patch.object(
                cc, "fetch_tip_version", side_effect=RuntimeError("gh down")
            ), mock.patch.object(
                cc, "find_open_channel_issue", return_value=existing
            ), mock.patch.object(cc, "apply_decision") as apply:
                rc = cc.main(
                    [
                        "--versions-json",
                        str(path),
                        "--pinned-version",
                        "4.1.2",
                        "--latest-release",
                        "4.1.2",
                        "--prerelease-tags",
                        "--fetch-tip-version",
                        "--apply",
                    ]
                )
        apply.assert_not_called()
        self.assertEqual(rc, 2)


class TestCloseDuplicateFreshness(unittest.TestCase):
    def test_closes_open_contract_drift_as_duplicate_of_channel(self) -> None:
        freshness = [{"number": 895, "title": "Upstream Coolify contract has changed"}]
        with mock.patch.object(
            cc, "find_open_freshness_issues", return_value=freshness
        ), mock.patch.object(cc.subprocess, "check_call") as check_call:
            cc.close_duplicate_freshness_issues("894")
        check_call.assert_called_once()
        args = check_call.call_args[0][0]
        self.assertEqual(args[:4], ["gh", "issue", "close", "895"])
        self.assertIn("--reason", args)
        self.assertIn("completed", args)
        comment = args[args.index("--comment") + 1]
        self.assertIn("Duplicate of #894", comment)

    def test_skips_when_no_freshness_issues(self) -> None:
        with mock.patch.object(
            cc, "find_open_freshness_issues", return_value=[]
        ), mock.patch.object(cc.subprocess, "check_call") as check_call:
            cc.close_duplicate_freshness_issues("894")
        check_call.assert_not_called()

    def test_apply_create_closes_leftover_freshness(self) -> None:
        decision = cc.Decision(
            action="open",
            reasons=["pin behind tip"],
            snapshot=cc.ChannelSnapshot(
                stable="4.3.23",
                nightly="4.4-rc.1",
                pin="4.3.21",
            ),
            title="coolify-channel: support Coolify 4.4 (pin is 4.3.21)",
            body="body",
            pin_behind_nightly=True,
            pin_behind_stable=True,
            pin_behind_prerelease=False,
            pin_behind_latest_release=True,
            pin_behind_tip_version=True,
            pin_behind_tip_api=True,
            nightly_ahead_of_stable=True,
        )
        with mock.patch.object(cc, "ensure_labels"), mock.patch.object(
            cc, "find_open_channel_issue", return_value=None
        ), mock.patch.object(
            cc.subprocess, "check_output", return_value="https://github.com/o/r/issues/894\n"
        ), mock.patch.object(cc, "apply_issue_labels"), mock.patch.object(
            cc, "close_duplicate_freshness_issues"
        ) as close_dup:
            rc = cc.apply_decision(decision)
        self.assertEqual(rc, 0)
        close_dup.assert_called_once_with("894")


if __name__ == "__main__":
    unittest.main()
