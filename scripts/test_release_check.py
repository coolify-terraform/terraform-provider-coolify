#!/usr/bin/env python3
"""Unit tests for scripts/release-check.py classification."""

import importlib.util
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "release-check.py"
_spec = importlib.util.spec_from_file_location("release_check", _SCRIPT)
rc = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(rc)


class TestClassify(unittest.TestCase):
    def test_import_mismatch_is_product(self):
        log = """
--- FAIL: TestAccStorageBackupResource_CRUD (1.00s)
    resource_acc_test.go:72: ImportStateVerify attributes not equivalent
    Difference is shown below
    missing_backup_notification_days: "0" != ""
"""
        self.assertEqual(rc.classify_log(log), "product")

    def test_429_is_flake(self):
        log = """
--- FAIL: TestAccDestinationDataSource (0.10s)
    Error: api error for /api/v1/version (status 429): Too Many Attempts.
"""
        self.assertEqual(rc.classify_log(log), "flake")

    def test_version_deadline_is_flake(self):
        log = """
--- FAIL: TestAccDestinationResource_CRUD (30.00s)
    Get "/api/v1/version": context deadline exceeded
"""
        self.assertEqual(rc.classify_log(log), "flake")

    def test_resource_deadline_is_product(self):
        log = """
--- FAIL: TestAccApplicationResource_CRUD (30.00s)
    context deadline exceeded
"""
        self.assertEqual(rc.classify_log(log), "product")

    def test_429_does_not_hide_another_failure(self):
        log = """
--- FAIL: TestAccApplicationResource_CRUD (1.00s)
    Error: attribute mismatch
--- FAIL: TestAccCleanup (0.10s)
    status 429: Too Many Attempts
"""
        self.assertEqual(rc.classify_log(log), "product")

    def test_timestamp_429_is_product(self):
        log = """
2026-10-08T12:34:56.1429871Z --- FAIL: TestAccApplicationResource_CRUD (1.00s)
    Error: status 422: The given data was invalid.
"""
        self.assertEqual(rc.classify_log(log), "product")

    def test_too_many_attempts_is_flake(self):
        log = """
--- FAIL: TestAccDestinationDataSource (0.10s)
    Error: Too Many Attempts.
"""
        self.assertEqual(rc.classify_log(log), "flake")


class TestRerunArgs(unittest.TestCase):
    def test_stable_uses_sha_and_latest(self):
        args = rc.rerun_workflow_args("abc", ["stable"])
        self.assertIn("--ref", args)
        self.assertEqual(args[args.index("--ref") + 1], "abc")
        self.assertIn("custom_image=latest", args)
        self.assertIn("profile=custom", args)
        self.assertIn("run_scenarios=false", args)

    def test_edge_does_not_use_latest_only(self):
        args = rc.rerun_workflow_args("abc", ["edge"])
        self.assertIn("profile=tip-only", args)
        self.assertNotIn("custom_image=latest", args)
        self.assertEqual(args[args.index("--ref") + 1], "abc")

    def test_floor_requests_floor_profile(self):
        args = rc.rerun_workflow_args("abc", ["floor"])
        self.assertIn("profile=floor-only", args)
        self.assertNotIn("custom_image=latest", args)

    def test_two_slots_use_all(self):
        args = rc.rerun_workflow_args("abc", ["edge", "stable"])
        self.assertIn("profile=all", args)
        self.assertNotIn("custom_image=latest", args)


