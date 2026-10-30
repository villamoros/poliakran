// Package architecture_test verifica automáticamente los límites entre
// contextos de ADR-0001 (LMS-TD-001). Se ejecuta con el resto de pruebas.
package architecture_test

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"strings"
	"testing"
)

const module = "github.com/villamoros/poliakran/backend/internal/"

// Paquetes externos que el dominio no puede usar: el dominio no conoce
// HTTP, bases de datos ni proveedores.
var forbiddenInDomain = []string{"net/http", "database/sql", "github.com/jackc/", "github.com/pressly/"}

func TestContextBoundaries(t *testing.T) {
	err := filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() || !strings.HasSuffix(path, ".go") || path == "architecture_test.go" {
			return err
		}
		parts := strings.Split(filepath.ToSlash(path), "/")
		owner := parts[0] // contexto o "platform"
		isDomain := len(parts) > 2 && parts[1] == "domain"

		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			ip := strings.Trim(imp.Path.Value, `"`)
			if target, ok := strings.CutPrefix(ip, module); ok {
				other := strings.Split(target, "/")[0]
				switch {
				case owner == "platform" && other != "platform":
					t.Errorf("%s: platform no puede depender del contexto %q", path, other)
				case owner != "platform" && other != owner && other != "platform":
					t.Errorf("%s: el contexto %q no puede importar el contexto %q (ADR-0001, regla 5)", path, owner, other)
				}
			}
			if isDomain {
				for _, bad := range forbiddenInDomain {
					if strings.HasPrefix(ip, bad) {
						t.Errorf("%s: el dominio no puede importar %q", path, ip)
					}
				}
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
