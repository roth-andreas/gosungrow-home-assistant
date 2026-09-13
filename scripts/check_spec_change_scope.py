#!/usr/bin/env python3
"""Enforce atomic specification and derived-artifact pull requests."""

from __future__ import annotations

import argparse
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
NORMATIVE_RE = re.compile(r"^specs/(?!decisions/)[^/]+\.md$")
DERIVED_RE = re.compile(
    r"^(?:main\.go|cmd/|cmdHassio/|iSolarCloud/|defaults/|addon/gosungrow/|"
    r"tools/|examples/|scripts/|\.agents/|\.github/workflows/|go\.(?:mod|sum)$|"
    r"repository\.yaml$|\.dockerignore$)"
)
PRODUCT_RE = re.compile(
    r"^(?:main\.go|cmd/|cmdHassio/|iSolarCloud/|defaults/|addon/gosungrow/|"
    r"tools/preview/[^/]+\.(?:js|html)|examples/)"
)
GOVERNANCE_SPEC_NAMES = {"governance.md", "completeness.md", "delivery.md"}


def changed_files(base: str) -> list[str]:
    result = subprocess.run(
        ["git", "diff", "--name-only", f"{base}...HEAD"],
        cwd=ROOT,
        check=True,
        text=True,
        capture_output=True,
    )
    return [line.strip().replace("\\", "/") for line in result.stdout.splitlines() if line.strip()]


def is_normative(path: str) -> bool:
    if not NORMATIVE_RE.match(path):
        return False
    current = ROOT / path
    if current.is_file():
        return "Status: Normative" in current.read_text(encoding="utf-8")
    return True  # Deleted top-level specification files are contract changes.


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument("--base", required=True, help="PR base commit SHA")
    parser.add_argument("--allow-spec-only", action="store_true")
    parser.add_argument("--allow-implementation-only", action="store_true")
    args = parser.parse_args()

    changed = changed_files(args.base)
    specifications = [path for path in changed if is_normative(path)]
    product_specifications = [
        path for path in specifications if Path(path).name not in GOVERNANCE_SPEC_NAMES
    ]
    derived = [path for path in changed if DERIVED_RE.match(path)]
    product = [path for path in changed if PRODUCT_RE.match(path)]
    errors: list[str] = []

    if specifications and not derived and not args.allow_spec_only:
        errors.append(
            "normative specs changed without a derived artifact; keep implementation in this PR "
            "or apply the spec-docs-only label for a reviewed non-behavioral correction"
        )
    if product_specifications and not product and not args.allow_spec_only:
        errors.append(
            "product behavior specs changed without product source or behavioral tests; "
            "keep their derived artifacts in this PR or apply the spec-docs-only label only "
            "for a reviewed non-behavioral correction"
        )
    if product and not specifications and not args.allow_implementation_only:
        errors.append(
            "product implementation changed without a normative spec; update specs in this PR "
            "or apply the behavior-neutral label for a reviewed non-behavioral change"
        )

    if errors:
        print("Specification change-scope validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(
        "Specification change-scope validation passed: "
        f"{len(specifications)} normative spec ({len(product_specifications)} product), "
        f"{len(derived)} derived, "
        f"and {len(product)} product file(s) changed."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
