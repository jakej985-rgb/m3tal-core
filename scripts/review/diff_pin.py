"""Git commit SHA pinning, merge-base detection, and patch extraction."""

from __future__ import annotations
import subprocess
import hashlib
import os
from typing import Tuple, Dict, Any, Optional


def run_git(cmd: list[str], cwd: str) -> str:
    """Execute git command and return stripped stdout."""
    res = subprocess.run(
        ["git"] + cmd,
        cwd=cwd,
        stdout=subprocess.PIPE,
        stderr=subprocess.PIPE,
        text=True,
        check=True,
    )
    return res.stdout.strip()


def detect_repo_root(target_dir: str) -> str:
    """Find the git root directory for a target path."""
    return run_git(["rev-parse", "--show-toplevel"], cwd=target_dir)


def get_current_branch(repo_dir: str) -> str:
    """Return current branch name or 'DETACHED_HEAD'."""
    try:
        branch = run_git(["rev-parse", "--abbrev-ref", "HEAD"], cwd=repo_dir)
        return branch if branch != "HEAD" else "DETACHED_HEAD"
    except subprocess.CalledProcessError:
        return "UNKNOWN"


def pin_commit_scope(
    repo_dir: str,
    base_ref: Optional[str] = None,
    head_ref: Optional[str] = None,
) -> Tuple[str, str]:
    """
    Establish exact base commit SHA and head commit SHA.
    
    If base_ref is not specified, tries:
    1. merge-base with origin/main
    2. merge-base with main
    3. HEAD~1
    4. First commit of repository
    """
    head_sha = run_git(["rev-parse", head_ref or "HEAD"], cwd=repo_dir)

    if base_ref:
        base_sha = run_git(["rev-parse", base_ref], cwd=repo_dir)
    else:
        # Try auto-detecting base
        candidates = ["origin/main", "main", "origin/master", "master"]
        base_sha = None
        for cand in candidates:
            try:
                base_sha = run_git(["merge-base", head_sha, cand], cwd=repo_dir)
                if base_sha and base_sha != head_sha:
                    break
            except subprocess.CalledProcessError:
                continue

        if not base_sha or base_sha == head_sha:
            try:
                base_sha = run_git(["rev-parse", "HEAD~1"], cwd=repo_dir)
            except subprocess.CalledProcessError:
                # Repo with single commit
                base_sha = head_sha

    return base_sha, head_sha


def extract_diff(
    repo_dir: str,
    base_sha: str,
    head_sha: str,
    include_working_tree: bool = True,
) -> Tuple[str, str]:
    """
    Extract unified diff between base and head (or working tree).
    Returns (diff_content, sha256_checksum).
    """
    if include_working_tree:
        # Include any unstaged/staged changes against base_sha
        diff_content = run_git(["diff", base_sha], cwd=repo_dir)
    else:
        diff_content = run_git(["diff", f"{base_sha}..{head_sha}"], cwd=repo_dir)

    checksum = hashlib.sha256(diff_content.encode("utf-8")).hexdigest()
    return diff_content, checksum


def snapshot_diff(
    repo_dir: str,
    out_dir: str,
    base_ref: Optional[str] = None,
    head_ref: Optional[str] = None,
    include_working_tree: bool = True,
) -> Dict[str, Any]:
    """
    Pin git scope, generate diff.patch, compute checksum, and save to out_dir.
    """
    os.makedirs(out_dir, exist_ok=True)
    base_sha, head_sha = pin_commit_scope(repo_dir, base_ref, head_ref)
    diff_text, diff_checksum = extract_diff(repo_dir, base_sha, head_sha, include_working_tree)

    patch_path = os.path.join(out_dir, "diff.patch")
    with open(patch_path, "w", encoding="utf-8") as f:
        f.write(diff_text)

    return {
        "repository_path": repo_dir,
        "branch": get_current_branch(repo_dir),
        "base_sha": base_sha,
        "head_sha": head_sha,
        "diff_checksum": diff_checksum,
        "diff_size_bytes": len(diff_text.encode("utf-8")),
        "patch_file": patch_path,
        "is_empty": len(diff_text.strip()) == 0,
    }
