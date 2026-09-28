package tools

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

const hubImportPath = "github.com/Sonora-Multiroom/sonora-cli/hub"

// forbiddenHTTP are net/http identifiers that would bypass the hub package.
var forbiddenHTTP = map[string]bool{
	"Get": true, "Post": true, "Head": true, "NewRequest": true, "DefaultClient": true,
}

// TestArchitecture guards Principles I and III: every hub call goes through
// the hub package, and no hub state is kept in package-level variables.
func TestArchitecture(t *testing.T) {
	root := filepath.Join("..", "..")
	fset := token.NewFileSet()
	for _, dir := range []string{"internal", "cmd"} {
		base := filepath.Join(root, dir)
		err := filepath.WalkDir(base, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			f, err := parser.ParseFile(fset, path, nil, 0)
			if err != nil {
				return err
			}
			checkFile(t, fset, f)
			return nil
		})
		if err != nil {
			t.Fatalf("walk %s: %v", base, err)
		}
	}
}

func checkFile(t *testing.T, fset *token.FileSet, f *ast.File) {
	t.Helper()
	httpName, hubName := "", ""
	for _, imp := range f.Imports {
		path, _ := strconv.Unquote(imp.Path.Value)
		name := path[strings.LastIndex(path, "/")+1:]
		if imp.Name != nil {
			name = imp.Name.Name
		}
		switch path {
		case "net/http":
			httpName = name
		case hubImportPath:
			hubName = name
		}
	}

	if httpName != "" {
		ast.Inspect(f, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == httpName && forbiddenHTTP[sel.Sel.Name] {
					t.Errorf("%s: http.%s bypasses the hub package (Principle I)", fset.Position(sel.Pos()), sel.Sel.Name)
				}
			}
			return true
		})
	}

	if hubName == "" {
		return
	}
	for _, decl := range f.Decls {
		gd, ok := decl.(*ast.GenDecl)
		if !ok || gd.Tok != token.VAR {
			continue
		}
		ast.Inspect(gd, func(n ast.Node) bool {
			if sel, ok := n.(*ast.SelectorExpr); ok {
				if x, ok := sel.X.(*ast.Ident); ok && x.Name == hubName {
					t.Errorf("%s: package-level variable uses hub.%s; hub state must not be kept (Principle III)", fset.Position(sel.Pos()), sel.Sel.Name)
				}
			}
			return true
		})
	}
}
