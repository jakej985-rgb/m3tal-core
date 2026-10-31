"""Unit test suite for M3TAL Review Engine components."""

import os
import shutil
import subprocess
import tempfile
import unittest

from scripts.review.schemas import (
    Severity,
    FindingCategory,
    JudgeVerdictType,
    RemediationStatus,
    Location,
    SpecialistFinding,
    JudgeVerdict,
)
from scripts.review.fingerprint import generate_finding_fingerprint, normalize_file_path, normalize_text
from scripts.review.validator import ReviewValidator
from scripts.review.report_builder import (
    build_final_report_json,
    build_markdown_report,
    build_pr_short_summary,
)
from scripts.review.engine import ReviewEngine


class FingerprintTest(unittest.TestCase):
    def test_determinism(self):
        fp1 = generate_finding_fingerprint("pkg/plugins/loader.go", "Security", "Unvalidated exec path", "spawnPlugin", 84)
        fp2 = generate_finding_fingerprint("pkg/plugins/loader.go", "Security", "Unvalidated exec path", "spawnPlugin", 84)
        self.assertEqual(fp1, fp2)
        self.assertEqual(len(fp1), 64)

    def test_path_normalization(self):
        fp1 = generate_finding_fingerprint("./pkg/plugins/loader.go", "Security", "Unvalidated exec path", "spawnPlugin")
        fp2 = generate_finding_fingerprint("pkg/plugins/loader.go", "Security", "Unvalidated exec path", "spawnPlugin")
        fp3 = generate_finding_fingerprint("/pkg/plugins/loader.go", "Security", "Unvalidated exec path", "spawnPlugin")
        self.assertEqual(fp1, fp2)
        self.assertEqual(fp2, fp3)

    def test_text_normalization(self):
        fp1 = generate_finding_fingerprint("pkg/a.go", "Logic", "Unchecked error return value", "foo")
        fp2 = generate_finding_fingerprint("pkg/a.go", "Logic", "  unchecked ERROR   return value!  ", "foo")
        self.assertEqual(fp1, fp2)

    def test_anchor_differentiation(self):
        fp1 = generate_finding_fingerprint("pkg/a.go", "Logic", "Error", "funcA", 10)
        fp2 = generate_finding_fingerprint("pkg/a.go", "Logic", "Error", "funcB", 10)
        self.assertNotEqual(fp1, fp2)


class ValidatorTest(unittest.TestCase):
    def setUp(self):
        self.validator_strict = ReviewValidator(block_on_medium=True)
        self.validator_lenient = ReviewValidator(block_on_medium=False)

    def test_severity_blocking(self):
        self.assertTrue(self.validator_strict.does_severity_block(Severity.CRITICAL))
        self.assertTrue(self.validator_strict.does_severity_block(Severity.HIGH))
        self.assertTrue(self.validator_strict.does_severity_block(Severity.MEDIUM))
        self.assertFalse(self.validator_strict.does_severity_block(Severity.LOW))
        self.assertFalse(self.validator_strict.does_severity_block(Severity.INFORMATIONAL))

        # Lenient medium
        self.assertFalse(self.validator_lenient.does_severity_block(Severity.MEDIUM))

    def test_deduplication(self):
        f1 = SpecialistFinding(
            finding_id="SEC-001",
            fingerprint="abc123hash",
            agent="review-security",
            category=FindingCategory.SECURITY,
            severity=Severity.HIGH,
            file="pkg/a.go",
            location=Location(start_line=10, end_line=12),
            problem="Issue A",
            scenario="Fail A",
            impact="Impact A",
            diff_attribution="Diff A",
            suggested_fix="Fix A",
        )
        f2 = SpecialistFinding(
            finding_id="LOG-001",
            fingerprint="abc123hash",  # Same fingerprint
            agent="review-logic",
            category=FindingCategory.LOGIC,
            severity=Severity.CRITICAL,  # Higher severity
            file="pkg/a.go",
            location=Location(start_line=10, end_line=12),
            problem="Issue A",
            scenario="Fail A",
            impact="Impact A",
            diff_attribution="Diff A",
            suggested_fix="Fix A",
        )
        deduped = self.validator_strict.deduplicate_findings([f1, f2])
        self.assertEqual(len(deduped), 1)
        self.assertEqual(deduped[0].severity, Severity.CRITICAL)

    def test_judge_verdict_evaluation(self):
        v1 = JudgeVerdict(
            finding_id="SEC-001",
            fingerprint="fp1",
            decision=JudgeVerdictType.CONFIRMED,
            rationale="Valid exploit path",
            effective_severity=Severity.HIGH,
            blocks_push=False,  # Validator should enforce True
        )
        v2 = JudgeVerdict(
            finding_id="SLOP-001",
            fingerprint="fp2",
            decision=JudgeVerdictType.DISMISSED,
            rationale="Style preference",
            effective_severity=Severity.LOW,
            blocks_push=True,  # Validator should enforce False
        )
        decision = self.validator_strict.evaluate_judge_verdicts(
            review_id="REV-001",
            iteration=1,
            timestamp="2026-10-31T00:00:00Z",
            verdicts=[v1, v2],
        )
        self.assertEqual(decision.overall_decision, "FAILED")
        self.assertEqual(decision.summary["blocking_count"], 1)
        self.assertTrue(decision.verdicts[0].blocks_push)
        self.assertFalse(decision.verdicts[1].blocks_push)

    def test_dismissed_retention(self):
        v = JudgeVerdict(
            finding_id="SLOP-001",
            fingerprint="fp_dismissed",
            decision=JudgeVerdictType.NEEDS_EVIDENCE,
            rationale="Retry",
            effective_severity=Severity.LOW,
            blocks_push=False,
        )
        decision = self.validator_strict.evaluate_judge_verdicts(
            review_id="REV-001",
            iteration=2,
            timestamp="2026-10-31T00:00:00Z",
            verdicts=[v],
            dismissed_fingerprints_history={"fp_dismissed"},
        )
        self.assertEqual(decision.verdicts[0].decision, JudgeVerdictType.DISMISSED)


