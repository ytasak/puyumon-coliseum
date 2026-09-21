package game

import (
	"strings"
	"testing"
	"unicode"

	"github.com/hajimehoshi/ebiten/v2"
)

// testSeed はtestで使う固定のseed。同じ対戦から始まるようにする。
const testSeed = 1

func newGame(t *testing.T) *Game {
	t.Helper()

	g, err := New(testSeed)
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

// どの場面でもDrawがpanicしない。
//
// lead選択・技の一覧・交代・再生中・決着で描くものが変わる。
// ヘッドレスではpixelを読めないので、組み立てが壊れていないことだけを見る。
func TestDrawDoesNotPanicInEveryPhase(t *testing.T) {
	t.Parallel()

	g := newGame(t)
	screen := ebiten.NewImage(LogicalWidth, LogicalHeight)
	scene := g.battle

	g.Draw(screen) // lead選択

	submitOrFatal(t, scene, command{kind: commandLead, index: 0})
	g.Draw(screen) // 技を選ぶ画面

	scene.menu = menuFight
	g.Draw(screen)
	scene.menu = menuSwitch
	g.Draw(screen)
	scene.menu = menuRoot

	c, ok := firstChoice(scene.view.Commands)
	if !ok {
		t.Fatal("選べる行動が無い")
	}
	submitOrFatal(t, scene, c)
	g.Draw(screen) // 再生中

	drainCues(t, scene)
	g.Draw(screen)

	playToFinish(t, scene)
	g.Draw(screen) // 決着
}

func TestOverlayTextIdentifiesTheBuild(t *testing.T) {
	t.Parallel()

	got := newGame(t).overlayText()

	for _, want := range []string{"PUYUMON", "640x360", "ticks: 0"} {
		if !strings.Contains(got, want) {
			t.Errorf("overlayText() = %q, want it to contain %q", got, want)
		}
	}
}

// 動作確認用テキストは相手の枠と重ならず、画面に収まる。
func TestOverlayTextFitsBesideTheOpponentPanel(t *testing.T) {
	t.Parallel()

	text := newGame(t).overlayText()
	width := len(text) * debugFontCharWidth

	if overlayTextOriginX+width > LogicalWidth {
		t.Errorf("%q が画面からはみ出す（右端 %d px）", text, overlayTextOriginX+width)
	}
	if overlayTextOriginX < infoPanels[foe].Max.X {
		t.Errorf("動作確認用テキストが相手の枠へかぶる（x %d < %d）", overlayTextOriginX, infoPanels[foe].Max.X)
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
