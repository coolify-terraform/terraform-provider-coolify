"""Tests for report-nightly-failure.py.

Run with:
    python3 -m unittest scripts.test_report_nightly_failure -v
"""

from __future__ import annotations

import importlib.util
import json
import sys
import unittest
from datetime import date
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "report-nightly-failure.py"
_spec = importlib.util.spec_from_file_location("report_nightly_failure", _SCRIPT)
rnf = importlib.util.module_from_spec(_spec)
sys.modules["report_nightly_failure"] = rnf
assert _spec.loader is not None
_spec.loader.exec_module(rnf)


def _snap(**kwargs: object) -> rnf.Snapshot:
    defaults = {
        "today": date(2026, 9, 16),
        "run_id": "35055147394",
        "head_sha": "7eb06fd1708fa90e5837b0f878b0c1ad4c8543df",
        "run_url": "https://github.com/coolify-terraform/terraform-provider-coolify/actions/runs/35055147394",
        "prepare": "success",
        "acceptance": "failure",
        "scenarios": "success",
        "failed_jobs": ["Acc (stable-latest)"],
    }
    defaults.update(kwargs)
    return rnf.Snapshot(**defaults)  # type: ignore[arg-type]


def _existing(number: int = 900, **state: object) -> dict[str, object]:
    payload = {
        "first_failed_on": "2026-09-11",
        "last_failed_on": "2026-09-15",
        "consecutive_days": 5,
        "signature": "Acc (stable-latest)",
        "run_id": "34959002473",
        "sha": "a31547ca9d51",
        "previous_issue": None,
    }
    payload.update(state)
    body = rnf.encode_state(payload) + "\n\nold body\n"
    return {"number": number, "title": "Nightly Acc red: Acc (stable-latest)", "body": body}


class TestClassifyRun(unittest.TestCase):
    def test_any_failure_wins(self) -> None:
        self.assertEqual(rnf.classify_run("success", "failure", "cancelled"), "failure")

    def test_cancelled_without_failure(self) -> None:
        self.assertEqual(rnf.classify_run("success", "cancelled", "success"), "cancelled")

    def test_skipped_scenarios_is_success(self) -> None:
        self.assertEqual(rnf.classify_run("success", "success", "skipped"), "success")


class TestConsecutiveDays(unittest.TestCase):
    def test_same_day_is_one(self) -> None:
        self.assertEqual(rnf.consecutive_days(date(2026, 9, 16), date(2026, 9, 16)), 1)

    def test_five_calendar_days(self) -> None:
        self.assertEqual(rnf.consecutive_days(date(2026, 9, 11), date(2026, 9, 16)), 6)

    def test_future_first_is_one(self) -> None:
        self.assertEqual(rnf.consecutive_days(date(2026, 9, 17), date(2026, 9, 16)), 1)


class TestTitle(unittest.TestCase):
    def test_first_day_omits_count(self) -> None:
        self.assertEqual(
            rnf.issue_title(["Acc (stable-latest)"], 1),
            "Nightly Acc red: Acc (stable-latest)",
        )

    def test_later_day_includes_count(self) -> None:
        self.assertEqual(
            rnf.issue_title(["Acc (stable-latest)"], 5),
            "Nightly Acc red: Acc (stable-latest) (5 consecutive days)",
        )

    def test_many_jobs_summarize(self) -> None:
        title = rnf.issue_title(
            ["Acc (floor-4.1.2)", "Acc (stable-latest)", "Acc (tip-edge)"],
            2,
        )
        self.assertIn("+ 2 more", title)
        self.assertIn("2 consecutive days", title)


