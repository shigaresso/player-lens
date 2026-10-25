package internal_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const modulePath = "github.com/shigaresso/player-lens/"

type sourceImports struct {
	name         string
	dependencies []string
	test         bool
}

func TestLayerDependencies(t *testing.T) {
	var sources []sourceImports
	helpers := make(map[string]bool)
	for _, directory := range []string{"internal", "cmd"} {
		err := filepath.WalkDir(filepath.Join("..", directory), func(filename string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() || !strings.HasSuffix(filename, ".go") {
				return nil
			}

			relative, err := filepath.Rel("..", filename)
			if err != nil {
				return err
			}

			file, err := parser.ParseFile(token.NewFileSet(), filename, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}

			source := sourceImports{name: filepath.ToSlash(relative), test: strings.HasSuffix(filename, "_test.go")}
			for _, imported := range file.Imports {
				dependency, err := strconv.Unquote(imported.Path.Value)
				if err != nil {
					return err
				}

				source.dependencies = append(source.dependencies, dependency)
				if !source.test && dependency == "testing" {
					helpers[modulePath+path.Dir(source.name)] = true
				}
			}

			sources = append(sources, source)
			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	for _, source := range sources {
		t.Run("レイヤーの依存を保つ/"+source.name, func(t *testing.T) {
			for _, dependency := range source.dependencies {
				for _, old := range []string{"steam", "reviewmcp", "reviewcli", "jsonoutput", "mcpaccess"} {
					if dependency == modulePath+"internal/"+old {
						t.Errorf("分類前のパッケージを参照しない: %s", dependency)
					}
				}

				if strings.HasPrefix(dependency, modulePath+"internal/review/") {
					t.Errorf("既存領域を review 配下へ統合しない: %s", dependency)
				}

				if source.test {
					continue
				}

				if helpers[dependency] {
					t.Errorf("本番コードからテスト helper を参照しない: %s", dependency)
				}

				checkLayerDependency(t, source.name, dependency)
			}
		})
	}
}

func checkLayerDependency(t *testing.T, source, dependency string) {
	t.Helper()
	parts := strings.Split(source, "/")
	if len(parts) < 4 || parts[0] != "internal" || (parts[2] != "domain" && parts[2] != "application") {
		return
	}

	first, _, _ := strings.Cut(dependency, "/")
	if !strings.Contains(first, ".") {
		return
	}

	allowed := make(map[string]bool)
	if parts[2] == "application" {
		allowed[modulePath+"internal/"+parts[1]+"/domain"] = true
	}

	// MCP の利用制限は収集入力のエラーを共有し、実行操作は Steam の利用側契約を使う。
	if parts[1] == "reviewmcp" {
		allowed[modulePath+"internal/steam/domain"] = true
		if parts[2] == "application" {
			allowed[modulePath+"internal/steam/application"] = true
		}
	}

	if !allowed[dependency] {
		t.Errorf("domain/application から外向きの依存を持たない: %s", dependency)
	}
}
