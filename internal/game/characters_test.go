package game

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// 6体すべてに見た目がある。
func TestEveryRosterCharacterHasALook(t *testing.T) {
	t.Parallel()

	all := roster.All()
	if len(characters) != len(all) {
		t.Errorf("定義が %d 体（rosterは %d 体）", len(characters), len(all))
	}
	for _, character := range all {
		look, ok := characters[character.ID]
		if !ok {
			t.Errorf("%s の見た目が無い", character.ID)
			continue
		}
		if len(look.Parts) == 0 {
			t.Errorf("%s の部品が空", character.ID)
		}
		if emojiFor(character.ID) == "" {
			t.Errorf("%s の代表Emojiが空", character.ID)
		}
	}
}

// 6体は見分けがつく。体のEmojiが重なっていない。
func TestCharacterLooksAreDistinct(t *testing.T) {
	t.Parallel()

	bodies := map[string]bool{}
	for id, look := range characters {
		body := look.Parts[0].Emoji
		if bodies[body] {
			t.Errorf("%s の体 %s が他と重複している", id, body)
		}
		bodies[body] = true
	}
}

// 部品は奥から手前の順に並んでいる。
func TestCharacterPartsAreOrderedByZ(t *testing.T) {
	t.Parallel()

	for id, look := range characters {
		for i := 1; i < len(look.Parts); i++ {
			if look.Parts[i-1].Z > look.Parts[i].Z {
				t.Errorf("%s の部品がZの昇順になっていない: %d の後に %d",
					id, look.Parts[i-1].Z, look.Parts[i].Z)
			}
		}
	}
}

// 使うEmojiはすべて同梱フォントのカラーglyphで描ける。
//
// 描けないEmojiを混ぜると、実機で1文字だけ白黒になったり欠けたりする。
func TestCharacterEmojiUseColorGlyphs(t *testing.T) {
	t.Parallel()

	emojis, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New()に失敗: %v", err)
	}

	for id, look := range characters {
		for _, part := range look.Parts {
			if !emojis.IsColorGlyph(part.Emoji) {
				t.Errorf("%s の %s が同梱フォントのカラーglyphでない", id, part.Emoji)
			}
		}
		if e := emojiFor(id); !emojis.IsColorGlyph(e) {
			t.Errorf("%s の代表Emoji %s が同梱フォントのカラーglyphでない", id, e)
		}
	}
}

// 定義の無いキャラクターでも描ける形を返す。
func TestCharacterForFallsBackToASinglePart(t *testing.T) {
	t.Parallel()

	look := characterFor("unknown")
	if len(look.Parts) != 1 {
		t.Fatalf("部品が %d 個（1個のはず）", len(look.Parts))
	}
}
