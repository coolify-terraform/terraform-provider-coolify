#!/usr/bin/env python3
"""Fail when a test mock cannot tell a missing JSON key from a zero value.

Coolify omits keys instead of sending the column default. A mock that
decodes the HTTP body into a non-pointer bool or number stores zero when
the provider omitted the key, then treats that zero as a sent value.

Pointers, map decodes, and acctest.DecodeOptional keep the distinction.
"""

from __future__ import annotations

import re
import sys
from pathlib import Path

ROOT = Path(__file__).resolve().parent.parent
SERVICE = ROOT / "internal" / "service"

FIELD_RE = re.compile(
    r"^\s*(\w+)\s+(\*?)([\w.]+)\s+`json:\"([^\"`]+)\"`",
    re.M,
)
FUNC_RE = re.compile(r"(?m)^func ")
DECODE_RE = re.compile(
    r"(?:json\.Unmarshal\([^,]+,\s*&(\w+)|json\.NewDecoder\([^)]*\)\.Decode\(&(\w+))"
)
STRUCT_RE = re.compile(r"type\s+(\w+)\s+struct\s*\{", re.M)
VAR_STRUCT_RE = re.compile(r"var\s+(\w+)\s+struct\s*\{", re.M)
SCALAR_TYPES = {"bool", "int", "int64", "float64", "float32"}


def brace_body(src: str, open_at: int) -> str:
    depth = 0
    for index in range(open_at, len(src)):
        if src[index] == "{":
            depth += 1
        elif src[index] == "}":
            depth -= 1
            if depth == 0:
                return src[open_at : index + 1]
    return ""


def struct_fields(src: str) -> dict[str, str]:
    """Map struct or var-struct name to its brace body."""
    found: dict[str, str] = {}
    for pattern in (STRUCT_RE, VAR_STRUCT_RE):
        for match in pattern.finditer(src):
            body = brace_body(src, match.end() - 1)
            if body:
                found[match.group(1)] = body
    return found


def bad_scalar_fields(body: str) -> list[str]:
    """Non-pointer bools and numbers tagged omitempty cannot show a missing key."""
    bad = []
    for match in FIELD_RE.finditer(body):
        go_name, star, typ, raw = match.group(1), match.group(2), match.group(3), match.group(4)
        if star or typ.split(".")[-1] not in SCALAR_TYPES:
            continue
        parts = raw.split(",")
        json_name = parts[0]
        if json_name in ("", "-") or "omitempty" not in parts:
            continue
        bad.append(f"{go_name} ({json_name})")
    return bad


def functions(src: str) -> list[str]:
    starts = [match.start() for match in FUNC_RE.finditer(src)]
    if not starts:
        return []
    chunks = []
    for index, start in enumerate(starts):
        end = starts[index + 1] if index + 1 < len(starts) else len(src)
        chunks.append(src[start:end])
    return chunks


def problems_in_source(src: str, label: str) -> list[str]:
    if "r.Body" not in src:
        return []
    file_types = {
        name: body
        for name, body in struct_fields(src).items()
        if re.search(rf"type\s+{re.escape(name)}\s+struct", src)
    }
    problems = []
    for chunk in functions(src):
        if "r.Body" not in chunk:
            continue
        if "DecodeOptional" in chunk or "DecodeObject" in chunk:
            continue
        local_structs = struct_fields(chunk)
        for match in DECODE_RE.finditer(chunk):
            target = match.group(1) or match.group(2) or ""
            body = local_structs.get(target) or file_types.get(target)
            if not body:
                continue
            for field in bad_scalar_fields(body):
                problems.append(
                    f"{label}: request decode into {target}.{field} "
                    "collapses a missing key to zero. Use a pointer or acctest.DecodeOptional."
                )
    return problems


def problems_in_file(path: Path) -> list[str]:
    return problems_in_source(path.read_text(), str(path.relative_to(ROOT)))


def check_repo(root: Path | None = None) -> list[str]:
    root = root or ROOT
    service = root / "internal" / "service"
    problems: list[str] = []
    if not service.is_dir():
        return [f"missing {service}"]
    for path in sorted(service.rglob("*_test.go")):
        problems.extend(problems_in_file(path))
    return problems


def main() -> int:
    problems = check_repo()
    if problems:
        print("request-decode: failed", file=sys.stderr)
        for problem in problems:
            print(f"  {problem}", file=sys.stderr)
        return 1
    print("request-decode: ok")
    return 0


if __name__ == "__main__":
    sys.exit(main())
