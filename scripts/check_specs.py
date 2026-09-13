#!/usr/bin/env python3
"""Validate GoSungrow's source-of-truth and derived-surface contracts."""

from __future__ import annotations

import hashlib
import re
import subprocess
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parents[1]
SPECS = ROOT / "specs"
REQ_RE = re.compile(r"\bREQ-([A-Z][A-Z0-9]*)-([0-9]{3})\b")
REQ_DEFINITION_RE = re.compile(
    r"^\s*-\s+\*\*(REQ-[A-Z][A-Z0-9]*-[0-9]{3})\*\*\s+(?:—|-)\s+",
    re.MULTILINE,
)
LINK_RE = re.compile(r"\[[^\]]+\]\(([^)]+\.md(?:#[^)]+)?)\)")
UNRESOLVED_RE = re.compile(r"\b(?:TODO|TBD)\b", re.IGNORECASE)
ENV_RE = re.compile(r"\b(?:GOSUNGROW_[A-Z][A-Z0-9_]*|SUPERVISOR_TOKEN)\b")
INVENTORY_PATTERN_RE = re.compile(r"^\| `([^`]+)` \|", re.MULTILINE)

EXPECTED_NORMATIVE = {
    "acceptance.md",
    "cli-and-addon.md",
    "completeness.md",
    "configuration.md",
    "cross-cutting.md",
    "dashboard-frontend.md",
    "dashboard-management.md",
    "dashboard-resolution.md",
    "dashboard-source-overrides.md",
    "data-normalization.md",
    "delivery.md",
    "domain-model.md",
    "governance.md",
    "home-assistant-entities.md",
    "isolarcloud-api.md",
    "mqtt.md",
    "product-and-architecture.md",
}

REQUIRED_PREFIXES = {
    "ACC", "ADDON", "API", "CARD", "CFG", "CLI", "COMP", "DASH",
    "DATA", "DOM", "GOV", "HA", "MQTT", "PROD", "REL", "RES", "SRC",
    "XCUT",
}

REQUIRED_SKILLS = {
    "plan-spec-change",
    "plan-spec-change-implementation",
}


def tracked_files() -> list[str]:
    result = subprocess.run(
        ["git", "ls-files"], cwd=ROOT, check=True, text=True, capture_output=True
    )
    return [line.strip().replace("\\", "/") for line in result.stdout.splitlines() if line.strip()]


def glob_regex(pattern: str) -> re.Pattern[str]:
    escaped = re.escape(pattern)
    escaped = escaped.replace(r"\*\*/", "(?:.*/)?")
    escaped = escaped.replace(r"\*\*", ".*")
    escaped = escaped.replace(r"\*", "[^/]*")
    escaped = escaped.replace(r"\?", "[^/]")
    return re.compile(f"^{escaped}$")


def validate_skills(errors: list[str]) -> None:
    skills_root = ROOT / ".agents" / "skills"
    actual = {path.name for path in skills_root.iterdir() if path.is_dir()}
    for missing in sorted(REQUIRED_SKILLS - actual):
        errors.append(f"missing required skill: .agents/skills/{missing}")

    for skill_name in sorted(actual):
        skill_dir = skills_root / skill_name
        skill_path = skill_dir / "SKILL.md"
        metadata_path = skill_dir / "agents" / "openai.yaml"
        if not skill_path.is_file():
            errors.append(f"{skill_dir.relative_to(ROOT)}: missing SKILL.md")
            continue
        text = skill_path.read_text(encoding="utf-8")
        match = re.match(r"\A---\n(.*?)\n---\n", text, re.DOTALL)
        if not match:
            errors.append(f"{skill_path.relative_to(ROOT)}: invalid YAML frontmatter boundary")
            continue
        fields: dict[str, str] = {}
        for line in match.group(1).splitlines():
            key, separator, value = line.partition(":")
            if not separator or not key.strip() or not value.strip():
                errors.append(f"{skill_path.relative_to(ROOT)}: invalid frontmatter line {line!r}")
                continue
            fields[key.strip()] = value.strip()
        if set(fields) != {"name", "description"}:
            errors.append(f"{skill_path.relative_to(ROOT)}: frontmatter must contain only name and description")
        if fields.get("name") != skill_name:
            errors.append(f"{skill_path.relative_to(ROOT)}: name must equal directory {skill_name!r}")
        if not re.fullmatch(r"[a-z0-9-]{1,64}", skill_name):
            errors.append(f"{skill_path.relative_to(ROOT)}: invalid skill name")
        description = fields.get("description", "")
        if not description or len(description) > 1024:
            errors.append(f"{skill_path.relative_to(ROOT)}: description must be 1-1024 characters")
        if not metadata_path.is_file():
            errors.append(f"{skill_dir.relative_to(ROOT)}: missing agents/openai.yaml")
        else:
            metadata = metadata_path.read_text(encoding="utf-8")
            for key in ("interface:", "display_name:", "short_description:", "default_prompt:"):
                if key not in metadata:
                    errors.append(f"{metadata_path.relative_to(ROOT)}: missing {key}")
            if f"${skill_name}" not in metadata:
                errors.append(f"{metadata_path.relative_to(ROOT)}: default_prompt must mention ${skill_name}")


