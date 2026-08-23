"""Tests for upload-social-preview.py helpers."""

from __future__ import annotations

import importlib.util
import sys
import unittest
from pathlib import Path

_SCRIPT = Path(__file__).resolve().parent / "upload-social-preview.py"
_spec = importlib.util.spec_from_file_location("upload_social_preview", _SCRIPT)
usp = importlib.util.module_from_spec(_spec)
sys.modules["upload_social_preview"] = usp
assert _spec.loader is not None
_spec.loader.exec_module(usp)


class TestIsAuthGate(unittest.TestCase):
    def test_login_url(self) -> None:
        self.assertTrue(usp.is_auth_gate("https://github.com/login", ""))

    def test_confirm_access_heading(self) -> None:
        self.assertTrue(
            usp.is_auth_gate(
                "https://github.com/coolify-terraform/terraform-provider-coolify/settings",
                "Confirm access",
            )
        )

    def test_settings_ok(self) -> None:
        self.assertFalse(
            usp.is_auth_gate(
                "https://github.com/coolify-terraform/terraform-provider-coolify/settings",
                "Settings: coolify-terraform/terraform-provider-coolify",
            )
        )


class TestIsConfirmResponse(unittest.TestCase):
    def test_put_200(self) -> None:
        self.assertTrue(
            usp.is_confirm_response(
                "PUT",
                "https://github.com/upload/repository-images/1969309",
                200,
            )
        )

    def test_policy_post_is_not_confirm(self) -> None:
        self.assertFalse(
            usp.is_confirm_response(
                "POST",
                "https://github.com/upload/policies/repository-images",
                201,
            )
        )

    def test_put_error_is_not_confirm(self) -> None:
        self.assertFalse(
            usp.is_confirm_response(
                "PUT",
                "https://github.com/upload/repository-images/1",
                422,
            )
        )


class TestPublicPreviewOk(unittest.TestCase):
    def test_png_200(self) -> None:
        self.assertTrue(usp.public_preview_ok(200, "image/png"))

    def test_azure_404_html_is_not_proof(self) -> None:
        self.assertFalse(usp.public_preview_ok(404, "text/html"))

    def test_og_url_host_alone_is_not_enough(self) -> None:
        # Callers must check HTTP status, not only "repository-images" in URL.
        self.assertFalse(usp.public_preview_ok(200, "text/html"))


class TestExtractConfirmHref(unittest.TestCase):
    def test_href(self) -> None:
        href = usp.extract_confirm_href(
            {
                "id": 1,
                "href": "https://repository-images.githubusercontent.com/1/abc",
            }
        )
        self.assertIn("repository-images", href)

    def test_empty(self) -> None:
        self.assertEqual(usp.extract_confirm_href(None), "")
        self.assertEqual(usp.extract_confirm_href({}), "")


if __name__ == "__main__":
    unittest.main()
