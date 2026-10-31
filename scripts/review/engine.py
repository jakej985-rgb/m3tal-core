"""Core Review Engine state machine, session management, and CLI runner."""

from __future__ import annotations
import argparse
import datetime
import json
import os
import sys
import time
from typing import Dict, Any, List, Optional, Set

# Ensure repository root is on sys.path when script is invoked directly
repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
if repo_root not in sys.path:
    sys.path.insert(0, repo_root)

from scripts.review.schemas import (
    ReviewState,
    Severity,
    JudgeVerdictType,
    RemediationStatus,
    SpecialistFinding,
    JudgeVerdict,
    JudgeDecision,
)
from scripts.review.fingerprint import generate_finding_fingerprint
from scripts.review.diff_pin import snapshot_diff, detect_repo_root
from scripts.review.validator import ReviewValidator
from scripts.review.report_builder import (
    build_final_report_json,
    build_markdown_report,
    build_pr_short_summary,
)


class ReviewEngine:
    def __init__(
        self,
        target_repo: str,
        max_iterations: int = 3,
        max_duration_seconds: int = 900,
        block_on_medium: bool = True,
        review_dir: Optional[str] = None,
    ):
        self.repo_dir = os.path.abspath(target_repo)
        self.max_iterations = max_iterations
        self.max_duration_seconds = max_duration_seconds
        self.validator = ReviewValidator(block_on_medium=block_on_medium)
        self.review_root = review_dir or os.path.join(self.repo_dir, ".m3tal-review")
        self.history_dir = os.path.join(self.review_root, "history")
        self.reports_dir = os.path.join(self.review_root, "reports")
        self.active_dir = os.path.join(self.review_root, "active")

        os.makedirs(self.history_dir, exist_ok=True)
        os.makedirs(self.reports_dir, exist_ok=True)
        os.makedirs(self.active_dir, exist_ok=True)

        self.review_id: Optional[str] = None
        self.session_dir: Optional[str] = None
        self.current_iteration: int = 1
        self.state: ReviewState = ReviewState.PENDING
        self.start_time: float = time.time()
        self.last_diff_checksum: Optional[str] = None
        self.base_sha: Optional[str] = None
        self.head_sha: Optional[str] = None
        self.branch: Optional[str] = None
        self.dismissed_fingerprints: Set[str] = set()
        self.findings_lifecycle: Dict[str, Dict[str, Any]] = {}
        self.agent_summaries: Dict[str, str] = {
            "review-syntax": "PENDING",
            "review-slop": "PENDING",
            "review-security": "PENDING",
            "review-logic": "PENDING",
            "review-architecture": "PENDING",
            "review-tests": "PENDING",
        }

    def init_session(
        self,
        base_ref: Optional[str] = None,
        head_ref: Optional[str] = None,
        custom_id: Optional[str] = None,
    ) -> Dict[str, Any]:
        """Initialize a new immutable review session snapshot."""
        now = datetime.datetime.now(datetime.timezone.utc)
        self.review_id = custom_id or f"REV-{now.strftime('%Y%m%d')}-{int(now.timestamp()) % 10000:04d}"
        self.session_dir = os.path.join(self.history_dir, self.review_id)
        os.makedirs(self.session_dir, exist_ok=True)
        os.makedirs(os.path.join(self.session_dir, "specialists"), exist_ok=True)
        os.makedirs(os.path.join(self.session_dir, "judge"), exist_ok=True)
        os.makedirs(os.path.join(self.session_dir, "remediation"), exist_ok=True)

        self.start_time = time.time()
        self.state = ReviewState.INITIALIZING

        # Snapshot diff
        snap = snapshot_diff(self.repo_dir, self.session_dir, base_ref, head_ref)
        self.base_sha = snap["base_sha"]
        self.head_sha = snap["head_sha"]
        self.branch = snap["branch"]
        self.last_diff_checksum = snap["diff_checksum"]

        metadata = {
            "review_id": self.review_id,
            "created_at": now.isoformat(),
            "repository": self.repo_dir,
            "branch": self.branch,
            "base_sha": self.base_sha,
            "head_sha": self.head_sha,
            "initial_diff_checksum": self.last_diff_checksum,
            "max_iterations": self.max_iterations,
            "max_duration_seconds": self.max_duration_seconds,
            "status": "INITIALIZED",
        }

        meta_path = os.path.join(self.session_dir, "metadata.json")
        with open(meta_path, "w", encoding="utf-8") as f:
            json.dump(metadata, f, indent=2)

        # Update active pointer
        active_session = {
            "active_review_id": self.review_id,
            "session_dir": self.session_dir,
            "current_iteration": 1,
            "state": "REVIEWING",
        }
        with open(os.path.join(self.active_dir, "session.json"), "w", encoding="utf-8") as f:
            json.dump(active_session, f, indent=2)

        self.state = ReviewState.REVIEWING
        return metadata

    def record_specialist_report(self, agent_name: str, raw_findings: List[Dict[str, Any]]) -> str:
        """Validate, fingerprint, and write a specialist's findings report."""
        if not self.session_dir:
            raise RuntimeError("Review session not initialized")

        validated_findings = []
        for rf in raw_findings:
            # Guarantee agent and fingerprint
            rf["agent"] = agent_name
            if not rf.get("fingerprint"):
                loc = rf.get("location", {})
                rf["fingerprint"] = generate_finding_fingerprint(
                    file_path=rf.get("file", ""),
                    category=rf.get("category", "Logic"),
                    problem_title=rf.get("problem", ""),
                    symbol=loc.get("symbol"),
                    start_line=loc.get("start_line"),
                )
            finding = SpecialistFinding.from_dict(rf)
            validated_findings.append(finding)

        out_file = os.path.join(self.session_dir, "specialists", f"{agent_name}.json")
        with open(out_file, "w", encoding="utf-8") as f:
            json.dump([f.to_dict() for f in validated_findings], f, indent=2)

        self.agent_summaries[agent_name] = "FAIL" if any(self.validator.does_severity_block(f.severity) for f in validated_findings) else "PASS"
        return out_file

    def evaluate_judge_decision(self, raw_verdicts: List[Dict[str, Any]]) -> JudgeDecision:
        """Evaluate judge verdicts individually, record decision, and update lifecycle."""
        if not self.session_dir:
            raise RuntimeError("Review session not initialized")

        self.state = ReviewState.JUDGING
        verdicts = [JudgeVerdict.from_dict(rv) for rv in raw_verdicts]
        now = datetime.datetime.now(datetime.timezone.utc).isoformat()

        decision = self.validator.evaluate_judge_verdicts(
            review_id=self.review_id or "REV-UNKNOWN",
            iteration=self.current_iteration,
            timestamp=now,
            verdicts=verdicts,
            dismissed_fingerprints_history=self.dismissed_fingerprints,
        )

        # Update findings lifecycle and dismissed history
        for v in decision.verdicts:
            if v.decision == JudgeVerdictType.DISMISSED:
                self.dismissed_fingerprints.add(v.fingerprint)

            if v.fingerprint not in self.findings_lifecycle:
                self.findings_lifecycle[v.fingerprint] = {
                    "finding_id": v.finding_id,
                    "fingerprint": v.fingerprint,
                    "severity": v.effective_severity.value,
                    "blocks_push": v.blocks_push,
                    "status": "Dismissed" if v.decision == JudgeVerdictType.DISMISSED else "New",
                    "history": [],
                }

            self.findings_lifecycle[v.fingerprint]["history"].append({
                "iteration": self.current_iteration,
                "decision": v.decision.value,
                "rationale": v.rationale,
                "blocks_push": v.blocks_push,
            })

        out_file = os.path.join(self.session_dir, "judge", f"validation-iter-{self.current_iteration}.json")
        with open(out_file, "w", encoding="utf-8") as f:
            json.dump(decision.to_dict(), f, indent=2)

        self.state = ReviewState.EVALUATING
        return decision

    def generate_remediation_request(self, decision: JudgeDecision) -> Optional[Dict[str, Any]]:
        """If blocking findings exist and iterations remain, generate remediation request payload."""
        if not self.session_dir:
            raise RuntimeError("Review session not initialized")

        blocking = [v for v in decision.verdicts if v.blocks_push and v.decision == JudgeVerdictType.CONFIRMED]
        if not blocking:
            return None

        # Check duration limit
        elapsed = time.time() - self.start_time
        if elapsed > self.max_duration_seconds:
            self.state = ReviewState.EXHAUSTED
            return None

        # Check iteration limit
        if self.current_iteration >= self.max_iterations:
            self.state = ReviewState.EXHAUSTED
            return None

        req = {
            "protocol": "m3tal.review.remediation",
            "version": "1.0",
            "review_id": self.review_id,
            "iteration": self.current_iteration,
            "max_iterations": self.max_iterations,
            "blocking_findings": [v.to_dict() for v in blocking],
        }

        req_file = os.path.join(self.session_dir, "remediation", f"iter-{self.current_iteration}-request.json")
        with open(req_file, "w", encoding="utf-8") as f:
            json.dump(req, f, indent=2)

        self.state = ReviewState.REMEDIATING
        return req

    def verify_remediation_and_advance(self, response_payload: Dict[str, Any]) -> str:
        """
        Record coding agent response, verify updated diff against previous checksum,
        and advance iteration. Returns next state ('RE_REVIEWING' or 'BLOCKED').
        """
        if not self.session_dir:
            raise RuntimeError("Review session not initialized")

        resp_file = os.path.join(self.session_dir, "remediation", f"iter-{self.current_iteration}-response.json")
        with open(resp_file, "w", encoding="utf-8") as f:
            json.dump(response_payload, f, indent=2)

        self.state = ReviewState.VERIFYING_DIFF

        # Capture updated diff
        snap = snapshot_diff(self.repo_dir, self.session_dir, self.base_sha, self.head_sha)
        new_checksum = snap["diff_checksum"]

        # Detect unchanged diff
        if new_checksum == self.last_diff_checksum:
            self.state = ReviewState.BLOCKED
            return "BLOCKED"

        self.last_diff_checksum = new_checksum
        self.current_iteration += 1
        self.state = ReviewState.RE_REVIEWING
        return "RE_REVIEWING"

    def finalize_review(self, status_override: Optional[str] = None, rationale: Optional[str] = None) -> Dict[str, Any]:
        """Compile final report, update latest symlinks/copies, and clear active state."""
        if not self.session_dir:
            raise RuntimeError("Review session not initialized")

        elapsed = time.time() - self.start_time
        final_status = status_override or (
            "PASSED" if self.state == ReviewState.PASSED else (
                "EXHAUSTED" if self.state == ReviewState.EXHAUSTED else (
                    "BLOCKED" if self.state == ReviewState.BLOCKED else "FAILED"
                )
            )
        )

        default_rationale = {
            "PASSED": "All required review gates passed with 0 blocking defects.",
            "FAILED": "Verified blocking findings require remediation.",
            "BLOCKED": "Review cannot safely continue (unchanged diff or fatal agent error).",
            "EXHAUSTED": f"Reached maximum remediation iterations ({self.max_iterations}) with unresolved defects.",
        }.get(final_status, "Review complete.")

        report_json = build_final_report_json(
            review_id=self.review_id or "REV-UNKNOWN",
            repo_path=self.repo_dir,
            branch=self.branch or "UNKNOWN",
            base_sha=self.base_sha or "UNKNOWN",
            head_sha=self.head_sha or "UNKNOWN",
            status=final_status,
            iterations_executed=self.current_iteration,
            max_iterations=self.max_iterations,
            duration_seconds=elapsed,
            findings_history=list(self.findings_lifecycle.values()),
            dismissed_findings=[{"fingerprint": fp} for fp in self.dismissed_fingerprints],
            agent_summaries=self.agent_summaries,
            rationale=rationale or default_rationale,
        )

        final_json_path = os.path.join(self.session_dir, "final-report.json")
        with open(final_json_path, "w", encoding="utf-8") as f:
            json.dump(report_json, f, indent=2)

        markdown_content = build_markdown_report(report_json)
        final_md_path = os.path.join(self.session_dir, "final-report.md")
        with open(final_md_path, "w", encoding="utf-8") as f:
            f.write(markdown_content)

        # Copy to latest reports
        latest_json_path = os.path.join(self.reports_dir, "latest.json")
        latest_md_path = os.path.join(self.reports_dir, "latest.md")
        with open(latest_json_path, "w", encoding="utf-8") as f:
            json.dump(report_json, f, indent=2)
        with open(latest_md_path, "w", encoding="utf-8") as f:
            f.write(markdown_content)

        return report_json

    def verify_push(self, head_ref: Optional[str] = None) -> bool:
        """
        Verify if the commit being pushed has a valid, passing review audit.
        Returns True if passed, False otherwise.
        """
        import subprocess

        try:
            head_sha = subprocess.check_output(
                ["git", "rev-parse", head_ref or "HEAD"],
                cwd=self.repo_dir,
                text=True,
            ).strip()
        except subprocess.SubprocessError:
            head_sha = "UNKNOWN"

        latest_json = os.path.join(self.reports_dir, "latest.json")
        if not os.path.exists(latest_json):
            return False

        try:
            with open(latest_json, "r", encoding="utf-8") as f:
                report = json.load(f)

            report_head = report.get("head_commit") or report.get("head_sha", "")
            status = report.get("status", "")
            final_dec = report.get("final_decision", {})
            blocking = final_dec.get("blocking_unresolved", 0)

            # Match exact head or prefix
            head_matches = (
                report_head == head_sha
                or (len(head_sha) >= 7 and report_head.startswith(head_sha))
                or (len(report_head) >= 7 and head_sha.startswith(report_head))
            )
            if head_matches and status == "PASSED" and blocking == 0:
                return True
        except (json.JSONDecodeError, OSError):
            return False

        return False


