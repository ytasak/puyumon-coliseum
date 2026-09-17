GOROOT_WASM := $(shell go env GOROOT)/lib/wasm
WEB_DIR     := web
DIST_DIR    := dist
SERVE_ADDR  := localhost:8080

.PHONY: run wasm serve dist fmt vet test build check clean

## Desktop版を起動する
run:
	go run ./cmd/game

## WASM binaryとGo WASM runtimeを $(WEB_DIR) へ生成する
wasm:
	cp $(GOROOT_WASM)/wasm_exec.js $(WEB_DIR)/wasm_exec.js
	GOOS=js GOARCH=wasm go build -o $(WEB_DIR)/main.wasm ./cmd/game

## WASM版をローカルHTTPで配信する（http://$(SERVE_ADDR)/）
serve: wasm
	go run ./cmd/serve -addr $(SERVE_ADDR) -dir $(WEB_DIR)

## 本番配信用の成果物を $(DIST_DIR) へ生成する。転送量を減らすため事前に圧縮する
## 配信側の設定は docs/wasm-delivery.md を参照
dist:
	rm -rf $(DIST_DIR)
	mkdir -p $(DIST_DIR)
	cp $(WEB_DIR)/index.html $(WEB_DIR)/iframe.html $(DIST_DIR)/
	cp $(GOROOT_WASM)/wasm_exec.js $(DIST_DIR)/wasm_exec.js
	GOOS=js GOARCH=wasm go build -o $(DIST_DIR)/main.wasm ./cmd/game
	gzip -9 -k -f $(DIST_DIR)/main.wasm
	@if command -v brotli >/dev/null 2>&1; then \
		brotli -q 11 -f -o $(DIST_DIR)/main.wasm.br $(DIST_DIR)/main.wasm; \
	else \
		echo 'warning: brotli が無いため main.wasm.br を生成していない（brew install brotli）'; \
	fi
	@echo
	@for f in $(DIST_DIR)/main.wasm $(DIST_DIR)/main.wasm.gz $(DIST_DIR)/main.wasm.br; do \
		if [ -f $$f ]; then printf '%-20s %10d bytes\n' $$f $$(wc -c < $$f); fi; \
	done

## 未フォーマットのファイルがあれば一覧して失敗する
fmt:
	@out=$$(gofmt -l .); \
	if [ -n "$$out" ]; then echo "$$out"; echo 'gofmt: unformatted files found'; exit 1; fi

vet:
	go vet ./...
	GOOS=js GOARCH=wasm go vet ./...

test:
	go test ./...

build:
	go build ./...

## 基本検証をまとめて実行する
check: fmt vet test build

clean:
	rm -f $(WEB_DIR)/main.wasm $(WEB_DIR)/wasm_exec.js
	rm -rf $(DIST_DIR)
