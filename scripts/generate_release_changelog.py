#!/usr/bin/env python3
"""
scripts/generate_release_changelog.py
Automated release change & impact analyzer for M3tal-core.
Extracts commits between releases, analyzes diff stats, and produces structured changelogs
highlighting What Changed, Why, and User/Architectural Impact across Go CLI, API, GUI, and DEB packaging.
"""

import sys
import os
import re
import subprocess
import argparse
import json
from datetime import datetime

ROOT_DIR = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))
VERSION_PATH = os.path.join(ROOT_DIR, "VERSION")
CHANGELOG_PATH = os.path.join(ROOT_DIR, "CHANGELOG.md")
GUI_PACKAGE_PATH = os.path.join(ROOT_DIR, "gui", "package.json")


def run_git(args):
    proc = subprocess.run(["git"] + args, stdout=subprocess.PIPE, stderr=subprocess.PIPE, text=True, cwd=ROOT_DIR)
    if proc.returncode != 0:
        return ""
    return proc.stdout.strip()


def get_latest_tags(count=5):
    out = run_git(["tag", "--sort=-creatordate"])
    if not out:
        return []
    tags = [t.strip() for t in out.split("\n") if t.strip()]
    return tags[:count]


def get_commits_in_range(start_ref, end_ref="HEAD"):
    range_str = f"{start_ref}..{end_ref}" if start_ref else end_ref
    raw = run_git(["log", range_str, "--pretty=format:%h%x1f%s%x1f%b%x1e"])
    if not raw:
        return []
    commits = []
    for entry in raw.split("\x1e"):
        parts = entry.strip().split("\x1f")
        if len(parts) >= 2:
            sha = parts[0].strip()
            subject = parts[1].strip()
            body = parts[2].strip() if len(parts) > 2 else ""
            if sha:
                commits.append({"sha": sha, "subject": subject, "body": body})
    return commits


def get_diff_stat(start_ref, end_ref="HEAD"):
    range_str = f"{start_ref}..{end_ref}" if start_ref else end_ref
    return run_git(["diff", range_str, "--stat"])


def categorize_commit(commit):
    subj = commit["subject"].lower()
    body = commit.get("body", "").lower()
    combined = f"{subj}\n{body}"

    if any(k in subj for k in ["break", "breaking", "remove", "deprecate"]) or "breaking change" in combined:
        return "Breaking"
    elif any(k in subj for k in ["ui", "gui", "webui", "dashboard", "frontend", "theme"]):
        return "UX"
    elif any(k in subj for k in ["deb", "package", "packaging", "nfpm", "postinst", "systemd", "desktop"]):
        return "Packaging"
    elif any(k in subj for k in ["fix", "bug", "repair", "resolve", "patch"]):
        return "Fixes"
    elif any(k in subj for k in ["arch", "engine", "plugin", "protocol", "internal", "daemon", "cli", "api", "core"]):
        return "Architecture"
    elif any(k in subj for k in ["ci", "workflow", "script", "test", "lint", "tool", "hub", "keyring"]):
        return "Toolchain"
    elif any(k in subj for k in ["doc", "docs", "readme", "spec", "audit", "plan"]):
        return "Docs"
    elif any(k in subj for k in ["feat", "add", "implement", "introduce"]):
        return "Features"
    return "Features"


def generate_draft(version, start_ref, end_ref="HEAD"):
    clean_version = version.lstrip("v")
    commits = get_commits_in_range(start_ref, end_ref)
    today = datetime.now().strftime("%Y-%m-%d")

    groups = {
        "Features": [],
        "Breaking": [],
        "Architecture": [],
        "UX": [],
        "Packaging": [],
        "Fixes": [],
        "Toolchain": [],
        "Docs": [],
    }

    for c in commits:
        cat = categorize_commit(c)
        if cat in groups:
            groups[cat].append(c)
        else:
            groups["Features"].append(c)

    lines = []
    lines.append(f"## [{clean_version}] - {today}\n")
    lines.append("### Overview & Release Highlights")
    if commits:
        lines.append(f"Release v{clean_version} delivers updates across {len(commits)} commits, providing system orchestration primitives, Debian distribution, and ecosystem automation.\n")
    else:
        lines.append(f"Release v{clean_version} establishing the core system baseline and package distribution.\n")

    section_titles = {
        "Features": "🚀 New Features & Capabilities",
        "Breaking": "⚠️ Breaking Changes & Migrations",
        "Architecture": "🏗️ Architecture & Core System Primitives",
        "UX": "🎨 Web GUI & Dashboard",
        "Packaging": "📦 Packaging & Debian Distribution",
        "Fixes": "🐛 Bug Fixes & Stability",
        "Toolchain": "🛠️ Developer Toolchains & CI/CD",
        "Docs": "📚 Documentation & Specifications",
    }

    for key, title in section_titles.items():
        if groups[key]:
            lines.append(f"### {title}")
            for c in groups[key]:
                lines.append(f"- **{c['subject']}** (`{c['sha']}`):")
                body_desc = c['body'].replace("\n", " ").strip() if c['body'] else ""
                if body_desc:
                    lines.append(f"  - *What Changed*: {body_desc}")
                else:
                    lines.append(f"  - *What Changed*: Implemented changes described in commit.")
                lines.append(f"  - *Why*: Enhanced system reliability, packaging compliance, or workflow automation.")
            lines.append("")

    return "\n".join(lines).strip()


def get_current_version():
    if os.path.exists(VERSION_PATH):
        try:
            with open(VERSION_PATH, "r", encoding="utf-8") as f:
                raw = f.read().strip()
                if raw:
                    return raw.lstrip("v")
        except Exception:
            pass
    return "0.1.0"


