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
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
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

	fmt.Printf("serving %s\n", root)
	for _, url := range serveURLs(*addr, interfaceIPs) {
		fmt.Printf("  %s\n", url)
	}
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

// serveURLs はaddrで待ち受けたときにブラウザから開けるURLを返す。
//
// 実機のiPhoneから開くには同じLANのIPアドレスが要る。0.0.0.0のような
// ワイルドカードをそのままURLにしても開けないため、ipsで得たアドレスを並べる。
func serveURLs(addr string, ips func() []string) []string {
	host, port, err := net.SplitHostPort(addr)
	if err != nil {
		// アドレスを解釈できなければ、そのまま出して判断は利用者に任せる。
		return []string{"http://" + addr + "/"}
	}

	if !isWildcard(host) {
		return []string{"http://" + net.JoinHostPort(host, port) + "/"}
	}

	urls := []string{"http://" + net.JoinHostPort("localhost", port) + "/"}
	for _, ip := range ips() {
		urls = append(urls, "http://"+net.JoinHostPort(ip, port)+"/")
	}
	return urls
}

// isWildcard はhostがすべてのinterfaceで待ち受ける指定かを返す。
func isWildcard(host string) bool {
	switch strings.ToLower(host) {
	case "", "0.0.0.0", "::", "[::]":
		return true
	}
	return false
}

// interfaceIPs はこのマシンがLAN上で持つIPv4アドレスを返す。
//
// loopbackとlink-localは実機から開けないため除く。
func interfaceIPs() []string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return nil
	}

	var ips []string
	for _, a := range addrs {
		ipnet, ok := a.(*net.IPNet)
		if !ok {
			continue
		}
		ip := ipnet.IP.To4()
		if ip == nil || ip.IsLoopback() || ip.IsLinkLocalUnicast() {
			continue
		}
		ips = append(ips, ip.String())
	}
	return ips
}
