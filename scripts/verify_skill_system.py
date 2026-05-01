#!/usr/bin/env python3
from __future__ import annotations

import argparse
import sys
from pathlib import Path


ROOT = Path(__file__).resolve().parent.parent
sys.path.insert(0, str(Path(__file__).resolve().parent))

import bootstrap_skill_system as bootstrap  # noqa: E402


REQUIRED_HEADINGS = [
    "## Purpose",
    "## Inputs",
    "## Outputs",
    "## Rules",
    "## Workflow",
    "## Definition of Done",
    "## Forbidden",
]

REQUIRED_FILES = [
    ROOT / "AGENTS.md",
    ROOT / "CLAUDE.md",
    ROOT / ".env.example",
    ROOT / "Makefile",
    ROOT / "README.md",
    ROOT / "docs" / "setup" / "recommended-order.md",
    ROOT / "docs" / "setup" / "implementation-prompts.md",
    ROOT / "docs" / "setup" / "setup-commands.md",
    ROOT / "docs" / "setup" / "verification-commands.md",
    ROOT / "docs" / "setup" / "skills-inventory.md",
    ROOT / "docs" / "setup" / "files-inventory.txt",
]


def expected_skill_slugs() -> list[str]:
    return [slug for group in bootstrap.SKILL_GROUPS for slug, _, _ in group["skills"]]


def verify() -> list[str]:
    errors: list[str] = []
    expected_slugs = expected_skill_slugs()

    for required_file in REQUIRED_FILES:
        if not required_file.exists():
            errors.append(f"Missing required file: {required_file.relative_to(ROOT)}")

    for tree in bootstrap.SKILL_TREES:
        if not tree.exists():
            errors.append(f"Missing skill tree: {tree.relative_to(ROOT)}")
            continue
        for slug in expected_slugs:
            skill_file = tree / slug / "SKILL.md"
            if not skill_file.exists():
                errors.append(f"Missing skill file: {skill_file.relative_to(ROOT)}")
                continue
            content = skill_file.read_text(encoding="utf-8")
            lines = content.splitlines()
            if len(lines) < 4 or lines[0] != "---" or lines[1] != f"name: {slug}" or not lines[2].startswith("description: ") or lines[3] != "---":
                errors.append(f"Invalid frontmatter in {skill_file.relative_to(ROOT)}")
            if f"name: {slug}" not in content:
                errors.append(f"Frontmatter name mismatch: {skill_file.relative_to(ROOT)}")
            for heading in REQUIRED_HEADINGS:
                if heading not in content:
                    errors.append(f"Missing heading '{heading}' in {skill_file.relative_to(ROOT)}")

    return errors


def list_skills() -> None:
    for group in bootstrap.SKILL_GROUPS:
        print(group["group"])
        for slug, _, _ in group["skills"]:
            print(f"  - {slug}")


def main() -> None:
    parser = argparse.ArgumentParser()
    parser.add_argument("--list", action="store_true", help="Print the expected skill inventory")
    args = parser.parse_args()

    if args.list:
        list_skills()
        return

    errors = verify()
    expected_count = len(expected_skill_slugs())
    actual_skill_files = sum(1 for tree in bootstrap.SKILL_TREES for _ in tree.glob("*/SKILL.md"))

    if errors:
        print("Verification failed:")
        for error in errors:
            print(f"- {error}")
        raise SystemExit(1)

    print(f"Verification passed: {expected_count} skills per tree, {actual_skill_files} SKILL.md files total.")


if __name__ == "__main__":
    main()
