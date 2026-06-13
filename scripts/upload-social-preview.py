#!/usr/bin/env python3
"""Upload the social preview image to GitHub Settings via Playwright CDP.

GitHub has no API for social preview upload. This script automates the
web UI via Chrome Remote Debugging (CDP).

Requires:
  - Chrome running with --remote-debugging-port=9222
  - Signed in to GitHub in the Chrome profile
  - Playwright: pip install playwright

Usage:
  python3 scripts/upload-social-preview.py
"""

import os
import sys

from playwright.sync_api import sync_playwright

REPO = "coolify-terraform/terraform-provider-coolify"
IMAGE_PATH = os.path.join(os.path.dirname(__file__), "..", "assets", "social-preview.png")


def main():
    image = os.path.abspath(IMAGE_PATH)
    if not os.path.exists(image):
        print(f"Image not found: {image}", file=sys.stderr)
        sys.exit(1)

    with sync_playwright() as p:
        browser = p.chromium.connect_over_cdp("http://localhost:9222")
        context = browser.contexts[0]
        page = context.new_page()
        try:
            print(f"Navigating to https://github.com/{REPO}/settings ...")
            page.goto(
                f"https://github.com/{REPO}/settings",
                wait_until="domcontentloaded",
                timeout=60000,
            )

            # Check for login redirect
            if "login" in page.url or "session" in page.url:
                print(
                    "Not signed in to GitHub. Please sign in via the Chrome window.",
                    file=sys.stderr,
                )
                sys.exit(1)

            # Wait for the Social preview heading
            page.wait_for_function(
                "() => [...document.querySelectorAll('h2')]"
                ".some(h => h.textContent.includes('Social preview'))",
                polling=200,
                timeout=15000,
            )

            # Scroll to it
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

            # Open the Edit dropdown
            page.locator("summary:has-text('Edit')").first.click()

            # Upload via file chooser
            with page.expect_file_chooser() as fc_info:
                page.locator("label[for='repo-image-file-input']").click()
            fc_info.value.set_files(image)

            # Wait for upload processing
            page.wait_for_function(
                "() => {"
                "  const fa = document.querySelector("
                "    'file-attachment.js-upload-repository-image');"
                "  return fa && !fa.classList.contains('is-default');"
                "}",
                polling=500,
                timeout=15000,
            )

            print("Social preview uploaded successfully.")

            # Verify via og:image
            page.goto(
                f"https://github.com/{REPO}",
                wait_until="domcontentloaded",
                timeout=30000,
            )
            og = page.evaluate(
                "() => document.querySelector("
                "'meta[property=\"og:image\"]')?.content"
            )
            if og and "repository-images" in og:
                print(f"Verified: {og}")
            else:
                print(f"Warning: og:image may not have updated yet: {og}")

        finally:
            page.close()
            browser.close()


if __name__ == "__main__":
    main()