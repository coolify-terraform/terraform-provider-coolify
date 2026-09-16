#!/usr/bin/env python3
"""Lock tfplugindocs version extraction from tools/go.mod."""

from __future__ import annotations

import os
import subprocess
import tempfile
import unittest
from pathlib import Path

ROOT = Path(__file__).resolve().parents[1]
SCRIPT = ROOT / "scripts" / "tfplugindocs-mod-version.sh"
MAKEFILE = ROOT / "GNUmakefile"
TOOLS_GOMOD = ROOT / "tools" / "go.mod"

# Old Makefile awk: print $2 on the first matching line.
# Single-line require makes $2 the module path, not the version.
OLD_AWK = r'/terraform-plugin-docs v[0-9]/ {print $2; exit}'


def run_script(go_mod: Path) -> subprocess.CompletedProcess[str]:
    return subprocess.run(
        ["bash", str(SCRIPT), str(go_mod)],
        cwd=ROOT,
        capture_output=True,
        text=True,
    )


def write_gomod(text: str) -> Path:
    tmp = tempfile.NamedTemporaryFile(
        mode="w",
        suffix=".mod",
        delete=False,
        encoding="utf-8",
    )
    tmp.write(text)
    tmp.close()
    return Path(tmp.name)


class TestTfplugindocsModVersion(unittest.TestCase):
    def tearDown(self) -> None:
        leftover = getattr(self, "_tmp", None)
        if leftover is not None and leftover.exists():
            leftover.unlink()

    def _mod(self, text: str) -> Path:
        path = write_gomod(text)
        self._tmp = path
        return path

    def test_missing_arg_exits_2(self) -> None:
        proc = subprocess.run(
            ["bash", str(SCRIPT)],
            cwd=ROOT,
            capture_output=True,
            text=True,
        )
        self.assertEqual(proc.returncode, 2)
        self.assertIn("usage", proc.stderr)

    def test_single_line_require(self) -> None:
        path = self._mod(
            "module example\n\n"
            "require github.com/hashicorp/terraform-plugin-docs v0.25.0\n"
        )
        proc = run_script(path)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "0.25.0")

    def test_block_style_require(self) -> None:
        path = self._mod(
            "module example\n\nrequire (\n"
            "\tgithub.com/hashicorp/terraform-plugin-docs v0.25.0\n"
            ")\n"
        )
        proc = run_script(path)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "0.25.0")

    def test_missing_package_prints_nothing(self) -> None:
        path = self._mod("module example\n\nrequire github.com/other/mod v1.0.0\n")
        proc = run_script(path)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        self.assertEqual(proc.stdout.strip(), "")

    def test_old_field_two_awk_is_module_path_on_single_line(self) -> None:
        path = self._mod(
            "module example\n\n"
            "require github.com/hashicorp/terraform-plugin-docs v0.25.0\n"
        )
        proc = subprocess.run(
            ["awk", OLD_AWK, str(path)],
            capture_output=True,
            text=True,
            check=True,
        )
        self.assertEqual(
            proc.stdout.strip(),
            "github.com/hashicorp/terraform-plugin-docs",
        )

    def test_old_field_two_awk_works_on_block_style(self) -> None:
        path = self._mod(
            "module example\n\nrequire (\n"
            "\tgithub.com/hashicorp/terraform-plugin-docs v0.25.0\n"
            ")\n"
        )
        proc = subprocess.run(
            ["awk", OLD_AWK, str(path)],
            capture_output=True,
            text=True,
            check=True,
        )
        self.assertEqual(proc.stdout.strip(), "v0.25.0")

    def test_tools_gomod_matches_script(self) -> None:
        proc = run_script(TOOLS_GOMOD)
        self.assertEqual(proc.returncode, 0, proc.stderr)
        version = proc.stdout.strip()
        self.assertRegex(version, r"^\d+\.\d+\.\d+")
        body = TOOLS_GOMOD.read_text(encoding="utf-8")
        self.assertIn(f"terraform-plugin-docs v{version}", body)

    def test_makefile_uses_script_and_command_v(self) -> None:
        text = MAKEFILE.read_text(encoding="utf-8")
        self.assertIn("scripts/tfplugindocs-mod-version.sh tools/go.mod", text)
        self.assertIn("command -v tfplugindocs", text)
        self.assertIn("Plain go install prints Version dev", text)
        self.assertNotIn(
            "awk '/terraform-plugin-docs",
            text,
            "version awk must live in the script, not inline in the Makefile",
        )


class TestCheckTfplugindocsMissingBinary(unittest.TestCase):
    def test_missing_binary_says_not_found(self) -> None:
        # Command-line PATH overrides Makefile export so BIN_DIR is not
        # prepended. /usr/bin:/bin is enough for /usr/bin/make on macOS.
        proc = subprocess.run(
            ["make", "PATH=/usr/bin:/bin", "check-tfplugindocs"],
            cwd=ROOT,
            capture_output=True,
            text=True,
            env={**os.environ, "PATH": "/usr/bin:/bin"},
        )
        combined = proc.stdout + proc.stderr
        self.assertNotEqual(proc.returncode, 0, combined)
        self.assertIn("Installed: not found", combined)
        self.assertNotRegex(combined, r"Installed: found\b")
