#!/usr/bin/env python3
"""Lock Recipe A/E/G triggers and the notes-branch apply path."""

from __future__ import annotations

import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
WORKFLOWS = ROOT / ".github" / "workflows"


def _on_block(text: str) -> str:
    start = text.index("\non:")
    rest = text[start + 1 :]
    end = rest.index("\njobs:")
    block = rest[:end]
    lines = []
    for line in block.splitlines():
        stripped = line.split("#", 1)[0].rstrip()
        if stripped:
            lines.append(stripped)
    return "\n".join(lines)


class WorkflowTriggerTests(unittest.TestCase):
    def test_ci_has_no_push_compile(self) -> None:
        on_block = _on_block((WORKFLOWS / "ci.yml").read_text(encoding="utf-8"))
        self.assertIn("pull_request:", on_block)
        self.assertIn("workflow_dispatch:", on_block)
        self.assertIn("schedule:", on_block)
        self.assertNotIn("push:", on_block)
        self.assertNotIn("tags:", on_block)
        self.assertNotIn("merge_group:", on_block)

    def test_codeql_has_no_push_compile(self) -> None:
        on_block = _on_block((WORKFLOWS / "codeql.yml").read_text(encoding="utf-8"))
        self.assertIn("pull_request:", on_block)
        self.assertIn("workflow_dispatch:", on_block)
        self.assertIn("schedule:", on_block)
        self.assertNotIn("push:", on_block)
        self.assertNotIn("tags:", on_block)

    def test_release_is_push_main_or_dispatch(self) -> None:
        text = (WORKFLOWS / "release.yml").read_text(encoding="utf-8")
        on_block = _on_block(text)
        self.assertIn("push:", on_block)
        self.assertIn("branches: [main]", on_block)
        self.assertIn("workflow_dispatch:", on_block)
        self.assertNotIn("pull_request:", on_block)
        self.assertNotIn("workflow_run:", on_block)
        self.assertNotIn("tags:", on_block)
        self.assertNotRegex(text, r"\bgo test\b")
        self.assertNotIn("chore/cleanup-release-notes", text)
        self.assertIn("scripts/apply-release-notes.sh", text)
        self.assertIn("continue-on-error: true", text)

    def test_release_please_job_does_not_compile(self) -> None:
        text = (WORKFLOWS / "release.yml").read_text(encoding="utf-8")
        start = text.index("name: Release Please")
        end = text.index("name: Release\n", start + 1)
        job = text[start:end]
        self.assertNotIn("go build", job)
        self.assertNotIn("go test", job)
        self.assertNotIn("goreleaser", job.lower())
        self.assertIn("googleapis/release-please-action@", job)

    def test_cache_go_is_cheap_main_job(self) -> None:
        text = (WORKFLOWS / "release.yml").read_text(encoding="utf-8")
        self.assertIn("name: Cache Go modules", text)
        start = text.index("name: Cache Go modules")
        end = text.index("name: Release Please")
        job = text[start:end]
        self.assertIn("cache: true", job)
        self.assertNotIn("go build", job)
        self.assertNotIn("go test", job)
        self.assertIn("github.event_name == 'push'", job)

    def test_apply_release_notes_is_dispatch_only(self) -> None:
        text = (WORKFLOWS / "apply-release-notes.yml").read_text(encoding="utf-8")
        on_block = _on_block(text)
        self.assertIn("workflow_dispatch:", on_block)
        self.assertNotIn("pull_request:", on_block)
        self.assertNotIn("push:", on_block)
        self.assertNotIn("go build", text)
        self.assertNotIn("goreleaser-action", text)
        self.assertIn("scripts/apply-release-notes.sh", text)

    def test_auto_approve_does_not_merge_release_prs(self) -> None:
        text = (WORKFLOWS / "auto-approve.yml").read_text(encoding="utf-8")
        self.assertNotIn("gh pr merge", text)
        self.assertNotIn("pulls.merge", text)

    def test_codeql_standin_on_release_please(self) -> None:
        sec = (WORKFLOWS / "codeql.yml").read_text(encoding="utf-8")
        self.assertIn("startsWith(github.head_ref, 'release-please')", sec)
        self.assertIn(
            "echo 'version-bump PR; CodeQL already ran on the feature PR'",
            sec,
        )
        for pin in (
            "github/codeql-action/init@",
            "github/codeql-action/analyze@",
        ):
            idx = sec.index(pin)
            window = sec[max(0, idx - 400) : idx]
            self.assertIn(
                "!startsWith(github.head_ref, 'release-please')", window, pin
            )

    def test_required_ci_gate_stays_named(self) -> None:
        ci = (WORKFLOWS / "ci.yml").read_text(encoding="utf-8")
        self.assertIn("name: CI", ci)
        self.assertIn("name: DCO", ci)
        self.assertNotIn("RELEASE_NOTES.md", ci)

    def test_ci_zizmor_pip_is_hash_pinned(self) -> None:
        # Scorecard PinnedDependenciesID (#36) flags unhashed pip install.
        ci = (WORKFLOWS / "ci.yml").read_text(encoding="utf-8")
        self.assertIn(
            "pip install --user --require-hashes -r .github/requirements/zizmor.txt",
            ci,
        )
        self.assertNotIn('pip install --user "zizmor==', ci)
        req = ROOT / ".github" / "requirements" / "zizmor.txt"
        self.assertTrue(req.is_file(), req)
        body = req.read_text(encoding="utf-8")
        self.assertIn("zizmor==1.16.1", body)
        self.assertIn("--hash=sha256:", body)

    def test_optional_jobs_skip_release_please(self) -> None:
        ci = (WORKFLOWS / "ci.yml").read_text(encoding="utf-8")
        acc = ci[ci.index("name: Acceptance Tests") :]
        scenarios = ci[ci.index("name: Scenario Tests") : ci.index("name: Acceptance Tests")]
        self.assertIn(
            "startsWith(github.head_ref, 'release-please')", acc.split("steps:")[0]
        )
        self.assertIn(
            "startsWith(github.head_ref, 'release-please')",
            scenarios.split("steps:")[0],
        )
        fossa = (WORKFLOWS / "fossa.yml").read_text(encoding="utf-8")
        self.assertIn("startsWith(github.head_ref, 'release-please')", fossa)


if __name__ == "__main__":
    unittest.main()
