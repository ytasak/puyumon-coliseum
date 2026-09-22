package game

import (
	"testing"

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
