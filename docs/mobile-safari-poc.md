# iPhone Safari / iframe 検証記録（YTA-10）

Ebitengine + WebAssembly + Composite Emoji という構成を、実際の配布環境に近い
Mobile Safari + iframe で検証した結果を残す。この構成を本実装へ採用するかの
Go / No-Go 判断の根拠になる。

**この文書は記入途中である。** チェックリストの結果と Go / No-Go は実機で検証してから記入する。
自動テストで代替できる項目ではないため、空欄を埋めずに完了扱いにしない。

## 検証環境

| 項目 | 内容 |
| --- | --- |
| 端末 | （記入: 例 iPhone 15 / iOS 18.x） |
| ブラウザ | （記入: Safari のバージョン） |
| 接続方法 | （記入: 同一 LAN の開発マシンへ直接 / トンネル経由） |
| 検証日 | （記入） |
| コミット | （記入: 検証したときの main の SHA） |

## 検証手順

1. 開発マシンで LAN へ公開する。

   ```sh
   make serve SERVE_ADDR=0.0.0.0:8080
   ```

   起動時に表示される `http://<LAN IP>:8080/` を iPhone の Safari で開く。
   開発マシンと iPhone が同じ Wi-Fi にいる必要がある。

2. 単体ページ `http://<LAN IP>:8080/` と、iframe 埋め込みページ
   `http://<LAN IP>:8080/iframe.html` の両方を確認する。

3. 画面の見方

   - 左上: PoC 識別用テキスト、`fps` / `tps`、tick、タップ回数、最後のタップ座標
   - 右下: 起動にかかった時間、`main.wasm` の転送量、`devicePixelRatio`、viewport サイズ、iframe 内かどうか
   - 中央: ナッシー型（`🌴` + `🥺 😫 🤪`）。待機アニメーションで上下する
   - 中央上: 待機中は `💤` が自動で出る
   - 下部: `ATTACK` / `HIT` / `EMPHASIS` のタップ領域。それぞれ `⚡` / `💥` / `❄️` が出る
   - タップした位置: 黄色い十字の印。**指を置いた場所と印がずれていれば座標変換が合っていない**

## Verification checklist

記入方法: 各項目に `OK` / `NG` / `未確認` と、気付いたことを書く。

### Runtime

| 項目 | 結果 | メモ |
| --- | --- | --- |
| WASM が正常に load / init する | | |
| reload 後も安定して起動する | | |
| console に継続的な fatal error が出ない | | |

### Rendering

| 項目 | 結果 | メモ |
| --- | --- | --- |
| Emoji が欠損・monochrome 化しない（キャラクター 4 種 + particle 4 種） | | |
| Composite Sprite の相対レイアウトが Desktop と大きく乖離しない | | |
| Canvas が viewport から不自然にはみ出さない | | |
| devicePixelRatio による実用上問題となる blur が発生しない | | |

### Input

| 項目 | 結果 | メモ |
| --- | --- | --- |
| tap を確実に取得できる（タップ回数が増える） | | |
| page scroll / zoom と競合しない | | |
| iframe 内でも入力座標が論理座標へ正しく変換される（印が指の位置に出る） | | |

### Performance

| 項目 | 結果 | メモ |
| --- | --- | --- |
| Idle 状態で目視できる継続的な stutter がない | | |
| animation / particle 発火時にも操作不能となる frame drop がない | | |
| 初回 load 時間 | | 画面右下の `boot` を記入 |
| `main.wasm` の転送量 | | 画面右下を記入。ビルド時のサイズは下の「既知の数値」を参照 |

### iframe

| 項目 | 結果 | メモ |
| --- | --- | --- |
| iframe 内で起動できる | | |
| focus / input に致命的な問題がない | | |
| 必要だった security / sandbox 設定 | | `sandbox="allow-scripts allow-same-origin"` で検証している |

## 既知の数値（ビルド時点）

| 項目 | 値 |
| --- | --- |
| `main.wasm` | 22,436,383 bytes（gzip 約 5.3 MB） |
| 同梱フォント | 1,474,284 bytes（`main.wasm` に含まれる） |

`main.wasm` の増分の内訳と削減の選択肢は [emoji-rendering.md](emoji-rendering.md) を参照。
`text/v2` のコードがフォント本体より大きいため、フォントのサブセット化だけでは大きく減らない。

## Go / No-Go

**（実機検証後に記入する。）**

判断は以下のいずれかを根拠付きで明示する。

- **Go** — Mobile Safari の iframe で安定起動し、Emoji 表現が意図を維持し、touch input が実用可能で、
  Battle 画面程度の描画負荷に十分な performance がある
- **Conditional Go** — 条件付きで採用する。下の「問題の記録」に、必要な対応を後続 Issue として特定する
- **No-Go** — Ebitengine 継続 / Emoji 方式変更 / Web UI への切替のどれを推奨するかを記す

最終的な技術選定の変更はこの文書だけで決めない。設計判断として Linear へ記録してから反映する。

### 問題の記録

問題があった場合は、項目ごとに次を残す。

| 再現条件 | 原因または有力仮説 | 影響範囲 | workaround 候補 |
| --- | --- | --- | --- |
| | | | |
