package leash

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenFrameworks lists web framework import paths that the SDK must never
// import. Carried over from 0.3 — the SDK targets framework-agnostic
// *http.Request consumers and bringing in any of these would force a
// dependency on consumers who haven't chosen that framework.
var forbiddenFrameworks = []string{
	"github.com/gin-gonic/gin",
	"github.com/labstack/echo",
	"github.com/gofiber/fiber",
	"github.com/go-chi/chi",
	"github.com/gorilla/mux",
	"github.com/golang-jwt/jwt", // 0.4 ships a stdlib-only JWT decoder
}

func TestNoForbiddenImports(t *testing.T) {
	t.Run("root_package", func(t *testing.T) {
		assertNoForbiddenImports(t, ".")
	})
	t.Run("integrations_package", func(t *testing.T) {
		assertNoForbiddenImports(t, "integrations")
	})
}

func assertNoForbiddenImports(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		path := filepath.Join(dir, name)
		t.Run(path, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("parse %s: %v", path, err)
			}
			for _, imp := range f.Imports {
				p := strings.Trim(imp.Path.Value, `"`)
				for _, banned := range forbiddenFrameworks {
					if strings.HasPrefix(p, banned) {
						t.Errorf("%s imports forbidden package %q", path, p)
					}
				}
			}
		})
	}
}
