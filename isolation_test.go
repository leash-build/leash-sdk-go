package leash

import (
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// forbiddenFrameworks lists web framework import paths that the SDK must never import.
var forbiddenFrameworks = []string{
	"github.com/gin-gonic/gin",
	"github.com/labstack/echo",
	"github.com/gofiber/fiber",
	"github.com/go-chi/chi",
	"github.com/gorilla/mux",
}

// --- Static Checks ---

func TestNoWebFrameworkImports(t *testing.T) {
	sdkFiles := []string{"leash.go", "types.go", "gmail.go", "calendar.go", "drive.go"}

	for _, filename := range sdkFiles {
		t.Run(filename, func(t *testing.T) {
			path := filepath.Join(".", filename)
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", filename, err)
			}

			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				for _, banned := range forbiddenFrameworks {
					if strings.HasPrefix(importPath, banned) {
						t.Errorf("file %s imports banned web framework package %q", filename, importPath)
					}
				}
			}
		})
	}
}

func TestAllGoFilesNoWebFrameworkImports(t *testing.T) {
	entries, err := os.ReadDir(".")
	if err != nil {
		t.Fatal(err)
	}

	for _, entry := range entries {
		name := entry.Name()
		if entry.IsDir() || !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}

		t.Run(name, func(t *testing.T) {
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, name, nil, parser.ImportsOnly)
			if err != nil {
				t.Fatalf("failed to parse %s: %v", name, err)
			}

			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				for _, banned := range forbiddenFrameworks {
					if strings.HasPrefix(importPath, banned) {
						t.Errorf("file %s imports banned web framework package %q", name, importPath)
					}
				}
			}
		})
	}
}

// --- Runtime Isolation ---

func TestSDKBuildsWithoutWebFrameworks(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping runtime isolation test in short mode")
	}

	// Create a temporary Go module that imports the SDK.
	tmpDir, err := os.MkdirTemp("", "leash-isolation-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tmpDir)

	// Resolve the absolute path to the SDK source.
	sdkDir, err := filepath.Abs(".")
	if err != nil {
		t.Fatalf("failed to resolve SDK directory: %v", err)
	}

	// Write go.mod with a replace directive pointing at the local SDK.
	goMod := `module isolation_test

go 1.21

require github.com/leash-build/leash-sdk-go v0.0.0

replace github.com/leash-build/leash-sdk-go => ` + sdkDir + `
`
	if err := os.WriteFile(filepath.Join(tmpDir, "go.mod"), []byte(goMod), 0644); err != nil {
		t.Fatal(err)
	}

	// Write a minimal main.go that imports the SDK and uses it.
	mainGo := `package main

import (
	"fmt"
	leash "github.com/leash-build/leash-sdk-go"
)

func main() {
	c := leash.New("test-token")
	fmt.Println(c.PlatformURL)
}
`
	if err := os.WriteFile(filepath.Join(tmpDir, "main.go"), []byte(mainGo), 0644); err != nil {
		t.Fatal(err)
	}

	// Run go mod tidy to resolve transitive dependencies from the local SDK.
	tidyCmd := exec.Command("go", "mod", "tidy")
	tidyCmd.Dir = tmpDir
	tidyCmd.Env = append(os.Environ(), "GOFLAGS=")
	if output, err := tidyCmd.CombinedOutput(); err != nil {
		t.Fatalf("go mod tidy failed:\n%s\n%v", string(output), err)
	}

	// Run go build in the temp directory. If the SDK pulls in any web
	// framework dependency that isn't available, this will fail.
	cmd := exec.Command("go", "build", "-o", filepath.Join(tmpDir, "test_binary"), ".")
	cmd.Dir = tmpDir
	cmd.Env = append(os.Environ(), "GOFLAGS=")
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("SDK failed to build in a clean module without web frameworks:\n%s\n%v", string(output), err)
	}
}