class TestCI(unittest.TestCase):
    def test_schedule_success_is_not_a_product_pass(self):
        runs = [
            {
                "event": "schedule",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-10-08T03:00:00Z",
                "databaseId": 1,
            }
        ]
        status, _latest = rc.classify_ci(runs)
        self.assertEqual(status, "missing")

    def test_older_dispatch_still_counts_when_schedule_is_newer(self):
        runs = [
            {
                "event": "workflow_dispatch",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-10-07T01:00:00Z",
                "databaseId": 2,
            },
            {
                "event": "schedule",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-10-08T03:00:00Z",
                "databaseId": 3,
            },
        ]
        status, latest = rc.classify_ci(runs)
        self.assertEqual(status, "pass")
        self.assertEqual(latest["databaseId"], 2)

    def test_skipped_product_jobs_are_not_ok(self):
        jobs = [
            {"name": "Test (0)", "conclusion": "skipped"},
            {"name": "Acceptance Tests (0)", "conclusion": "skipped"},
            {"name": "Scenario Tests (core, 25)", "conclusion": "skipped"},
        ]
        self.assertFalse(rc.product_ci_jobs_ok(jobs))

    def test_product_jobs_ok(self):
        jobs = [
            {"name": "Test (0)", "conclusion": "success"},
            {"name": "Test (1)", "conclusion": "success"},
            {"name": "Acceptance Tests (0)", "conclusion": "success"},
            {"name": "Acceptance Tests (1)", "conclusion": "success"},
            {"name": "Scenario Tests (core, 25)", "conclusion": "success"},
            {"name": "CI", "conclusion": "success"},
        ]
        self.assertTrue(rc.product_ci_jobs_ok(jobs))

    def test_newer_failure_hides_older_success(self):
        runs = [
            {"conclusion": "success", "status": "completed", "createdAt": "2026-10-08T01:00:00Z"},
            {"conclusion": "failure", "status": "completed", "createdAt": "2026-10-08T02:00:00Z", "url": "https://example.test/fail"},
        ]
        status, latest = rc.classify_ci(runs)
        self.assertEqual(status, "fail")
        self.assertEqual(latest["url"], "https://example.test/fail")

    def test_newer_success_hides_older_failure(self):
        runs = [
            {"conclusion": "failure", "status": "completed", "createdAt": "2026-10-08T01:00:00Z"},
            {"conclusion": "success", "status": "completed", "createdAt": "2026-10-08T03:00:00Z"},
        ]
        status, _ = rc.classify_ci(runs)
        self.assertEqual(status, "pass")


def _product_jobs(conclusion="success"):
    return [
        {"name": "Test (0)", "conclusion": conclusion},
        {"name": "Acceptance Tests (0)", "conclusion": conclusion},
        {"name": "Scenario Tests (core, 25)", "conclusion": conclusion},
    ]


class TestSameTreeCI(unittest.TestCase):
    def test_same_tree_success_counts(self):
        runs = [
            {
                "databaseId": 9,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "pr",
                "url": "https://example.test/ci",
            }
        ]
        status, run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-a", lambda _run: _product_jobs())
        self.assertEqual(status, "pass")
        self.assertEqual(run["databaseId"], 9)

    def test_different_tree_does_not_count(self):
        runs = [
            {
                "databaseId": 9,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "other",
            }
        ]
        status, run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-b", lambda _run: _product_jobs())
        self.assertEqual(status, "missing")
        self.assertIsNone(run)

    def test_skipped_product_jobs_do_not_count(self):
        runs = [
            {
                "databaseId": 9,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "pr",
            }
        ]
        jobs = [
            {"name": "Test (0)", "conclusion": "skipped"},
            {"name": "Acceptance Tests (0)", "conclusion": "skipped"},
            {"name": "Scenario Tests (core, 25)", "conclusion": "skipped"},
        ]
        status, _run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-a", lambda _run: jobs)
        self.assertEqual(status, "missing")

    def test_newer_same_tree_failure_wins(self):
        runs = [
            {
                "databaseId": 1,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T15:00:00Z",
                "headSha": "old",
            },
            {
                "databaseId": 2,
                "event": "pull_request",
                "conclusion": "failure",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "new",
                "url": "https://example.test/fail",
            },
        ]
        status, run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-a", lambda _run: _product_jobs())
        self.assertEqual(status, "fail")
        self.assertEqual(run["databaseId"], 2)

    def test_schedule_run_is_ignored(self):
        runs = [
            {
                "databaseId": 3,
                "event": "schedule",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T17:00:00Z",
                "headSha": "monday",
            },
            {
                "databaseId": 4,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "pr",
            },
        ]
        status, run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-a", lambda _run: _product_jobs())
        self.assertEqual(status, "pass")
        self.assertEqual(run["databaseId"], 4)

    def test_open_run_on_another_tree_is_ignored(self):
        runs = [
            {
                "databaseId": 8,
                "event": "pull_request",
                "conclusion": "",
                "status": "in_progress",
                "createdAt": "2026-11-28T18:00:00Z",
                "headSha": "other",
            },
            {
                "databaseId": 4,
                "event": "pull_request",
                "conclusion": "success",
                "status": "completed",
                "createdAt": "2026-11-28T16:00:00Z",
                "headSha": "pr",
            },
        ]

        def trees(sha):
            return "tree-b" if sha == "other" else "tree-a"

        status, run = rc.select_same_tree_ci("tree-a", runs, trees, lambda _run: _product_jobs())
        self.assertEqual(status, "pass")
        self.assertEqual(run["databaseId"], 4)

    def test_open_run_on_the_same_tree_is_running(self):
        runs = [
            {
                "databaseId": 8,
                "event": "pull_request",
                "conclusion": "",
                "status": "in_progress",
                "createdAt": "2026-11-28T18:00:00Z",
                "headSha": "pr",
            }
        ]
        status, run = rc.select_same_tree_ci("tree-a", runs, lambda _sha: "tree-a", lambda _run: _product_jobs())
        self.assertEqual(status, "running")
        self.assertEqual(run["databaseId"], 8)

    def test_unknown_tree_is_missing(self):
        status, run = rc.select_same_tree_ci("", [], lambda _sha: "tree-a", lambda _run: _product_jobs())
        self.assertEqual(status, "missing")
        self.assertIsNone(run)


