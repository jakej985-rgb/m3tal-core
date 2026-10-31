#!/usr/bin/env bash
# ==============================================================================
# M3TAL Review Engine: Git Pre-Push Hook Installer
# Installs multi-agent review verification into .git/hooks/pre-push
# ==============================================================================
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
CORE_DIR="$(cd "${SCRIPT_DIR}/../.." && pwd)"
ENGINE_SCRIPT="${CORE_DIR}/scripts/review/engine.py"

TARGET_REPOS=(
    "/home/m3tal/apps/M3tal-core"
    "/home/m3tal/apps/m3tal-plugin-page"
    "/home/m3tal/apps/M3tal-Hub"
    "/home/m3tal/apps/m3tal-apt-key"
)

# Allow passing custom target repo as argument
if [ $# -gt 0 ]; then
    TARGET_REPOS=("$@")
fi

HOOK_CONTENT='#!/bin/sh
# ==============================================================================
# M3TAL Review Engine Pre-Push Hook
# Enforces multi-agent code review approval before pushing
# ==============================================================================
ENGINE_SCRIPT="/home/m3tal/apps/M3tal-core/scripts/review/engine.py"

# Read standard input: <local_ref> <local_sha> <remote_ref> <remote_sha>
while read -r local_ref local_sha remote_ref remote_sha; do
    # Skip branch deletion
    if [ "$local_sha" = "0000000000000000000000000000000000000000" ] || [ -z "$local_sha" ]; then
        continue
    fi

    if [ -f "$ENGINE_SCRIPT" ]; then
        python3 "$ENGINE_SCRIPT" --repo . --head "$local_sha" --action verify-push
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo ""
            echo "================================================================================"
            echo ">>> M3TAL Review Gate: PUSH BLOCKED!"
            echo ">>> To review and approve changes with the Multi-Agent Review Engine, run:"
            echo "    /home/m3tal/apps/M3tal-core/.agents/agents/review-orchestrator/agent.md"
            echo ">>> (Use '\''git push --no-verify'\'' only for emergency human-authorized bypass)"
            echo "================================================================================"
            exit $EXIT_CODE
        fi
    fi
done

# Entire CLI hooks (if installed)
if command -v entire >/dev/null 2>&1; then
    entire hooks git pre-push "$1" || true
fi

exit 0
'

for repo in "${TARGET_REPOS[@]}"; do
    if [ -d "${repo}/.git" ]; then
        HOOK_FILE="${repo}/.git/hooks/pre-push"
        echo "[m3tal-review] Installing pre-push hook to ${HOOK_FILE}..."
        mkdir -p "${repo}/.git/hooks"
        cat << 'EOF' > "${HOOK_FILE}"
#!/bin/sh
# ==============================================================================
# M3TAL Review Engine Pre-Push Hook
# Enforces multi-agent code review approval before pushing
# ==============================================================================
ENGINE_SCRIPT="/home/m3tal/apps/M3tal-core/scripts/review/engine.py"

# Read standard input: <local_ref> <local_sha> <remote_ref> <remote_sha>
while read -r local_ref local_sha remote_ref remote_sha; do
    # Skip branch deletion
    if [ "$local_sha" = "0000000000000000000000000000000000000000" ] || [ -z "$local_sha" ]; then
        continue
    fi

    if [ -f "$ENGINE_SCRIPT" ]; then
        python3 "$ENGINE_SCRIPT" --repo . --head "$local_sha" --action verify-push
        EXIT_CODE=$?
        if [ $EXIT_CODE -ne 0 ]; then
            echo ""
            echo "================================================================================"
            echo ">>> M3TAL Review Gate: PUSH BLOCKED!"
            echo ">>> To review and approve changes with the Multi-Agent Review Engine, run:"
            echo "    /home/m3tal/apps/M3tal-core/.agents/agents/review-orchestrator/agent.md"
            echo ">>> (Use 'git push --no-verify' only for emergency human-authorized bypass)"
            echo "================================================================================"
            exit $EXIT_CODE
        fi
    fi
done

# Entire CLI hooks (if installed)
if command -v entire >/dev/null 2>&1; then
    entire hooks git pre-push "$1" || true
fi

exit 0
EOF
        chmod +x "${HOOK_FILE}"
        echo "[m3tal-review] ✓ Successfully installed in ${repo}"
    else
        echo "[m3tal-review] Skipping ${repo} (not a git repository)"
    fi
done

echo "[m3tal-review] All pre-push hooks installed and active."
