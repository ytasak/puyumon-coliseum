package uifont

import (
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// 同梱フォントを解析できる。
func TestNewParsesTheBundledFont(t *testing.T) {
	t.Parallel()

	face, err := New()
	if err != nil {
		t.Fatalf("New()に失敗: %v", err)
	}
	if face.Source == nil {
		t.Fatal("faceにSourceが無い")
	}
	if face.Size != Size {
		t.Errorf("Size = %v, want %v", face.Size, Size)
	}
}

// 日本語の主要な文字種を描ける。
//
// 画面に出す文字そのものの網羅は internal/game 側で見る。ここでは
// 「同梱したフォントが日本語フォントである」ことだけを確かめる。
func TestBundledFontCoversJapanese(t *testing.T) {
	t.Parallel()

	face, err := New()
	if err != nil {
		t.Fatalf("New()に失敗: %v", err)
	}

	groups := map[string]string{
		"ひらがな": "あいうえおがぎぐげござじずぜぞぱぴぷぺぽっゃゅょー、。",
		"カタカナ": "アイウエオガギグゲゴザジズゼゾパピプペポッャュョー・",
		"漢字":   "相手攻撃防御素早特殊命中回避状態異常倒回復急所効果",
		"英数記号": "0123456789ABCXYZabcxyz/-:()！？",
	}
	for name, s := range groups {
		if missing := MissingGlyphs(face, s); len(missing) > 0 {
			t.Errorf("%s に字形の無い文字がある: %q", name, missing)
		}
	}
}

// 字形の無い文字を見つけられる。
//
// MissingGlyphs が常に空を返すだけの関数になっていないことを確かめる。
// Emojiは別フォント（internal/emoji）で描いており、UIフォントには無い。
func TestMissingGlyphsReportsAbsentRunes(t *testing.T) {
	t.Parallel()

	face, err := New()
	if err != nil {
		t.Fatalf("New()に失敗: %v", err)
	}

	missing := MissingGlyphs(face, "🐂🌴")
	if len(missing) != 2 {
		t.Fatalf("MissingGlyphs(Emoji) = %q, 2文字とも無いはず", missing)
	}

	// 重複は1度だけ報告する。
	if got := MissingGlyphs(face, "🐂🐂🐂"); len(got) != 1 {
		t.Errorf("MissingGlyphs(同じEmoji3つ) = %q, 1件のはず", got)
	}
}

// 表示幅は全角と半角で変わる。
//
// 固定幅として扱うと日本語の中央揃えと折り返しがずれるので、実測できる
// ことを固定しておく。
func TestMeasureDistinguishesFullAndHalfWidth(t *testing.T) {
	t.Parallel()

	face, err := New()
	if err != nil {
		t.Fatalf("New()に失敗: %v", err)
	}

	full, _ := text.Measure("ああああ", face, LineHeight)
	half, _ := text.Measure("aaaa", face, LineHeight)

	if full <= half {
		t.Errorf("全角4文字 %.0f px が半角4文字 %.0f px 以下になっている", full, half)
	}
	if want := float64(4 * Size); full != want {
		t.Errorf("全角4文字 = %.0f px, want %.0f px", full, want)
	}
}
