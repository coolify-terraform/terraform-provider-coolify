"""Tests for generate-openapi.py."""

import importlib.util
import unittest
from pathlib import Path

# Import the script as a module (filename contains hyphens).
_spec = importlib.util.spec_from_file_location(
    "generate_openapi", Path(__file__).parent / "generate-openapi.py"
)
gen = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(gen)


# ---------------------------------------------------------------------------
# _build_property
# ---------------------------------------------------------------------------


class TestBuildProperty(unittest.TestCase):
    def test_string_field(self):
        prop = gen._build_property("user_name", {"type": "string"})
        self.assertEqual(prop["type"], "string")
        self.assertEqual(prop["description"], "User name.")

    def test_boolean_field(self):
        prop = gen._build_property("is_active", {"type": "boolean"})
        self.assertEqual(prop["type"], "boolean")

    def test_integer_types(self):
        for ct in ("integer", "bigInteger", "smallInteger", "tinyInteger"):
            prop = gen._build_property("count", {"type": ct})
            self.assertEqual(prop["type"], "integer", f"failed for {ct}")

    def test_float_types(self):
        for ct in ("float", "decimal"):
            prop = gen._build_property("amount", {"type": ct})
            self.assertEqual(prop["type"], "number", f"failed for {ct}")

    def test_text_maps_to_string(self):
        for ct in ("text", "longText"):
            prop = gen._build_property("body", {"type": ct})
            self.assertEqual(prop["type"], "string", f"failed for {ct}")

    def test_timestamp_has_format(self):
        prop = gen._build_property("created_at", {"type": "timestamp"})
        self.assertEqual(prop["type"], "string")
        self.assertEqual(prop["format"], "date-time")

    def test_json_maps_to_object(self):
        prop = gen._build_property("metadata", {"type": "json"})
        self.assertEqual(prop["type"], "object")

    def test_enum_maps_to_string(self):
        prop = gen._build_property("status", {"type": "enum"})
        self.assertEqual(prop["type"], "string")

    def test_default_value_included(self):
        prop = gen._build_property("port", {"type": "integer", "default": 22})
        self.assertEqual(prop["default"], 22)

    def test_no_default_when_none(self):
        prop = gen._build_property("port", {"type": "integer"})
        self.assertNotIn("default", prop)

    def test_enum_values_included(self):
        prop = gen._build_property(
            "status", {"type": "enum", "enum_values": ["running", "stopped"]}
        )
        self.assertEqual(prop["enum"], ["running", "stopped"])

    def test_max_length_for_string(self):
        prop = gen._build_property("name", {"type": "string", "max_length": 255})
        self.assertEqual(prop["maxLength"], 255)

    def test_max_length_not_added_for_non_string(self):
        prop = gen._build_property("count", {"type": "integer", "max_length": 10})
        self.assertNotIn("maxLength", prop)

    def test_unknown_type_defaults_to_string(self):
        prop = gen._build_property("weird", {"type": "unknownType"})
        self.assertEqual(prop["type"], "string")


# ---------------------------------------------------------------------------
# _patch_property
# ---------------------------------------------------------------------------


class TestPatchProperty(unittest.TestCase):
    def test_fixes_type(self):
        prop = {"type": "string"}
        gen._patch_property(prop, {"type": "integer"})
        self.assertEqual(prop["type"], "integer")

    def test_adds_default(self):
        prop = {"type": "integer"}
        gen._patch_property(prop, {"type": "integer", "default": 3600})
        self.assertEqual(prop["default"], 3600)

    def test_adds_enum(self):
        prop = {"type": "string"}
        gen._patch_property(
            prop, {"type": "enum", "enum_values": ["a", "b"]}
        )
        self.assertEqual(prop["enum"], ["a", "b"])

    def test_timestamp_format(self):
        prop = {"type": "string"}
        gen._patch_property(prop, {"type": "timestamp"})
        self.assertEqual(prop["format"], "date-time")

    def test_preserves_existing_nullable(self):
        prop = {"type": "string", "nullable": True}
        gen._patch_property(prop, {"type": "boolean"})
        self.assertEqual(prop["type"], "boolean")
        self.assertTrue(prop["nullable"])


# ---------------------------------------------------------------------------
# build_schema_from_contract
# ---------------------------------------------------------------------------


class TestBuildSchemaFromContract(unittest.TestCase):
    def test_basic_model(self):
        model = {
            "fields": {
                "uuid": {"type": "string"},
                "name": {"type": "string"},
                "port": {"type": "integer", "nullable": True},
            }
        }
        schema = gen.build_schema_from_contract(model)
        self.assertEqual(schema["type"], "object")
        self.assertIn("uuid", schema["properties"])
        self.assertIn("name", schema["properties"])
        self.assertIn("port", schema["properties"])
        # nullable field gets nullable annotation
        self.assertTrue(schema["properties"]["port"]["nullable"])
        # non-nullable field does not
        self.assertNotIn("nullable", schema["properties"]["uuid"])

    def test_properties_sorted(self):
        model = {
            "fields": {
                "zebra": {"type": "string"},
                "alpha": {"type": "string"},
                "middle": {"type": "string"},
            }
        }
        schema = gen.build_schema_from_contract(model)
        keys = list(schema["properties"].keys())
        self.assertEqual(keys, ["alpha", "middle", "zebra"])

    def test_settings_fields_merged(self):
        model = {
            "fields": {"uuid": {"type": "string"}},
            "settings_fields": {"is_enabled": {"type": "boolean"}},
        }
        schema = gen.build_schema_from_contract(model)
        self.assertIn("uuid", schema["properties"])
        self.assertIn("is_enabled", schema["properties"])

    def test_settings_fields_override_fields(self):
        model = {
            "fields": {"port": {"type": "string"}},
            "settings_fields": {"port": {"type": "integer"}},
        }
        schema = gen.build_schema_from_contract(model)
        # settings_fields wins because {**fields, **settings_fields}
        self.assertEqual(schema["properties"]["port"]["type"], "integer")

    def test_empty_fields(self):
        model = {"fields": {}}
        schema = gen.build_schema_from_contract(model)
        self.assertEqual(schema["type"], "object")
        self.assertEqual(schema["properties"], {})

    def test_missing_fields_key(self):
        model = {}
        schema = gen.build_schema_from_contract(model)
        self.assertEqual(schema["properties"], {})


