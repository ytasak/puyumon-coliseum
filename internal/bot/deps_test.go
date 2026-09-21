package bot

import (
	"go/parser"
	"go/token"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// modulePath はこのリポジトリのGo module path。
const modulePath = "github.com/ytasak/puyumon-coliseum"

// Botはsessionの外側のactorなので、production codeから
// singleplayer / simulation / Ebitengineのいずれにも依存しない。
//
// このpackage自身のimportだけでは不十分で、battleを経由して
// 引き込まれていないことまで確かめる必要がある。
// testからsingleplayerを使うのは可なので、非test fileだけを見る。
func TestProductionDependenciesStayInsideBattle(t *testing.T) {
	t.Parallel()

	forbidden := []string{
		modulePath + "/internal/singleplayer",
		modulePath + "/internal/simulation",
		modulePath + "/internal/game",
	}

	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatalf("module rootの解決に失敗: %v", err)
	}

	visited := map[string]bool{}
	queue := []string{modulePath + "/internal/bot"}
	for len(queue) > 0 {
		pkgPath := queue[0]
		queue = queue[1:]
		if visited[pkgPath] {
			continue
		}
		visited[pkgPath] = true

		dir := filepath.Join(root, strings.TrimPrefix(pkgPath, modulePath+"/"))
		if _, err := os.Stat(dir); err != nil {
			t.Fatalf("%s のdirectoryが見つからない: %v", pkgPath, err)
		}

		for _, imported := range productionImports(t, dir) {
			if strings.Contains(imported, "ebiten") {
				t.Errorf("%s が %s をimportしている（Botの依存に入っている）", pkgPath, imported)
				continue
			}
			for _, banned := range forbidden {
				if imported == banned {
					t.Errorf("%s が %s をimportしている（Botはsessionの外側のactorなので依存しない）", pkgPath, imported)
				}
			}
			if strings.HasPrefix(imported, modulePath+"/") {
				queue = append(queue, imported)
			}
		}
	}

	// 依存を辿れていること自体も確かめる。
	if !visited[modulePath+"/internal/battle"] {
		t.Errorf("依存を辿れていない: %v", visited)
	}
}

// productionImports はdirectory内の非test fileがimportしているpathを返す。
func productionImports(t *testing.T, dir string) []string {
	t.Helper()

	fset := token.NewFileSet()
	skipTests := func(info os.FileInfo) bool { return !strings.HasSuffix(info.Name(), "_test.go") }
	pkgs, err := parser.ParseDir(fset, dir, skipTests, parser.ImportsOnly)
	if err != nil {
		t.Fatalf("%s のparseに失敗: %v", dir, err)
	}

	var imports []string
	for _, pkg := range pkgs {
		for _, file := range pkg.Files {
			for _, spec := range file.Imports {
				imports = append(imports, strings.Trim(spec.Path.Value, `"`))
			}
		}
	}
	return imports
}
