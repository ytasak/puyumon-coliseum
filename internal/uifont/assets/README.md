# 同梱フォント

## DotGothic16-Regular.ttf

| 項目 | 内容 |
| --- | --- |
| 配布元 | <https://github.com/google/fonts/tree/main/ofl/dotgothic16> |
| 元ファイル名 | `DotGothic16-Regular.ttf` |
| 上流 | [fontworks-fonts/DotGothic16](https://github.com/fontworks-fonts/DotGothic16) |
| SHA-256 | `3ad9af88726d42b40f7f365f0dcac785af73cf20ea6f1d5b44e57cc21150b8f1` |
| サイズ | 2,069,236 bytes |
| 字形 | 16px のドット字形を想定した日本語ゴシック。全角 16px / 半角 8px |
| 変更点 | **無し。** Google Fonts の配布物をそのまま同梱している |

ライセンス全文は同じディレクトリの `OFL.txt`（配布元のものをそのまま置いた）を参照。

## 採用理由

- **論理解像度 640x360 に合う。** 16px 設計なので 1 行に全角 40 文字入り、対戦の
  1 行メッセージが収まる。既存のレイアウトが前提にしていた 16px の行高とも一致する
- **出典とライセンスが追える。** Google Fonts に `OFL.txt` ごとホストされており、
  上流リポジトリも公開されている
- **ドット字形が画面に合う。** Emoji とドット絵的な盤面に対して浮かない

候補として PixelMplus12（1,274,396 bytes / M+ BITMAP FONTS License）も代表 266 字の
収録を確認したが、2013 年から更新が無く配布が個人リリースの zip であるため採らなかった。
比較の経緯は Linear の YTA-35 を参照。

## ライセンスと表示義務

SIL Open Font License 1.1（Copyright 2020 The DotGothic16 Project Authors）。

OFL は次を求める。

- **ライセンス全文を同梱すること** — `OFL.txt` を同じディレクトリに置いている
- **著作権表示を保持すること** — `OFL.txt` の先頭に含まれる
- **フォント単体を有償で売らないこと** — ゲームへの同梱は該当しない
- **Reserved Font Name を改変版へ使わないこと** — 本リポジトリはフォントを改変していない

ゲームを配布する際も、`OFL.txt` をビルド成果物側に残すこと。

## WASM サイズへの影響（実測）

`GOOS=js GOARCH=wasm go build ./cmd/game` の比較。

| 対象 | サイズ |
| --- | --- |
| 日本語化前（`bf9a341`） | 22,879,751 bytes（gzip 5,642,282） |
| 日本語化後 | 23,040,144 bytes（gzip 5,975,904） |
| 差 | **+160,393 bytes（+0.7%）**（gzip +333,622 bytes / +5.9%） |

フォント自体は 2,069,236 bytes あるが、**同じ変更で `ebitenutil` を使わなくなった**ため
純増はこれだけに収まっている。`ebitenutil` は `image/png` を引き込み、最小構成の WASM で
実測 2,206,584 bytes を占めていた。

サブセット化は行っていない。ビルド依存（`fonttools`）を増やさない判断で、
必要になった場合の検討条件は `docs/wasm-delivery.md` を参照。
