# ぷゆもんコロシアム 155 Battle

初代155ルール準拠の3vs3 Emojiターン制対戦ゲーム。
Client は Go + [Ebitengine](https://ebitengine.org/) で実装し、最終的に WebAssembly / iframe での配信を想定する。

仕様・設計の Single Source of Truth は Linear の
[ぷゆもんコロシアム 155 Battle](https://linear.app/ytask/project/ぷゆもんコロシアム-155-battle-2372a3a4229f) プロジェクト。
開発の進め方は Project Document「AI Development Protocol」に従う。

現在のリポジトリ状態は Milestone「Ebitengine / WASM Emoji PoC」を一通り実装した、
Desktop と WebAssembly の両方で起動し、タップでアニメーションを発火できる
Composite Emoji のゲームクライアント。

実機 iPhone Safari / iframe での検証は完了しており、**この構成を本実装へ採用する（Go）**という結論。
検証結果と持ち越した課題は [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md) にある。

ここから Generation I 準拠の Battle Engine を `internal/battle` に実装していく。
現在あるのは domain model と seeded RNG までで、ダメージ計算などの mechanics はこれから追加する。
対戦仕様の正は Linear の Project Document「Battle Rules Specification」。
active / reserve や multi-turn state といった用語も、同 Document の Glossary の意味で使う。

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

ウィンドウを縦長にすると、ゲーム画面の代わりに横持ちを促す画面が出る。横に広げれば元に戻る。

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

`main.wasm` は約 21 MB ある。`make serve` は圧縮しないので初回ロードには時間がかかる。
本番配信では事前圧縮して配るため、実際の転送量は brotli で約 3.8 MB になる。
配信手順は [docs/wasm-delivery.md](docs/wasm-delivery.md)、サイズの内訳は
[docs/emoji-rendering.md](docs/emoji-rendering.md) を参照。

ロード中は受信量（`読み込み中... 2.4MB`）を出し、10 秒を超えたら注意書きを足す。
失敗したときは理由と再読み込みボタンを出す。回線を絞った実測値と判断の根拠は
[docs/loading-experience.md](docs/loading-experience.md) にある。

### 実機（iPhone Safari）

同じ Wi-Fi にいる iPhone から開くには、LAN へ公開して配信する。

```sh
make serve SERVE_ADDR=0.0.0.0:8080
```

起動時に表示される `http://<LAN IP>:8080/` を iPhone の Safari で開く。
iframe 埋め込みの確認は `http://<LAN IP>:8080/iframe.html`。
横長の枠と縦長の枠を並べてあり、**縦長の枠では横持ちを促す画面が出る**のが正しい挙動。

検証手順とチェックリストは [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md) にある。

ビルドだけ行う場合は次のとおり。

```sh
make wasm         # web/main.wasm と web/wasm_exec.js を生成する
make dist         # 本番配信用の成果物を dist/ へ生成する
make clean        # 生成物を削除する
```

`make dist` は非圧縮・gzip・brotli の `main.wasm` と bootstrap ページ一式を `dist/` へ出す。
brotli コマンドが必要（macOS は `brew install brotli`）。
配信側に必要なレスポンスヘッダと実測サイズは [docs/wasm-delivery.md](docs/wasm-delivery.md) にある。

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
internal/battle/            Battle Engine。domain model・タイプ相性・ダメージ・状態異常・turn resolver（UI 非依存）
internal/roster/            6 キャラクターと固定 move set のデータ（UI 非依存）
internal/simulation/        Battle Engine を UI なしで回す simulation harness（UI 非依存）
docs/mobile-safari-poc.md   iPhone Safari / iframe 検証の手順と記録（YTA-10）
docs/wasm-delivery.md       本番配信時の圧縮手順と実測サイズ（YTA-12）
docs/loading-experience.md  初回ロード中の表示と、回線別のロード時間（YTA-13）
internal/emoji/             Emoji 素材のフォント読み込みとセル画像のキャッシュ
internal/emoji/assets/      同梱フォントと、その出典・ライセンス
docs/emoji-rendering.md     カラー Emoji 描画の検証記録（YTA-7）
web/index.html              Go WASM runtime と main.wasm をロードする bootstrap
web/iframe.html             iframe 埋め込み確認用ページ
```

`web/main.wasm` と `web/wasm_exec.js` は `make wasm` が、`dist/` は `make dist` が生成するため
commit していない。

ゲームロジックと描画を `main` へ集中させず `internal/game` に閉じている。
platform 固有の処理（ウィンドウ設定・HTTP 配信・ブラウザ bootstrap）は `cmd` と `web` に置き、
`internal/game` へ持ち込まない。
後続の Battle Engine は UI 非依存の別 package として追加し、`internal/game` から状態として参照する。

`internal/emoji` は Emoji を「UI テキスト」ではなく「スプライト素材」として供給する層で、
画面構成（どこに何を並べるか）は持たない。画面構成は `internal/game` 側に置く。

`internal/sprite` はキャラクターの**定義**（どの Emoji をどこに置くか）と**描画**を分けている。
定義と座標計算は Ebitengine に依存せず、描画だけが `Renderer` に閉じている。
キャラクターごとの分岐は描画側に持たせないため、新しいキャラクターの追加は定義を増やすだけで済む。

`internal/battle` は対戦そのものを表す層で、UI からも Ebitengine からも独立している。
状態（`BattleState`）・行動（`Action`）・結果（`Event`）だけを持ち、描画や入力を知らない。
乱数は `RNG` として外から渡すため、同じ初期状態・同じ行動列・同じ seed からいつでも同じ結果を再現できる。
キャラクターや技は識別子（`SpeciesID` / `MoveID`）で参照するだけで、domain 側にキャラクター固有の分岐を持たせない。
タイプ相性のようなゲーム定義はデータとして持ち、ロジックはそれを解釈するだけにしている。
1 turn の解決は `Resolver` が担い、状態・行動・乱数から次の状態と Event 列を返す。UI は呼ばない。

`internal/roster` は「誰がどんな技を持つか」というゲーム定義だけを持ち、ルールは持たない。
`battle.Data` へ変換して engine へ渡すので、キャラクターや技が増えても engine のコードは変わらない。

`internal/simulation` は Battle Engine を UI なしで回すための入口で、対戦のルールを一切持たない。
行動順もダメージも状態異常も `internal/battle` の `Resolver` が決め、キャラクターと技は `internal/roster` から取る。
この層の責務は「scenario を進める」「結果をまとめる」「上限で止める」の 3 つだけで、
engine のロジックを test 側へ写し取らないための境界でもある。
後続のバランス検証がそのまま呼べるように、test helper ではなく production のコードとして置いている。

`internal/anim` はアニメーションの**状態**だけを持ち、キャラクター定義も base transform も持たない。
描画に使う transform は毎回 base から計算し直すため、再生を繰り返してもずれが蓄積しない。
Ebitengine に依存しないので、動きの検証は描画コンテキストなしで行える。

## 技術的な判断

Linear に明示されていない箇所について、以下を採用した。変更が必要になった場合は Linear 側の仕様を先に更新する。

| 項目 | 採用した内容 | 理由 |
| --- | --- | --- |
| Go module path | `github.com/ytasak/puyumon-coliseum` | GitHub へ push する前提。Product Owner 確認済み |
| Ebitengine | v2.10.2 | 実行時点の最新安定版 |
| 論理解像度 | 640x360 (16:9) 固定 | **横持ち前提の確定値**（YTA-11）。初代準拠の対面レイアウト（両者のアクティブ・控え・HP・技 4 つ・メッセージ）を 1 画面へ収めるため横長を採る。`internal/game/layout.go` の定数一箇所で管理し、実ウィンドウサイズとは分離する |
| 縦持ちで開かれたとき | ゲーム画面の代わりに横持ちを促す画面を出す | iOS Safari はページから画面の向きをロックできず、促すことしかできない。縦長のまま出すと 16:9 が高さの 3 分の 1 ほどしか使えない。`Update` は止めないので、横にすればそのまま続きが見える |
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
| 本番配信の圧縮 | `make dist` で事前圧縮した `.br` / `.gz` を生成し、配信側は `Content-Encoding` を付けて返す | brotli -q11 で 21.4 MB → 3.8 MB。21 MB の `.wasm` は CDN の自動圧縮のサイズ上限を超えやすく、事前圧縮のほうが確実。圧縮は配信の設定なのでゲーム側のコードは変えない |
| `-ldflags="-s -w"` | 使わない | 圧縮後で 0.06 MB しか減らない一方、panic 時のシンボルを失う |
| Battle Engine の位置 | `internal/battle`。Ebitengine を import しない | 仕様書が求める UI 非依存の pure domain。import していないことを test で検査しており、うっかり依存が入れば落ちる |
| チームの表現 | `[3]Pokemon` の固定長配列 + 場に出ている index | 本作は 3 体固定なので型で表せる。交代しても index が変わらないため、Event から常に同じ index で指せる |
| `Action` / `Event` | 非公開メソッドを持つ interface で閉じる | package の外から別の行動や Event を足せない。resolver は type switch で漏れなく扱える。Gen I の技データを `Move` と名付ける余地を残すため、行動側は `MoveAction` / `SwitchAction` とした |
| 乱数 | splitmix64 を自前実装し、`RNG` interface で注入する | 生成列をこのリポジトリのコードだけで固定する。標準ライブラリの版差で golden test が壊れるのを避ける。global state に依存しないので、同じ seed から必ず同じ対戦を再現できる |
| 乱数を状態に含めるか | 含めない。`BattleState` は値として複製できる pure な状態のままにする | 状態を複製しても乱数列が分岐しない。seed の管理は resolver 側の責務になる |
| キャラクター・技の参照 | `SpeciesID` / `MoveID` という識別子だけを持つ | 実データの定義は後続 Issue の範囲。domain 側がキャラクター名で分岐しない構造を最初から守る |
| キャラクター・技のデータ | `internal/roster` に置き、`battle.Data` へ変換して渡す | engine はデータを解釈するだけにする。実データを engine 側へ置かない |
| 能力値の求め方 | Gen I の式。**DV 15・stat exp 最大で固定** | 全キャラがテンプレートなので個体差を持たない。同じ種族値と Level から必ず同じ実数値になる。個体差を入れるなら引数を足す |
| 155 のレベル配分 | 配布された 3 体へ、強い順に **50 / 50 / 55** | どの組み合わせでも合計 155 に収まる。Product Owner 判断（Linear の YTA-20 が正） |
| 公開表示名 | 内部 ID だけ確定し、`DisplayName` は空のまま | IP 方針。データ構造で分離してあるので、あとから表示名を入れられる |
| 技の効果 | `MoveEffect` の列挙と発生率をデータに持ち、resolver が列挙で分岐する | キャラクター名で分岐しない。効果が増えても列挙とデータの追加で済む |
| 追加効果が起きない条件 | 相手が既に状態異常、または**技のタイプと相手のタイプが一致**していたら起きない | Gen I の仕様。ノーマル技はノーマルを麻痺させず、こおり技はこおりを凍らせない |
| 相手を倒した turn | 追加効果も反動も起きない。ただし**吸収・自爆・多段は起きる** | ROM は相手が倒れた時点で処理を終えるが、この 3 つだけは「倒しても最後まで処理する」側に入っている |
| 自爆 | 命中判定より前に使用者を戦闘不能にし、相手の防御を半分にして計算する | **外れても倒れる**のが Gen I |
| Rest | 状態異常を消して全回復し、sleep counter を 2 にする | Gen I の固定値。HP 満タンなら失敗する |
| Recover / Rest のバグ | 再現しない | 「HP 差の下位バイトが 0 なら失敗する」は obscure な挙動なので、Battle Rules に判断が入るまで入れない |
| turn の解決 | `ResolveTurn` が state / Action / RNG から次の状態と Event 列を返す。UI を呼ばない | 仕様書の方向性どおり。進行順は Linear の YTA-18 に書いた turn pipeline が正 |
| 技・キャラクターのデータ | `Data` struct を外から渡し、engine は定義の形だけを持つ | 実データは別 Issue の範囲。実装が 1 つしかない段階で interface を切らない方針に従った |
| replacement の表現 | 状態に専用フラグを置かず、`NeedsReplacement` で導出する。解決は `ResolveReplacement` で、turn は進めない | 「active が戦闘不能かつ控えが残っている」ことから決まるので、状態を二重に持たない |
| 乱数の消費順 | 命中 → 急所 → ダメージ（pipeline と同じ順） | 実機はダメージ計算のあとに命中判定を行うが、ROM との bit 互換は非目標。順序を 1 つに統一して追いやすくする |
| 交代時の reset | stat stages・反動・継続中の技を、退く側と出る側の両方で初期化する | Gen I ではこれらが場の側に紐づくため。major status と残り PP は引き継ぐ |
| Event の追加 | `MoveMissed` / `Unaffected` / `ActionBlocked` を足した | 仕様書の Event 一覧は例示。外れた・相性で通らない・状態異常で動けないを区別できないと、UI が「何が起きたか」を復元できない |
| ねむりの起床ターン | 目を覚ました turn は行動できない | Gen I の挙動。現代世代と違う点で、誤解されやすいので test でも固定している |
| こおり | 自然解凍しない。解除は技の側から状態を消して行う | Gen I には自然解凍が無い。現代世代の「毎ターン 20% で溶ける」を持ち込まない |
| 行動を妨げた理由の返し方 | 真偽値ではなく `StatusBlock`（ねむり / 起床 / こおり / まひ）で返す | 理由ごとに見せ方が変わる。Event の組み立ては turn resolver に任せ、この層は mechanics に絞る |
| 継続ダメージの順序 | 「先攻が動く → 先攻の継続ダメージ → 後攻が動く → 後攻の継続ダメージ」。**先攻が継続ダメージで倒れたら後攻はその turn 行動しない** | Gen I は turn の終わりへ一括しない。実機は手番のあとに戦闘不能を見つけるとその場で faint の処理へ移るため、後攻の技実行まで進まない |
| Gen I のダメージ計算 | 出荷 ROM と同じ順序で整数演算する。相性は合成値を一度に掛けず、防御側のタイプごとに順に適用する | 掛ける順序と切り捨ての位置が変わると結果が変わる。期待値は pokered の `engine/battle/core.asm` から起こした別実装（Python）で独立に求め、golden test として固定した |
| 急所 | base Speed 依存。通常技は `floor(base Speed / 2)`、高急所技はその 8 倍で 255 頭打ち。急所時は能力変化を無視し、Level を 2 倍にして計算する | 現代世代の固定確率ではない。base Speed は species データ側から渡す |
| 能力値が 255 を超えたとき | 攻守とも 1/4 にしてから計算する | Gen I が 1 byte へ収めるための処理。能力を上げたときの結果に影響するので落とせない |
| 防御側の能力値が 0 のとき | 1 として扱う | 実機は 0 除算で停止する。フリーズは再現できないため最小値で代替する |
| 1/256 miss | 再現しない。命中率が最大（255）なら必中 | Issue の指示。再現する場合は Battle Rules Specification を先に更新する |
| 乱数の消費数 | 命中判定もダメージ乱数も、結果によらず常に 1 つ消費する | ROM と bit 互換ではないので、消費数を一定にして追いやすさを優先した |
| Gen I のタイプ相性 | 出荷 ROM と同じ 82 エントリをデータとして持ち、表に無い組み合わせは等倍 | 現代世代の知識で「直して」しまう事故を防ぐ。データは [pokered](https://github.com/pret/pokered) の `data/types/type_matchups.asm` と突き合わせた。後の世代で変わった相性は個別の test でも固定している |
| simulation harness の位置 | `internal/simulation` に production package として置く | 後続の Balance milestone からそのまま呼べるようにする。test helper にすると test からしか使えない。Product Owner 判断（YTA-21） |
| simulation の責務 | scenario を進める・結果をまとめる・上限で止める、の 3 つだけ | 対戦のルールを二重に持たない。ここに判定を書くと engine と食い違っても気づけない |
| turn 上限 | 対戦ルールには入れず simulation 側だけが持つ。到達したら `TurnLimitReached` を立て、`BattleState.Status` は `Ongoing` のままにする | 終わらない simulation を止めるための安全装置であって、ゲームのルールではない。勝敗を捏造しない |
| 行動の選び方 | `FirstUsable`（使える先頭の技）と `Script`（決めた順に返す）の 2 つだけ | 賢い Bot は Out of scope。大量 simulation と golden scenario に必要な最小限にとどめる |
| `Config` が持つもの | Chooser の instance ではなく作り方（`ChooserFactory`）。`Run` ごとに新しい Chooser を作る | `Script` は「どこまで使ったか」を持つ。instance を持たせると同じ `Config` の 2 回目が途中から始まり、「同じ初期状態・同じ行動列・同じ seed」でなくなる。並列に回しても Chooser を共有しない |
| 統合 golden の固定範囲 | 代表 scenario 1 本だけ Event 全文と final state を固定する。20 通りの総当たりでは固定しない | 目的は「mechanics を通した結果が意図せず変わったこと」の検出。個々の値の正しさは YTA-16〜YTA-19 の独立 golden が正 |
| 代表 scenario の作り方 | 両者の行動を script で固定し、急所・ねむり・こおりが 1 本へ収まる seed を選んだ | 交代・状態異常・急所・戦闘不能・反動（YTA-19）を 11 turn で通せる。含めるべき挙動が抜けた scenario へ差し替わらないよう、内容の検査も test に置いた |
| 再現性の確かめ方 | golden 値ではなく、同じ Config を 2 回実行して state と Event 列が完全に一致することで見る | golden は「変わったこと」を、この test は「毎回同じであること」を担保する。役割が違う |
| headless の担保 | 依存を辿って Ebitengine が混ざっていないことを test で検査する | package 自身の import を見るだけでは、`internal/battle` や `internal/roster` 経由の混入を防げない |
| ゴースト技 → エスパー | 0×（効かない）。出荷されたとおり | 本来は効果ばつぐんの意図だったとされる実装ミスだが、初代の対戦を決定づけた挙動のため維持する。Product Owner 判断 |
| 相性倍率の持ち方 | 100 を等倍とする整数 | 0.25 / 0.5 / 2 / 4 を誤差なく扱える。ただし Gen I はダメージへ防御側のタイプごとに掛けて都度切り捨てるため、ダメージ計算では合成値ではなく `Against` をタイプごとに使う |
| stage 倍率の持ち方 | 分子・分母のまま持つ（`25/100` 〜 `4/1`） | Gen I は整数演算で掛けるので、小数へ直すと端数の出る値で結果がずれる。値は pokered の `data/battle/stat_modifiers.asm` と一致 |
| 命中・回避の stage | 能力値と同じ倍率表を使う | Gen I では同一。別の表になるのは Gen II 以降 |
| stage 適用時の頭打ち | `ApplyStage` では 1〜999 の頭打ちをしない | どこで頭打ちにするかは能力値・ダメージ計算側で決める。倍率の適用だけを純粋に行う |
| ロード中の表示 | テキストのみ。受信量と、10 秒を超えたときの注意書き | ローディング画面のアートワークは Out of scope。HTML 側で Emoji を出すと OS のフォントで描かれ、起動後の Twemoji と絵柄が変わる。揃えるには 1.4 MB のフォントを別に読ませることになり、待ち時間を減らす目的と逆を向く |
| 受信量の取り方 | `fetch` した body を自前で数え、同じ内容を `instantiateStreaming` へ渡す | `instantiateStreaming` は受信量を教えてくれない。streaming のまま渡すので起動は遅くならない。圧縮配信では解凍後のバイト数になる（転送量は起動後に Resource Timing から出す） |
| ロード失敗時 | 理由と再読み込みボタンを出す。自動リトライもタイムアウトによる中断もしない | 実機では console を見られないことがある。iframe 内ではブラウザの再読み込みが埋め込みページ全体に及ぶため、枠の中だけやり直せるようにする。遅いだけの回線を打ち切ると、あと少しで終わる読み込みを捨てることになる |
| フォントのサブセット化 | 現時点では行わない | 使う Emoji が未確定で、`fonttools` をビルド依存に追加することになる。再検討の条件は [docs/wasm-delivery.md](docs/wasm-delivery.md) |

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

ロード中の表示と失敗時の表示はゲーム本体が動く前の HTML / JS 側にあるため `go test` の対象外。
回線を絞ったブラウザでの手動確認で担保しており、確認した内容は
[docs/loading-experience.md](docs/loading-experience.md) に残している。

Emoji についても色そのものは検証できないため、必須 Emoji が同梱フォントの
単一カラーグリフへ解決されるところまでを自動テストで確認し、実際の色は目視確認で担保している。

`internal/battle` は Ebitengine に依存しないため、描画 context なしで全部テストできる。
状態の検証（取り得ない値を弾くか）、seeded RNG の再現性、`Action` / `Event` モデルの妥当性に加えて、
**この package が Ebitengine を import していないこと自体**も test で確かめている。

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

## 持ち越した課題

PoC の結論は Go だが、本実装までに扱う必要がある点が残っている。
詳細は [docs/mobile-safari-poc.md](docs/mobile-safari-poc.md) の「持ち越す課題」を参照。

- **配信時の圧縮** — 手順は [docs/wasm-delivery.md](docs/wasm-delivery.md) で確定済み（brotli で 3.8 MB）。
  残るのは配信基盤の選定と、実配信での転送量の確認
- **モバイル回線での初回ロード** — ロード中の表示は実装済み（[docs/loading-experience.md](docs/loading-experience.md)）。
  Chrome の throttling では Fast 4G 4.1 秒 / Slow 4G 22.8 秒。**実機のモバイル回線では未計測**で、
  公開 HTTPS のエンドポイントが必要
- ~~**縦持ちでの画面の使い方**~~ — YTA-11 で決着。横持ち前提を維持し、縦長で開かれたときは横持ちを促す
