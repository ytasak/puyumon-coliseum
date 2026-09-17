package emoji

import "testing"

func newSet(t *testing.T) *Set {
	t.Helper()

	s, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	return s
}

// 同梱フォントが壊れていればここで落ちる。
func TestNewLoadsBundledFont(t *testing.T) {
	t.Parallel()

	newSet(t)
}

// Emojiの表示サイズはセル画像の拡大縮小で決めるため、どのEmojiでも
// 同じ大きさの正方セルが返る必要がある。
func TestImageIsAlwaysOneSquareCell(t *testing.T) {
	t.Parallel()

	s := newSet(t)
	for _, e := range []string{"🌴", "⚡", "❄️"} {
		b := s.Image(e).Bounds()
		if b.Dx() != CellSize || b.Dy() != CellSize {
			t.Errorf("Image(%q) bounds = %dx%d, want %dx%d", e, b.Dx(), b.Dy(), CellSize, CellSize)
		}
	}
}

// セル画像は1度だけ生成し、以降は同じ画像を使い回す。
// 毎フレームのglyph初期化とセル画像の作り直しを避けるための要件。
func TestImageIsGeneratedOncePerEmoji(t *testing.T) {
	t.Parallel()

	s := newSet(t)

	first := s.Image("🌴")
	if second := s.Image("🌴"); second != first {
		t.Error("Image() returned a different image for the same emoji; the cell should be cached")
	}
	if other := s.Image("🥺"); other == first {
		t.Error("Image() returned the same image for different emoji")
	}
}

// 素材として使えないものはIsColorGlyphで弾けること。
func TestIsColorGlyphRejectsWhatIsNotASingleColorGlyph(t *testing.T) {
	t.Parallel()

	s := newSet(t)
	for _, tc := range []struct {
		name string
		in   string
	}{
		{"empty", ""},
		{"two emoji", "🌴🥺"},
		{"ascii letter", "A"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			if s.IsColorGlyph(tc.in) {
				t.Errorf("IsColorGlyph(%q) = true, want false", tc.in)
			}
		})
	}
}
