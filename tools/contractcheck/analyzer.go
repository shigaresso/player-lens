// Package contractcheck は本番型の分類・生成と更新の境界・利用側の引数の再検査を検査する。
package contractcheck

import (
	"go/ast"
	"go/constant"
	"go/token"
	"go/types"
	"path/filepath"
	"reflect"
	"strings"

	"golang.org/x/tools/go/analysis"
)

type boundaryFact struct{ File string }

func (*boundaryFact) AFact() {}

func (f *boundaryFact) String() string { return "controlled in " + f.File }

// Analyzer は保証対象型の生成・更新位置と利用側の再検査、本番型の分類漏れを検査する。
var Analyzer = &analysis.Analyzer{
	Name:       "contractcheck",
	Doc:        "生成と更新を対象型の定義ファイルに限定する",
	Run:        run,
	FactTypes:  []analysis.Fact{new(boundaryFact)},
	ResultType: reflect.TypeFor[map[string]string](),
}

func run(pass *analysis.Pass) (any, error) {
	definitions := make(map[string]string)
	for _, file := range pass.Files {
		registerTypes(pass, file, definitions)
	}

	for _, file := range pass.Files {
		if len(definitions) > 0 || pass.Pkg.Path() == project || strings.HasPrefix(pass.Pkg.Path(), project+"/") {
			checkArguments(pass, file)
		}

		parents := parentNodes(file)
		aliases := make(map[types.Object]*types.Named)
		ast.Inspect(file, func(node ast.Node) bool {
			switch n := node.(type) {
			case *ast.CompositeLit:
				checkLiteral(pass, n)
			case *ast.ValueSpec:
				if len(n.Values) == 0 {
					for _, name := range n.Names {
						checkZero(pass, name, pass.TypesInfo.TypeOf(name))
					}
				}

				for i, value := range n.Values {
					if i < len(n.Names) {
						if isNil(pass, value) {
							checkZero(pass, value, pass.TypesInfo.TypeOf(n.Names[i]))
						}

						trackReference(pass, n.Names[i], value, aliases)
					}
				}
			case *ast.AssignStmt:
				for i, left := range n.Lhs {
					checkUpdate(pass, left)
					if i < len(n.Rhs) {
						if id, ok := n.Rhs[i].(*ast.Ident); ok && id.Name == "nil" {
							checkZero(pass, left, pass.TypesInfo.TypeOf(left))
						}

						trackReference(pass, left, n.Rhs[i], aliases)
					}
				}
			case *ast.IncDecStmt:
				checkUpdate(pass, n.X)
			case *ast.ReturnStmt:
				results := returnTypes(pass, n, parents)
				for i, value := range n.Results {
					if isNil(pass, value) && results != nil && i < results.Len() {
						checkZero(pass, value, results.At(i).Type())
					}

					if owner := referenceOwner(pass, value, aliases); owner != nil {
						pass.Reportf(value.Pos(), "%s の内部参照をそのまま返さない", owner.Obj().Name())
					}
				}
			case *ast.FuncType:
				if n.Results != nil {
					for _, result := range n.Results.List {
						for _, name := range result.Names {
							checkZero(pass, name, pass.TypesInfo.TypeOf(name))
						}
					}
				}
			case *ast.CallExpr:
				checkCall(pass, n, aliases)
			case *ast.IndexExpr:
				if _, ok := pass.TypesInfo.TypeOf(n.X).Underlying().(*types.Map); ok && !commaOK(parents[n]) {
					checkLookup(pass, n)
				}
			case *ast.UnaryExpr:
				if n.Op == token.ARROW && !commaOK(parents[n]) {
					checkLookup(pass, n)
				}

				if n.Op == token.AND {
					if owner := storageOwner(pass, n.X); owner != nil {
						checkPosition(pass, n, owner, "内部への参照取得")
					}
				}
			}

			return true
		})
	}

	return definitions, nil
}

func isNil(pass *analysis.Pass, expr ast.Expr) bool {
	id, ok := expr.(*ast.Ident)
	return ok && pass.TypesInfo.ObjectOf(id) == types.Universe.Lookup("nil")
}

func returnTypes(pass *analysis.Pass, node ast.Node, parents map[ast.Node]ast.Node) *types.Tuple {
	for parent := parents[node]; parent != nil; parent = parents[parent] {
		var t types.Type
		switch n := parent.(type) {
		case *ast.FuncDecl:
			t = pass.TypesInfo.ObjectOf(n.Name).Type()
		case *ast.FuncLit:
			t = pass.TypesInfo.TypeOf(n)
		}

		if t != nil {
			return t.(*types.Signature).Results()
		}
	}

	return nil
}

func deref(t types.Type) types.Type {
	for t != nil {
		p, ok := t.Underlying().(*types.Pointer)
		if !ok {
			return t
		}

		t = p.Elem()
	}

	return nil
}

