"""Data models and JSON serialization schemas for M3TAL Review Engine."""

from __future__ import annotations
from dataclasses import dataclass, field, asdict
from typing import List, Optional, Dict, Any
from enum import Enum
import json


class FindingCategory(str, Enum):
    SECURITY = "Security"
    LOGIC = "Logic"
    SYNTAX = "Syntax"
    ARCHITECTURE = "Architecture"
    SLOP = "Slop"
    TESTS = "Tests"


class Severity(str, Enum):
    CRITICAL = "Critical"
    HIGH = "High"
    MEDIUM = "Medium"
    LOW = "Low"
    INFORMATIONAL = "Informational"


class JudgeVerdictType(str, Enum):
    CONFIRMED = "Confirmed"
    DISMISSED = "Dismissed"
    NEEDS_EVIDENCE = "Needs Evidence"


class RemediationStatus(str, Enum):
    NEW = "New"
    FIXED = "Fixed"
    UNRESOLVED = "Unresolved"
    REOPENED = "Reopened"
    ACCEPTED_RISK = "Accepted Risk"


class ReviewState(str, Enum):
    PENDING = "PENDING"
    INITIALIZING = "INITIALIZING"
    REVIEWING = "REVIEWING"
    JUDGING = "JUDGING"
    EVALUATING = "EVALUATING"
    PASSED = "PASSED"
    FAILED = "FAILED"
    REMEDIATING = "REMEDIATING"
    AWAITING_FIX = "AWAITING_FIX"
    VERIFYING_DIFF = "VERIFYING_DIFF"
    RE_REVIEWING = "RE_REVIEWING"
    BLOCKED = "BLOCKED"
    EXHAUSTED = "EXHAUSTED"


@dataclass
class Location:
    start_line: int
    end_line: int
    symbol: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        d = {"start_line": self.start_line, "end_line": self.end_line}
        if self.symbol:
            d["symbol"] = self.symbol
        return d


@dataclass
class SpecialistFinding:
    finding_id: str
    fingerprint: str
    agent: str
    category: FindingCategory
    severity: Severity
    file: str
    location: Location
    problem: str
    scenario: str
    impact: str
    diff_attribution: str
    suggested_fix: str
    status: RemediationStatus = RemediationStatus.NEW

    def to_dict(self) -> Dict[str, Any]:
        return {
            "finding_id": self.finding_id,
            "fingerprint": self.fingerprint,
            "agent": self.agent,
            "category": self.category.value if isinstance(self.category, FindingCategory) else str(self.category),
            "severity": self.severity.value if isinstance(self.severity, Severity) else str(self.severity),
            "file": self.file,
            "location": self.location.to_dict(),
            "problem": self.problem,
            "scenario": self.scenario,
            "impact": self.impact,
            "diff_attribution": self.diff_attribution,
            "suggested_fix": self.suggested_fix,
            "status": self.status.value if isinstance(self.status, RemediationStatus) else str(self.status),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> SpecialistFinding:
        loc_data = data.get("location", {})
        location = Location(
            start_line=loc_data.get("start_line", 0),
            end_line=loc_data.get("end_line", 0),
            symbol=loc_data.get("symbol"),
        )
        return cls(
            finding_id=data["finding_id"],
            fingerprint=data["fingerprint"],
            agent=data.get("agent", "unknown"),
            category=FindingCategory(data["category"]),
            severity=Severity(data["severity"]),
            file=data["file"],
            location=location,
            problem=data["problem"],
            scenario=data.get("scenario", ""),
            impact=data.get("impact", ""),
            diff_attribution=data.get("diff_attribution", ""),
            suggested_fix=data.get("suggested_fix", ""),
            status=RemediationStatus(data.get("status", "New")),
        )


@dataclass
class JudgeVerdict:
    finding_id: str
    fingerprint: str
    decision: JudgeVerdictType
    rationale: str
    effective_severity: Severity
    blocks_push: bool
    remediation_status: RemediationStatus = RemediationStatus.NEW

    def to_dict(self) -> Dict[str, Any]:
        return {
            "finding_id": self.finding_id,
            "fingerprint": self.fingerprint,
            "decision": self.decision.value if isinstance(self.decision, JudgeVerdictType) else str(self.decision),
            "rationale": self.rationale,
            "effective_severity": self.effective_severity.value if isinstance(self.effective_severity, Severity) else str(self.effective_severity),
            "blocks_push": self.blocks_push,
            "remediation_status": self.remediation_status.value if isinstance(self.remediation_status, RemediationStatus) else str(self.remediation_status),
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> JudgeVerdict:
        return cls(
            finding_id=data["finding_id"],
            fingerprint=data["fingerprint"],
            decision=JudgeVerdictType(data["decision"]),
            rationale=data["rationale"],
            effective_severity=Severity(data["effective_severity"]),
            blocks_push=data["blocks_push"],
            remediation_status=RemediationStatus(data.get("remediation_status", "New")),
        )


@dataclass
class JudgeDecision:
    review_id: str
    iteration: int
    timestamp: str
    verdicts: List[JudgeVerdict]
    summary: Dict[str, int]
    overall_decision: str  # "PASSED" or "FAILED"

    def to_dict(self) -> Dict[str, Any]:
        return {
            "review_id": self.review_id,
            "iteration": self.iteration,
            "timestamp": self.timestamp,
            "verdicts": [v.to_dict() for v in self.verdicts],
            "summary": self.summary,
            "overall_decision": self.overall_decision,
        }

    @classmethod
    def from_dict(cls, data: Dict[str, Any]) -> JudgeDecision:
        return cls(
            review_id=data["review_id"],
            iteration=data["iteration"],
            timestamp=data["timestamp"],
            verdicts=[JudgeVerdict.from_dict(v) for v in data.get("verdicts", [])],
            summary=data.get("summary", {}),
            overall_decision=data.get("overall_decision", "FAILED"),
        )


@dataclass
class RemediationRequest:
    protocol: str
    version: str
    review_id: str
    iteration: int
    max_iterations: int
    blocking_findings: List[Dict[str, Any]]

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)


@dataclass
class RemediationResponse:
    protocol: str
    version: str
    review_id: str
    iteration: int
    status: str  # "COMPLETED", "PARTIAL", "UNABLE_TO_FIX"
    addressed_findings: List[str]
    modified_files: List[str]
    changes_summary: str
    unresolved_findings: List[Dict[str, Any]]
    notes: Optional[str] = None

    def to_dict(self) -> Dict[str, Any]:
        return asdict(self)
