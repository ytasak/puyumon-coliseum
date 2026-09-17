package battle

import (
	"go/parser"
	"go/token"
	"strings"
	"testing"
)

// Battle domainはUIから独立していることが前提なので、Ebitengineをimportしない。
// package doc中の説明だけでは守れないため、importを実際に確かめる。
func TestPackageDoesNotImportEbitengine(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	pkgs, err := parser.ParseDir(fset, ".", nil, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parse package: %v", err)
	}

	for name, pkg := range pkgs {
		for path, file := range pkg.Files {
			for _, spec := range file.Imports {
				imported := strings.Trim(spec.Path.Value, `"`)
				if strings.Contains(imported, "ebiten") {
					t.Errorf("%s (package %s) imports %s", path, name, imported)
				}
			}
		}
	}
}
