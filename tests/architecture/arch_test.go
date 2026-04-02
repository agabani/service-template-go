package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	module      = "github.com/agabani/service-template-go"
	adaptersDir = "../../internal/adapters"
	adaptersPkg = module + "/internal/adapters/"
	domainDir   = "../../internal/domain"
)

// TestDomainIsolation verifies that no domain package imports adapters or config.
// Domain packages must remain framework-free; they may only depend on the
// standard library and external packages that carry no infrastructure concerns.
//
// This rule is enforced automatically for any new package added under
// internal/domain/ — no configuration change is required.
func TestDomainIsolation(t *testing.T) {
	forbidden := []string{
		module + "/internal/adapters",
		module + "/internal/config",
	}

	err := filepath.WalkDir(domainDir, func(path string, d fs.DirEntry, _ error) error {
		if d.IsDir() || !strings.HasSuffix(path, ".go") {
			return nil
		}
		fset := token.NewFileSet()
		f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			importPath := strings.Trim(imp.Path.Value, `"`)
			for _, prefix := range forbidden {
				if importPath == prefix || strings.HasPrefix(importPath, prefix+"/") {
					t.Errorf(
						"%s imports forbidden package %q — domain must not depend on adapters or config",
						path, importPath,
					)
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatalf("walk %s: %v", domainDir, err)
	}
}

// TestAdapterIsolation verifies that no adapter sub-package imports a sibling
// adapter sub-package. Adapters must communicate through domain interfaces.
//
// This rule is enforced automatically for any new adapter added under
// internal/adapters/ — no configuration change is required.
func TestAdapterIsolation(t *testing.T) {
	entries, err := os.ReadDir(adaptersDir)
	if err != nil {
		t.Fatalf("read adapters dir: %v", err)
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		adapterName := entry.Name()
		adapterPath := filepath.Join(adaptersDir, adapterName)

		err := filepath.WalkDir(adapterPath, func(path string, d fs.DirEntry, _ error) error {
			if d.IsDir() || !strings.HasSuffix(path, ".go") {
				return nil
			}
			fset := token.NewFileSet()
			f, err := parser.ParseFile(fset, path, nil, parser.ImportsOnly)
			if err != nil {
				return err
			}
			for _, imp := range f.Imports {
				importPath := strings.Trim(imp.Path.Value, `"`)
				if !strings.HasPrefix(importPath, adaptersPkg) {
					continue
				}
				// Extract the top-level adapter name from the import path.
				// e.g. ".../adapters/postgres/..." → "postgres"
				rest := strings.TrimPrefix(importPath, adaptersPkg)
				importedAdapter := strings.SplitN(rest, "/", 2)[0]
				if importedAdapter != adapterName {
					t.Errorf(
						"%s imports sibling adapter %q — use a domain interface instead\n\timport: %s",
						path, importedAdapter, importPath,
					)
				}
			}
			return nil
		})
		if err != nil {
			t.Errorf("walk %s: %v", adapterPath, err)
		}
	}
}
