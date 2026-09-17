package sprite

import (
	"testing"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

func newRenderer(t *testing.T) *Renderer {
	t.Helper()

	emojis, err := emoji.New()
	if err != nil {
		t.Fatalf("emoji.New() returned error: %v", err)
	}
	return NewRenderer(emojis)
}

// Drawはゲームループ外から呼んでもpanicしない。
// ヘッドレスではgraphics contextが無いため描画結果のpixelは検証できず、
// 描画コマンドの組み立てが壊れていないことのみを確認する。
func TestDrawDoesNotPanic(t *testing.T) {
	t.Parallel()

	r := newRenderer(t)
	character := NewCharacter(
		Part{Emoji: "🌴", Scale: 1, Z: 0},
		Part{Emoji: "🥺", X: -0.2, Y: -0.2, Scale: 0.3, Rotation: -0.1, Z: 1},
	)

	dst := ebiten.NewImage(320, 180)
	r.Draw(dst, character, Transform{X: 160, Y: 90, Scale: 100})
	r.Draw(dst, character, Transform{X: 160, Y: 90, Scale: 60, Rotation: 0.4})
}

// 静的なキャラクターをoffscreen imageへ描いて使い回せる構造であることを確認する。
// キャッシュ自体はPoCでは実装しないが、dstを差し替えられることが前提になる。
func TestDrawAcceptsAnOffscreenDestination(t *testing.T) {
	t.Parallel()

	r := newRenderer(t)
	character := NewCharacter(Part{Emoji: "🌴", Scale: 1})

	const size = emoji.CellSize
	offscreen := ebiten.NewImage(size, size)
	r.Draw(offscreen, character, Transform{X: size / 2, Y: size / 2, Scale: size})

	if b := offscreen.Bounds(); b.Dx() != size || b.Dy() != size {
		t.Errorf("offscreen bounds = %dx%d, want %dx%d", b.Dx(), b.Dy(), size, size)
	}
}

// 空のCharacterでも描画は落ちない。
func TestDrawWithNoParts(t *testing.T) {
	t.Parallel()

	newRenderer(t).Draw(ebiten.NewImage(64, 64), NewCharacter(), Transform{Scale: 1})
}
