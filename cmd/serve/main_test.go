package main

import (
	"net/http"
	"net/http/httptest"
	"slices"
	"testing"
	"testing/fstest"
)

// WebAssembly.instantiateStreaming は Content-Type が application/wasm でないと失敗する。
// 配信側がこれを満たしていることを検証する。
func TestHandlerServesWasmWithCorrectContentType(t *testing.T) {
	t.Parallel()

	fsys := fstest.MapFS{
		"main.wasm":  {Data: []byte("\x00asm\x01\x00\x00\x00")},
		"index.html": {Data: []byte("<!DOCTYPE html>")},
	}
	h := handler(fsys)

	tests := []struct {
		path            string
		wantContentType string
	}{
		{"/main.wasm", "application/wasm"},
		{"/", "text/html; charset=utf-8"}, // FileServerFS は /index.html を / へ301するため、rootを検証する
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			rec := httptest.NewRecorder()
			h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, tt.path, nil))

			if rec.Code != http.StatusOK {
				t.Fatalf("GET %s = %d, want %d", tt.path, rec.Code, http.StatusOK)
			}
			if got := rec.Header().Get("Content-Type"); got != tt.wantContentType {
				t.Errorf("GET %s Content-Type = %q, want %q", tt.path, got, tt.wantContentType)
			}
		})
	}
}

// 再buildした.wasmが古いキャッシュで差し替わらない事故を防ぐ。
func TestHandlerDisablesCaching(t *testing.T) {
	t.Parallel()

	h := handler(fstest.MapFS{"main.wasm": {Data: []byte("\x00asm\x01\x00\x00\x00")}})
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/main.wasm", nil))

	if got, want := rec.Header().Get("Cache-Control"), "no-store"; got != want {
		t.Errorf("Cache-Control = %q, want %q", got, want)
	}
}

// 実機から開くURLを案内できることを確認する。
// ワイルドカードのアドレスをそのまま出しても実機のSafariでは開けない。
func TestServeURLs(t *testing.T) {
	t.Parallel()

	ips := func() []string { return []string{"192.168.1.23"} }

	tests := []struct {
		name string
		addr string
		want []string
	}{
		{
			name: "explicit host",
			addr: "localhost:8080",
			want: []string{"http://localhost:8080/"},
		},
		{
			name: "wildcard lists reachable addresses",
			addr: "0.0.0.0:8080",
			want: []string{"http://localhost:8080/", "http://192.168.1.23:8080/"},
		},
		{
			name: "port only is a wildcard too",
			addr: ":8080",
			want: []string{"http://localhost:8080/", "http://192.168.1.23:8080/"},
		},
		{
			name: "unparsable address is passed through",
			addr: "not-an-address",
			want: []string{"http://not-an-address/"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := serveURLs(tt.addr, ips)
			if !slices.Equal(got, tt.want) {
				t.Errorf("serveURLs(%q) = %v, want %v", tt.addr, got, tt.want)
			}
		})
	}
}
