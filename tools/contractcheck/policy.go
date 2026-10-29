package contractcheck

import (
	"go/ast"
	"go/types"
	"path/filepath"
	"strings"

	"golang.org/x/tools/go/analysis"
)

var project string

func init() {
	Analyzer.Flags.StringVar(&project, "project", "github.com/shigaresso/player-lens", "型の分類を必須にするアプリケーションのモジュールパス")
}

func registerTypes(pass *analysis.Pass, file *ast.File, definitions map[string]string) {
	filename := filepath.Base(pass.Fset.Position(file.Pos()).Filename)
	owned := (pass.Pkg.Path() == project || strings.HasPrefix(pass.Pkg.Path(), project+"/")) && !strings.HasSuffix(filename, "_test.go")
	ast.Inspect(file, func(node ast.Node) bool {
		group, ok := node.(*ast.GenDecl)
		if !ok {
			return true
		}

		for _, spec := range group.Specs {
			s, ok := spec.(*ast.TypeSpec)
			if !ok {
				continue
			}

			kind, reason := typePolicy(group.Doc, s.Doc)
			obj := pass.TypesInfo.Defs[s.Name]
			switch kind {
			case "controlled":
				registerControlled(pass, s, obj, filename, definitions)
			case "data":
				if reason == "" {
					pass.Reportf(s.Pos(), "対象外の理由を記載する")
				}
			case "dependency":
				switch obj.Type().Underlying().(type) {
				case *types.Interface, *types.Signature:
				default:
					pass.Reportf(s.Pos(), "依存の契約は interface または関数型とする")
				}
			default:
				if owned || kind != "" {
					pass.Reportf(s.Pos(), "生成・更新の分類を指定する（controlled・dependency・data と理由）")
				}
			}
		}

		return true
	})
}

func typePolicy(groups ...*ast.CommentGroup) (string, string) {
	kind, reason := "", ""
	for _, group := range groups {
		if group == nil {
			continue
		}

		for _, comment := range group.List {
			text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
			if !strings.HasPrefix(text, "contractcheck:") {
				continue
			}

			if kind != "" {
				return "duplicate", ""
			}

			fields := strings.SplitN(strings.TrimPrefix(text, "contractcheck:"), " ", 2)
			kind = fields[0]
			if len(fields) == 2 {
				reason = strings.TrimSpace(fields[1])
			}
		}
	}

	return kind, reason
}

func registerControlled(pass *analysis.Pass, spec *ast.TypeSpec, obj types.Object, filename string, definitions map[string]string) {
	structure, ok := obj.Type().Underlying().(*types.Struct)
	if !ok {
		pass.Reportf(spec.Pos(), "対象型は struct とする")
		return
	}

	for i := 0; i < structure.NumFields(); i++ {
		if structure.Field(i).Exported() {
			pass.Reportf(spec.Pos(), "対象型のフィールドは非公開にする")
			return
		}
	}

	pass.ExportObjectFact(obj, &boundaryFact{File: filename})
	definitions[obj.Name()] = filename
}
