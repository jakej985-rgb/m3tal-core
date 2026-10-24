#!/usr/bin/env bash
# ==============================================================================
# scripts/bump-version.sh
# Semantic Version Auto-Bumping & Release Synchronization System for M3TAL Core
#
# Analyzes git history using Conventional Commits (feat: -> minor, fix: -> patch,
# BREAKING CHANGE: -> major) and synchronizes:
#   - VERSION
#   - gui/package.json
#   - CHANGELOG.md
#
# Usage:
#   ./scripts/bump-version.sh [auto|minor|patch|major|check|set <version>]
# ==============================================================================

set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
ROOT_DIR="$(cd "${SCRIPT_DIR}/.." && pwd)"

PYTHON_SCRIPT="${SCRIPT_DIR}/generate_release_changelog.py"

MODE="${1:-auto}"

case "$MODE" in
  check)
    echo "🔍 Inspecting semantic version bump requirement..."
    python3 "${PYTHON_SCRIPT}" --bump auto
    ;;

  auto|minor|patch|major)
    echo "🚀 Executing semantic version bump (${MODE})..."
    python3 "${PYTHON_SCRIPT}" --bump "${MODE}" --apply
    NEW_VER=$(cat "${ROOT_DIR}/VERSION" | tr -d '[:space:]')
    echo "✅ Version successfully bumped to ${NEW_VER}"
    ;;

  set)
    if [ -z "${2:-}" ]; then
      echo "❌ Error: 'set' mode requires a target version (e.g. ./scripts/bump-version.sh set 0.2.0)"
      exit 1
    fi
    TARGET_VER="${2#v}"
    echo "📌 Setting explicit version to v${TARGET_VER}..."
    python3 "${PYTHON_SCRIPT}" --version "${TARGET_VER}" --apply
    NEW_VER=$(cat "${ROOT_DIR}/VERSION" | tr -d '[:space:]')
    echo "✅ Version explicitly set to ${NEW_VER}"
    ;;

  help|--help|-h)
    echo "Usage: ./scripts/bump-version.sh [command]"
    echo ""
    echo "Commands:"
    echo "  auto           Analyze commits since last tag and apply calculated bump (default)"
    echo "  check          Dry-run check of calculated bump without changing files"
    echo "  minor          Force a minor version bump (e.g. 0.1.20 -> 0.2.0)"
    echo "  patch          Force a patch version bump (e.g. 0.2.0 -> 0.2.1)"
    echo "  major          Force a major version bump (e.g. 0.2.0 -> 1.0.0)"
    echo "  set <ver>      Set explicit version (e.g. ./scripts/bump-version.sh set 0.2.0)"
    ;;

  *)
    echo "❌ Unknown command: $MODE. Run './scripts/bump-version.sh --help' for usage."
    exit 1
    ;;
esac
