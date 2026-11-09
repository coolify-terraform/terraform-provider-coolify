#!/usr/bin/env python3
"""Classify a Coolify /register HTML body for test bootstrap.

form: the name field is in the server HTML, so Playwright can register.
closed: Coolify hid the form because the root setup window is not open.
pending: startup message, or any other body. The caller reloads.
"""

from __future__ import annotations

import sys


def classify_register_html(html: str) -> str:
    """Return form, closed, or pending. A visible name field wins."""
    if 'name="name"' in html:
        return "form"
    if "Setup window closed" in html:
        return "closed"
    return "pending"


def main() -> None:
    print(classify_register_html(sys.stdin.read()))


if __name__ == "__main__":
    main()