class TestDecide(unittest.TestCase):
    def test_first_red_opens(self) -> None:
        d = rnf.decide(_snap())
        self.assertEqual(d.action, "open")
        self.assertEqual(d.consecutive_days, 1)
        self.assertIn("Acc (stable-latest)", d.title)
        self.assertNotIn("consecutive days", d.title)
        self.assertIn("nightly-failure-state", d.body)

    def test_later_day_same_signature_replaces(self) -> None:
        d = rnf.decide(_snap(), existing=_existing())
        self.assertEqual(d.action, "replace")
        self.assertEqual(d.consecutive_days, 6)
        self.assertIn("6 consecutive days", d.title)
        self.assertIn("#900", d.body)

    def test_same_day_same_signature_updates(self) -> None:
        d = rnf.decide(
            _snap(run_id="35055147394"),
            existing=_existing(last_failed_on="2026-09-16", run_id="34959002473"),
        )
        self.assertEqual(d.action, "update")
        self.assertEqual(d.consecutive_days, 6)

    def test_same_run_is_noop(self) -> None:
        d = rnf.decide(
            _snap(run_id="35055147394"),
            existing=_existing(last_failed_on="2026-09-16", run_id="35055147394"),
        )
        self.assertEqual(d.action, "none")

    def test_new_signature_same_day_replaces(self) -> None:
        d = rnf.decide(
            _snap(failed_jobs=["Acc (tip-edge)"]),
            existing=_existing(last_failed_on="2026-09-16"),
        )
        self.assertEqual(d.action, "replace")
        self.assertIn("Acc (tip-edge)", d.title)

    def test_green_closes(self) -> None:
        d = rnf.decide(_snap(acceptance="success", failed_jobs=[]), existing=_existing())
        self.assertEqual(d.action, "close")
        self.assertIn("green", d.close_comment)
        self.assertIn("5 consecutive day", d.close_comment)

    def test_green_without_issue_is_noop(self) -> None:
        d = rnf.decide(_snap(acceptance="success", failed_jobs=[]))
        self.assertEqual(d.action, "none")

    def test_cancelled_is_noop(self) -> None:
        d = rnf.decide(
            _snap(acceptance="cancelled", failed_jobs=[]),
            existing=_existing(),
        )
        self.assertEqual(d.action, "none")


class TestFailedJobsFromRun(unittest.TestCase):
    def test_skips_gate_and_non_failures(self) -> None:
        jobs = rnf.failed_jobs_from_run(
            [
                {"name": "Acc (stable-latest)", "conclusion": "failure"},
                {"name": "Acc (tip-edge)", "conclusion": "success"},
                {"name": "Nightly gate", "conclusion": "failure"},
                {"name": "Prepare matrix", "conclusion": "failure"},
            ]
        )
        self.assertEqual(jobs, ["Acc (stable-latest)"])


class TestApplyDecision(unittest.TestCase):
    def test_replace_creates_then_closes_old(self) -> None:
        calls: list[list[str]] = []

        def gh(args: list[str]) -> str:
            calls.append(args)
            if args[:2] == ["issue", "create"]:
                return "https://github.com/coolify-terraform/terraform-provider-coolify/issues/901\n"
            return ""

        d = rnf.decide(_snap(), existing=_existing())
        rc = rnf.apply_decision(d, existing=_existing(), gh=gh, assignee="SebTardif")
        self.assertEqual(rc, 0)
        created = [c for c in calls if c[:2] == ["issue", "create"]]
        self.assertEqual(len(created), 1)
        self.assertIn("--assignee", created[0])
        self.assertIn("SebTardif", created[0])
        self.assertIn("nightly-failure", created[0])
        closed = [c for c in calls if c[:2] == ["issue", "close"]]
        self.assertEqual(closed[0][2], "900")
        comment = closed[0][closed[0].index("--comment") + 1]
        self.assertIn("Superseded by #901", comment)
        self.assertIn("6 consecutive days", comment)

    def test_close_green(self) -> None:
        calls: list[list[str]] = []

        def gh(args: list[str]) -> str:
            calls.append(args)
            return ""

        d = rnf.decide(_snap(acceptance="success", failed_jobs=[]), existing=_existing())
        rc = rnf.apply_decision(d, existing=_existing(), gh=gh, assignee="SebTardif")
        self.assertEqual(rc, 0)
        self.assertTrue(any(c[:2] == ["issue", "close"] for c in calls))
        self.assertFalse(any(c[:2] == ["issue", "create"] for c in calls))


class TestWorkflowWiring(unittest.TestCase):
    def test_nightly_gate_runs_reporter(self) -> None:
        text = (
            Path(__file__).resolve().parents[1]
            / ".github"
            / "workflows"
            / "coolify-nightly.yml"
        ).read_text(encoding="utf-8")
        self.assertIn("scripts/report-nightly-failure.py --apply", text)
        self.assertIn("issues: write", text)
        self.assertIn("NIGHTLY_FAILURE_ASSIGNEE", text)

    def test_issue_triage_skips_bot_inbox(self) -> None:
        text = (
            Path(__file__).resolve().parents[1]
            / ".github"
            / "workflows"
            / "issue-triage.yml"
        ).read_text(encoding="utf-8")
        self.assertIn("Nightly Acc red:", text)
        self.assertIn("nightly-failure", text)


class TestFindOpenIssue(unittest.TestCase):
    def test_prefers_state_marker(self) -> None:
        def gh(args: list[str]) -> str:
            self.assertNotIn("--search", args)
            return json.dumps(
                [
                    {"number": 2, "title": "other", "body": "no marker"},
                    {"number": 1, "title": "red", "body": rnf.encode_state({"run_id": "1"})},
                ]
            )

        got = rnf.find_open_issue(gh)
        self.assertEqual(got["number"], 1)


if __name__ == "__main__":
    unittest.main()
