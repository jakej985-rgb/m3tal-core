package main

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

func TestArchitectureConstraints(t *testing.T) {
	// Root paths to search relative to the cli test directory
	pathsToCheck := []string{"."}

	for _, path := range pathsToCheck {
		err := filepath.WalkDir(path, func(filePath string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() || !strings.HasSuffix(filePath, ".go") {
				return nil
			}

			// Parse file imports only
			fset := token.NewFileSet()
			fileAST, err := parser.ParseFile(fset, filePath, nil, parser.ImportsOnly)
			if err != nil {
				t.Errorf("failed to parse file %s: %v", filePath, err)
				return nil
			}

			for _, imp := range fileAST.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if strings.HasPrefix(importPath, "github.com/jakej985-rgb/m3tal-core/core/") {
					t.Errorf("Architecture Violation: File %s imports forbidden package %q", filePath, importPath)
				}
			}
			return nil
		})
		if err != nil {
			t.Fatalf("failed to walk path %s: %v", path, err)
		}
	}
}
