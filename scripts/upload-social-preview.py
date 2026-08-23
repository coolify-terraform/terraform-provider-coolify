#!/usr/bin/env python3
"""Upload the social preview image to GitHub Settings via Playwright CDP.

GitHub has no API for social preview upload. This script automates the
web UI via Chrome Remote Debugging (CDP).

The file-attachment widget is: POST /upload/policies/repository-images,
POST the file to S3, then PUT /upload/repository-images/{id}. That PUT
is the confirm. Leaving the settings page before it returns 200 aborts
the blob. A new og:image URL is not proof the public PNG exists.

Requires:
  - Chrome running with --remote-debugging-port=9222
  - Signed in to GitHub in the Chrome profile
  - Playwright: pip install playwright

Usage:
  python3 scripts/upload-social-preview.py
"""

from __future__ import annotations

import os
import sys
import threading
from typing import Any

REPO = "coolify-terraform/terraform-provider-coolify"
IMAGE_PATH = os.path.join(os.path.dirname(__file__), "..", "assets", "social-preview.png")


def is_auth_gate(url: str, heading: str) -> bool:
    """True when the tab is login, OAuth, or sudo Confirm access."""
    lower = (url or "").lower()
    if any(
        s in lower
        for s in ("github.com/login", "/login/oauth", "confirm_access", "/sessions")
    ):
        return True
    return (heading or "").strip() in (
        "Sign in to GitHub",
        "Confirm access",
        "Authorize",
    )


def is_confirm_response(method: str, url: str, status: int) -> bool:
    """True when GitHub accepted the uploaded repository image asset."""
    return (
        (method or "").upper() == "PUT"
        and "/upload/repository-images/" in (url or "")
        and status == 200
    )


def public_preview_ok(status: int, content_type: str) -> bool:
    """True when the public repository-images URL actually serves an image."""
    return status == 200 and "image/" in (content_type or "")


def extract_confirm_href(payload: dict[str, Any] | None) -> str:
    if not payload:
        return ""
    href = payload.get("href")
    return href if isinstance(href, str) else ""


def main() -> None:
    from playwright.sync_api import sync_playwright

    image = os.path.abspath(IMAGE_PATH)
    if not os.path.exists(image):
        print(f"Image not found: {image}", file=sys.stderr)
        sys.exit(1)

    with sync_playwright() as p:
        browser = p.chromium.connect_over_cdp("http://localhost:9222", timeout=5000)
        if not browser.contexts:
            print(
                "No browser contexts found. Open at least one tab in Chrome and try again.",
                file=sys.stderr,
            )
            browser.close()
            sys.exit(1)
        context = browser.contexts[0]
        page = context.new_page()
        confirmed: dict[str, Any] = {"href": ""}
        done = threading.Event()
        try:
            print(f"PLAN: upload {image}")
            print(f"DO: navigate https://github.com/{REPO}/settings")
            page.goto(
                f"https://github.com/{REPO}/settings",
                wait_until="domcontentloaded",
                timeout=60000,
            )

            heading = page.evaluate(
                "() => ((document.querySelector('h1')||{}).innerText||'')"
            )
            if is_auth_gate(page.url, heading):
                print(f"LOGIN_IN_PROGRESS url={page.url} heading={heading!r}")
                sys.exit(2)

            page.wait_for_function(
                "() => [...document.querySelectorAll('h2')]"
                ".some(h => h.textContent.includes('Social preview'))",
                polling=200,
                timeout=15000,
            )

            def on_response(resp: Any) -> None:
                if not is_confirm_response(resp.request.method, resp.url, resp.status):
                    return
                href = ""
                try:
                    href = extract_confirm_href(resp.json())
                except Exception:
                    href = ""
                confirmed["href"] = href
                done.set()

            page.on("response", on_response)

            page.evaluate(
                """() => {
                for (const h of document.querySelectorAll('h2')) {
                    if (h.textContent.includes('Social preview')) {
                        h.scrollIntoView({ behavior: 'instant', block: 'center' });
                        return;
                    }
                }
            }"""
            )

            page.locator("summary:has-text('Edit')").first.click()
            with page.expect_file_chooser() as fc_info:
                page.locator("label[for='repo-image-file-input']").click()
            fc_info.value.set_files(image)
            print("DO: file selected; waiting for PUT /upload/repository-images/")

            if not done.wait(20):
                print(
                    "FAIL: confirm PUT /upload/repository-images/ did not return 200",
                    file=sys.stderr,
                )
                sys.exit(1)

            print(f"OK: confirm 200 href={confirmed['href']}")
            print(
                "NEXT: GET the href; success is image/* 200, "
                "not merely a repository-images og:image URL"
            )
        finally:
            page.close()
            browser.close()


if __name__ == "__main__":
    main()