func storageOwner(pass *analysis.Pass, expr ast.Expr) *types.Named {
	switch n := expr.(type) {
	case *ast.SelectorExpr:
		selection := pass.TypesInfo.Selections[n]
		if selection != nil && selection.Kind() == types.FieldVal {
			if owner, _ := controlled(pass, deref(selection.Recv())); owner != nil {
				return owner
			}
		}

		return storageOwner(pass, n.X)
	case *ast.IndexExpr:
		return storageOwner(pass, n.X)
	case *ast.StarExpr:
		if owner, _ := controlled(pass, deref(pass.TypesInfo.TypeOf(n))); owner != nil {
			return owner
		}

		return storageOwner(pass, n.X)
	case *ast.ParenExpr:
		return storageOwner(pass, n.X)
	}

	return nil
}

func checkUpdate(pass *analysis.Pass, expr ast.Expr) {
	if owner := storageOwner(pass, expr); owner != nil {
		checkPosition(pass, expr, owner, "更新")
	}
}

func checkPosition(pass *analysis.Pass, node ast.Node, named *types.Named, operation string) {
	fact := new(boundaryFact)
	if pass.ImportObjectFact(named.Obj(), fact) && !inside(pass, node, named, fact) {
		pass.Reportf(node.Pos(), "%s の%sは定義ファイル %s に限定する", named.Obj().Name(), operation, fact.File)
	}
}

func referenceType(t types.Type, seen map[types.Type]bool) bool {
	if t == nil || seen[t] {
		return false
	}

	seen[t] = true
	switch shape := t.Underlying().(type) {
	case *types.Pointer, *types.Slice, *types.Map, *types.Chan:
		return true
	case *types.Struct:
		for i := 0; i < shape.NumFields(); i++ {
			if referenceType(shape.Field(i).Type(), seen) {
				return true
			}
		}
	case *types.Array:
		return shape.Len() > 0 && referenceType(shape.Elem(), seen)
	}

	return false
}

func referenceOwner(pass *analysis.Pass, expr ast.Expr, aliases map[types.Object]*types.Named) *types.Named {
	if !referenceType(pass.TypesInfo.TypeOf(expr), make(map[types.Type]bool)) {
		return nil
	}

	switch n := expr.(type) {
	case *ast.SelectorExpr:
		return storageOwner(pass, n)
	case *ast.StarExpr:
		return referenceOwner(pass, n.X, aliases)
	case *ast.IndexExpr:
		return referenceOwner(pass, n.X, aliases)
	case *ast.ParenExpr:
		return referenceOwner(pass, n.X, aliases)
	case *ast.Ident:
		return aliases[pass.TypesInfo.ObjectOf(n)]
	case *ast.CallExpr:
		if builtinName(pass, n.Fun) == "append" && len(n.Args) > 0 {
			return referenceOwner(pass, n.Args[0], aliases)
		}
	}

	return nil
}

func trackReference(pass *analysis.Pass, left, right ast.Expr, aliases map[types.Object]*types.Named) {
	owner := referenceOwner(pass, right, aliases)
	if owner == nil {
		return
	}

	checkPosition(pass, right, owner, "内部への参照取得")
	if id, ok := left.(*ast.Ident); ok {
		switch pass.TypesInfo.TypeOf(right).Underlying().(type) {
		case *types.Pointer, *types.Slice, *types.Map, *types.Chan:
			aliases[pass.TypesInfo.ObjectOf(id)] = owner
		}
	}
}

func parentNodes(root ast.Node) map[ast.Node]ast.Node {
	parents := make(map[ast.Node]ast.Node)
	var stack []ast.Node
	ast.Inspect(root, func(node ast.Node) bool {
		if node == nil {
			stack = stack[:len(stack)-1]
			return false
		}

		if len(stack) > 0 {
			parents[node] = stack[len(stack)-1]
		}

		stack = append(stack, node)
		return true
	})
	return parents
}

func commaOK(node ast.Node) bool {
	switch n := node.(type) {
	case *ast.AssignStmt:
		return len(n.Lhs) == 2 && len(n.Rhs) == 1
	case *ast.ValueSpec:
		return len(n.Names) == 2 && len(n.Values) == 1
	}

	return false
}

func checkLookup(pass *analysis.Pass, expr ast.Expr) {
	for _, named := range zeroTypes(pass, pass.TypesInfo.TypeOf(expr), make(map[types.Type]bool)) {
		pass.Reportf(expr.Pos(), "%s の取得は存在確認用の bool と一緒に受け取る", named.Obj().Name())
	}
}

func checkLiteral(pass *analysis.Pass, literal *ast.CompositeLit) {
	t := pass.TypesInfo.TypeOf(literal)
	if named, _ := controlled(pass, t); named != nil {
		checkCreation(pass, literal, t)
		return
	}

	switch shape := t.Underlying().(type) {
	case *types.Struct:
		provided := make(map[string]bool)
		for i, expr := range literal.Elts {
			if pair, ok := expr.(*ast.KeyValueExpr); ok {
				name := pair.Key.(*ast.Ident).Name
				provided[name] = true
				for j := 0; j < shape.NumFields(); j++ {
					if shape.Field(j).Name() == name && isNil(pass, pair.Value) {
						checkZero(pass, pair.Value, shape.Field(j).Type())
					}
				}
			} else {
				provided[shape.Field(i).Name()] = true
				if isNil(pass, expr) {
					checkZero(pass, expr, shape.Field(i).Type())
				}
			}
		}

		for i := 0; i < shape.NumFields(); i++ {
			field := shape.Field(i)
			if !provided[field.Name()] {
				checkZero(pass, literal, field.Type())
			}
		}
	case *types.Array:
		checkNilElements(pass, literal, shape.Elem())
		if int64(len(literal.Elts)) < shape.Len() {
			checkZero(pass, literal, shape.Elem())
		}
	case *types.Slice:
		checkNilElements(pass, literal, shape.Elem())
	}
}

