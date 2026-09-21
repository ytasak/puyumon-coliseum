// Package uifont はゲーム画面の日本語テキストを描くための同梱フォントを提供する。
//
// 実行環境のシステムフォントやCDNには依存しない。Desktopとブラウザで同じ字形を
// 出すため、同梱したフォントだけで描画結果を固定する。これは internal/emoji が
// Emojiに対して取っているのと同じ方針で、経路も同じ text/v2 を使う。
//
// このpackageはフォントの読み込みと、描ける文字かどうかの判定だけを持つ。
// 色・位置・文言は描画側の関心事なので持たない。
// 同梱フォントの出典とライセンスは assets/README.md を参照。
package uifont

import (
	"bytes"
	_ "embed"
	"fmt"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// dotGothic16 は同梱した日本語UIフォント。
//
//go:embed assets/DotGothic16-Regular.ttf
var dotGothic16 []byte

// Size は画面テキストの標準サイズ（論理座標のpx）。
//
// DotGothic16は16pxのドット字形を想定して作られている。論理解像度640x360に対して
// 全角16pxなら1行に40文字入り、対戦の1行メッセージが収まる。
const Size = 16

// LineHeight は複数行を積むときの行送り。
//
// 16pxの字形に対して詰まりすぎない値として1.25倍を取る。
const LineHeight = Size * 1.25

// New は同梱フォントを解析したfaceを返す。
//
// フォントの解析はここで1度だけ行う。描画contextを必要としないので、
// ゲームループの開始前に呼べる。
func New() (*text.GoTextFace, error) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(dotGothic16))
	if err != nil {
		return nil, fmt.Errorf("uifont: parse bundled font: %w", err)
	}
	return &text.GoTextFace{Source: source, Size: Size}, nil
}

// MissingGlyphs は face で描けない文字を、現れた順に重複なく返す。
//
// 同梱フォントに字形が無い文字は .notdef（GID 0）へ解決され、画面では
// 豆腐になる。表示する文字列をこれに通せば、豆腐が出ないことをtestで
// 固定できる。glyph画像を作らないので描画contextは要らない。
func MissingGlyphs(face *text.GoTextFace, s string) []rune {
	var missing []rune
	seen := map[rune]bool{}

	// 1文字ずつ見る。まとめて渡すとligatureや合字でruneとglyphが1対1で
	// 対応しなくなり、どの文字が欠けたのかを言えなくなる。
	for _, r := range s {
		if seen[r] {
			continue
		}
		seen[r] = true

		glyphs := text.AppendLazyGlyphs(nil, string(r), face, nil)
		for _, g := range glyphs {
			if g.GID == 0 {
				missing = append(missing, r)
				break
			}
		}
	}
	return missing
}
