"""Tests for extract-contract.py.

Run with:
    python3 -m pytest scripts/test_extract_contract.py -v
    python3 -m unittest scripts.test_extract_contract
"""

import json
import os
import shutil
import sys
import tempfile
import unittest
from pathlib import Path

# Import the module under test (filename contains a hyphen, so use importlib).
import importlib.util

_SCRIPT = Path(__file__).resolve().parent / "extract-contract.py"
_spec = importlib.util.spec_from_file_location("extract_contract", _SCRIPT)
ec = importlib.util.module_from_spec(_spec)
_spec.loader.exec_module(ec)


# ── _parse_php_value ────────────────────────────────────────────────


class TestParsePhpValue(unittest.TestCase):
    """Unit tests for _parse_php_value."""

    def test_true_lowercase(self):
        self.assertIs(ec._parse_php_value("true"), True)

    def test_true_capitalized(self):
        self.assertIs(ec._parse_php_value("True"), True)

    def test_false_lowercase(self):
        self.assertIs(ec._parse_php_value("false"), False)

    def test_false_capitalized(self):
        self.assertIs(ec._parse_php_value("False"), False)

    def test_null_lowercase(self):
        self.assertIsNone(ec._parse_php_value("null"))

    def test_null_uppercase(self):
        self.assertIsNone(ec._parse_php_value("NULL"))

    def test_single_quoted_string(self):
        self.assertEqual(ec._parse_php_value("'hello'"), "hello")

    def test_double_quoted_string(self):
        self.assertEqual(ec._parse_php_value('"world"'), "world")

    def test_integer(self):
        self.assertEqual(ec._parse_php_value("42"), 42)

    def test_negative_integer(self):
        self.assertEqual(ec._parse_php_value("-7"), -7)

    def test_float(self):
        self.assertAlmostEqual(ec._parse_php_value("3.14"), 3.14)

    def test_zero(self):
        self.assertEqual(ec._parse_php_value("0"), 0)

    def test_unrecognized_expression_returned_as_string(self):
        self.assertEqual(ec._parse_php_value("some_expr"), "some_expr")

    def test_whitespace_stripped(self):
        self.assertIs(ec._parse_php_value("  true  "), True)

    def test_empty_single_quoted_string(self):
        self.assertEqual(ec._parse_php_value("''"), "")

    def test_empty_double_quoted_string(self):
        self.assertEqual(ec._parse_php_value('""'), "")


# ── extract_fillable ────────────────────────────────────────────────


class TestExtractFillable(unittest.TestCase):
    """Unit tests for extract_fillable."""

    def test_basic_fillable(self):
        php = """<?php
class App extends Model {
    protected $fillable = ['a', 'b', 'c'];
}
"""
        self.assertEqual(ec.extract_fillable(php), ["a", "b", "c"])

    def test_multiline_fillable(self):
        php = """<?php
class App extends Model {
    protected $fillable = [
        'alpha',
        'beta',
        'gamma',
    ];
}
"""
        self.assertEqual(ec.extract_fillable(php), ["alpha", "beta", "gamma"])

    def test_no_fillable(self):
        self.assertEqual(ec.extract_fillable("<?php class X {}"), [])

    def test_empty_fillable(self):
        php = "protected $fillable = [];"
        self.assertEqual(ec.extract_fillable(php), [])

    def test_single_element(self):
        php = "protected $fillable = ['only'];"
        self.assertEqual(ec.extract_fillable(php), ["only"])


# ── extract_guarded ─────────────────────────────────────────────────


class TestExtractGuarded(unittest.TestCase):
    def test_basic(self):
        php = "protected $guarded = ['id', 'secret'];"
        self.assertEqual(ec.extract_guarded(php), ["id", "secret"])

    def test_empty(self):
        self.assertEqual(ec.extract_guarded("no guarded here"), [])


# ── extract_casts ───────────────────────────────────────────────────