func checkNilElements(pass *analysis.Pass, literal *ast.CompositeLit, element types.Type) {
	for _, expr := range literal.Elts {
		if pair, ok := expr.(*ast.KeyValueExpr); ok {
			expr = pair.Value
		}

		if isNil(pass, expr) {
			checkZero(pass, expr, element)
		}
	}
}

func checkCall(pass *analysis.Pass, call *ast.CallExpr, aliases map[types.Object]*types.Named) {
	if pass.TypesInfo.Types[call.Fun].IsType() {
		if len(call.Args) == 1 && !types.Identical(pass.TypesInfo.TypeOf(call.Fun), pass.TypesInfo.TypeOf(call.Args[0])) {
			if isNil(pass, call.Args[0]) {
				checkZero(pass, call, pass.TypesInfo.TypeOf(call))
			} else {
				checkCreation(pass, call, pass.TypesInfo.TypeOf(call))
				if _, pointer := pass.TypesInfo.TypeOf(call.Args[0]).Underlying().(*types.Pointer); pointer {
					if owner, _ := controlled(pass, deref(pass.TypesInfo.TypeOf(call.Args[0]))); owner != nil {
						checkPosition(pass, call, owner, "内部への参照取得")
					}
				}
			}
		}

		return
	}

	name := builtinName(pass, call.Fun)
	if name != "len" && name != "cap" {
		for i, arg := range call.Args {
			if name == "copy" && i > 0 {
				continue
			}

			if owner := referenceOwner(pass, arg, aliases); owner != nil {
				checkPosition(pass, arg, owner, "内部への参照取得")
			}
		}
	}

	if len(call.Args) == 0 {
		return
	}

	switch name {
	case "new":
		if pass.TypesInfo.Types[call.Args[0]].IsType() {
			checkZero(pass, call, pass.TypesInfo.TypeOf(call.Args[0]))
		}
	case "make":
		slice, ok := pass.TypesInfo.TypeOf(call).Underlying().(*types.Slice)
		if !ok || len(call.Args) < 2 {
			return
		}

		length := pass.TypesInfo.Types[call.Args[1]].Value
		if length == nil || constant.Sign(length) != 0 {
			checkZero(pass, call, slice.Elem())
		}
	}
}

func builtinName(pass *analysis.Pass, expr ast.Expr) string {
	id, ok := expr.(*ast.Ident)
	if ok {
		if builtin, ok := pass.TypesInfo.Uses[id].(*types.Builtin); ok {
			return builtin.Name()
		}
	}

	return ""
}

func checkZero(pass *analysis.Pass, node ast.Node, t types.Type) {
	for _, named := range zeroTypes(pass, t, make(map[types.Type]bool)) {
		checkCreation(pass, node, named)
	}
}

func zeroTypes(pass *analysis.Pass, t types.Type, seen map[types.Type]bool) []*types.Named {
	if t == nil || seen[t] {
		return nil
	}

	seen[t] = true
	if named, _ := controlled(pass, t); named != nil {
		return []*types.Named{named}
	}

	switch shape := t.Underlying().(type) {
	case *types.Pointer:
		return zeroTypes(pass, shape.Elem(), seen)
	case *types.Array:
		if shape.Len() > 0 {
			return zeroTypes(pass, shape.Elem(), seen)
		}
	case *types.Struct:
		var result []*types.Named
		for i := 0; i < shape.NumFields(); i++ {
			result = append(result, zeroTypes(pass, shape.Field(i).Type(), seen)...)
		}

		return result
	}

	return nil
}

func controlled(pass *analysis.Pass, t types.Type) (*types.Named, *boundaryFact) {
	if t == nil {
		return nil, nil
	}

	named, ok := types.Unalias(t).(*types.Named)
	if !ok {
		return nil, nil
	}

	fact := new(boundaryFact)
	if !pass.ImportObjectFact(named.Obj(), fact) {
		return nil, nil
	}

	return named, fact
}

func checkCreation(pass *analysis.Pass, node ast.Node, t types.Type) {
	named, fact := controlled(pass, t)
	if named == nil || inside(pass, node, named, fact) {
		return
	}

	pass.Reportf(node.Pos(), "%s の生成は定義ファイル %s に限定する", named.Obj().Name(), fact.File)
}

func inside(pass *analysis.Pass, node ast.Node, named *types.Named, fact *boundaryFact) bool {
	return named.Obj().Pkg() == pass.Pkg && filepath.Base(pass.Fset.Position(node.Pos()).Filename) == fact.File
}
