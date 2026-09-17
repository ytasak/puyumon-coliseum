package game

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

// YTA-7が要求する必須Emojiは12種。重複していると1画面での比較にならない。
func TestPoCEmojiIsTheRequiredGlyphSet(t *testing.T) {
	t.Parallel()

	const want = 12
	if got := len(pocEmoji); got != want {
		t.Errorf("len(pocEmoji) = %d, want %d", got, want)
	}

	seen := make(map[string]bool, len(pocEmoji))
	for _, e := range pocEmoji {
		if seen[e] {
			t.Errorf("pocEmoji contains %q more than once", e)
		}
		seen[e] = true
	}
}

// 必須Emojiがすべて同梱フォントの単一カラーglyphへ解決できることを確認する。
//
// ヘッドレスではpixelを読み出せないため色そのものは検証できない。
// ここではEbitengineがカラーglyphとして扱っているか（グレースケールの
// アウトラインや.notdefへ落ちていないか）までを保証し、実際の見た目は
// Desktop起動とブラウザでの目視確認で担保する。
func TestPoCEmojiResolveToSingleColorGlyphs(t *testing.T) {
	t.Parallel()

	set, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New() returned error: %v", err)
	}

	for _, e := range pocEmoji {
		if !set.IsColorGlyph(e) {
			t.Errorf("IsColorGlyph(%q) = false, want true; the bundled font cannot render it as a color glyph", e)
		}
	}
}

// 倍率サンプルに使うEmojiも必須Emojiの中から選ぶ。
func TestScaleSampleEmojiIsOneOfThePoCEmoji(t *testing.T) {
	t.Parallel()

	for _, e := range pocEmoji {
		if e == scaleSampleEmoji {
			return
		}
	}
	t.Errorf("scaleSampleEmoji = %q, want one of pocEmoji", scaleSampleEmoji)
}
