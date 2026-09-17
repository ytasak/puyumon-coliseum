# iPhone Safari / iframe 検証記録（YTA-10）

Ebitengine + WebAssembly + Composite Emoji という構成を、実際の配布環境に近い
Mobile Safari + iframe で検証した結果。この構成を本実装へ採用するかの根拠になる。

## 結論: Go

Go の条件をすべて満たした。Ebitengine + WASM + Composite Emoji を本実装へ採用してよい。

| Go の条件 | 結果 |
| --- | --- |
| Mobile Safari iframe で安定起動 | 満たす |
| Emoji 表現が意図を維持 | 満たす |
| touch input が実用可能 | 満たす |
| Battle 画面程度の描画負荷に十分な performance | 満たす |

**ただし配布時の転送量には対応が要る。** 詳細は「持ち越す課題」を参照。
これは採用可否を左右するものではなく、配信方法の設定で対処できる範囲と判断した。

## 検証環境

| 項目 | 内容 |
| --- | --- |
| 端末 | iPhone 13 mini |
| OS | iOS 26 系 |
| ブラウザ | Safari（検証時点の最新） |
| 接続方法 | 同一 LAN の開発マシンへ直接（`make serve SERVE_ADDR=0.0.0.0:8081`） |
| 検証日 | 2026-09-17 |
| コミット | `34f5fd3` |
| 検証者 | Product Owner |

実機が報告した値:

```
boot 1.19s / 21.4MB over the wire
dpr 3 / 375x610
```

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

Product Owner が実機で一通り操作し、単体ページと iframe 埋め込みの両方で問題なしと報告した。
個別に異常が報告された項目は無い。

### Runtime

| 項目 | 結果 | メモ |
| --- | --- | --- |
| WASM が正常に load / init する | OK | 起動まで 1.19 秒（LAN・非圧縮） |
| reload 後も安定して起動する | OK | |
| console に継続的な fatal error が出ない | OK | 起動失敗時は画面へ出す作りだが、失敗の表示は出ていない |

### Rendering

| 項目 | 結果 | メモ |
| --- | --- | --- |
| Emoji が欠損・monochrome 化しない | OK | キャラクター 4 種（`🌴 🥺 😫 🤪`）と particle 4 種（`⚡ 💥 ❄️ 💤`） |
| Composite Sprite の相対レイアウトが Desktop と大きく乖離しない | OK | |
| Canvas が viewport から不自然にはみ出さない | OK | viewport 375x610（縦持ち）。論理解像度 640x360 は横長なので上下に余白が出る |
| devicePixelRatio による実用上問題となる blur が発生しない | OK | **dpr 3 で blur なし。** Emoji セルを 128px でラスタライズする方式が高精細な端末でも成立することの確認になる |

### Input

| 項目 | 結果 | メモ |
| --- | --- | --- |
| tap を確実に取得できる | OK | |
| page scroll / zoom と競合しない | OK | `touch-action: none` と `user-scalable=no` が効いている |
| iframe 内でも入力座標が論理座標へ正しく変換される | OK | タップ位置のマーカーで確認。**Ebitengine 側の座標変換に手を入れる必要は無い** |

### Performance

| 項目 | 結果 | メモ |
| --- | --- | --- |
| Idle 状態で目視できる継続的な stutter がない | OK | |
| animation / particle 発火時にも操作不能となる frame drop がない | OK | |
| 初回 load 時間 | 1.19 秒 | **LAN かつ非圧縮での値。モバイル回線では未検証** |
| `main.wasm` の転送量 | 21.4 MB | ビルドサイズとほぼ一致しており、開発サーバが圧縮していないことを示す |

### iframe

| 項目 | 結果 | メモ |
| --- | --- | --- |
| iframe 内で起動できる | OK | |
| focus / input に致命的な問題がない | OK | |
| 必要だった security / sandbox 設定 | `allow-scripts allow-same-origin` | この 2 つで動作した。追加の緩和は不要だった |

## 既知の数値（検証したコミット時点）

| 項目 | 値 |
| --- | --- |
| `main.wasm` | 22,436,447 bytes（21.4 MiB） |
| `main.wasm`（gzip） | 5,616,946 bytes（5.4 MiB） |
| 同梱フォント | 1,474,284 bytes（`main.wasm` に含まれる） |

`main.wasm` の増分の内訳と削減の選択肢は [emoji-rendering.md](emoji-rendering.md) を参照。
`text/v2` のコードがフォント本体より大きいため、フォントのサブセット化だけでは大きく減らない。

## 持ち越す課題

採用可否には影響しないが、本実装までに対応が要る点。

### 1. 配信時の圧縮

実機が受け取った 21.4 MB はビルドサイズとほぼ同じで、`cmd/serve` が圧縮していないことを示す。
`cmd/serve` は動作確認用なのでこのままでよいが、**本番の配信では gzip / brotli を有効にする**。
gzip で約 5.4 MB、brotli ならさらに小さくなる。

### 2. モバイル回線での初回ロード

1.19 秒という値は LAN で非圧縮という好条件のもの。**モバイル回線での実測はしていない。**
圧縮後 5.4 MB でも回線によっては体感できる待ち時間になる。
初回ロード中の表示（現状は `loading WebAssembly...` のみ）を含めて、配信方法を決める段階で扱う。

### 3. 縦持ちでの画面の使い方

viewport 375x610 の縦持ちに対し、論理解像度 640x360 は横長なので上下に余白が出る。
PoC では問題にならなかったが、Battle 画面のレイアウトを決めるときに
縦持ちを前提にするのか、横持ちを促すのかを決める必要がある。

これらは後続 Issue として Linear へ起票する対象。
