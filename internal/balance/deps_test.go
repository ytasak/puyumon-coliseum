package balance

import (
	"go/parser"
	"go/token"
	"os"
	"strings"
	"testing"
)

// modulePath はこのリポジトリのGo module path。
const modulePath = "github.com/ytasak/puyumon-coliseum"

// allowedImports はこのpackageのproduction codeがimportしてよいリポジトリ内のpackage。
//
// この3つの依存の先までEbitengineが混ざっていないことは internal/simulation のdeps testが辿る。
// ここで見るのは、balance自身が別の層へ依存を伸ばしていないこと。
var allowedImports = map[string]bool{
	modulePath + "/internal/battle":     true,
	modulePath + "/internal/roster":     true,
	modulePath + "/internal/simulation": true,
}

// balanceはheadlessで回せることが前提なので、UIやEbitengineへ依存しない。
func TestDependenciesStayHeadless(t *testing.T) {
	t.Parallel()

	fset := token.NewFileSet()
	skipTests := func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }
	pkgs, err := parser.ParseDir(fset, ".", skipTests, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("parseに失敗: %v", err)
	}

	for _, pkg := range pkgs {
		for name, file := range pkg.Files {
			for _, spec := range file.Imports {
				imported := strings.Trim(spec.Path.Value, `"`)
				switch {
				case strings.Contains(imported, "ebiten"):
					t.Errorf("%s が %s をimportしている", name, imported)
				case strings.HasPrefix(imported, modulePath+"/") && !allowedImports[imported]:
					t.Errorf("%s が %s をimportしている（許可していない依存）", name, imported)
				}
			}
		}
	}
}