def main():
    parser = argparse.ArgumentParser(description="M3TAL Review Engine CLI")
    parser.add_argument("--repo", default=".", help="Target repository directory")
    parser.add_argument("--base", default=None, help="Base commit / branch reference")
    parser.add_argument("--head", default=None, help="Head commit reference")
    parser.add_argument("--max-iterations", type=int, default=3, help="Max remediation iterations")
    parser.add_argument("--timeout", type=int, default=900, help="Max review duration seconds")
    parser.add_argument("--action", choices=["init", "summary-pr", "status", "verify-push"], default="status")

    args = parser.parse_args()
    engine = ReviewEngine(
        target_repo=args.repo,
        max_iterations=args.max_iterations,
        max_duration_seconds=args.timeout,
    )

    if args.action == "init":
        meta = engine.init_session(args.base, args.head)
        print(json.dumps(meta, indent=2))
    elif args.action == "summary-pr":
        latest_json = os.path.join(engine.reports_dir, "latest.json")
        if os.path.exists(latest_json):
            with open(latest_json, "r", encoding="utf-8") as f:
                data = json.load(f)
            print(build_pr_short_summary(data))
        else:
            print("No latest review found in .m3tal-review/reports/latest.json")
    elif args.action == "verify-push":
        if engine.verify_push(args.head):
            head_desc = args.head or "HEAD"
            print(f"✓ M3TAL Review Engine: Code review PASSED for {head_desc}")
            sys.exit(0)
        else:
            print("❌ M3TAL Review Engine: PUSH BLOCKED!")
            print("The commit being pushed has not passed local multi-agent code review.")
            print("Run the Review Orchestrator to audit and approve before pushing:")
            print("  /home/m3tal/apps/M3tal-core/.agents/agents/review-orchestrator/agent.md")
            print("Or initialize a new review session:")
            print(f"  python3 /home/m3tal/apps/M3tal-core/scripts/review/engine.py --repo {engine.repo_dir} --action init")
            sys.exit(1)
    else:
        print(f"M3TAL Review Engine initialized for: {engine.repo_dir}")


if __name__ == "__main__":
    main()
