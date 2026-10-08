import json
import os
from pathlib import Path
import shutil
import subprocess
import sys
import tempfile
import unittest

ROOT = Path(__file__).resolve().parents[1]


class IntegrationCheckTests(unittest.TestCase):
    def setUp(self):
        directory = tempfile.TemporaryDirectory()
        self.addCleanup(directory.cleanup)
        self.root = Path(directory.name)
        (self.root / "scripts").mkdir()
        shutil.copy2(ROOT / "scripts/go-check.sh", self.root / "scripts/go-check.sh")
        (self.root / "scripts/verify_go_vcs.sh").write_text(
            '#!/usr/bin/env bash\nexit "${VCS_EXIT:-0}"\n')
        executable = self.root / "go"
        executable.write_text(f"#!{sys.executable}\n" + '''
import json
import os
import sys
from pathlib import Path
args = sys.argv[1:]
with Path("calls.jsonl").open("a") as output:
    output.write(json.dumps(args) + "\\n")
if args[0] == "list":
    if os.environ.get("LIST_ERROR"):
        print("package metadata failed", file=sys.stderr)
        sys.exit(23)
    if "-buildvcs=false" not in os.environ.get("GOFLAGS", "").split():
        print("error obtaining VCS status: exit status 128", file=sys.stderr)
        sys.exit(128)
    if not os.environ.get("LIST_EMPTY"):
        print("server_integration_test.go")
elif args[0] == "test":
    sys.exit(int(os.environ.get("TEST_EXIT", "0")))
''')
        executable.chmod(0o755)
        self.env = {**os.environ, "PATH": f"{self.root}:{os.environ['PATH']}",
                    "GOFLAGS": "-buildvcs=false"}

    def run_check(self):
        return subprocess.run(["bash", "scripts/go-check.sh", "integration"],
                              cwd=self.root, env=self.env, capture_output=True,
                              text=True, timeout=10)

    def calls(self):
        return [json.loads(line) for line in (self.root / "calls.jsonl").read_text().splitlines()]

    def test_metadata_does_not_require_vcs_and_still_runs_race_tests(self):
        result = self.run_check()
        self.assertEqual(0, result.returncode, result.stderr)
        self.assertEqual(["test", "-tags=integration", "-race", "-count=1", "./..."],
                         self.calls()[-1])

    def test_metadata_failure_is_not_misreported_as_missing_tests(self):
        self.env["LIST_ERROR"] = "1"
        result = self.run_check()
        self.assertEqual(23, result.returncode)
        self.assertIn("package metadata failed", result.stderr)
        self.assertNotIn("結合テストが存在しない", result.stderr)
        self.assertEqual(1, len(self.calls()))

    def test_missing_integration_tests_still_fail(self):
        self.env["LIST_EMPTY"] = "1"
        result = self.run_check()
        self.assertEqual(1, result.returncode)
        self.assertIn("結合テストが存在しない", result.stderr)
        self.assertEqual(1, len(self.calls()))

    def test_race_test_failure_is_preserved(self):
        self.env["TEST_EXIT"] = "7"
        self.assertEqual(7, self.run_check().returncode)

    def test_vcs_regression_failure_stops_integration_check(self):
        self.env["VCS_EXIT"] = "19"
        self.assertEqual(19, self.run_check().returncode)


if __name__ == "__main__":
    unittest.main()
