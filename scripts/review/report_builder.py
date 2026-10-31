"""Report builder for machine-readable JSON and human-readable Markdown outputs."""

from __future__ import annotations
import json
import os
import sys

repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
if repo_root not in sys.path:
    sys.path.insert(0, repo_root)

from typing import Dict, Any, List
from scripts.review.schemas import (
    JudgeDecision,
    JudgeVerdictType,
    Severity,
    RemediationStatus,
)


def build_final_report_json(
    review_id: str,
    repo_path: str,
    branch: str,
    base_sha: str,
    head_sha: str,
    status: str,
    iterations_executed: int,
    max_iterations: int,
    duration_seconds: float,
    findings_history: List[Dict[str, Any]],
    dismissed_findings: List[Dict[str, Any]],
    agent_summaries: Dict[str, str],
    rationale: str,
) -> Dict[str, Any]:
    """Construct complete machine-readable audit report dictionary."""
    blocking_unresolved = sum(
        1 for f in findings_history
        if f.get("status") in (RemediationStatus.NEW.value, RemediationStatus.UNRESOLVED.value, RemediationStatus.REOPENED.value)
        and f.get("blocks_push", False)
    )

    return {
        "$schema": "https://json-schema.org/draft/2020-12/schema",
        "review_id": review_id,
        "repository": repo_path,
        "branch": branch,
        "base_commit": base_sha,
        "head_commit": head_sha,
        "status": status,
        "iterations_executed": iterations_executed,
        "max_iterations": max_iterations,
        "duration_seconds": round(duration_seconds, 2),
        "agent_summaries": agent_summaries,
        "findings_history": findings_history,
        "dismissed_findings": dismissed_findings,
        "final_decision": {
            "verdict": status,
            "blocking_unresolved": blocking_unresolved,
            "rationale": rationale,
        },
    }


def build_markdown_report(report_data: Dict[str, Any]) -> str:
    """Format human-readable Markdown report conforming to Section 10 specification."""
    review_id = report_data["review_id"]
    repo = report_data["repository"].split("/")[-1]
    branch = report_data["branch"]
    status = report_data["status"]
    iters = report_data["iterations_executed"]
    duration = report_data["duration_seconds"]

    lines = [
        "# M3TAL REVIEW REPORT",
        "",
        f"**Review:** {review_id}  ",
        f"**Repository:** {repo}  ",
        f"**Branch:** {branch}  ",
        f"**Status:** {status}  ",
        f"**Iterations:** {iters}  ",
        f"**Duration:** {duration}s  ",
        "",
        "## AGENT SUMMARY",
        "",
        "| Specialist | Result |",
        "| :--- | :--- |",
    ]

    summaries = report_data.get("agent_summaries", {})
    for agent, res in summaries.items():
        lines.append(f"| {agent} | {res} |")

    lines.extend(["", "## FINDINGS", ""])

    findings = report_data.get("findings_history", [])
    if not findings:
        lines.append("No defects detected. Clean review.")
    else:
        for f in findings:
            fid = f.get("finding_id", "N/A")
            sev = f.get("severity", "Medium")
            prob = f.get("problem", "Defect")
            file_loc = f"{f.get('file', '')}:{f.get('location', {}).get('start_line', '')}"
            lines.append(f"### [{sev.upper()}] {fid}")
            lines.append(f"**Location:** `{file_loc}`  ")
            lines.append(f"**Problem:** {prob}  ")
            if f.get("scenario"):
                lines.append(f"**Failure Scenario:** {f['scenario']}  ")
            if f.get("suggested_fix"):
                lines.append(f"**Suggested Fix:** {f['suggested_fix']}  ")
            lines.append("")

    lines.extend(["## REMEDIATION", ""])
    if not findings:
        lines.append("None required.")
    else:
        for f in findings:
            fid = f.get("finding_id", "N/A")
            f_stat = f.get("status", "NEW").upper()
            lines.append(f"- **{fid}**: {f_stat}")

    lines.extend([
        "",
        "## FINAL DECISION",
        "",
        f"**{status}**",
        "",
        f"**Reason:** {report_data.get('final_decision', {}).get('rationale', 'Review process complete.')}",
        "",
    ])

    return "\n".join(lines)


def build_pr_short_summary(report_data: Dict[str, Any]) -> str:
    """Generate concise PR / commit markdown summary."""
    status_emoji = "✅" if report_data["status"] == "PASSED" else "❌"
    lines = [
        f"### {status_emoji} M3TAL Review: {report_data['status']} ({report_data['review_id']})",
        f"- **Scope:** `{report_data['branch']}` ({report_data['base_commit'][:8]}..{report_data['head_commit'][:8]})",
        f"- **Iterations:** {report_data['iterations_executed']} | **Duration:** {report_data['duration_seconds']}s",
    ]

    blocking = report_data.get("final_decision", {}).get("blocking_unresolved", 0)
    if blocking > 0:
        lines.append(f"- **Blocking Issues:** {blocking} unresolved")
    else:
        lines.append("- **Blocking Issues:** 0 (Clean)")

    findings = report_data.get("findings_history", [])
    if findings:
        lines.append("\n**Key Findings:**")
        for f in findings[:5]:  # Top 5
            lines.append(f"- `[{f.get('severity')}]` {f.get('finding_id')}: {f.get('problem')} ({f.get('status')})")

    return "\n".join(lines)
