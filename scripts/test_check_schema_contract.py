#!/usr/bin/env python3
"""Unit tests for scripts/check-schema-contract.py."""

import importlib.util
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "check-schema-contract.py"
_spec = importlib.util.spec_from_file_location("check_schema_contract", _SCRIPT)
sc = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(sc)


SCHEMA = '''
"enable_ipv6": schema.BoolAttribute{
    Optional: true,
    PlanModifiers: []planmodifier.Bool{
        boolplanmodifier.RequiresReplace(),
    },
},
"monitoring": schema.BoolAttribute{
    Optional: true,
    Default: booldefault.StaticBool(true),
},
'''


class TestCreateOnly(unittest.TestCase):
    def test_requires_replace_present(self):
        self.assertEqual(
            sc.missing_requires_replace(SCHEMA, ["enable_ipv6"]),
            [],
        )

    def test_requires_replace_missing(self):
        self.assertEqual(
            sc.missing_requires_replace(SCHEMA, ["monitoring"]),
            ["monitoring"],
        )

    def test_field_not_in_schema_is_skipped(self):
        self.assertEqual(
            sc.missing_requires_replace(SCHEMA, ["cloud_provider_token_id"]),
            [],
        )


class TestVolumeBackup(unittest.TestCase):
    def test_schema_default_is_forbidden(self):
        self.assertTrue(sc.forbidden_schema_defaults(SCHEMA))
        self.assertFalse(sc.forbidden_schema_defaults("Optional: true,\n"))

    def test_import_ignore_must_list_non_identity_attrs(self):
        src = '''
        "storage_uuid": schema.StringAttribute{Required: true},
        "timeout": schema.Int64Attribute{Optional: true},
        '''
        self.assertEqual(
            sc.missing_import_ignores(src, ["uuid"], {"storage_uuid"}),
            ["timeout"],
        )
        self.assertEqual(
            sc.missing_import_ignores(src, ["timeout"], {"storage_uuid"}),
            [],
        )


class TestRepo(unittest.TestCase):
    def test_current_tree_passes(self):
        self.assertEqual(sc.check_repo(), [])


if __name__ == "__main__":
    unittest.main()