# ---------------------------------------------------------------------------
# patch_schema
# ---------------------------------------------------------------------------


class TestPatchSchema(unittest.TestCase):
    def test_patches_existing_property(self):
        schema = {"properties": {"port": {"type": "string"}}}
        model = {"fields": {"port": {"type": "integer", "default": 22}}}
        result = gen.patch_schema(schema, model)
        self.assertEqual(result["properties"]["port"]["type"], "integer")
        self.assertEqual(result["properties"]["port"]["default"], 22)

    def test_adds_missing_property(self):
        schema = {"properties": {"uuid": {"type": "string"}}}
        model = {"fields": {"name": {"type": "string"}}}
        result = gen.patch_schema(schema, model)
        self.assertIn("name", result["properties"])

    def test_merges_settings_fields(self):
        schema = {"properties": {}}
        model = {
            "fields": {"uuid": {"type": "string"}},
            "settings_fields": {"concurrent_builds": {"type": "integer"}},
        }
        result = gen.patch_schema(schema, model)
        self.assertIn("uuid", result["properties"])
        self.assertIn("concurrent_builds", result["properties"])

    def test_removes_required(self):
        schema = {"properties": {"uuid": {"type": "string"}}, "required": ["uuid"]}
        model = {"fields": {"uuid": {"type": "string"}}}
        result = gen.patch_schema(schema, model)
        self.assertNotIn("required", result)

    def test_properties_sorted(self):
        schema = {"properties": {"z_field": {"type": "string"}}}
        model = {"fields": {"a_field": {"type": "string"}, "z_field": {"type": "string"}}}
        result = gen.patch_schema(schema, model)
        keys = list(result["properties"].keys())
        self.assertEqual(keys, sorted(keys))


# ---------------------------------------------------------------------------
# patch_request_bodies
# ---------------------------------------------------------------------------


class TestPatchRequestBodies(unittest.TestCase):
    def test_removes_environment_uuid_from_required(self):
        spec = {
            "paths": {
                "/api/v1/applications": {
                    "post": {
                        "requestBody": {
                            "content": {
                                "application/json": {
                                    "schema": {
                                        "required": [
                                            "name",
                                            "environment_uuid",
                                            "server_uuid",
                                        ]
                                    }
                                }
                            }
                        }
                    }
                }
            }
        }
        gen.patch_request_bodies(spec, {})
        schema = spec["paths"]["/api/v1/applications"]["post"]["requestBody"][
            "content"
        ]["application/json"]["schema"]
        self.assertNotIn("environment_uuid", schema["required"])
        self.assertIn("name", schema["required"])

    def test_removes_required_key_when_empty(self):
        spec = {
            "paths": {
                "/api/v1/test": {
                    "post": {
                        "requestBody": {
                            "content": {
                                "application/json": {
                                    "schema": {"required": ["environment_uuid"]}
                                }
                            }
                        }
                    }
                }
            }
        }
        gen.patch_request_bodies(spec, {})
        schema = spec["paths"]["/api/v1/test"]["post"]["requestBody"]["content"][
            "application/json"
        ]["schema"]
        self.assertNotIn("required", schema)

    def test_no_op_without_environment_uuid(self):
        spec = {
            "paths": {
                "/api/v1/test": {
                    "post": {
                        "requestBody": {
                            "content": {
                                "application/json": {
                                    "schema": {"required": ["name"]}
                                }
                            }
                        }
                    }
                }
            }
        }
        gen.patch_request_bodies(spec, {})
        schema = spec["paths"]["/api/v1/test"]["post"]["requestBody"]["content"][
            "application/json"
        ]["schema"]
        self.assertEqual(schema["required"], ["name"])

    def test_handles_missing_request_body(self):
        spec = {"paths": {"/api/v1/test": {"get": {}}}}
        gen.patch_request_bodies(spec, {})  # should not raise


# ---------------------------------------------------------------------------
# TYPE_MAP coverage
# ---------------------------------------------------------------------------


class TestTypeMap(unittest.TestCase):
    def test_all_contract_types_mapped(self):
        expected_types = {
            "string", "text", "longText", "boolean", "integer",
            "bigInteger", "smallInteger", "tinyInteger", "float",
            "decimal", "timestamp", "json", "enum",
        }
        self.assertEqual(set(gen.TYPE_MAP.keys()), expected_types)


if __name__ == "__main__":
    unittest.main()
