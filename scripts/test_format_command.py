import os
from pathlib import Path
import shutil
import subprocess
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class FormatCommandTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name).resolve()
        (self.root / "scripts").mkdir()
        for filename in ("go-check.sh", "check-go.sh"):
            shutil.copy2(ROOT / "scripts" / filename, self.root / "scripts" / filename)
        for name in ("gofmt", "docker"):
            executable = self.root / name
            executable.write_text('''#!/usr/bin/env bash
printf '%s\\n' "$@" > calls.txt
if [[ "${TOOL_EXIT:-0}" != 0 ]]; then exit "$TOOL_EXIT"; fi
if [[ "${UNFORMATTED:-0}" == 1 && "$1" == -l ]]; then echo 'source with spaces.go'; fi
''')
            executable.chmod(0o755)
        self.env = {**os.environ, "PATH": f"{self.root}:{os.environ['PATH']}"}

    def run_check(self, script, mode):
        return subprocess.run(["bash", f"scripts/{script}", mode], cwd=self.root,
                              env=self.env, capture_output=True, text=True, timeout=10)

    def test_format_reports_without_writing_and_preserves_failure(self):
        for name, unformatted, tool_exit, expected in (
                ("整形済み", "0", "0", 0), ("未整形", "1", "0", 1),
                ("ツール失敗", "0", "23", 23)):
            with self.subTest(name):
                self.env.update(UNFORMATTED=unformatted, TOOL_EXIT=tool_exit)
                result = self.run_check("go-check.sh", "format")
                self.assertEqual(expected, result.returncode, result.stderr)
                self.assertEqual(["-l", "."], (self.root / "calls.txt").read_text().splitlines())
                if unformatted == "1":
                    self.assertIn("source with spaces.go", result.stderr)

    def test_only_explicit_fix_mounts_writable_source(self):
        for mode in ("format", "format-fix"):
            with self.subTest(mode):
                result = self.run_check("check-go.sh", mode)
                self.assertEqual(0, result.returncode, result.stderr)
                args = (self.root / "calls.txt").read_text().splitlines()
                self.assertEqual(mode, args[-1])
                self.assertEqual(mode == "format-fix", "--volume" in args)
                if mode == "format-fix":
                    self.assertIn(f"{self.root}:/src:rw", args)

    def test_fix_runs_gofmt_write_and_preserves_failure(self):
        for tool_exit in (0, 17):
            with self.subTest(tool_exit):
                self.env["TOOL_EXIT"] = str(tool_exit)
                result = self.run_check("go-check.sh", "format-fix")
                self.assertEqual(tool_exit, result.returncode, result.stderr)
                self.assertEqual(["-w", "."], (self.root / "calls.txt").read_text().splitlines())


if __name__ == "__main__":
    unittest.main()
