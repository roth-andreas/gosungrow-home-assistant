#!/usr/bin/env python3
"""Validate the normative GoSungrow specification without third-party packages."""

from __future__ import annotations

import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SPECS = ROOT / "specs"
REQ_RE = re.compile(r"\bREQ-[A-Z][A-Z0-9]*-[0-9]{3}\b")
LINK_RE = re.compile(r"\[[^\]]+\]\(([^)]+\.md(?:#[^)]+)?)\)")
UNRESOLVED_RE = re.compile(r"\b(?:TODO|TBD)\b", re.IGNORECASE)


def tracked_files() -> list[str]:
    result = subprocess.run(
        ["git", "ls-files"], cwd=ROOT, check=True, text=True, capture_output=True
    )
    return [line.strip().replace("\\", "/") for line in result.stdout.splitlines() if line.strip()]


def main() -> int:
    errors: list[str] = []
    documents = sorted(SPECS.rglob("*.md"))
    if not documents:
        errors.append("specs/ contains no Markdown documents")

    owners: dict[str, Path] = {}
    all_text: dict[Path, str] = {}
    normative: list[Path] = []

    for path in documents:
        text = path.read_text(encoding="utf-8")
        all_text[path] = text
        if "Status: Normative" in text:
            normative.append(path)
            for heading in ("Scope:", "Prohibited behavior"):
                if heading not in text:
                    errors.append(f"{path.relative_to(ROOT)}: missing {heading!r}")
            scrubbed = re.sub(r"`[^`]*`", "", text)
            if UNRESOLVED_RE.search(scrubbed):
                errors.append(f"{path.relative_to(ROOT)}: unresolved TODO/TBD marker")
            if not REQ_RE.search(text):
                errors.append(f"{path.relative_to(ROOT)}: no normative requirement IDs")

        for requirement in REQ_RE.findall(text):
            if "Status: Informative" in text:
                continue
            previous = owners.get(requirement)
            if previous and previous != path:
                errors.append(
                    f"duplicate {requirement}: {previous.relative_to(ROOT)} and {path.relative_to(ROOT)}"
                )
            owners[requirement] = path

        for target in LINK_RE.findall(text):
            relative_target = target.split("#", 1)[0]
            resolved = (path.parent / relative_target).resolve()
            if not resolved.is_file():
                errors.append(
                    f"{path.relative_to(ROOT)}: broken Markdown link {target!r}"
                )

    expected_normative = {
        "governance.md",
        "product-and-architecture.md",
        "domain-model.md",
        "isolarcloud-api.md",
        "data-normalization.md",
        "mqtt.md",
        "home-assistant-entities.md",
        "dashboard-management.md",
        "dashboard-resolution.md",
        "dashboard-source-overrides.md",
        "dashboard-frontend.md",
        "cli-and-addon.md",
        "cross-cutting.md",
        "delivery.md",
        "acceptance.md",
    }
    actual_normative = {path.name for path in normative}
    for missing in sorted(expected_normative - actual_normative):
        errors.append(f"missing normative specification: specs/{missing}")

    traceability = all_text.get(SPECS / "traceability.md", "")
    tracked = tracked_files()
    tests = [
        path
        for path in tracked
        if path.endswith("_test.go")
        or path.endswith(".test.cjs")
        or (path.endswith(".mjs") and Path(path).name.startswith("test_"))
    ]
    for test in tests:
        if f"`{test}`" not in traceability:
            errors.append(f"traceability.md does not cover behavioral test {test}")

    required_prefixes = {
        "GOV", "PROD", "DOM", "API", "DATA", "MQTT", "HA", "DASH",
        "RES", "SRC", "CARD", "CLI", "ADDON", "XCUT", "REL", "ACC",
    }
    present_prefixes = {req.split("-")[1] for req in owners}
    for prefix in sorted(required_prefixes - present_prefixes):
        errors.append(f"no requirements found for area {prefix}")

    if errors:
        print("Specification validation failed:", file=sys.stderr)
        for error in errors:
            print(f"- {error}", file=sys.stderr)
        return 1

    print(
        f"Specification validation passed: {len(normative)} normative documents, "
        f"{len(owners)} unique requirements, {len(tests)} traced behavioral test files."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