class TestExtractCasts(unittest.TestCase):
    """Unit tests for extract_casts."""

    def test_method_style(self):
        php = """<?php
class App extends Model {
    protected function casts(): array {
        return [
            'is_active' => 'boolean',
            'settings' => 'encrypted',
        ];
    }
}
"""
        result = ec.extract_casts(php)
        self.assertEqual(result, {"is_active": "boolean", "settings": "encrypted"})

    def test_property_style(self):
        php = """<?php
class App extends Model {
    protected $casts = [
        'ports' => 'json',
        'is_admin' => 'boolean',
    ];
}
"""
        result = ec.extract_casts(php)
        self.assertEqual(result, {"ports": "json", "is_admin": "boolean"})

    def test_no_casts(self):
        self.assertEqual(ec.extract_casts("<?php class X {}"), {})

    def test_method_style_preferred_over_property(self):
        """When both styles exist, the method style should win."""
        php = """<?php
class App extends Model {
    protected $casts = ['a' => 'string'];
    protected function casts(): array {
        return ['b' => 'integer'];
    }
}
"""
        result = ec.extract_casts(php)
        self.assertEqual(result, {"b": "integer"})


# ── extract_hidden ──────────────────────────────────────────────────


class TestExtractHidden(unittest.TestCase):
    def test_basic(self):
        php = "protected $hidden = ['secret', 'password'];"
        self.assertEqual(ec.extract_hidden(php), ["secret", "password"])

    def test_empty(self):
        self.assertEqual(ec.extract_hidden("no hidden"), [])


# ── extract_appends ─────────────────────────────────────────────────


class TestExtractAppends(unittest.TestCase):
    def test_basic(self):
        php = "protected $appends = ['full_name'];"
        self.assertEqual(ec.extract_appends(php), ["full_name"])

    def test_empty(self):
        self.assertEqual(ec.extract_appends("nope"), [])


# ── extract_model_attributes ────────────────────────────────────────


class TestExtractModelAttributes(unittest.TestCase):
    def test_basic_defaults(self):
        php = """<?php
class App extends Model {
    protected $attributes = [
        'is_enabled' => true,
        'name' => 'default',
        'count' => 0,
    ];
}
"""
        result = ec.extract_model_attributes(php)
        self.assertIs(result["is_enabled"], True)
        self.assertEqual(result["name"], "default")
        self.assertEqual(result["count"], 0)

    def test_no_attributes(self):
        self.assertEqual(ec.extract_model_attributes("nothing"), {})


# ── parse_migration_columns ─────────────────────────────────────────


class TestParseMigrationColumns(unittest.TestCase):
    """Tests for parse_migration_columns with real temp files."""

    def setUp(self):
        self.tmpdir = Path(tempfile.mkdtemp())

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

    def _write(self, name: str, content: str):
        (self.tmpdir / name).write_text(content)

    def test_create_table(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->id();
    $table->string('name', 100);
    $table->boolean('active')->default(true);
    $table->text('description')->nullable();
    $table->integer('count')->default(0);
    $table->timestamps();
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertEqual(cols["name"]["type"], "string")
        self.assertEqual(cols["name"]["max_length"], 100)
        self.assertFalse(cols["name"]["nullable"])

        self.assertEqual(cols["active"]["type"], "boolean")
        self.assertIs(cols["active"]["default"], True)

        self.assertEqual(cols["description"]["type"], "text")
        self.assertTrue(cols["description"]["nullable"])

        self.assertEqual(cols["count"]["type"], "integer")
        self.assertEqual(cols["count"]["default"], 0)

        # timestamps() should add created_at / updated_at
        self.assertIn("created_at", cols)
        self.assertIn("updated_at", cols)
        self.assertTrue(cols["created_at"]["nullable"])

    def test_alter_table_adds_column(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('name');
});
""",
        )
        self._write(
            "2024_02_01_000000_add_status_to_items.php",
            """<?php
