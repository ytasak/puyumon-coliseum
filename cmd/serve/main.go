// Command serve はWASMビルド済みのゲームクライアントをローカルHTTPで配信する。
//
// ブラウザは file:// からWebAssemblyをstreaming instantiateできないため、
// 動作確認には静的ファイルサーバが必要になる。外部ツールへ依存せずに
// 手順を再現できるよう、標準ライブラリだけで実装している。
package main

import (
	"flag"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"path/filepath"
)

func main() {
	addr := flag.String("addr", "localhost:8080", "listen address")
	dir := flag.String("dir", "web", "directory to serve")
	flag.Parse()

	root, err := filepath.Abs(*dir)
	if err != nil {
		log.Fatalf("resolve dir: %v", err)
	}
	if _, err := os.Stat(root); err != nil {
		log.Fatalf("serve dir: %v", err)
	}

	fmt.Printf("serving %s on http://%s/\n", root, *addr)
	if err := http.ListenAndServe(*addr, handler(os.DirFS(root))); err != nil {
		log.Fatalf("listen: %v", err)
	}
}

// handler は静的ファイルを配信するhandlerを返す。
//
// キャッシュを無効化しているのは、再buildした.wasmがブラウザのキャッシュで
// 差し替わらず、古いbinaryを検証してしまう事故を防ぐため。
func handler(fsys fs.FS) http.Handler {
	files := http.FileServerFS(fsys)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Cache-Control", "no-store")
		files.ServeHTTP(w, r)
	})
}