class TestSlots(unittest.TestCase):
    def test_latest_failure_wins_over_older_success(self):
        jobs = [
            {"name": "Acc (stable-latest)", "conclusion": "success", "startedAt": "2026-10-08T01:00:00Z"},
            {"name": "Acc (custom-latest)", "conclusion": "failure", "startedAt": "2026-10-08T02:00:00Z", "databaseId": 9},
        ]
        matched = rc.slot_jobs(jobs, rc.STABLE_JOBS)
        self.assertEqual(rc.latest_conclusion(matched), "fail")

    def test_missing_slot(self):
        self.assertEqual(rc.latest_conclusion([]), "missing")

    def test_in_progress_nightly_hides_older_success(self):
        runs = [
            {"databaseId": 1, "status": "completed", "conclusion": "success", "createdAt": "2026-10-08T01:00:00Z"},
            {"databaseId": 2, "status": "in_progress", "conclusion": "", "createdAt": "2026-10-08T02:00:00Z"},
        ]
        jobs = {
            1: [{"name": "Acc (tip-edge)", "conclusion": "success", "startedAt": "2026-10-08T01:00:00Z"}],
            2: [],
        }
        states = rc.nightly_slot_states(runs, jobs)
        self.assertEqual(states["edge"][0], "running")
        self.assertEqual(states["stable"][0], "running")

    def test_prepare_failure_does_not_reuse_older_matrix(self):
        runs = [
            {"databaseId": 1, "status": "completed", "conclusion": "success", "createdAt": "2026-10-08T01:00:00Z"},
            {"databaseId": 2, "status": "completed", "conclusion": "failure", "createdAt": "2026-10-08T02:00:00Z"},
        ]
        jobs = {
            1: [{"name": "Acc (tip-edge)", "conclusion": "success", "startedAt": "t1"}],
            2: [{"name": "Prepare matrix", "conclusion": "failure", "startedAt": "t2"}],
        }
        states = rc.nightly_slot_states(runs, jobs)
        self.assertEqual(states["edge"][0], "fail")
        self.assertEqual(states["floor"][0], "fail")

    def test_custom_stable_failure_keeps_older_edge(self):
        runs = [
            {"databaseId": 1, "status": "completed", "conclusion": "success", "createdAt": "2026-10-08T01:00:00Z"},
            {"databaseId": 2, "status": "completed", "conclusion": "failure", "createdAt": "2026-10-08T02:00:00Z"},
        ]
        jobs = {
            1: [
                {"name": "Acc (tip-edge)", "conclusion": "success", "startedAt": "t1"},
                {"name": "Acc (stable-latest)", "conclusion": "success", "startedAt": "t1"},
                {"name": "Acc (floor-4.1.2)", "conclusion": "success", "startedAt": "t1"},
            ],
            2: [{"name": "Acc (custom-latest)", "conclusion": "failure", "startedAt": "t2", "databaseId": 9}],
        }
        states = rc.nightly_slot_states(runs, jobs)
        self.assertEqual(states["stable"][0], "fail")
        self.assertEqual(states["edge"][0], "pass")
        self.assertEqual(states["floor"][0], "pass")


if __name__ == "__main__":
    unittest.main()
