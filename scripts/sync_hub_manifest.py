#!/usr/bin/env python3
"""
Sync M3tal Core version and changelog into M3tal-Hub apps/manifest.yml.
"""
import argparse
import datetime
import os
import re
import subprocess
import sys
import yaml


def get_latest_commit_message():
    try:
        res = subprocess.run(
            ["git", "log", "-1", "--pretty=%s"],
            capture_output=True,
            text=True,
            check=True,
        )
        msg = res.stdout.strip()
        if msg:
            return msg
    except Exception:
        pass
    return "Core system primitives and native utilities."


def sync_manifest(manifest_path: str, version: str, tag: str, notes: list = None):
    if not os.path.isfile(manifest_path):
        raise FileNotFoundError(f"Manifest not found: {manifest_path}")

    with open(manifest_path, "r", encoding="utf-8") as f:
        content = f.read()

    match = re.search(r"  - id: m3tal-core\b.*?(?=\n  - id: |\Z)", content, re.DOTALL)
    if not match:
        raise ValueError("Could not find '  - id: m3tal-core' block in manifest.yml")

    block = match.group(0)
    parsed = yaml.safe_load(block)
    if not parsed or not isinstance(parsed, list):
        raise ValueError("Failed to parse m3tal-core block as YAML list")

    item = parsed[0]
    clean_version = version.lstrip("v")
    item["version"] = clean_version
    item["status"] = "active"

    # Filter out legacy changelog entries with obsolete versions (e.g. 1.1.62)
    current_changelog = item.get("changelog", [])
    filtered_changelog = [
        c
        for c in current_changelog
        if not str(c.get("version", "")).startswith("1.")
    ]

    today_str = datetime.date.today().isoformat()
    if not notes:
        commit_msg = get_latest_commit_message()
        notes = [commit_msg, "Debian package (.deb) binary distribution."]

    # Check if entry for this version already exists
    existing = next((c for c in filtered_changelog if str(c.get("version", "")).lstrip("v") == clean_version), None)
    if existing:
        existing["date"] = today_str
        existing["tag"] = "Release"
        existing["notes"] = notes
    else:
        filtered_changelog.insert(
            0,
            {
                "version": clean_version,
                "date": today_str,
                "tag": "Release",
                "notes": notes,
            },
        )

    item["changelog"] = filtered_changelog

    # Dump updated YAML for this single block
    dumped = yaml.dump([item], sort_keys=False, default_flow_style=False)
    # Ensure correct 2-space base indent for root list elements in manifest
    updated_block = "\n".join(("  " + line if line.strip() else "") for line in dumped.strip().splitlines()) + "\n"

    new_content = content[:match.start()] + updated_block + content[match.end():]

    with open(manifest_path, "w", encoding="utf-8") as f:
        f.write(new_content)

    print(f"Successfully synced m3tal-core v{clean_version} to {manifest_path}")


def main():
    parser = argparse.ArgumentParser(description="Sync M3tal Core version to M3tal-Hub")
    parser.add_argument("--manifest", required=True, help="Path to M3tal-Hub apps/manifest.yml")
    parser.add_argument("--version", required=True, help="Version string (e.g. 0.1.0)")
    parser.add_argument("--tag", default="", help="Tag name (e.g. v0.1.0)")
    parser.add_argument("--note", action="append", help="Changelog notes")

    args = parser.parse_args()
    sync_manifest(args.manifest, args.version, args.tag, args.note)


if __name__ == "__main__":
    main()
