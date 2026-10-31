"""Judge decision validator, deduplicator, and severity-blocking calculator."""

from __future__ import annotations
from typing import List, Dict, Any, Set, Tuple, Optional
import os
import sys

repo_root = os.path.abspath(os.path.join(os.path.dirname(__file__), "..", ".."))
if repo_root not in sys.path:
    sys.path.insert(0, repo_root)

from scripts.review.schemas import (
    JudgeDecision,
    JudgeVerdict,
    JudgeVerdictType,
    Severity,
    SpecialistFinding,
    RemediationStatus,
)


class ReviewValidator:
    def __init__(self, block_on_medium: bool = True):
        self.block_on_medium = block_on_medium

    def does_severity_block(self, severity: Severity) -> bool:
        """Determine if a severity level blocks review/push."""
        if severity in (Severity.CRITICAL, Severity.HIGH):
            return True
        if severity == Severity.MEDIUM and self.block_on_medium:
            return True
        return False

    def deduplicate_findings(
        self,
        findings: List[SpecialistFinding],
    ) -> List[SpecialistFinding]:
        """
        Deduplicate specialist findings based on stable fingerprints.
        Preserves highest severity if duplicate fingerprints exist.
        """
        seen: Dict[str, SpecialistFinding] = {}
        severity_order = {
            Severity.CRITICAL: 5,
            Severity.HIGH: 4,
            Severity.MEDIUM: 3,
            Severity.LOW: 2,
            Severity.INFORMATIONAL: 1,
        }

        for f in findings:
            fp = f.fingerprint
            if fp not in seen:
                seen[fp] = f
            else:
                # Keep highest severity
                existing_weight = severity_order.get(seen[fp].severity, 0)
                new_weight = severity_order.get(f.severity, 0)
                if new_weight > existing_weight:
                    seen[fp] = f

        return list(seen.values())

    def evaluate_judge_verdicts(
        self,
        review_id: str,
        iteration: int,
        timestamp: str,
        verdicts: List[JudgeVerdict],
        dismissed_fingerprints_history: Optional[Set[str]] = None,
    ) -> JudgeDecision:
        """
        Process and enforce rules on judge verdicts:
        1. Ensure stable fingerprints.
        2. Filter or flag previously dismissed findings.
        3. Enforce severity-based blocking rules.
        4. Calculate summary counts and overall decision.
        """
        dismissed_history = dismissed_fingerprints_history or set()
        validated_verdicts: List[JudgeVerdict] = []
        confirmed_count = 0
        dismissed_count = 0
        needs_evidence_count = 0
        blocking_count = 0

        for v in verdicts:
            # If finding was previously dismissed in an earlier iteration and no new evidence:
            if v.fingerprint in dismissed_history and v.decision != JudgeVerdictType.CONFIRMED:
                v.decision = JudgeVerdictType.DISMISSED
                v.blocks_push = False

            # Enforce severity blocking logic
            if v.decision == JudgeVerdictType.CONFIRMED:
                confirmed_count += 1
                v.blocks_push = self.does_severity_block(v.effective_severity)
                if v.blocks_push:
                    blocking_count += 1
            elif v.decision == JudgeVerdictType.DISMISSED:
                dismissed_count += 1
                v.blocks_push = False
            elif v.decision == JudgeVerdictType.NEEDS_EVIDENCE:
                needs_evidence_count += 1
                # Needs Evidence blocks only if critical/high until clarified
                v.blocks_push = self.does_severity_block(v.effective_severity)
                if v.blocks_push:
                    blocking_count += 1

            validated_verdicts.append(v)

        overall = "PASSED" if blocking_count == 0 else "FAILED"

        summary = {
            "total_inspected": len(validated_verdicts),
            "confirmed": confirmed_count,
            "dismissed": dismissed_count,
            "needs_evidence": needs_evidence_count,
            "blocking_count": blocking_count,
        }

        return JudgeDecision(
            review_id=review_id,
            iteration=iteration,
            timestamp=timestamp,
            verdicts=validated_verdicts,
            summary=summary,
            overall_decision=overall,
        )
