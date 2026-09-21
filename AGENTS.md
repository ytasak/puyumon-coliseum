# AGENTS.md

このファイルは、**ぷゆもんコロシアム 155 Battle** リポジトリでコーディングエージェントが作業するときに
毎回守る恒久的な開発規約を定義する。Codex と Claude Code のどちらからも読まれる共通の規約である。

ゲーム仕様・機能要件・バランス値・キャラクター定義はここに書かない。それらは Linear にある。
本ファイルは「何を作るか」ではなく「どう開発するか」だけを扱う。

## 0. このリポジトリ

- Go + [Ebitengine](https://ebitengine.org/) の 2D ターン制対戦ゲームクライアント。最終的に WebAssembly / iframe で配信する
- Go module: `github.com/ytasak/puyumon-coliseum`
- 役割分担
  - Product Owner: ユーザー。面白さ・操作感・採否を決定する
  - Design / Architecture / PM: ChatGPT。仕様・設計・Issue 分解・Acceptance Criteria を Linear へ残す
  - Implementation Agent: コーディングエージェント（Codex / Claude Code）。Linear Issue とリポジトリを読み、実装・test・build・必要な documentation 更新を行う

参照先:

- Linear Project: <https://linear.app/ytask/project/ぷゆもんコロシアム-155-battle-2372a3a4229f>
- Linear Document「AI Development Protocol」: <https://linear.app/ytask/document/ai-development-protocol-e0d820551903>

本ファイルは AI Development Protocol をリポジトリ側から補完するものであり、置き換えるものではない。

### このファイルの置き場所について

- 本リポジトリの開発規約は **`AGENTS.md` 1 枚**に集約する。Codex はこれを読み、Claude Code も
  `CLAUDE.md` が無い場合は `AGENTS.md` を読む
- **`CLAUDE.md` / `CLAUDE.local.md` を追加しない。** 追加すると Claude Code が `AGENTS.md` を
  読まなくなり、規約が片方のエージェントにしか効かない状態になる。
  どうしても個人用の追記が必要な場合は `AGENTS.md` を直接更新するか、
  Claude Code の **Project instructions** 設定を `claude-md-and-agents-md` にしてから追加する

## 1. Source of Truth

**Linear が Single Source of Truth。**

| 対象 | 正とするもの |
| --- | --- |
| gameplay / product / feature 仕様、Acceptance Criteria | Linear |
| リポジトリでの作業方法、恒久的 engineering convention | AGENTS.md |

- 会話の中だけで決まった実装仕様は未確定として扱う。Linear へ反映されるまで実装しない
- コードを先に正として Linear を後追いさせない。仕様変更が必要なら Linear を先に更新する
- Linear とリポジトリ、または Linear と本ファイルが矛盾し、上の表では判断できない場合は**作業を停止して報告する**

## 2. 作業開始手順

Issue を渡されたら、コードを変更する前に必ずこの順序で読む。

1. 指定された Linear Issue を**全文**読む
2. blocking / related Issue を確認する
3. 関連する Linear Document を読む
4. リポジトリの既存実装・README・tests・dependency を確認する
5. Issue の要求と既存実装の差分を理解する

**Issue title だけを見て実装を開始しない。**

読み終えたら、コード変更前に短い実装計画（変更する package、追加する主要型/API、既存コードへの影響、test strategy）を提示する。
Issue に書かれていない大きな architecture 変更が必要だと判明した時点で実装を止め、設計判断を求める。

## 3. Scope discipline

- **1 Issue 単位で作業する。** 後続 Issue を先回りして実装しない
- Issue に明示された **Out of scope を実装しない**。「ついで」に直さない
- 「将来必要になりそう」だけを理由にした先行実装・過剰な abstraction・大規模 refactor を行わない
- Issue scope 外の変更が不可避になった場合は、勝手に広げず停止して報告する
- 既知の問題を見つけても、scope 外なら実装せず報告に残す

## 4. Architecture 原則

恒久的に守る原則。具体的な package 構造は Issue と Linear の設計で決める。

- **domain logic を UI / framework から分離する。** battle などの game domain logic は、描画を持たない package として実装する
- **Ebitengine 固有コードを battle / domain logic へ侵入させない。** domain 側が `github.com/hajimehoshi/ebiten/v2` を import しない状態を保つ
- **platform 固有処理（Desktop / WASM / browser bootstrap）を core logic から分離する。** Desktop 版と WASM 版でゲームコードを fork しない
- **data-driven にできるゲーム定義を、キャラクター固有の条件分岐として実装しない。** キャラクター・技・相性などはデータとして定義し、ロジックはデータを解釈する
- **deterministic test が可能な設計を優先する。** 乱数・時刻・入力は外から与えられる形にし、隠れたグローバル状態に依存しない
- **画面座標系の定数は一箇所で管理し、実ウィンドウ / canvas サイズと分離する**

### 現在の構造（固定ではない）

```text
cmd/<name>/     エントリポイント。ウィンドウ / プラットフォーム設定と起動のみ。ロジックを置かない
internal/game/  Ebitengine のゲームループ（Update / Draw / Layout）と描画
```

新しい package 境界は、必要になった Issue の中で導入する。本ファイルで先に確定させない。

## 5. Go conventions

基本 verification は次の 3 つ。

```sh
gofmt -l .      # 出力が空であること
go vet ./...
go test ./...
```

- 既存コードの Go convention・命名・コメントの粒度を尊重する。周囲のコードと同じ書き方をする
- 不要な interface / factory / wrapper / dependency を追加しない
- **interface は実際に抽象境界が必要になった時点で導入する。** 実装が 1 つしかない段階で interface を切らない
- error は握りつぶさず、呼び出し元が判断できる形で返す。`main` 以外で `log.Fatal` しない
- exported な識別子には doc comment を書く
- `go mod tidy` を実行しても `go.mod` / `go.sum` に差分が出ない状態を保つ

### テストについて

- test は実装詳細ではなく Issue の要求を検証する
- Ebitengine は graphics context の無いヘッドレス環境で pixel の読み出し（`Image.At` / `ReadPixels`）ができず panic する。
  **描画結果そのものは `go test` で検証できない**。見た目は目視または実起動による確認で担保し、その旨を報告に明記する
- domain logic を Ebitengine 非依存に保つことは、テスト可能性を維持するための実利でもある

## 6. Dependencies

- 新しい dependency を安易に追加しない。標準ライブラリと既存 dependency で足りるなら追加しない
- 追加する場合は必要性を説明できることを条件とする
- 特に architecture へ影響する以下の種類の dependency は、**Issue に明示されていなければ実装前に確認する**
  - framework
  - rendering library
  - networking library
  - persistence library
  - serialization 方式
- dependency を追加・更新したら、選定理由を README に残す
- generated / build artifact を commit しない

## 7. 停止条件と自律判断の境界

### 推測で進めず、作業を停止して報告する

- gameplay 仕様が複数解釈でき、結果がゲーム性に影響する
- Acceptance Criteria 同士が矛盾している
- architecture の大幅な変更が必要
- public API / persistent data format / network protocol の変更が必要
- 主要 dependency / framework の追加が必要
- security / license / 配布条件に重大な不確実性がある
- Linear の仕様とリポジトリの実装が根本的に矛盾している
- Issue scope 外の変更が不可避

### 自律的に判断してよい

- private function / private type の命名
- private type の局所的な構造
- file 分割
- idiomatic Go 上の軽微な判断
- Linear に明示されていない軽微な実装詳細（既存設計と Go の慣習に従う）

自律判断で技術的な選択をした場合は、理由を README またはコードコメントに残し、報告の Deviations にも書く。

## 8. Verification

実装完了を報告する前に、この順序で検証する。

1. **format** — `gofmt -l .`
2. **static verification** — `go vet ./...`
3. **tests** — `go test ./...`
4. **Issue 固有の build / check** — WASM build、実起動、ブラウザ確認など Issue が要求するもの
5. **Acceptance Criteria との照合** — 1 項目ずつ確認する

守ること:

- **実行していない検証を成功したと報告しない**
- 検証不能な Acceptance Criterion を「完了」と推測しない。何が検証できていないかを明示する
- manual verification が必要な場合は、**誰が何をどう確認すべきか**を手順として書く
- 検証用の一時コードはリポジトリに残さない

## 9. 完了報告フォーマット

Issue 実装後は次の形式で報告する。

```text
Summary             何を実装したか
Changed             主な変更ファイル / package
Verification        実行した検証コマンドと結果（実測値）
Acceptance Criteria 各項目の達成状況（達成 / 未達 / 検証不能を明示）
Deviations          Issue から変更した点。無ければ None
Remaining risks     手動確認が必要な点、未解決リスク、既知の制約
```

Issue を閉じること自体を目的にしない。目的は Issue の意図と Acceptance Criteria を、
リポジトリの整合性を保ったまま実装することである。
Issue のステータス変更は Product Owner の確認を経る。ただし **PR がマージされた時点で Done への変更は
許可済み**とし、その都度確認を取らない。マージ前に Done にはしない。

## 10. Git discipline

- **原則 1 Issue = 1 branch。**
- **branch name は英語で付ける。** Linear の Issue title が日本語でも、適切な英語へ翻訳して slug 化する。
  形式は `<handle>/<issue-id>-<english-slug>`（例: `vividnasubi/yta-5-set-up-ebitengine-project`）。
  Linear が生成する branch name は日本語を含むためそのまま使わない。
  branch name に Issue ID を含めておけば Linear が PR を自動リンクする。
  Issue に紐づかない作業は `chore/<english-slug>` のように英語で付ける
- **branch を切るときは `git worktree` を使う。** 作業中のブランチを切り替えず、未コミットの変更を
  他の作業へ持ち込まないため。worktree はリポジトリ外の sibling directory に作る（リポジトリ内に置く場合は
  `.gitignore` へ追加する）。作業が終わったら `git worktree remove` で片付ける
- commit を Issue scope に限定する。unrelated な refactor を混在させない
- 1 タスク完了ごとに、後からリバートしやすい単位で commit する
- ビルドエラー・テスト失敗・中途半端な実装の状態で commit しない。
  ただし型変更とそれに伴う呼び出し側修正など、**壊さずに分割できない変更は 1 commit にまとめる**
- secret / credential / local environment file を commit しない
- commit message の subject は `YTA-<番号> <英語の要約>` とし、本文は日本語でよい

```text
YTA-5 Set up the Ebitengine game client project

Ebitengineを採用した最小のゲームクライアントを追加する。

- ...
```

- push と PR 作成は、Issue の実装・検証が済んだら個別の事前確認なしで実行してよい
- PR は最新 head の実際の差分・tests・Acceptance Criteria・checks をレビューし、blocking な問題が
  無ければマージしてよい。マージ前に Issue を Done にしない
- リモートリポジトリ作成・force push・公開・deploy など、上記以外の外部へ影響する操作は
  実行前にユーザーの確認を取る
- PR を作成する場合は、Linear Issue へのリンク、Summary、Implementation notes、検証結果、
  manual verification 手順、Known limitations を記載する。PR 作成だけで Issue を Done にしない

## 11. ドキュメントと言語

- ユーザーへの報告、コードコメント、README、commit body は日本語
- 識別子、package 名、commit subject は英語
- Linear に明示されていない技術的判断をした場合は、**README の「技術的な判断」に項目と理由を追記する**
- 起動方法・build 手順を変えたら README を同じ commit で更新する
- README にゲームバランスや個別 Issue の仕様を書かない。仕様は Linear を参照させる
