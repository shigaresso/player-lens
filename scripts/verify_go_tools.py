"""実際の Go 整形・リンターが正常な入力と違反を区別することを検証する。"""

import shutil
import subprocess
import tempfile
from pathlib import Path

import staged_format

ROOT = Path(__file__).resolve().parents[1]


def compose(*args):
    return subprocess.run(["docker", "compose", *args], cwd=ROOT, capture_output=True,
                          text=True, timeout=600)


def verify_lint(directory):
    module = directory / "module"
    module.mkdir()
    (module / "go.mod").write_text("module example.invalid/lintfixture\n\ngo 1.27.1\n")
    (module / ".golangci.yml").write_bytes((ROOT / ".golangci.yml").read_bytes())
    error_group = '''package example
import "strconv"
func Value() (int, error) {
    value, err := strconv.Atoi("1")
    if err != nil {
        return 0, err
    }

    return value, nil
}
'''
    cleanup_group = '''package example
func prepare() (func(), error) { return func() {}, nil }
func Run() error {
    release, err := prepare()
    if err != nil {
        return err
    }
    defer release()

    return nil
}
'''
    cases = [
        ("正常", "source.go", "package example\nfunc Value() int { return 1 }\n", None),
        ("未確認エラー", "source.go", 'package example\nimport "os"\nfunc Run() { os.Chdir("/") }\n', "errcheck"),
        ("結合テストの未確認エラー", "source_integration_test.go",
         '//go:build integration\n\npackage example\nimport ("os"; "testing")\nfunc TestRun(t *testing.T) { os.Chdir("/") }\n', "errcheck"),
        ("不正な正規表現", "source.go",
         'package example\nimport "regexp"\nfunc Value() string { return regexp.MustCompile("[").String() }\n', "staticcheck"),
        ("無効な代入", "source.go", "package example\nfunc Value() int { value := 1; value = 2; return value }\n", "ineffassign"),
        ("未使用コード", "source.go", "package example\nfunc unusedValue() int { return 1 }\n", "unused"),
        ("処理とエラー確認のセット", "source.go", error_group, None),
        ("処理とエラー確認の分断", "source.go",
         error_group.replace('strconv.Atoi("1")\n', 'strconv.Atoi("1")\n\n'), "wsl_v5"),
        ("エラー対応と次の処理の区切りなし", "source.go",
         error_group.replace("}\n\n    return", "}\n    return"), "wsl_v5"),
        ("結合テストのエラー対応後の区切りなし", "source_integration_test.go",
         '//go:build integration\n\npackage example\nimport "testing"\n'
         'func TestRun(t *testing.T) {\n'
         '    if t.Failed() {\n        t.Fatal("failed")\n    }\n    t.Log("done")\n}\n', "wsl_v5"),
        ("資源取得とエラー確認とdeferのまとまり", "source.go", cleanup_group, None),
        ("連続deferのまとまり", "source.go",
         'package example\nfunc Value() int {\n'
         '    first, second := func() {}, func() {}\n    defer first()\n    defer second()\n\n'
         '    return 1\n}\n', None),
        ("最後のdeferと次の処理の区切りなし", "source.go",
         cleanup_group.replace("defer release()\n\n", "defer release()\n"), "wsl_v5"),
    ]
    for name, filename, source, expected in cases:
        path = module / filename
        path.write_text(source)
        try:
            result = compose("run", "--rm", "--no-deps", "-T", "--volume",
                             f"{module}:/src:ro", "lint")
            if (expected is None and result.returncode != 0) or (expected is not None and
                    (result.returncode != 1 or f"({expected})" not in result.stdout + result.stderr)):
                raise RuntimeError(f"{name} の判定が異なる:\n{result.stdout}\n{result.stderr}")
            if path.read_text() != source:
                raise RuntimeError(f"{name} の検査で入力が変更された")
            print(f"リンター: {name}を確認した", flush=True)
        finally:
            path.unlink()


def verify_staged_format(directory):
    repo = directory / "index"
    repo.mkdir()
    subprocess.run(["git", "init", "-q", str(repo)], check=True)
    path = repo / "source with spaces.go"
    good = b"package example\n\nvar Value = 1\n"
    bad = b"package example;var Value=2\n"
    for name, staged, worktree, failure in (
            ("ステージ済みが整形済み", good, bad, False),
            ("ステージ済みだけが未整形", bad, good, True)):
        path.write_bytes(staged)
        subprocess.run(["git", "add", "--", path.name], cwd=repo, check=True)
        path.write_bytes(worktree)
        rejected = False
        try:
            staged_format.check_staged(repo, formatter=lambda _, snapshot:
                                       staged_format.format_snapshot(ROOT, snapshot))
        except ValueError:
            rejected = True
        index = subprocess.run(["git", "show", f":{path.name}"], cwd=repo,
                               capture_output=True, check=True).stdout
        if rejected != failure or index != staged or path.read_bytes() != worktree:
            raise RuntimeError(f"{name} の判定または index・作業ツリーの保持が異なる")
        print(f"整形: {name}の判定と入力の保持を確認した", flush=True)


def verify_format_commands(directory):
    source = directory / "format commands"
    (source / "scripts").mkdir(parents=True)
    shutil.copy2(ROOT / "scripts/go-check.sh", source / "scripts/go-check.sh")
    path = source / "source with spaces.go"
    good = b"package example\n\nvar Value = 1\n"
    bad = b"package example;var Value=1\n"
    for content, expected in ((good, 0), (bad, 1)):
        path.write_bytes(content)
        result = compose("run", "--rm", "--no-deps", "-T", "--volume",
                         f"{source}:/src:ro", "verify", "bash", "scripts/go-check.sh", "format")
        if result.returncode != expected or path.read_bytes() != content:
            raise RuntimeError(f"全体の整形検査の判定または入力の保持が異なる: {result.stderr}")
    result = compose("run", "--rm", "--no-deps", "-T", "--volume",
                     f"{source}:/src:rw", "format", "bash", "scripts/go-check.sh", "format-fix")
    if result.returncode or path.read_bytes() != good:
        raise RuntimeError(f"明示的な整形修正が失敗した: {result.stderr}")
    print("整形: 全体検査と明示的な修正を確認した", flush=True)


def main():
    result = compose("build", "verify", "lint")
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    with tempfile.TemporaryDirectory(prefix="player lens go tools ") as directory:
        root = Path(directory)
        verify_lint(root)
        verify_staged_format(root)
        verify_format_commands(root)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(f"実 Go ツール検証が失敗した: {error}")