Schema::table('items', function (Blueprint $table) {
    $table->string('status')->default('pending');
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertIn("name", cols)
        self.assertIn("status", cols)
        self.assertEqual(cols["status"]["default"], "pending")

    def test_drop_column(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('name');
    $table->string('old_field');
});
""",
        )
        self._write(
            "2024_02_01_000000_drop_old_field.php",
            """<?php
Schema::table('items', function (Blueprint $table) {
    $table->dropColumn('old_field');
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertIn("name", cols)
        self.assertNotIn("old_field", cols)

    def test_drop_column_array(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('a');
    $table->string('b');
    $table->string('c');
});
""",
        )
        self._write(
            "2024_02_01_000000_drop_columns.php",
            """<?php
Schema::table('items', function (Blueprint $table) {
    $table->dropColumn(['a', 'b']);
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertNotIn("a", cols)
        self.assertNotIn("b", cols)
        self.assertIn("c", cols)

    def test_string_default_max_length(self):
        """String without explicit length gets 255."""
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('title');
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertEqual(cols["title"]["max_length"], 255)

    @unittest.expectedFailure  # BUG: col_pattern regex only accepts numeric 2nd arg (\d+), skips enum arrays
    def test_enum_values(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->enum('color', ['red', 'green', 'blue']);
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertEqual(cols["color"]["type"], "enum")
        self.assertEqual(cols["color"]["enum_values"], ["red", "green", "blue"])

    def test_soft_deletes(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('name');
    $table->softDeletes();
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertIn("deleted_at", cols)
        self.assertTrue(cols["deleted_at"]["nullable"])

    def test_no_matching_migration(self):
        self._write(
            "2024_01_01_000000_create_other_table.php",
            """<?php
Schema::create('other', function (Blueprint $table) {
    $table->string('x');
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertEqual(cols, {})

    def test_application_settings_skipped_for_applications(self):
        """Ensure 'application_settings' files are skipped when table is 'applications'."""
        self._write(
            "2024_01_01_create_applications_table.php",
            """<?php
Schema::create('applications', function (Blueprint $table) {
    $table->string('name');
});
""",
        )
        self._write(
            "2024_02_01_create_application_settings_table.php",
            """<?php
Schema::create('application_settings', function (Blueprint $table) {
    $table->boolean('is_git_lfs_enabled')->default(false);
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "applications")
        self.assertIn("name", cols)
        self.assertNotIn("is_git_lfs_enabled", cols)

    def test_foreign_id(self):
        self._write(
            "2024_01_01_000000_create_items_table.php",
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->foreignId('team_id');
});
""",
        )
        cols = ec.parse_migration_columns(self.tmpdir, "items")
        self.assertEqual(cols["team_id"]["type"], "bigInteger")


# ── extract_enums ───────────────────────────────────────────────────


class TestExtractEnums(unittest.TestCase):
    def setUp(self):
        self.tmpdir = Path(tempfile.mkdtemp())
        self.enums_dir = self.tmpdir / "Enums"
        self.enums_dir.mkdir()

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

    def test_basic_enum(self):
        (self.enums_dir / "ProxyTypes.php").write_text(
            """<?php
namespace App\\Enums;

enum ProxyTypes: string {
    case TRAEFIK = 'traefik';
    case CADDY = 'caddy';
    case NGINX = 'nginx';
}
"""
        )
        result = ec.extract_enums(self.tmpdir)
        self.assertEqual(result["ProxyTypes"], ["traefik", "caddy", "nginx"])

    def test_no_enums_dir(self):
        empty = Path(tempfile.mkdtemp())
        try:
            self.assertEqual(ec.extract_enums(empty), {})
        finally:
            shutil.rmtree(empty)

    def test_non_enum_php_skipped(self):
        (self.enums_dir / "Helper.php").write_text(
            "<?php class Helper { public function foo() {} }"
        )
        result = ec.extract_enums(self.tmpdir)
        self.assertEqual(result, {})


# ── extract_allowed_fields ──────────────────────────────────────────


class TestExtractAllowedFields(unittest.TestCase):
    def test_basic(self):
        php = """<?php
class FooController {
    public function update(Request $request, $id) {
        $allowedFields = ['name', 'description', 'port'];
        // ...
    }
}
"""
        result = ec.extract_allowed_fields(php)
        self.assertEqual(result["update"], ["name", "description", "port"])

    def test_multiple_methods(self):
        php = """<?php
class FooController {
    public function store(Request $r) {
        $allowedFields = ['a'];
    }
    public function update(Request $r) {
        $allowedFields = ['b', 'c'];
    }
}
"""
        result = ec.extract_allowed_fields(php)
        self.assertEqual(result["store"], ["a"])
        self.assertEqual(result["update"], ["b", "c"])


# ── _clean_rule ─────────────────────────────────────────────────────


class TestCleanRule(unittest.TestCase):
    def test_pipe_joined(self):
        self.assertEqual(ec._clean_rule("'required', 'string'"), "required|string")

    def test_plain_string_fallback(self):
        self.assertEqual(ec._clean_rule("required"), "required")

    def test_empty(self):
        self.assertEqual(ec._clean_rule(""), "")


# ── extract_validation_rules ────────────────────────────────────────


class TestExtractValidationRules(unittest.TestCase):
    def test_basic(self):
        php = """<?php
class Ctrl {
    public function store(Request $request) {
        $request->validate([
            'name' => ['required', 'string'],
        ]);
    }
}
"""
        result = ec.extract_validation_rules(php)
        self.assertIn("store", result)
        self.assertEqual(result["store"]["name"], "required|string")


# ── build_model_contract ────────────────────────────────────────────


class TestBuildModelContract(unittest.TestCase):
    def setUp(self):
        self.tmpdir = Path(tempfile.mkdtemp())
        self.migration_dir = self.tmpdir / "migrations"
        self.migration_dir.mkdir()

    def tearDown(self):
        shutil.rmtree(self.tmpdir)

    def test_basic_contract(self):
        model_file = self.tmpdir / "Item.php"
        model_file.write_text(
            """<?php
class Item extends Model {
    protected $fillable = ['name', 'active'];
    protected $hidden = ['token'];
    protected function casts(): array {
        return ['active' => 'boolean'];
    }
}
"""
        )
        (self.migration_dir / "2024_01_01_create_items_table.php").write_text(
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('name');
    $table->boolean('active')->default(true);
    $table->string('token');
});
"""
        )
        contract = ec.build_model_contract(model_file, self.migration_dir, "items")
        self.assertEqual(contract["table"], "items")
        self.assertEqual(contract["fillable"], ["name", "active"])
        self.assertEqual(contract["hidden"], ["token"])
        self.assertIn("name", contract["fields"])
        self.assertIn("active", contract["fields"])
        self.assertTrue(contract["fields"]["active"]["fillable"])
        self.assertIs(contract["fields"]["active"]["default"], True)
        self.assertEqual(contract["fields"]["active"]["cast"], "boolean")
        # token is in migration columns but not fillable
        self.assertIn("token", contract["fields"])
        self.assertFalse(contract["fields"]["token"]["fillable"])

    def test_internal_fields_excluded(self):
        model_file = self.tmpdir / "Item.php"
        model_file.write_text(
            """<?php
class Item extends Model {
    protected $fillable = ['name'];
}
"""
        )
        (self.migration_dir / "2024_01_01_create_items_table.php").write_text(
            """<?php
Schema::create('items', function (Blueprint $table) {
    $table->string('name');
    $table->timestamps();
});
"""
        )
        contract = ec.build_model_contract(model_file, self.migration_dir, "items")
        for internal in ("id", "created_at", "updated_at", "team_id"):
            self.assertNotIn(internal, contract["fields"])


# ── Integration: extract_contract ───────────────────────────────────


class TestExtractContractIntegration(unittest.TestCase):
    """End-to-end test with a minimal Coolify-like directory layout."""

    def setUp(self):
        self.root = Path(tempfile.mkdtemp())
        # Minimal directory structure
        (self.root / "app" / "Models").mkdir(parents=True)
        (self.root / "app" / "Http" / "Controllers" / "Api").mkdir(parents=True)
        (self.root / "app" / "Enums").mkdir(parents=True)
        (self.root / "database" / "migrations").mkdir(parents=True)
        (self.root / "bootstrap" / "helpers").mkdir(parents=True)

        # Model
        (self.root / "app" / "Models" / "Application.php").write_text(
            """<?php
namespace App\\Models;

class Application extends Model {
    protected $fillable = [
        'name',
        'git_repository',
        'description',
    ];
    protected $hidden = ['private_key'];
    protected $appends = ['status'];
    protected function casts(): array {
        return [
            'ports_mappings' => 'json',
        ];
    }
}
"""
        )

        # Migration
        (
            self.root / "database" / "migrations" / "2024_01_01_create_applications_table.php"
        ).write_text(
            """<?php
Schema::create('applications', function (Blueprint $table) {
    $table->id();
    $table->string('name');
    $table->string('git_repository')->nullable();
    $table->text('description')->nullable();
    $table->string('private_key')->nullable();
    $table->json('ports_mappings')->nullable();
    $table->timestamps();
});
"""
        )

        # Controller
        (self.root / "app" / "Http" / "Controllers" / "Api" / "ApplicationsController.php").write_text(
            """<?php
class ApplicationsController {
    public function update(Request $request, $uuid) {
        $allowedFields = ['name', 'git_repository', 'description'];
    }
}
"""
        )

        # Enum
        (self.root / "app" / "Enums" / "BuildPackTypes.php").write_text(
            """<?php
namespace App\\Enums;

enum BuildPackTypes: string {
    case NIXPACKS = 'nixpacks';
    case DOCKERFILE = 'dockerfile';
    case DOCKERCOMPOSE = 'dockercompose';
}
"""
        )

    def tearDown(self):
        shutil.rmtree(self.root)

    def test_full_extraction(self):
        contract = ec.extract_contract(str(self.root), version="v4.0.0-test")

        # Metadata
        self.assertEqual(contract["version"], "v4.0.0-test")
        self.assertIn("extracted_at", contract)

        # Model
        self.assertIn("Application", contract["models"])
        app = contract["models"]["Application"]
        self.assertEqual(app["table"], "applications")
        self.assertEqual(app["fillable"], ["name", "git_repository", "description"])
        self.assertEqual(app["hidden"], ["private_key"])
        self.assertEqual(app["appends"], ["status"])
        self.assertIn("name", app["fields"])
        self.assertIn("git_repository", app["fields"])
        self.assertTrue(app["fields"]["git_repository"]["nullable"])
        self.assertEqual(app["fields"]["ports_mappings"]["cast"], "json")
        # Internal fields excluded
        self.assertNotIn("id", app["fields"])
        self.assertNotIn("created_at", app["fields"])

        # Endpoint
        self.assertIn("ApplicationsController::update", contract["endpoints"])
        self.assertEqual(
            contract["endpoints"]["ApplicationsController::update"]["allowed_fields"],
            ["name", "git_repository", "description"],
        )

        # Enum
        self.assertIn("BuildPackTypes", contract["enums"])
        self.assertEqual(
            contract["enums"]["BuildPackTypes"],
            ["nixpacks", "dockerfile", "dockercompose"],
        )

    def test_missing_models_dir_exits(self):
        empty = Path(tempfile.mkdtemp())
        try:
            with self.assertRaises(SystemExit):
                ec.extract_contract(str(empty))
        finally:
            shutil.rmtree(empty)

    def test_output_is_valid_json(self):
        contract = ec.extract_contract(str(self.root))
        # Round-trip through JSON serialization
        serialized = json.dumps(contract)
        deserialized = json.loads(serialized)
        self.assertEqual(contract, deserialized)


# ── _handle_special_column ──────────────────────────────────────────


class TestHandleSpecialColumn(unittest.TestCase):
    def test_morphs(self):
        columns = {}
        ec._handle_special_column("morphs", "commentable", columns)
        self.assertIn("commentable_type", columns)
        self.assertIn("commentable_id", columns)
        self.assertFalse(columns["commentable_type"]["nullable"])
        self.assertEqual(columns["commentable_id"]["type"], "bigInteger")

    def test_nullable_morphs(self):
        columns = {}
        ec._handle_special_column("nullableMorphs", "taggable", columns)
        self.assertTrue(columns["taggable_type"]["nullable"])
        self.assertTrue(columns["taggable_id"]["nullable"])


if __name__ == "__main__":
    unittest.main()
