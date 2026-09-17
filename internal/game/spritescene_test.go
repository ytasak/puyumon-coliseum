package game

import (
	"slices"
	"testing"
	"unicode"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// ナッシー型はIssueが指定した素材（body 🌴 / faces 🥺 😫 🤪）で構成する。
func TestNassyIsBuiltFromTheSpecifiedMaterials(t *testing.T) {
	t.Parallel()

	var got []string
	for _, p := range nassy.Parts {
		got = append(got, p.Emoji)
	}
	slices.Sort(got)

	want := []string{"🌴", "🤪", "😫", "🥺"}
	slices.Sort(want)
	if !slices.Equal(got, want) {
		t.Errorf("nassy parts = %v, want %v", got, want)
	}

	if body := nassy.Parts[0]; body.Emoji != "🌴" {
		t.Errorf("first drawn part = %q, want the body %q", body.Emoji, "🌴")
	}
}

// 描画順はZの昇順で、bodyが最初（もっとも奥）に来る。
func TestPoCCharactersAreOrderedByZ(t *testing.T) {
	t.Parallel()

	for _, c := range pocCharacters {
		t.Run(c.label, func(t *testing.T) {
			for i := 1; i < len(c.character.Parts); i++ {
				if prev, cur := c.character.Parts[i-1].Z, c.character.Parts[i].Z; prev > cur {
					t.Errorf("part %d has Z=%d after Z=%d; parts must be ordered back to front", i, cur, prev)
				}
			}
		})
	}
}

// PoCキャラクターに使うEmojiは、すべて同梱フォントでカラー描画できる必要がある。
func TestPoCCharacterPartsUseColorGlyphs(t *testing.T) {
	t.Parallel()

	set, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New() returned error: %v", err)
	}

	for _, c := range pocCharacters {
		for _, p := range c.character.Parts {
			if !set.IsColorGlyph(p.Emoji) {
				t.Errorf("%s: IsColorGlyph(%q) = false, want true", c.label, p.Emoji)
			}
		}
	}
}

// Scaleが0の部品は見えない。定義の書き間違いを拾う。
func TestPoCCharacterPartsHavePositiveScale(t *testing.T) {
	t.Parallel()

	for _, c := range pocCharacters {
		for _, p := range c.character.Parts {
			if p.Scale <= 0 {
				t.Errorf("%s: part %q has Scale = %v, want > 0", c.label, p.Emoji, p.Scale)
			}
		}
	}
}

// 画面のラベルはebitenutil.DebugPrintAtの組み込みASCIIフォントで描くため、
// 非ASCII文字が混ざると表示できない。
func TestSceneLabelsAreASCIIOnly(t *testing.T) {
	t.Parallel()

	labels := make([]string, 0, len(pocCharacters)+len(transformVariants))
	for _, c := range pocCharacters {
		labels = append(labels, c.label)
	}
	for _, v := range transformVariants {
		labels = append(labels, v.label)
	}

	for _, label := range labels {
		for _, r := range label {
			if r > unicode.MaxASCII {
				t.Errorf("label %q contains non-ASCII rune %q", label, r)
			}
		}
	}
}

// 下段のtransformは、全体のscaleかrotationのどちらかを基準から変えている。
// すべて同じ見た目では「一体として変換される」ことを確認できない。
func TestTransformVariantsDifferFromTheBaseTransform(t *testing.T) {
	t.Parallel()

	base := sprite.Transform{Scale: characterScale}
	for _, v := range transformVariants {
		if v.scale == base.Scale && v.rotation == base.Rotation {
			t.Errorf("variant %q is identical to the base transform", v.label)
		}
	}
}
