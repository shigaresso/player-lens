import json
import os
from pathlib import Path
import subprocess
import sys
import tempfile
import unittest

import pr_ci_scope


class PRScopeTests(unittest.TestCase):
    def test_metadata_changes_and_full_check_fallback(self):
        cases = [
            ("本文編集は軽量", {"action": "edited", "changes": {"body": {"from": "旧本文"}}}, False),
            ("タイトル編集は軽量", {"action": "edited", "changes": {"title": {"from": "旧題"}}}, False),
            ("本文とタイトルは軽量", {"action": "edited", "changes": {"body": {}, "title": {}}}, False),
            ("統合先変更は全検査", {"action": "edited", "changes": {"base": {}}}, True),
            ("本文と統合先は全検査", {"action": "edited", "changes": {"body": {}, "base": {}}}, True),
            ("未知の変更は全検査", {"action": "edited", "changes": {"unknown": {}}}, True),
            ("空の変更情報は全検査", {"action": "edited", "changes": {}}, True),
            ("変更情報の欠落は全検査", {"action": "edited"}, True),
            ("不正な変更情報は全検査", {"action": "edited", "changes": []}, True),
        ]
        cases.extend((f"{action} のメタデータ検査は軽量", {"action": action}, False)
                     for action in ("opened", "synchronize", "reopened", "ready_for_review", "labeled", "unlabeled"))
        for name, event, expected in cases:
            with self.subTest(name):
                self.assertEqual(expected, pr_ci_scope.needs_full_checks(event))

    def test_unknown_action_is_rejected(self):
        for event in ({}, {"action": "unknown"}):
            with self.subTest(event):
                with self.assertRaises(ValueError):
                    pr_ci_scope.needs_full_checks(event)


class PRScopeCLITests(unittest.TestCase):
    def test_appends_scope_without_replacing_existing_outputs(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            event = root / "event.json"
            output = root / "outputs.txt"
            output.write_text("existing=value\n")
            for action, changes, expected in (
                    ("edited", {"body": {}}, "false"), ("edited", {"base": {}}, "true")):
                with self.subTest(changes):
                    event.write_text(json.dumps({"action": action, "changes": changes}))
                    result = self.run_cli(event, output)
                    self.assertEqual(0, result.returncode, result.stderr)
                    self.assertTrue(output.read_text().endswith(f"full_checks={expected}\n"))
                    self.assertTrue(output.read_text().startswith("existing=value\n"))

    def test_invalid_event_does_not_publish_successful_scope(self):
        with tempfile.TemporaryDirectory() as directory:
            root = Path(directory)
            event, output = root / "event.json", root / "outputs.txt"
            for name, content in (("壊れた JSON", "{"), ("未知のイベント", '{"action":"unknown"}'),
                                  ("オブジェクトではない JSON", "[]")):
                with self.subTest(name):
                    event.write_text(content)
                    result = self.run_cli(event, output)
                    self.assertNotEqual(0, result.returncode)
                    self.assertFalse(output.exists())

    def run_cli(self, event, output):
        return subprocess.run([sys.executable, str(Path(pr_ci_scope.__file__)), str(event)],
                              env={**os.environ, "GITHUB_OUTPUT": str(output)},
                              capture_output=True, text=True, timeout=10)


if __name__ == "__main__":
    unittest.main()
