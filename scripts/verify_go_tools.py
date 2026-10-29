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
        ("nil と成功を同時に返さない", "source.go",
         "package example\ntype Value struct{}\nfunc New() (*Value, error) { return nil, nil }\n", "nilnil"),
        ("後始末を含む生成関数の nil と成功を同時に返さない", "source.go",
         "package example\ntype Value struct{}\nfunc New() (*Value, func(), error) { return nil, nil, nil }\n", "nilnil"),
        ("生成エラーを空識別子へ捨てない", "source.go",
         'package example\nimport "os"\nfunc Run() { _ = os.Chdir("/") }\n', "errcheck"),
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
                             f"{module}:/src:ro", "lint", "golangci-lint", "run",
                             "--config", ".golangci.yml", "./...")
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


def verify_contracts(directory):
    module = directory / "contracts"
    module.mkdir()
    (module / "go.mod").write_text("module example.invalid/contracts\n\ngo 1.27.1\n")
    value = """package example
//contractcheck:controlled
type Value struct{ number int }
func New() *Value { return &Value{number: 1} }
func (v *Value) Read() int { return v.number }
"""
    cases = [
        ("生成済みの依存を確認せず利用できる", "contractcheck", value,
         "package example\nfunc Run() int { return New().Read() }\n", None),
        ("生成境界の迂回", "contractcheck", value,
         "package example\nfunc Run() *Value { return &Value{} }\n", "生成は定義ファイル"),
        ("定義ファイル外からのポインター更新", "contractcheck", value,
         "package example\nfunc Run(v *Value) { v.number = 0 }\n", "更新は定義ファイル"),
        ("nil にならない生成と利用", "nilaway", value,
         "package example\nfunc Run() int { return New().Read() }\n", None),
        ("関数をまたぐ nil の受け渡し", "nilaway",
         "package example\ntype Value struct{ number int }\nfunc New() *Value { return nil }\n",
         "package example\nfunc use(v *Value) int { return v.number }; func Run() int { return use(New()) }\n", "Potential nil panic"),
        ("interface に入れた nil ポインターの受け渡し", "nilaway",
         "package example\ntype Reader interface{ Read() int }; type Value struct{ number int }; func (v *Value) Read() int { return v.number }; func New() Reader { var v *Value; return v }\n",
         "package example\nfunc Run() int { return New().Read() }\n", "Potential nil panic"),
        ("生成後の更新で依存が nil になる経路", "nilaway",
         "package example\ntype Value struct{ number *int }\nfunc New() *Value { return &Value{number: new(int)} }; func (v *Value) Reset() { v.number = nil }\n",
         "package example\nfunc Run() int { v := New(); v.Reset(); return *v.number }\n", "Potential nil panic"),
        ("map の存在しないキーからの nil", "nilaway", value,
         'package example\nfunc Run() int { values := map[string]*Value{}; return values["missing"].Read() }\n', "Potential nil panic"),
    ]
    for name, tool, definition, usage, expected in cases:
        files = {"value.go": definition, "use.go": usage}
        for filename, source in files.items():
            (module / filename).write_text(source)
        command = [tool]
        if tool == "nilaway":
            command += ["-include-pkgs=example.invalid/contracts", "-pretty-print=false",
                        "-experimental-struct-init"]
        result = compose("run", "--rm", "--no-deps", "-T", "--volume",
                         f"{module}:/src:ro", "lint", *command, "./...")
        output = result.stdout + result.stderr
        if (expected is None and result.returncode != 0) or (expected is not None and
                (result.returncode == 0 or expected not in output)):
            raise RuntimeError(f"{name} の判定が異なる:\n{output}")
        if any((module / filename).read_text() != source for filename, source in files.items()):
            raise RuntimeError(f"{name} の検査で入力が変更された")
        print(f"生成契約: {name}と入力保持を確認した", flush=True)


