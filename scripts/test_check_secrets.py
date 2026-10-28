import contextlib
import io
import json
from pathlib import Path
import subprocess
import tempfile
import unittest
from unittest.mock import patch

import check_secrets


class ScannerTests(unittest.TestCase):
    def test_findings_fail_without_exposing_secret_or_process_output(self):
        with tempfile.TemporaryDirectory() as directory:
            repo = Path(directory)
            marker = "DO_NOT_PRINT_CANDIDATE"
            reports = []

            def run(args, **kwargs):
                if args[-1] == "version":
                    return subprocess.CompletedProcess(args, 0, "8.30.1\n", "")
                report = Path(args[args.index("--report-path") + 1])
                reports.append(report)
                report.write_text(json.dumps([{
                    "RuleID": "generic-api-key", "File": "config.txt", "StartLine": 2,
                    "Match": marker, "Secret": marker,
                }]))
                return subprocess.CompletedProcess(args, 3, marker, marker)

            with self.assertRaises(ValueError) as raised:
                check_secrets.scan(repo, "git", run=run)
            self.assertIn("config.txt", str(raised.exception))
            self.assertNotIn(marker, str(raised.exception))
            self.assertEqual(1, len(reports))
            self.assertFalse(reports[0].exists())


class CLITests(unittest.TestCase):
    def test_cli_fails_without_printing_untrusted_process_errors(self):
        marker = "DO_NOT_PRINT_PROCESS_DIAGNOSTIC"
        for error in [OSError(marker), subprocess.TimeoutExpired(marker, 1),
                      ValueError(marker)]:
            with self.subTest(error=type(error).__name__):
                output = io.StringIO()
                with patch("check_secrets.scan", side_effect=error), contextlib.redirect_stderr(output):
                    result = check_secrets.main(["staged"])
                self.assertEqual(1, result)
                self.assertNotIn(marker, output.getvalue())
                self.assertIn("失敗", output.getvalue())


class ReportTests(unittest.TestCase):
    def test_malformed_reports_fail_instead_of_being_treated_as_clean(self):
        for payload in [None, {}, [None], [{"RuleID": [], "File": 12, "StartLine": "invalid"}]]:
            with self.subTest(payload=payload):
                def run(args, **kwargs):
                    if args[-1] == "version":
                        return subprocess.CompletedProcess(args, 0, "8.30.1\n", "")
                    Path(args[args.index("--report-path") + 1]).write_text(json.dumps(payload))
                    return subprocess.CompletedProcess(args, 0, "", "")

                with self.assertRaises(ValueError):
                    check_secrets.scan(Path.cwd(), "git", run=run)


class ToolFailureTests(unittest.TestCase):
    def test_unavailable_or_wrong_version_and_runtime_failure_are_rejected(self):
        cases = [
            ("版違い", "0.0.0", 0),
            ("版取得失敗", "8.30.1", 2),
            ("実行失敗", "8.30.1", 0),
        ]
        for name, version, status in cases:
            with self.subTest(name=name):
                def run(args, **kwargs):
                    if args[-1] == "version":
                        return subprocess.CompletedProcess(args, status, version, "DO_NOT_PRINT")
                    return subprocess.CompletedProcess(args, 2, "DO_NOT_PRINT", "DO_NOT_PRINT")

                with self.assertRaises(ValueError) as raised:
                    check_secrets.scan(Path.cwd(), "git", run=run)
                self.assertNotIn("DO_NOT_PRINT", str(raised.exception))

    def test_environment_cannot_replace_rules_or_disable_redaction(self):
        calls = []

        def run(args, **kwargs):
            calls.append((args, kwargs))
            if args[-1] == "version":
                return subprocess.CompletedProcess(args, 0, "8.30.1", "")
            Path(args[args.index("--report-path") + 1]).write_text("[]")
            return subprocess.CompletedProcess(args, 0, "", "")

        with patch.dict("os.environ", {"GITLEAKS_CONFIG": "untrusted", "GITLEAKS_CONFIG_TOML": "untrusted"}):
            check_secrets.scan(Path.cwd(), "git", staged=True, run=run)
        args, kwargs = calls[-1]
        self.assertIn("--redact=100", args)
        self.assertIn("--staged", args)
        self.assertIn(str(check_secrets.ROOT / ".gitleaks.toml"), args)
        self.assertFalse(any(k.startswith("GITLEAKS_") for k in kwargs["env"]))
