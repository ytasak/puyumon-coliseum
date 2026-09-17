# ぷゆもんコロシアム 155 Battle

初代155ルール準拠の3vs3 Emojiターン制対戦ゲーム。
Client は Go + [Ebitengine](https://ebitengine.org/) で実装し、最終的に WebAssembly / iframe での配信を想定する。

仕様・設計の Single Source of Truth は Linear の
[ぷゆもんコロシアム 155 Battle](https://linear.app/ytask/project/ぷゆもんコロシアム-155-battle-2372a3a4229f) プロジェクト。
開発の進め方は Project Document「AI Development Protocol」に従う。

現在のリポジトリ状態は Milestone「Ebitengine / WASM Emoji PoC」の
[YTA-10](https://linear.app/ytask/issue/YTA-10) までを実装した、Desktop と WebAssembly の両方で
起動し、タップでアニメーションを発火できる Composite Emoji のゲームクライアント。
実機 iPhone Safari での検証結果は [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md) に記録する。

## 必要環境

- Go 1.25 以上
- Desktop 実行に必要な OS 側の依存は [Ebitengine の Install 手順](https://ebitengine.org/en/documents/install.html) を参照
  （macOS は Xcode Command Line Tools のみで動作する）

## 起動方法

### Desktop

```sh
make run          # go run ./cmd/game と同じ
```

論理解像度 640x360 を 2 倍したウィンドウ（1280x720）が開く。画面には次が表示される。

- 左上: PoC 識別用テキスト、`fps` / `tps`、tick、タップ回数、最後のタップ座標
- 中央: ナッシー型（`🌴` + `🥺 😫 🤪`）。待機アニメーションで上下し、待機中は `💤` が出る
- 下部: `ATTACK` / `HIT` / `EMPHASIS` のタップ領域。それぞれ `⚡` / `💥` / `❄️` が出る
- タップした位置: 黄色い十字の印

動き終わったあとに元の位置・大きさへ戻り、繰り返しても少しずつずれていかなければ成立している。
印が押した場所に出ていれば、入力座標が論理座標へ正しく変換されている。

ウィンドウを閉じるとアプリケーションが終了する。

### WebAssembly

```sh
make serve        # WASM をビルドして http://localhost:8080/ で配信する
```

ブラウザで <http://localhost:8080/> を開くとゲームが起動する。
Desktop 版と同じ `cmd/game` をそのまま `GOOS=js GOARCH=wasm` でビルドしており、
WASM 用のコード分岐や build tag は無い。

Desktop 版と同じ内容が表示され、Emoji の見た目が Desktop と一致していれば WASM 側も成立している。

iframe へ埋め込んだ状態を確認する場合は <http://localhost:8080/iframe.html> を開く。
枠内の tick カウンタが増え続けていれば iframe 内でも Update / Draw が継続している。

`main.wasm` は約 21 MB（gzip 約 5.3 MB）ある。初回ロードには時間がかかる。
内訳と削減の選択肢は [docs/emoji-rendering.md](docs/emoji-rendering.md) を参照。

### 実機（iPhone Safari）

同じ Wi-Fi にいる iPhone から開くには、LAN へ公開して配信する。

```sh
make serve SERVE_ADDR=0.0.0.0:8080
```

起動時に表示される `http://<LAN IP>:8080/` を iPhone の Safari で開く。
iframe 埋め込みの確認は `http://<LAN IP>:8080/iframe.html`。

検証手順とチェックリストは [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md) にある。

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
Makefile                    build / run / 検証手順
cmd/game/main.go            エントリポイント。ウィンドウ設定とゲームループの起動のみ
cmd/serve/main.go           WASM 動作確認用のローカル静的ファイルサーバ
internal/game/game.go       Game 型（ebiten.Game の Update / Draw）と描画
internal/game/layout.go     論理解像度の定数と Layout
internal/game/spritescene.go PoC の画面構成、キャラクター定義、タップ領域
internal/game/input.go      マウス / タッチの取得
internal/sprite/            複数 Emoji を 1 体として定義・描画する Composite Sprite 層
internal/anim/              アニメーションの状態管理と Emoji particle
docs/mobile-safari-poc.md   iPhone Safari / iframe 検証の手順と記録（YTA-10）
internal/emoji/             Emoji 素材のフォント読み込みとセル画像のキャッシュ
internal/emoji/assets/      同梱フォントと、その出典・ライセンス
docs/emoji-rendering.md     カラー Emoji 描画の検証記録（YTA-7）
web/index.html              Go WASM runtime と main.wasm をロードする bootstrap
web/iframe.html             iframe 埋め込み確認用ページ
```

`web/main.wasm` と `web/wasm_exec.js` は `make wasm` が生成するため commit していない。

ゲームロジックと描画を `main` へ集中させず `internal/game` に閉じている。
platform 固有の処理（ウィンドウ設定・HTTP 配信・ブラウザ bootstrap）は `cmd` と `web` に置き、
`internal/game` へ持ち込まない。
後続の Battle Engine は UI 非依存の別 package として追加し、`internal/game` から状態として参照する。

`internal/emoji` は Emoji を「UI テキスト」ではなく「スプライト素材」として供給する層で、
画面構成（どこに何を並べるか）は持たない。画面構成は `internal/game` 側に置く。

`internal/sprite` はキャラクターの**定義**（どの Emoji をどこに置くか）と**描画**を分けている。
定義と座標計算は Ebitengine に依存せず、描画だけが `Renderer` に閉じている。
キャラクターごとの分岐は描画側に持たせないため、新しいキャラクターの追加は定義を増やすだけで済む。

`internal/anim` はアニメーションの**状態**だけを持ち、キャラクター定義も base transform も持たない。
描画に使う transform は毎回 base から計算し直すため、再生を繰り返してもずれが蓄積しない。
Ebitengine に依存しないので、動きの検証は描画コンテキストなしで行える。

## 技術的な判断

Linear に明示されていない箇所について、以下を採用した。変更が必要になった場合は Linear 側の仕様を先に更新する。

| 項目 | 採用した内容 | 理由 |
| --- | --- | --- |
| Go module path | `github.com/ytasak/puyumon-coliseum` | GitHub へ push する前提。Product Owner 確認済み |
| Ebitengine | v2.10.2 | 実行時点の最新安定版 |
| 論理解像度 | 640x360 (16:9) 固定 | iframe 埋め込みとモバイル横持ちを想定した暫定値。`internal/game/layout.go` の定数一箇所で管理し、実ウィンドウサイズとは分離する |
| 初期ウィンドウ | 論理解像度の 2 倍 (1280x720) | Desktop で確認しやすいサイズ。論理解像度には影響しない |
| 画面テキスト | `ebitenutil.DebugPrintAt`（組み込み ASCII フォント） | 識別用テキストは PoC 用途なので追加フォントを持ち込まない。ASCII のみで構成している。日本語の UI フォントは別 Issue の範囲 |
| Emoji フォント | Twemoji Mozilla 0.7.0（COLRv0）を同梱 | Ebitengine v2.10 は COLRv0 を描画できるが **COLRv1 は非対応**。候補中もっとも小さく（1.4 MB）、ベクターで拡大に強く、送り幅が正方 1em で共通の描画単位を定義しやすい。比較の実測値は [docs/emoji-rendering.md](docs/emoji-rendering.md) |
| Emoji の描画単位 | 128px 四方のセル画像 1 枚 = Emoji 1 文字。アンカーは em box の中心 | 表示サイズをセルの拡大縮小だけで決め、Emoji ごとの位置補正を不要にする。セルをキャッシュするので毎フレームのグリフ生成も起きない |
| Emoji のフォールバック | OS のフォントにフォールバックしない | Desktop とブラウザで同じ絵を出すため、同梱フォントだけで描画結果を固定する |
| キャラクターの座標系 | character-local 座標。1.0 = Emoji セル 1 個分、原点はキャラクター中心 | 画面サイズや表示倍率と定義を切り離す。キャラクターごとに暗黙の基準を作らないため、アンカーは全部品でセル中心に統一する |
| `Transform.Scale` の単位 | character-local 座標 1.0 あたりの pixel 数 | 「キャラクターを何 px で出すか」を 1 つの値で決められる。部品の大きさと部品間の距離が必ず同じ倍率で変わる |
| 描画順 | `NewCharacter` が `Z` の昇順へ 1 度だけ並べ替え、`Draw` はその順に描く | 描画のたびにソートしない。同じ `Z` は定義順を保つので、定義側で順序を明示できる |
| Composite のキャッシュ | 実装しない。`Draw` が `dst` を引数に取るので offscreen へ描いて使い回せる構造にとどめる | PoC のキャラクターは数部品しかなく、キャッシュしても得られるものが少ない。必要になった時点で呼び出し側が offscreen を用意すればよい |
| 2 体目のキャラクター（クジラ型） | 定義だけ追加し、描画コードは増やさない | 「新キャラクター追加に専用 draw function を必要としない」ことを画面とテストの両方で確認するため。デザインとしては未確定 |
| アニメーションの時間単位 | Ebitengine の tick（60 TPS）。長さも tick で持つ | フレーム数にも実時間にも依存せず、テストから同じ単位で進められる。`Update` を呼んだ回数がそのまま時間になる |
| ずれ（drift）の防ぎ方 | 状態を足し込まず、毎 tick base transform から計算し直す | 「アニメーション後に必ず base へ戻る」を設計で保証する。戻し忘れが起きる余地を作らない |
| 複数アニメーションの競合 | 単発アニメーションの同時再生を禁止し、再生中の要求は順番待ちへ積む | Issue が PoC で許容している方式。どの動きが出ているかが常に 1 つに定まる。中断は行わない |
| 動きの大きさの単位 | character-local 座標（`1.0` = Emoji セル 1 個分）で持ち、`Transform.Scale` を掛ける | キャラクターを拡大縮小しても動きの見た目の比率が変わらない |
| particle の消え方 | 終盤で縮めて消す。アルファは使わない | `Renderer` に色・透明度の引数を増やさずに済む。PoC で必要な「一定時間後に消滅」は満たせる |
| 入力の取得 | マウスとタッチの両方を拾い、座標変換は Ebitengine へ任せる | Desktop・ブラウザ・iframe の中で同じコードになる。変換が合っているかは画面のタップマーカーで実機確認する |
| タッチとページの競合 | canvas へ `touch-action: none`、viewport で `user-scalable=no` | これが無いと、ゲームを操作したつもりでページがスクロール・ズームしてしまい、入力の確認にならない |
| 実機での情報表示 | 起動時間・転送量・`devicePixelRatio`・viewport・iframe 内かどうかを画面へ出す | 実機では console を見られないことがある。チェックリストの記入に必要な値を画面から読めるようにする |
| iframe の sandbox | `allow-scripts allow-same-origin` を明示 | 埋め込み側が制限をかけた状態を再現する。この 2 つは WASM の起動と同一オリジンの `main.wasm` 取得に必要な最小限 |
| 配信時の URL 表示 | ワイルドカードで待ち受けたら LAN の IP を並べる | `http://0.0.0.0:8080/` は実機から開けない。実機検証のたびに IP を調べ直さずに済む |
| `wasm_exec.js` | `make wasm` が GOROOT からコピーし、commit しない | Go の同梱物なのでツールチェーンとバージョンを一致させる。生成物を commit しない方針とも揃う |
| ローカル配信サーバ | 標準ライブラリだけの Go 実装（`cmd/serve`） | `.wasm` の Content-Type が `application/wasm` でないと `instantiateStreaming` が失敗する。Go の `mime` なら確実で、外部ツールへの依存も増えない |
| 配信時のキャッシュ | `Cache-Control: no-store` | 再ビルドした `.wasm` が古いキャッシュのまま検証される事故を防ぐ |

### 直接 dependency

- `github.com/hajimehoshi/ebiten/v2` — 2D engine

それ以外は Ebitengine の推移的依存のみ（`go.mod` を参照）。
`ebiten/v2/text/v2` を使い始めたことで `go-text/typesetting` などが indirect dependency として増えたが、
いずれも Ebitengine 側の依存であり、直接依存は増やしていない。

### 同梱アセット

- `internal/emoji/assets/TwemojiMozilla.ttf` — Twemoji Mozilla 0.7.0（カラー Emoji フォント）

出典・SHA-256・ライセンス全文は `internal/emoji/assets/` に置いている。

## テストについて

`go test ./...` はヘッドレス環境で実行される。Ebitengine は graphics context が無いと
pixel の読み出し（`Image.At`）ができないため、描画結果そのものは検証していない。
`Draw` については「ゲームループ外から呼んでも panic しない」ことのみを確認し、
実際の見た目は Desktop 起動とブラウザでの目視確認で担保する。

`cmd/serve` については、`.wasm` の Content-Type とキャッシュ無効化を `httptest` で検証している。

Emoji についても色そのものは検証できないため、必須 Emoji が同梱フォントの
単一カラーグリフへ解決されるところまでを自動テストで確認し、実際の色は目視確認で担保している。

Composite Sprite の合成計算（`sprite.Part.Place`）は Ebitengine に依存しない純粋関数なので、
描画コンテキストなしで検証している。全体の移動・拡大・回転で部品どうしの位置関係が保たれることは
ここでテストしており、目視確認に頼っていない。

`internal/anim` も同様に Ebitengine へ依存しないため、アニメーション後に base へ戻ること、
順番待ちが要求順に再生されること、particle が生成から消滅まで正しく扱われることを
すべてテストで確認している。毎 tick 呼ばれる `Particles.Update` が確保を行わないことも
`testing.AllocsPerRun` で検証している。

## クレジット

本リポジトリは次のアセットを同梱している。配布時もこの表示を成果物側に残すこと。

- Emoji のデザイン: [Twemoji](https://github.com/twitter/twemoji) © Twitter, Inc. and other contributors、
  [CC-BY 4.0](https://creativecommons.org/licenses/by/4.0/) で利用
- フォント化: [twemoji-colr](https://github.com/mozilla/twemoji-colr) © Mozilla Foundation、Apache License 2.0

## 未対応

Milestone「Ebitengine / WASM Emoji PoC」の実装は一通り揃っている。
残るのは実機 iPhone Safari での検証と、その結果にもとづく Go / No-Go 判断。
手順と記録先は [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md)。
