package contractcheck

import (
	"go/ast"
	"go/token"
	"go/types"
	"strings"

	"golang.org/x/tools/go/analysis"
)

func checkArguments(pass *analysis.Pass, file *ast.File) {
	for _, decl := range file.Decls {
		function, ok := decl.(*ast.FuncDecl)
		if !ok {
			ast.Inspect(decl, func(node ast.Node) bool {
				if literal, ok := node.(*ast.FuncLit); ok {
					inspectArguments(pass, literal.Body, argumentOrigins(pass, literal.Type, nil))
					return false
				}

				return true
			})
			continue
		}

		if function.Body == nil {
			continue
		}

		parameters := argumentOrigins(pass, function.Type, nil)
		optional, constructor := argumentPolicy(pass, function, parameters)
		if constructor {
			continue
		}

		for object := range parameters {
			if optional[object.Name()] {
				parameters[object] = "?" + object.Name()
			}
		}

		inspectArguments(pass, function.Body, parameters)
	}
}

func argumentOrigins(pass *analysis.Pass, signature *ast.FuncType, inherited map[types.Object]string) map[types.Object]string {
	origins := make(map[types.Object]string)
	for object, name := range inherited {
		origins[object] = name
	}

	if signature.Params != nil {
		for _, field := range signature.Params.List {
			for _, name := range field.Names {
				object := pass.TypesInfo.Defs[name]
				if !types.Identical(object.Type(), types.Universe.Lookup("error").Type()) {
					origins[object] = name.Name
				}
			}
		}
	}

	return origins
}

func argumentPolicy(pass *analysis.Pass, function *ast.FuncDecl, parameters map[types.Object]string) (map[string]bool, bool) {
	optional := make(map[string]bool)
	constructor := false
	if function.Doc == nil {
		return optional, false
	}

	for _, comment := range function.Doc.List {
		text := strings.TrimSpace(strings.TrimPrefix(comment.Text, "//"))
		if text == "contractcheck:constructor" {
			constructor = true
		}

		if strings.HasPrefix(text, "contractcheck:optional") {
			fields := strings.Fields(strings.TrimPrefix(text, "contractcheck:optional"))
			if len(fields) < 2 {
				pass.Reportf(function.Pos(), "任意引数の理由を記載する")
				continue
			}

			found := false
			for object := range parameters {
				if object.Name() == fields[0] {
					found = true
					if !nilable(object.Type()) {
						pass.Reportf(function.Pos(), "任意引数 %s は nil を許容する型とする", fields[0])
					}
				}
			}

			if !found {
				pass.Reportf(function.Pos(), "任意引数 %s が存在しない", fields[0])
				continue
			}

			optional[fields[0]] = true
		}
	}

	if constructor && !constructsOwnedType(pass, function) {
		pass.Reportf(function.Pos(), "生成関数は定義ファイルの対象型を返す")
		constructor = false
	}

	return optional, constructor
}

func constructsOwnedType(pass *analysis.Pass, function *ast.FuncDecl) bool {
	if function.Recv != nil {
		return false
	}

	signature := pass.TypesInfo.Defs[function.Name].Type().(*types.Signature)
	for i := 0; i < signature.Results().Len(); i++ {
		if named, fact := controlled(pass, deref(signature.Results().At(i).Type())); named != nil && inside(pass, function, named, fact) {
			return true
		}
	}

	return false
}

func origin(pass *analysis.Pass, expression ast.Expr, origins map[types.Object]string) string {
	switch expression := expression.(type) {
	case *ast.Ident:
		return origins[pass.TypesInfo.ObjectOf(expression)]
	case *ast.ParenExpr:
		return origin(pass, expression.X, origins)
	case *ast.CallExpr:
		if pass.TypesInfo.Types[expression.Fun].IsType() && len(expression.Args) == 1 {
			return origin(pass, expression.Args[0], origins)
		}
	}

	return ""
}

func inspectArguments(pass *analysis.Pass, root ast.Node, origins map[types.Object]string) {
	ast.Inspect(root, func(node ast.Node) bool {
		switch n := node.(type) {
		case *ast.FuncLit:
			inspectArguments(pass, n.Body, argumentOrigins(pass, n.Type, origins))
			return false
		case *ast.AssignStmt:
			sources := make([]string, len(n.Rhs))
			for i, right := range n.Rhs {
				sources[i] = origin(pass, right, origins)
			}

			for i, left := range n.Lhs {
				if id, ok := left.(*ast.Ident); ok {
					object := pass.TypesInfo.ObjectOf(id)
					delete(origins, object)
					if i < len(sources) {
						if name := sources[i]; name != "" {
							origins[object] = name
						}
					}
				}
			}
		case *ast.ValueSpec:
			for i, name := range n.Names {
				if i < len(n.Values) {
					if from := origin(pass, n.Values[i], origins); from != "" {
						origins[pass.TypesInfo.Defs[name]] = from
					}
				}
			}
		case *ast.BinaryExpr:
			if n.Op == token.EQL || n.Op == token.NEQ {
				value := n.X
				if nilArgument(pass, n.X) {
					value = n.Y
				} else if !nilArgument(pass, n.Y) {
					return true
				}

				if name := origin(pass, value, origins); name != "" && !strings.HasPrefix(name, "?") {
					pass.Reportf(n.Pos(), "必須引数 %s の存在を利用側で再検査しない", name)
				}
			}
		case *ast.SwitchStmt:
			if name := origin(pass, n.Tag, origins); name != "" && !strings.HasPrefix(name, "?") {
				for _, statement := range n.Body.List {
					clause := statement.(*ast.CaseClause)
					for _, value := range clause.List {
						if nilArgument(pass, value) {
							pass.Reportf(clause.Pos(), "必須引数 %s の存在を利用側で再検査しない", name)
						}
					}
				}
			}
		case *ast.SelectorExpr:
			switch n.Sel.Name {
			case "Validate", "Valid", "IsValid", "IsZero", "IsNil":
				if name := origin(pass, n.X, origins); name != "" {
					if named, _ := controlled(pass, deref(pass.TypesInfo.TypeOf(n.X))); named != nil {
						pass.Reportf(n.Pos(), "検証済み引数 %s を利用側で再検査しない", strings.TrimPrefix(name, "?"))
					}
				}
			}
		}

		return true
	})
}

func nilArgument(pass *analysis.Pass, expression ast.Expr) bool {
	if isNil(pass, expression) {
		return true
	}

	switch expression := expression.(type) {
	case *ast.ParenExpr:
		return nilArgument(pass, expression.X)
	case *ast.CallExpr:
		return len(expression.Args) == 1 && pass.TypesInfo.Types[expression.Fun].IsType() && nilArgument(pass, expression.Args[0])
	}

	return false
}

func nilable(t types.Type) bool {
	switch t.Underlying().(type) {
	case *types.Pointer, *types.Interface, *types.Map, *types.Slice, *types.Chan, *types.Signature:
		return true
	}

	return false
}