def validate_inventory(tracked: list[str], inventory: str, errors: list[str]) -> None:
    patterns = INVENTORY_PATTERN_RE.findall(inventory)
    if not patterns:
        errors.append("specs/source-inventory.md: no tracked path patterns found")
        return
    matchers = [(pattern, glob_regex(pattern)) for pattern in patterns]
    for tracked_path in tracked:
        if not any(matcher.fullmatch(tracked_path) for _, matcher in matchers):
            errors.append(f"source-inventory.md does not classify tracked file {tracked_path}")


def validate_environment_catalog(tracked: list[str], catalog: str, errors: list[str]) -> None:
    inputs: set[str] = set()
    for tracked_path in tracked:
        if not tracked_path.endswith((".go", ".sh", ".yaml", ".yml", "Dockerfile")):
            continue
        path = ROOT / tracked_path
        try:
            inputs.update(ENV_RE.findall(path.read_text(encoding="utf-8")))
        except UnicodeDecodeError:
            continue
    for input_name in sorted(inputs):
        if f"`{input_name}`" not in catalog:
            errors.append(f"configuration.md does not classify environment input {input_name}")


def validate_decision_index(all_text: dict[Path, str], errors: list[str]) -> None:
    decisions = SPECS / "decisions"
    index = all_text.get(decisions / "README.md", "")
    for decision in sorted(decisions.glob("ADR-*.md")):
        if f"({decision.name})" not in index:
            errors.append(f"specs/decisions/README.md does not index {decision.name}")


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
            definitions = REQ_DEFINITION_RE.findall(text)
            if not definitions:
                errors.append(f"{path.relative_to(ROOT)}: no normative requirement definitions")
            for requirement in definitions:
                previous = owners.get(requirement)
                if previous:
                    errors.append(
                        f"duplicate {requirement}: {previous.relative_to(ROOT)} and {path.relative_to(ROOT)}"
                    )
                owners[requirement] = path

        for target in LINK_RE.findall(text):
            relative_target = target.split("#", 1)[0]
            resolved = (path.parent / relative_target).resolve()
            if not resolved.is_file():
                errors.append(f"{path.relative_to(ROOT)}: broken Markdown link {target!r}")

    for path, text in all_text.items():
        for area, number in REQ_RE.findall(text):
            requirement = f"REQ-{area}-{number}"
            if requirement not in owners:
                errors.append(f"{path.relative_to(ROOT)}: unknown requirement reference {requirement}")

    actual_normative = {path.name for path in normative}
    for missing in sorted(EXPECTED_NORMATIVE - actual_normative):
        errors.append(f"missing normative specification: specs/{missing}")

    traceability = all_text.get(SPECS / "traceability.md", "")
    tracked = tracked_files()
    tests = [
        path for path in tracked
        if path.endswith("_test.go")
        or path.endswith(".test.cjs")
        or (path.endswith(".mjs") and Path(path).name.startswith("test_"))
    ]
    for test in tests:
        if f"`{test}`" not in traceability:
            errors.append(f"traceability.md does not cover behavioral test {test}")

    present_prefixes = {requirement.split("-")[1] for requirement in owners}
    for prefix in sorted(REQUIRED_PREFIXES - present_prefixes):
        errors.append(f"no requirements found for area {prefix}")
    verification = traceability.partition("## Behavioral tests")[2]
    if not verification:
        errors.append("traceability.md: missing Behavioral tests verification section")
    for prefix in sorted(present_prefixes):
        if f"REQ-{prefix}-" not in verification and f"REQ-{prefix}-*" not in verification:
            errors.append(f"traceability.md contains no verification evidence for area {prefix}")

    validate_inventory(tracked, all_text.get(SPECS / "source-inventory.md", ""), errors)
    validate_environment_catalog(tracked, all_text.get(SPECS / "configuration.md", ""), errors)
    validate_decision_index(all_text, errors)
    validate_skills(errors)

    if not (ROOT / "docs" / "spec-change-workflow.md").is_file():
        errors.append("missing docs/spec-change-workflow.md")

    if errors:
        print("Specification validation failed:", file=sys.stderr)
        for error in sorted(set(errors)):
            print(f"- {error}", file=sys.stderr)
        return 1

    digest = hashlib.sha256(
        "".join(sorted(owners)).encode("utf-8")
    ).hexdigest()[:12]
    print(
        f"Specification validation passed: {len(normative)} normative documents, "
        f"{len(owners)} unique requirements, {len(tests)} traced behavioral test files, "
        f"{len(tracked)} classified tracked files (contract {digest})."
    )
    return 0


if __name__ == "__main__":
    raise SystemExit(main())
