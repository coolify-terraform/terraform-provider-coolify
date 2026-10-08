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

    def test_custom_replace_modifier_counts(self):
        src = '''
        "environment_name": schema.StringAttribute{
            PlanModifiers: []planmodifier.String{flex.EnvironmentNamePlan()},
        },
        '''
        self.assertEqual(sc.missing_requires_replace(src, ["environment_name"]), [])

    def test_no_replace_phrase_is_exempt(self):
        src = '''
        "autogenerate_domain": schema.BoolAttribute{
            Default: booldefault.StaticBool(true),
            MarkdownDescription: "changing this after create has no effect and does not force replacement.",
        },
        '''
        self.assertEqual(sc.missing_requires_replace(src, ["autogenerate_domain"]), [])


class TestDefaultReplace(unittest.TestCase):
    def test_missing_read_fill_fails(self):
        schema = '''
        "enable_ipv4": schema.BoolAttribute{
            Default: booldefault.StaticBool(true),
            PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
        },
        '''
        package = 'EnableIPv4 types.Bool `tfsdk:"enable_ipv4"`\n'
        self.assertNotEqual(sc.default_replace_gaps(schema, package), [])

    def test_matching_null_fill_passes(self):
        schema = '''
        "enable_ipv4": schema.BoolAttribute{
            Default: booldefault.StaticBool(true),
            PlanModifiers: []planmodifier.Bool{boolplanmodifier.RequiresReplace()},
        },
        '''
        package = '''
        EnableIPv4 types.Bool `tfsdk:"enable_ipv4"`
        if model.EnableIPv4.IsNull() {
            model.EnableIPv4 = types.BoolValue(true)
        }
        '''
        self.assertEqual(sc.default_replace_gaps(schema, package), [])


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


class TestDescriptionFlatten(unittest.TestCase):
    def test_string_to_framework_on_description_fails(self):
        src = 'model.Description = flex.StringToFramework(key.Description)\n'
        self.assertEqual(
            sc.description_string_to_framework_lines(src),
            [1],
        )

    def test_string_from_api_and_comments_pass(self):
        src = '''
// Description uses StringToFramework in older code.
model.Description = flex.StringFromAPI(key.Description, model.Description)
model.PublicKey = flex.StringToFramework(key.PublicKey)
'''
        self.assertEqual(sc.description_string_to_framework_lines(src), [])


class TestRepo(unittest.TestCase):
    def test_current_tree_passes(self):
        self.assertEqual(sc.check_repo(), [])


if __name__ == "__main__":
    unittest.main()
