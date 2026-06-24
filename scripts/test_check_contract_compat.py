"""Tests for check-contract-compat.py.

Run with:
    python3 -m pytest scripts/test_check_contract_compat.py -v
    python3 -m unittest scripts.test_check_contract_compat
"""

import importlib.util
import json
import tempfile
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "check-contract-compat.py"
_spec = importlib.util.spec_from_file_location("check_contract_compat", _SCRIPT)
cc = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(cc)


class TestParseVersion(unittest.TestCase):
    def test_simple(self):
        self.assertEqual(cc._parse_version("coolify-v4.1.2"), (4, 1, 2))

    def test_major_only(self):
        self.assertEqual(cc._parse_version("coolify-v4"), (4,))

    def test_no_match(self):
        self.assertEqual(cc._parse_version("unknown"), (0,))


class TestCompareEndpoints(unittest.TestCase):
    def _make_contract(self, endpoints: dict) -> dict:
        return {"version": "test", "endpoints": endpoints}

    def test_identical_fields(self):
        c1 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name", "desc"]},
        })
        c2 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name", "desc"]},
        })
        results = cc.compare_endpoints({"v4.1.0": c1, "v4.1.2": c2})
        self.assertEqual(results, [])

    def test_new_field_detected(self):
        c1 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name"]},
        })
        c2 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name", "color"]},
        })
        results = cc.compare_endpoints({"v4.1.0": c1, "v4.1.2": c2})
        self.assertEqual(len(results), 1)
        self.assertEqual(results[0]["endpoint"], "Ctrl::update")
        fields = [f["field"] for f in results[0]["version_dependent"]]
        self.assertEqual(fields, ["color"])

    def test_removed_field_detected(self):
        c1 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name", "old_field"]},
        })
        c2 = self._make_contract({
            "Ctrl::update": {"allowed_fields": ["name"]},
        })
        results = cc.compare_endpoints({"v4.1.0": c1, "v4.1.2": c2})
        self.assertEqual(len(results), 1)
        fields = [f["field"] for f in results[0]["version_dependent"]]
        self.assertEqual(fields, ["old_field"])

    def test_endpoint_in_one_version_only(self):
        c1 = self._make_contract({})
        c2 = self._make_contract({
            "Ctrl::create": {"allowed_fields": ["name"]},
        })
        # Endpoint only in one version, fewer than 2 versions, skipped
        results = cc.compare_endpoints({"v4.1.0": c1, "v4.1.2": c2})
        self.assertEqual(results, [])


if __name__ == "__main__":
    unittest.main()