import subprocess
import tempfile
import unittest
from pathlib import Path
from unittest.mock import Mock

import staged_format


class StagedFormatTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory(prefix="player lens index ")
        self.addCleanup(directory.cleanup)
        self.repo = Path(directory.name)
        self.git("init", "-q")

    def git(self, *args):
        return subprocess.run(["git", *args], cwd=self.repo, check=True,
                              capture_output=True).stdout

    def test_checks_index_content_without_changing_worktree_or_index(self):
        path = self.repo / "source with spaces.go"
        staged = b"package example\n\nvar Value = 1\n"
        unstaged = b"package example;var Value=2\n"
        path.write_bytes(staged)
        self.git("add", "--", path.name)
        path.write_bytes(unstaged)
        snapshots = []

        def formatter(repo, snapshot):
            self.assertEqual(self.repo, repo)
            self.assertEqual(staged, (snapshot / path.name).read_bytes())
            snapshots.append(snapshot)

        staged_format.check_staged(self.repo, formatter=formatter)
        self.assertEqual(1, len(snapshots))
        self.assertFalse(snapshots[0].exists())
        self.assertEqual(unstaged, path.read_bytes())
        self.assertEqual(staged, self.git("show", f":{path.name}"))

    def test_non_go_changes_and_deleted_go_do_not_start_formatter(self):
        path = self.repo / "deleted.go"
        path.write_text("package example\n")
        self.git("add", path.name)
        self.git("-c", "user.name=検証", "-c", "user.email=test@example.invalid",
                 "commit", "-qm", "baseline")
        self.git("rm", path.name)
        (self.repo / "notes.md").write_text("メモ\n")
        self.git("add", "notes.md")
        formatter = Mock()
        staged_format.check_staged(self.repo, formatter=formatter)
        formatter.assert_not_called()

    def test_formatter_failure_is_preserved_and_snapshot_is_removed(self):
        path = self.repo / "nested" / "bad.go"
        path.parent.mkdir()
        path.write_text("package example;var Value=1\n")
        self.git("add", ".")
        snapshots = []

        def formatter(repo, snapshot):
            snapshots.append(snapshot)
            raise ValueError("未整形")

        with self.assertRaisesRegex(ValueError, "未整形"):
            staged_format.check_staged(self.repo, formatter=formatter)
        self.assertFalse(snapshots[0].exists())

    def test_docker_formatter_rejects_unformatted_output(self):
        run = Mock(side_effect=[subprocess.CompletedProcess([], 0, "build output", ""),
                               subprocess.CompletedProcess([], 0, "/format-input/bad.go\n", "")])
        with self.assertRaisesRegex(ValueError, "bad.go"):
            staged_format.format_snapshot(self.repo, self.repo, run=run)

    def test_build_output_is_not_treated_as_unformatted_source(self):
        run = Mock(side_effect=[subprocess.CompletedProcess([], 0, "build output", ""),
                               subprocess.CompletedProcess([], 0, "", "")])
        staged_format.format_snapshot(self.repo, self.repo, run=run)
        command = run.call_args.args[0]
        self.assertIn(f"{self.repo}:/format-input:ro", command)
        self.assertNotIn("-w", command)

    def test_docker_failure_is_not_ignored(self):
        error = subprocess.CalledProcessError(125, ["docker"], stderr="daemon stopped")
        run = Mock(side_effect=error)
        with self.assertRaises(subprocess.CalledProcessError) as raised:
            staged_format.format_snapshot(self.repo, self.repo, run=run)
        self.assertIs(error, raised.exception)


if __name__ == "__main__":
    unittest.main()
