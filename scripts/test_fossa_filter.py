"""Tests for fossa-filter.py."""

import json
import os
import sys
import tempfile
import unittest

# Add scripts directory to path
sys.path.insert(0, os.path.dirname(__file__))
from importlib import import_module

fossa_filter = import_module("fossa-filter")


class TestFossaFilter(unittest.TestCase):
    """Test the FOSSA false-positive filter."""

    def _run_filter(self, issues):
        """Write issues to a temp file and run main()."""
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(issues, f)
            f.flush()
            old_argv = sys.argv
            sys.argv = ["fossa-filter.py", f.name]
            try:
                return fossa_filter.main()
            finally:
                sys.argv = old_argv
                os.unlink(f.name)

    def test_all_false_positives_pass(self):
        """All known false positives should result in exit 0."""
        issues = [
            {"package": "golang.org/x/text", "license": "CC-BY-SA-3.0", "type": "policy_conflict"},
            {"package": "golang.org/x/text", "license": "CC-BY-SA-1.0", "type": "policy_conflict"},
            {"package": "golang.org/x/crypto", "license": "openssl-ssleay", "type": "policy_flag"},
            {"package": "github.com/hashicorp/terraform-plugin-framework", "license": "MPL-2.0", "type": "policy_flag"},
            {"package": "github.com/hashicorp/go-cleanhttp", "license": "MPL-2.0", "type": "policy_flag"},
        ]
        self.assertEqual(self._run_filter(issues), 0)

    def test_genuine_issue_fails(self):
        """A non-whitelisted issue should result in exit 1."""
        issues = [
            {"package": "golang.org/x/text", "license": "CC-BY-SA-3.0", "type": "policy_conflict"},
            {"package": "sketchy-package", "license": "AGPL-3.0", "type": "policy_conflict"},
        ]
        self.assertEqual(self._run_filter(issues), 1)

    def test_empty_issues_pass(self):
        """No issues should result in exit 0."""
        self.assertEqual(self._run_filter([]), 0)

    def test_object_wrapper(self):
        """Issues wrapped in an object with 'issues' key should work."""
        data = {"issues": [
            {"package": "github.com/hashicorp/cli", "license": "MPL-2.0", "type": "policy_flag"},
        ]}
        with tempfile.NamedTemporaryFile(mode="w", suffix=".json", delete=False) as f:
            json.dump(data, f)
            f.flush()
            old_argv = sys.argv
            sys.argv = ["fossa-filter.py", f.name]
            try:
                result = fossa_filter.main()
            finally:
                sys.argv = old_argv
                os.unlink(f.name)
        self.assertEqual(result, 0)

    def test_is_false_positive(self):
        """Unit test the is_false_positive function directly."""
        self.assertTrue(fossa_filter.is_false_positive("golang.org/x/text", "CC-BY-SA-3.0"))
        self.assertTrue(fossa_filter.is_false_positive("golang.org/x/text", "CC-BY-SA-4.0"))
        self.assertTrue(fossa_filter.is_false_positive("golang.org/x/crypto", "openssl-ssleay"))
        self.assertTrue(fossa_filter.is_false_positive("github.com/hashicorp/terraform-plugin-go", "MPL-2.0"))
        self.assertFalse(fossa_filter.is_false_positive("sketchy-package", "GPL-3.0"))
        self.assertFalse(fossa_filter.is_false_positive("golang.org/x/text", "MIT"))


if __name__ == "__main__":
    unittest.main()