class ReportBuilderTest(unittest.TestCase):
    def test_reports(self):
        report_data = {
            "review_id": "REV-20261031-001",
            "repository": "/home/m3tal/apps/m3tal-core",
            "branch": "main",
            "base_commit": "1a40b28014",
            "head_commit": "d67a6cc496",
            "status": "PASSED",
            "iterations_executed": 1,
            "duration_seconds": 12.5,
            "agent_summaries": {"review-syntax": "PASS", "review-security": "PASS"},
            "findings_history": [],
            "dismissed_findings": [],
            "final_decision": {"verdict": "PASSED", "blocking_unresolved": 0, "rationale": "All gates green."},
        }

        md = build_markdown_report(report_data)
        self.assertIn("# M3TAL REVIEW REPORT", md)
        self.assertIn("REV-20261031-001", md)
        self.assertIn("All gates green.", md)

        pr_summary = build_pr_short_summary(report_data)
        self.assertIn("✅ M3TAL Review: PASSED", pr_summary)
        self.assertIn("**Blocking Issues:** 0 (Clean)", pr_summary)


class EngineLifecycleTest(unittest.TestCase):
    def setUp(self):
        self.test_dir = tempfile.mkdtemp()
        subprocess.run(["git", "init"], cwd=self.test_dir, check=True, stdout=subprocess.DEVNULL)
        subprocess.run(["git", "config", "user.name", "Test"], cwd=self.test_dir, check=True)
        subprocess.run(["git", "config", "user.email", "test@test.local"], cwd=self.test_dir, check=True)

        with open(os.path.join(self.test_dir, "file.txt"), "w") as f:
            f.write("Initial commit\n")
        subprocess.run(["git", "add", "file.txt"], cwd=self.test_dir, check=True)
        subprocess.run(["git", "commit", "-m", "init"], cwd=self.test_dir, check=True)

    def tearDown(self):
        shutil.rmtree(self.test_dir, ignore_errors=True)

    def test_engine_clean_lifecycle(self):
        engine = ReviewEngine(target_repo=self.test_dir, max_iterations=3)
        meta = engine.init_session()
        self.assertIsNotNone(meta["review_id"])

        # Record clean specialist reports
        engine.record_specialist_report("review-syntax", [])
        engine.record_specialist_report("review-security", [])

        # Judge confirms clean
        decision = engine.evaluate_judge_decision([])
        self.assertEqual(decision.overall_decision, "PASSED")

        req = engine.generate_remediation_request(decision)
        self.assertIsNone(req)  # No remediation needed

        report = engine.finalize_review(status_override="PASSED")
        self.assertEqual(report["status"], "PASSED")

    def test_unchanged_diff_blocked(self):
        # Add a change
        with open(os.path.join(self.test_dir, "file.txt"), "a") as f:
            f.write("A defect line\n")

        engine = ReviewEngine(target_repo=self.test_dir, max_iterations=3)
        engine.init_session()

        # Simulate finding
        v = {
            "finding_id": "SEC-001",
            "fingerprint": "fp_test",
            "decision": "Confirmed",
            "rationale": "Defect",
            "effective_severity": "High",
            "blocks_push": True,
        }
        decision = engine.evaluate_judge_decision([v])
        req = engine.generate_remediation_request(decision)
        self.assertIsNotNone(req)

        # Coding agent claims completed, but makes NO changes to working tree
        resp = {
            "status": "COMPLETED",
            "addressed_findings": ["SEC-001"],
            "modified_files": [],
        }
        next_state = engine.verify_remediation_and_advance(resp)
        self.assertEqual(next_state, "BLOCKED")


if __name__ == "__main__":
    unittest.main()
