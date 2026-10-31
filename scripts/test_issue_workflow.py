from pathlib import Path
import re
import unittest


class IssueWorkflowTriggerTests(unittest.TestCase):
    def test_issue_checks_run_for_changes_and_reopening(self):
        workflow = Path(__file__).resolve().parent.parent / ".github/workflows/issues.yml"
        trigger = re.search(r"(?m)^  issues:\n    types: \[([^\]\n]+)\]$", workflow.read_text())
        self.assertIsNotNone(trigger, "issues の対象イベントを確認できない")
        actions = {action.strip() for action in trigger.group(1).split(",")}
        self.assertEqual({"opened", "edited", "reopened", "labeled", "unlabeled"}, actions)


if __name__ == "__main__":
    unittest.main()
