# ぷゆもんコロシアム 155 Battle

初代155ルール準拠の3vs3 Emojiターン制対戦ゲーム。
Client は Go + [Ebitengine](https://ebitengine.org/) で実装し、最終的に WebAssembly / iframe での配信を想定する。

仕様・設計の Single Source of Truth は Linear の
[ぷゆもんコロシアム 155 Battle](https://linear.app/ytask/project/ぷゆもんコロシアム-155-battle-2372a3a4229f) プロジェクト。
開発の進め方は Project Document「AI Development Protocol」に従う。

現在のリポジトリ状態は Milestone「Ebitengine / WASM Emoji PoC」の
[YTA-6](https://linear.app/ytask/issue/YTA-6) までを実装した、Desktop と WebAssembly の両方で
起動する最小のゲームクライアント。

## 必要環境

- Go 1.25 以上
- Desktop 実行に必要な OS 側の依存は [Ebitengine の Install 手順](https://ebitengine.org/en/documents/install.html) を参照
  （macOS は Xcode Command Line Tools のみで動作する）

## 起動方法

### Desktop

```sh
make run          # go run ./cmd/game と同じ
```

論理解像度 640x360 を 2 倍したウィンドウ（1280x720）が開き、単色背景に PoC 識別用テキストと
tick カウンタが表示される。tick カウンタが増え続けていればゲームループが動作している。

ウィンドウを閉じるとアプリケーションが終了する。

### WebAssembly

```sh
make serve        # WASM をビルドして http://localhost:8080/ で配信する
```

ブラウザで <http://localhost:8080/> を開くとゲームが起動する。
Desktop 版と同じ `cmd/game` をそのまま `GOOS=js GOARCH=wasm` でビルドしており、
WASM 用のコード分岐や build tag は無い。

iframe へ埋め込んだ状態を確認する場合は <http://localhost:8080/iframe.html> を開く。
枠内の tick カウンタが増え続けていれば iframe 内でも Update / Draw が継続している。

ビルドだけ行う場合は次のとおり。

```sh
make wasm         # web/main.wasm と web/wasm_exec.js を生成する
make clean        # 生成物を削除する
```

`file://` では `WebAssembly.instantiateStreaming` が使えないため、必ず HTTP 経由で開くこと。
配信ポートを変える場合は `make serve SERVE_ADDR=localhost:9000` のように指定する。

## 開発コマンド

```sh
make check        # fmt + vet + test + build をまとめて実行する
```

個別に実行する場合は次のとおり。

```sh
gofmt -l .        # 未フォーマットのファイルを一覧（出力が無ければ OK）
go vet ./...
GOOS=js GOARCH=wasm go vet ./...
go test ./...
go build ./...
```

## ディレクトリ構成

```text
Makefile                build / run / 検証手順
cmd/game/main.go        エントリポイント。ウィンドウ設定とゲームループの起動のみ
cmd/serve/main.go       WASM 動作確認用のローカル静的ファイルサーバ
internal/game/game.go   Game 型（ebiten.Game の Update / Draw）と描画
internal/game/layout.go 論理解像度の定数と Layout
web/index.html          Go WASM runtime と main.wasm をロードする bootstrap
web/iframe.html         iframe 埋め込み確認用ページ
```

`web/main.wasm` と `web/wasm_exec.js` は `make wasm` が生成するため commit していない。

ゲームロジックと描画を `main` へ集中させず `internal/game` に閉じている。
platform 固有の処理（ウィンドウ設定・HTTP 配信・ブラウザ bootstrap）は `cmd` と `web` に置き、
`internal/game` へ持ち込まない。
後続の Battle Engine は UI 非依存の別 package として追加し、`internal/game` から状態として参照する。

## 技術的な判断

Linear に明示されていない箇所について、以下を採用した。変更が必要になった場合は Linear 側の仕様を先に更新する。

| 項目 | 採用した内容 | 理由 |
| --- | --- | --- |
| Go module path | `github.com/ytasak/puyumon-coliseum` | GitHub へ push する前提。Product Owner 確認済み |
| Ebitengine | v2.10.2 | 実行時点の最新安定版 |
| 論理解像度 | 640x360 (16:9) 固定 | iframe 埋め込みとモバイル横持ちを想定した暫定値。`internal/game/layout.go` の定数一箇所で管理し、実ウィンドウサイズとは分離する |
| 初期ウィンドウ | 論理解像度の 2 倍 (1280x720) | Desktop で確認しやすいサイズ。論理解像度には影響しない |
| 画面テキスト | `ebitenutil.DebugPrintAt`（組み込み ASCII フォント） | フォント同梱・日本語 / Emoji 描画は YTA-7 の範囲。この段階で追加の dependency を持ち込まないため。識別用テキストは ASCII のみで構成している |
| `wasm_exec.js` | `make wasm` が GOROOT からコピーし、commit しない | Go の同梱物なのでツールチェーンとバージョンを一致させる。生成物を commit しない方針とも揃う |
| ローカル配信サーバ | 標準ライブラリだけの Go 実装（`cmd/serve`） | `.wasm` の Content-Type が `application/wasm` でないと `instantiateStreaming` が失敗する。Go の `mime` なら確実で、外部ツールへの依存も増えない |
| 配信時のキャッシュ | `Cache-Control: no-store` | 再ビルドした `.wasm` が古いキャッシュのまま検証される事故を防ぐ |

### 直接 dependency

- `github.com/hajimehoshi/ebiten/v2` — 2D engine

それ以外は Ebitengine の推移的依存のみ（`go.mod` を参照）。

## テストについて

`go test ./...` はヘッドレス環境で実行される。Ebitengine は graphics context が無いと
pixel の読み出し（`Image.At`）ができないため、描画結果そのものは検証していない。
`Draw` については「ゲームループ外から呼んでも panic しない」ことのみを確認し、
実際の見た目は Desktop 起動とブラウザでの目視確認で担保する。

`cmd/serve` については、`.wasm` の Content-Type とキャッシュ無効化を `httptest` で検証している。

## 未対応（後続 Issue）

- カラー Emoji 描画（YTA-7）
- Composite Emoji Sprite（YTA-8）
- Sprite アニメーション（YTA-9）
- iPhone Safari / iframe 検証（YTA-10）