def update_version_file(new_semver):
    tag = f"v{new_semver.lstrip('v')}"
    with open(VERSION_PATH, "w", encoding="utf-8") as f:
        f.write(tag + "\n")
    return True


def update_gui_package_json(new_semver):
    if not os.path.exists(GUI_PACKAGE_PATH):
        return False
    try:
        with open(GUI_PACKAGE_PATH, "r", encoding="utf-8") as f:
            pkg = json.load(f)
        pkg["version"] = new_semver.lstrip("v")
        with open(GUI_PACKAGE_PATH, "w", encoding="utf-8") as f:
            json.dump(pkg, f, indent=2)
            f.write("\n")
        return True
    except Exception:
        return False


def determine_bump_level(commits):
    has_breaking = False
    has_minor = False

    for c in commits:
        subj = c["subject"]
        body = c.get("body", "")
        combined = f"{subj}\n{body}"

        if re.search(r"^[a-z]+(\([^\)]+\))?!:", subj) or "BREAKING CHANGE:" in combined:
            has_breaking = True
            break

        if re.search(r"^feat(\([^\)]+\))?:", subj):
            has_minor = True
        elif any(k in subj.lower() for k in ["feat", "feature", "introduce", "implement"]):
            has_minor = True

    if has_breaking:
        return "major"
    elif has_minor:
        return "minor"
    return "patch"


def calculate_next_version(current_semver, bump_type="auto", commits=None):
    if commits is None:
        commits = []

    parts = [int(p) for p in current_semver.split(".")]
    while len(parts) < 3:
        parts.append(0)
    major, minor, patch = parts[0], parts[1], parts[2]

    if bump_type == "auto":
        bump_type = determine_bump_level(commits)

    if bump_type == "major":
        next_semver = f"{major + 1}.0.0"
    elif bump_type == "minor":
        next_semver = f"{major}.{minor + 1}.0"
    else:  # patch
        next_semver = f"{major}.{minor}.{patch + 1}"

    return next_semver, bump_type


def prepend_to_changelog(new_entry):
    if not os.path.exists(CHANGELOG_PATH):
        with open(CHANGELOG_PATH, "w", encoding="utf-8") as f:
            f.write("# Changelog\n\nAll notable changes to M3tal Core will be documented in this file.\n\n" + new_entry + "\n")
        return True

    with open(CHANGELOG_PATH, "r", encoding="utf-8") as f:
        content = f.read()

    match = re.search(r"(## \[[0-9].*?\])", content)
    if match:
        idx = match.start()
        updated = content[:idx] + new_entry + "\n\n---\n\n" + content[idx:]
    else:
        updated = content.strip() + "\n\n" + new_entry + "\n"

    with open(CHANGELOG_PATH, "w", encoding="utf-8") as f:
        f.write(updated)
    return True


def release_entry_exists(content, semver):
    clean_ver = semver.lstrip("v")
    pattern = rf"^## \[(?:v)?{re.escape(clean_ver)}\]"
    return re.search(pattern, content, re.MULTILINE) is not None


def main():
    parser = argparse.ArgumentParser(description="Generate release changelog and automate semantic version bumping for M3tal-core")
    parser.add_argument("--version", default=None, help="Target version string (e.g. 0.1.2)")
    parser.add_argument("--since", default=None, help="Starting git tag/ref")
    parser.add_argument("--until", default="HEAD", help="Ending git tag/ref")
    parser.add_argument("--output", default=None, help="Path to write output markdown")
    parser.add_argument("--force-generate", action="store_true", help="Always synthesize from git history")
    parser.add_argument("--bump", choices=["auto", "major", "minor", "patch"], help="Compute semantic version bump")
    parser.add_argument("--apply", action="store_true", help="Apply version bump to project files and CHANGELOG.md")
    args = parser.parse_args()

    current_semver = get_current_version()

    start_tag = args.since
    if not start_tag:
        tags = get_latest_tags(2)
        if tags:
            # If current HEAD matches the newest tag, use previous tag as start
            current_head = run_git(["rev-parse", "HEAD"])
            latest_tag_commit = run_git(["rev-parse", f"{tags[0]}^{{commit}}"])
            if current_head == latest_tag_commit and len(tags) > 1:
                start_tag = tags[1]
            else:
                start_tag = tags[0]

    commits = get_commits_in_range(start_tag, args.until)

    target_semver = current_semver
    bump_reason = None

    if args.bump:
        target_semver, bump_reason = calculate_next_version(current_semver, args.bump, commits)
        print(f"Current version : {current_semver}")
        print(f"Calculated bump : {bump_reason.upper()} -> {target_semver}")

    if args.version:
        target_semver = args.version.strip().lstrip("v")

    content = generate_draft(target_semver, start_tag, args.until)

    if args.apply:
        if os.path.exists(CHANGELOG_PATH):
            with open(CHANGELOG_PATH, "r", encoding="utf-8") as f:
                existing_changelog = f.read()
            if release_entry_exists(existing_changelog, target_semver):
                print(f"Notice: CHANGELOG.md already contains an entry for {target_semver}. Skipping duplicate prepend.")
            else:
                print(f"Prepending release section for v{target_semver} to CHANGELOG.md...")
                prepend_to_changelog(content)
        else:
            print(f"Creating CHANGELOG.md with release section for v{target_semver}...")
            prepend_to_changelog(content)

        print(f"Updating VERSION to v{target_semver}...")
        update_version_file(target_semver)
        update_gui_package_json(target_semver)
        print(f"Successfully applied version {target_semver} across M3tal-core project files.")

    if args.output:
        with open(args.output, "w", encoding="utf-8") as f:
            f.write(content + "\n")
        print(f"Changelog written to {args.output}")
    elif not args.apply:
        print(content)


if __name__ == "__main__":
    main()