def verify_application_contracts(directory):
    module = directory / "application"
    module.mkdir()
    tracked = subprocess.run(["git", "ls-files", "-z", "--", "go.mod", "go.sum", "cmd", "internal"],
                             cwd=ROOT, capture_output=True, check=True).stdout
    for name in tracked.decode().split("\0"):
        if name:
            destination = module / name
            destination.parent.mkdir(parents=True, exist_ok=True)
            shutil.copy2(ROOT / name, destination)
    server = module / "cmd/server/run_server.go"
    server.write_text(server.read_text() + '''
func invalidMCPFetcherProbe(config reviewmcp.HTTPConfig) error {
    limits, err := serveradapter.ReadMCPLimits(os.Getenv)
    if err != nil { return err }
    _, err = reviewmcp.NewHandler(nil, limits, config)
    return err
}
func invalidTypedMCPFetcherProbe(config reviewmcp.HTTPConfig) error {
    limits, err := serveradapter.ReadMCPLimits(os.Getenv)
    if err != nil { return err }
    fetcher, cleanup, err := serveradapter.NewSteamFetcher()
    if err != nil { return err }
    defer cleanup()
    fetcher = nil
    _, err = reviewmcp.NewHandler(fetcher, limits, config)
    return err
}
''')
    collection = module / "internal/steam/application/collect_reviews.go"
    collection.write_text(collection.read_text() + '''
func invalidCollectionFetcherProbe() (steamdomain.Collection, error) {
    return CollectReviews(context.Background(), nil, 440, steamdomain.CollectionOptions{})
}
func invalidCollectionStateProbe() { _ = steamdomain.CollectionState{} }
''')
    mcp_server = module / "internal/reviewmcp/adapter/server.go"
    mcp_server.write_text(mcp_server.read_text() + '''
func invalidControlledConstructionProbe() {
    _ = mcpdomain.Limits{}
    _ = HTTPConfig{}
}
''')
    originals = {path: path.read_bytes() for path in module.rglob("*") if path.is_file()}
    result = compose("run", "--rm", "--no-deps", "-T", "--volume", f"{module}:/src:ro",
                     "lint", "contractcheck", "./...")
    output = result.stdout + result.stderr
    if result.returncode == 0 or any(
            f"{name} の生成は定義ファイル" not in output
            for name in ("Limits", "HTTPConfig", "CollectionState")):
        raise RuntimeError(f"実アプリケーションの生成境界を拒否しなかった:\n{output}")
    result = compose("run", "--rm", "--no-deps", "-T", "--volume", f"{module}:/src:ro",
                     "lint", "nilaway", "-include-pkgs=github.com/shigaresso/player-lens",
                     "-pretty-print=false", "-experimental-struct-init",
                     "-group-error-messages=false", "./cmd/server", "./internal/steam/application")
    output = result.stdout + result.stderr
    if (result.returncode == 0 or output.count("NewHandler()") < 2 or
            "CollectReviews()" not in output):
        raise RuntimeError(f"実アプリケーションの必須引数を拒否しなかった:\n{output}")
    if any(path.read_bytes() != source for path, source in originals.items()):
        raise RuntimeError("実アプリケーションの契約検査で入力が変更された")
    print("実アプリ: NewHandler の nil・型付き nil と CollectReviews の nil を拒否した", flush=True)
    print("実アプリ: Limits・HTTPConfig・CollectionState の生成境界を検査した", flush=True)


def main():
    result = compose("build", "verify", "lint")
    if result.returncode:
        raise RuntimeError(result.stdout + result.stderr)
    with tempfile.TemporaryDirectory(prefix="player lens go tools ") as directory:
        root = Path(directory)
        verify_lint(root)
        verify_contracts(root)
        verify_application_contracts(root)
        verify_staged_format(root)
        verify_format_commands(root)


if __name__ == "__main__":
    try:
        main()
    except (OSError, ValueError, RuntimeError, subprocess.SubprocessError) as error:
        raise SystemExit(f"実 Go ツール検証が失敗した: {error}")
