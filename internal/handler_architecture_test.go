package internal_test

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

func TestHandlerPlacement(t *testing.T) {
	entries := 0
	for _, directory := range []string{"cmd", "internal"} {
		err := filepath.WalkDir(filepath.Join("..", directory), func(filename string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}

			if entry.IsDir() || !strings.HasSuffix(filename, ".go") || strings.HasSuffix(filename, "_test.go") {
				return nil
			}

			file, err := parser.ParseFile(token.NewFileSet(), filename, nil, 0)
			if err != nil {
				return err
			}

			if !hasRequestHandler(file) {
				return nil
			}

			entries++
			if !handlerLocation(filename, file.Name.Name) {
				t.Errorf("受付処理は adapter/handler の package handler に置く: %s", filename)
			}

			return nil
		})
		if err != nil {
			t.Fatal(err)
		}
	}

	if entries == 0 {
		t.Fatal("受付処理の検査対象が見つからなかった")
	}
}

func TestHandlerPlacementRules(t *testing.T) {
	for _, tc := range []struct {
		name    string
		file    string
		source  string
		allowed bool
	}{
		{"HTTPメソッドの実装をhandler内に許可する", "cmd/server/adapter/handler/health.go", `package handler; import web "net/http"; type H struct{}; func(H) ServeHTTP(w web.ResponseWriter, r *web.Request) {}`, true},
		{"HTTPメソッドの実装をhandler外で拒否する", "cmd/server/adapter/health.go", `package adapter; import "net/http"; type H struct{}; func(H) ServeHTTP(w http.ResponseWriter, r *http.Request) {}`, false},
		{"handlerの生成関数をhandler外で拒否する", "internal/reviewmcp/adapter/handler.go", `package adapter; import web "net/http"; func New() (web.Handler, error) { return nil, nil }`, false},
		{"HTTP関数の実装をhandler外で拒否する", "cmd/server/adapter/health.go", `package adapter; import "net/http"; func Health(w http.ResponseWriter, r *http.Request) {}`, false},
		{"MCPの実行関数の生成をhandler外で拒否する", "internal/reviewmcp/adapter/collection.go", `package adapter; import protocol "github.com/modelcontextprotocol/go-sdk/mcp"; func New() protocol.ToolHandler { return nil }`, false},
		{"HandlerFuncの構築をhandler外で拒否する", "internal/mcpaccess/adapter/local.go", `package adapter; import "net/http"; var protect = http.HandlerFunc(next)`, false},
		{"直接のHTTP登録callbackをhandler外で拒否する", "cmd/server/adapter/routes.go", `package adapter; func register() { mux.HandleFunc("/healthz", func(w Writer, r *Request) {}) }`, false},
		{"直接のツール登録callbackをhandler外で拒否する", "internal/reviewmcp/adapter/routes.go", `package adapter; func register() { server.AddTool(tool, func(ctx Context, r *Request) {}) }`, false},
		{"handlerフォルダでも別のpackage名を拒否する", "cmd/server/adapter/handler/health.go", `package adapter; import "net/http"; func New() http.Handler { return nil }`, false},
		{"生成済みhandlerの登録を外側に許可する", "cmd/server/adapter/routes.go", `package adapter; import "net/http"; func Register(mux *http.ServeMux, h http.Handler) { mux.Handle("GET /healthz", h) }`, true},
		{"SDKのServer構築callbackを外側に許可する", "internal/reviewmcp/adapter/server.go", `package adapter; import "net/http"; func register() { sdk.NewStreamableHTTPHandler(func(r *http.Request) *Server { return newServer() }) }`, true},
		{"外向きのHTTPクライアントを現状のadapterに許可する", "internal/steam/adapter/client.go", `package adapter; import "net/http"; func Fetch() (*http.Response, error) { return nil, nil }`, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			file, err := parser.ParseFile(token.NewFileSet(), tc.file, tc.source, 0)
			if err != nil {
				t.Fatal(err)
			}

			allowed := !hasRequestHandler(file) || handlerLocation(tc.file, file.Name.Name)
			if allowed != tc.allowed {
				t.Fatalf("配置判定=%v; want=%v", allowed, tc.allowed)
			}
		})
	}
}

func handlerLocation(filename, packageName string) bool {
	return strings.Contains(filepath.ToSlash(filename), "/adapter/handler/") && packageName == "handler"
}

func hasRequestHandler(file *ast.File) bool {
	imports := make(map[string]string)
	for _, spec := range file.Imports {
		name, err := strconv.Unquote(spec.Path.Value)
		if err != nil {
			continue
		}

		alias := path.Base(name)
		if spec.Name != nil {
			alias = spec.Name.Name
		}

		imports[alias] = name
	}

	imported := func(expr ast.Expr, packagePath, name string) bool {
		selector, ok := expr.(*ast.SelectorExpr)
		if !ok {
			return false
		}

		alias, ok := selector.X.(*ast.Ident)
		return ok && imports[alias.Name] == packagePath && selector.Sel.Name == name
	}
	const sdk = "github.com/modelcontextprotocol/go-sdk/mcp"
	found := false
	ast.Inspect(file, func(node ast.Node) bool {
		switch node := node.(type) {
		case *ast.FuncDecl:
			if node.Recv != nil && node.Name.Name == "ServeHTTP" {
				found = true
			}
		case *ast.FuncType:
			if node.Results != nil {
				for _, result := range node.Results.List {
					if imported(result.Type, "net/http", "Handler") || imported(result.Type, "net/http", "HandlerFunc") || imported(result.Type, sdk, "ToolHandler") {
						found = true
					}
				}
			}

			if node.Params != nil {
				params := node.Params.List
				if len(params) == 2 && imported(params[0].Type, "net/http", "ResponseWriter") {
					if request, ok := params[1].Type.(*ast.StarExpr); ok && imported(request.X, "net/http", "Request") {
						found = true
					}
				}
			}
		case *ast.CallExpr:
			if imported(node.Fun, "net/http", "HandlerFunc") {
				found = true
			}

			if selector, ok := node.Fun.(*ast.SelectorExpr); ok && (selector.Sel.Name == "HandleFunc" || selector.Sel.Name == "AddTool") {
				for _, arg := range node.Args {
					if _, ok := arg.(*ast.FuncLit); ok {
						found = true
					}
				}
			}
		}

		return !found
	})
	return found
}
