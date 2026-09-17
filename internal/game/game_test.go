package game

import (
	"strings"
	"testing"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

func newGame(t *testing.T) *Game {
	t.Helper()

	g, err := New()
	if err != nil {
		t.Fatalf("New() returned error: %v", err)
	}
	return g
}

func TestNewStartsAtZeroTicks(t *testing.T) {
	t.Parallel()

	if got := newGame(t).Ticks(); got != 0 {
		t.Errorf("New().Ticks() = %d, want 0", got)
	}
}

// ゲームループが継続動作していることを、Updateがerrorを返さずtickを進め続けることで検証する。
func TestUpdateAdvancesTicksWithoutError(t *testing.T) {
	t.Parallel()

	const frames = 600 // 60 TPSで約10秒相当

	g := newGame(t)
	for i := 1; i <= frames; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("Update() at frame %d returned error: %v", i, err)
		}
		if got := g.Ticks(); got != uint64(i) {
			t.Fatalf("Ticks() after %d updates = %d, want %d", i, got, i)
		}
	}
}

// Drawはゲームループ外から呼んでもpanicしない。
// ヘッドレスではgraphics contextが無いため描画結果のpixelは検証できず、
// 描画コマンドの組み立てが壊れていないことのみを確認する。
func TestDrawDoesNotPanic(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	screen := ebiten.NewImage(LogicalWidth, LogicalHeight)

	g.Draw(screen)

	if err := g.Update(); err != nil {
		t.Fatalf("Update() returned error: %v", err)
	}
	g.Draw(screen)
}

func TestOverlayTextIdentifiesThePoC(t *testing.T) {
	t.Parallel()

	got := newGame(t).overlayText()

	for _, want := range []string{"PUYUMON COLISEUM", "YTA-7", "640x360", "ticks: 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("overlayText() = %q, want it to contain %q", got, want)
		}
	}
}

func TestOverlayTextReflectsTicks(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	for i := 0; i < 3; i++ {
		if err := g.Update(); err != nil {
			t.Fatalf("Update() returned error: %v", err)
		}
	}

	if got, want := g.overlayText(), "ticks: 3"; !strings.Contains(got, want) {
		t.Errorf("overlayText() = %q, want it to contain %q", got, want)
	}
}

// ebitenutil.DebugPrintは組み込みのASCIIフォントで描画するため、
// 識別用テキストに非ASCII文字が混ざると表示できない。
// フォント同梱とEmoji描画はYTA-7で扱う。
func TestOverlayTextIsASCIIOnly(t *testing.T) {
	t.Parallel()

	for _, r := range newGame(t).overlayText() {
		if r > unicode.MaxASCII {
			t.Errorf("overlayText() contains non-ASCII rune %q", r)
		}
	}
}
