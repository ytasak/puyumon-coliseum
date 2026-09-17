GOROOT_WASM := $(shell go env GOROOT)/lib/wasm
WEB_DIR     := web
SERVE_ADDR  := localhost:8080

.PHONY: run wasm serve fmt vet test build check clean

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
