"""Tests for upsert-social-preview-issue.py."""

from __future__ import annotations

import importlib.util
import json
import sys
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "upsert-social-preview-issue.py"
_spec = importlib.util.spec_from_file_location("upsert_social_preview_issue", _SCRIPT)
usp = importlib.util.module_from_spec(_spec)
sys.modules["upsert_social_preview_issue"] = usp
assert _spec.loader is not None
_spec.loader.exec_module(usp)


class TestPlanUpsert(unittest.TestCase):
    def test_empty_creates(self) -> None:
        plan = usp.plan_upsert([])
        self.assertEqual(plan, {"action": "create", "canonical": None, "close": []})

    def test_one_updates(self) -> None:
        plan = usp.plan_upsert([{"number": 774, "title": usp.ISSUE_TITLE}])
        self.assertEqual(plan, {"action": "update", "canonical": 774, "close": []})

    def test_keeps_oldest_closes_extras(self) -> None:
        plan = usp.plan_upsert(
            [
                {"number": 812, "title": usp.ISSUE_TITLE},
                {"number": 774, "title": usp.ISSUE_TITLE},
            ]
        )
        self.assertEqual(plan["action"], "update")
        self.assertEqual(plan["canonical"], 774)
        self.assertEqual(plan["close"], [812])


class TestCollectCandidates(unittest.TestCase):
    def test_unions_title_and_label_and_sorts(self) -> None:
        got = usp.collect_candidates(
            [{"number": 812, "title": usp.ISSUE_TITLE}],
            [{"number": 774, "title": usp.ISSUE_TITLE}],
        )
        self.assertEqual([i["number"] for i in got], [774, 812])

    def test_dedupes_same_number(self) -> None:
        got = usp.collect_candidates(
            [{"number": 774, "title": usp.ISSUE_TITLE}],
            [{"number": 774, "title": usp.ISSUE_TITLE}],
        )
        self.assertEqual(len(got), 1)


class TestUpsert(unittest.TestCase):
    def test_create_when_none_open(self) -> None:
        calls: list[list[str]] = []

        def gh(args: list[str]) -> str:
            calls.append(args)
            if args[:2] == ["issue", "list"]:
                return "[]"
            if args[:2] == ["issue", "create"]:
                return "https://github.com/org/repo/issues/900\n"
            raise AssertionError(args)

        plan = usp.upsert("body text", gh=gh)
        self.assertEqual(plan["action"], "create")
        create = [c for c in calls if c[:2] == ["issue", "create"]]
        self.assertEqual(len(create), 1)
        self.assertIn("--label", create[0])
        self.assertIn(f"{usp.LABEL},{usp.READY_LABEL}", create[0])

    def test_updates_oldest_and_closes_duplicate(self) -> None:
        calls: list[list[str]] = []

        def gh(args: list[str]) -> str:
            calls.append(args)
            if args[:2] == ["issue", "list"] and "--search" in args:
                return json.dumps(
                    [
                        {"number": 812, "title": usp.ISSUE_TITLE},
                        {"number": 774, "title": usp.ISSUE_TITLE},
                    ]
                )
            if args[:2] == ["issue", "list"]:
                return "[]"
            return ""

        plan = usp.upsert("new body", gh=gh)
        self.assertEqual(plan["canonical"], 774)
        self.assertEqual(plan["close"], [812])
        edits = [c for c in calls if c[:2] == ["issue", "edit"] and c[2] == "774"]
        self.assertTrue(any("--body" in c for c in edits))
        closes = [c for c in calls if c[:2] == ["issue", "close"]]
        self.assertEqual(closes[0][2], "812")
        self.assertIn("Duplicate of #774", " ".join(closes[0]))

    def test_ignores_other_titles_in_search(self) -> None:
        def gh(args: list[str]) -> str:
            if args[:2] == ["issue", "list"] and "--search" in args:
                return json.dumps(
                    [
                        {"number": 1, "title": "Unrelated"},
                        {"number": 774, "title": usp.ISSUE_TITLE},
                    ]
                )
            if args[:2] == ["issue", "list"]:
                return "[]"
            return ""

        plan = usp.upsert("body", gh=gh)
        self.assertEqual(plan["canonical"], 774)
        self.assertEqual(plan["close"], [])


if __name__ == "__main__":
    unittest.main()
