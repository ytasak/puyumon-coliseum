package game

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// 出しうるcueはすべて文章になる。
func TestEveryCueHasAMessage(t *testing.T) {
	t.Parallel()

	mine := battleui.Ref{Side: viewer, Index: 0, Species: roster.SpeciesBull}
	theirs := battleui.Ref{Side: foe, Index: 1, Species: roster.SpeciesStar}

	cues := []battleui.Cue{
		battleui.MoveUsedCue{Actor: mine, Move: roster.MoveSlam},
		battleui.DamageCue{Target: theirs, Amount: 42},
		battleui.DamageCue{Target: theirs, Amount: 42, Critical: true},
		battleui.HealCue{Target: mine, Amount: 40},
		battleui.StatusCue{Target: theirs, Status: battle.Paralysis, Applied: true},
		battleui.StatusCue{Target: theirs, Status: battle.Sleep},
		battleui.StatStageCue{Target: mine, Stat: battle.StatSpeed, Delta: 2},
		battleui.StatStageCue{Target: mine, Stat: battle.StatSpecial, Delta: -1},
		battleui.MultiHitCue{Actor: mine, Hits: 3},
		battleui.MissCue{Actor: mine, Target: theirs},
		battleui.UnaffectedCue{Actor: mine, Target: theirs},
		battleui.FailedCue{Actor: mine},
		battleui.BlockedCue{Target: mine, Reason: battleui.BlockSleep},
		battleui.BlockedCue{Target: mine, Reason: battleui.BlockRecharge},
		battleui.SwitchOutCue{Target: mine},
		battleui.SwitchInCue{Target: theirs},
		battleui.FaintCue{Target: theirs},
	}

	for _, cue := range cues {
		message := cueMessage(cue, viewer)
		if message == "" {
			t.Errorf("%T の文章が空", cue)
			continue
		}
		assertDrawable(t, message)
		assertFitsOnScreen(t, message)
	}
}

// 相手のことには「相手の」を付け、自分のことには付けない。
func TestMessagesTellTheSidesApart(t *testing.T) {
	t.Parallel()

	mine := battleui.Ref{Side: viewer, Index: 0, Species: roster.SpeciesBull}
	theirs := battleui.Ref{Side: foe, Index: 0, Species: roster.SpeciesBull}

	if got := cueMessage(battleui.FaintCue{Target: mine}, viewer); got != "ブルは倒れた" {
		t.Errorf("自分側 = %q", got)
	}
	if got := cueMessage(battleui.FaintCue{Target: theirs}, viewer); got != "相手のブルは倒れた" {
		t.Errorf("相手側 = %q", got)
	}
}

// 急所は文章にも出る。
func TestCriticalHitIsMentioned(t *testing.T) {
	t.Parallel()

	target := battleui.Ref{Side: foe, Index: 0, Species: roster.SpeciesStar}
	plain := cueMessage(battleui.DamageCue{Target: target, Amount: 10}, viewer)
	critical := cueMessage(battleui.DamageCue{Target: target, Amount: 10, Critical: true}, viewer)

	if plain == critical {
		t.Error("急所かどうかで文章が変わらない")
	}
}

// 決着は自分から見た言い方になる。
func TestResultMessageIsFromTheViewersSide(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		result battleui.Result
		want   string
	}{
		{name: "未決着", result: battleui.Result{}, want: ""},
		{name: "勝ち", result: battleui.Result{Decided: true, Winner: viewer}, want: "あなたの勝ち！"},
		{name: "負け", result: battleui.Result{Decided: true, Winner: foe}, want: "あなたの負け"},
		{name: "引き分け", result: battleui.Result{Decided: true, Draw: true}, want: "引き分け"},
	}
	for _, tc := range tests {
		if got := resultMessage(tc.result, viewer); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// 1試合を通して出る文章はすべて同梱フォントで描けて、画面に収まる。
func TestMessagesThroughAWholeMatchAreDrawable(t *testing.T) {
	t.Parallel()

	scene := startedScene(t, 3)
	seen := map[string]bool{}

	for step := 0; scene.view.Phase != singleplayer.PhaseFinished && step < maxSceneSteps; step++ {
		for scene.busy() {
			if err := scene.update(); err != nil {
				t.Fatalf("update()に失敗: %v", err)
			}
			seen[scene.message] = true
		}
		seen[scene.message] = true

		c, ok := firstChoice(scene.view.Commands)
		if !ok {
			if err := scene.update(); err != nil {
				t.Fatalf("update()に失敗: %v", err)
			}
			continue
		}
		submitOrFatal(t, scene, c)
	}

	if len(seen) < 3 {
		t.Fatalf("集まった文章が %d 種類しかない", len(seen))
	}
	for message := range seen {
		assertDrawable(t, message)
		assertFitsOnScreen(t, message)
	}
}

// assertDrawable は同梱フォントで全文字を描けることを確かめる。
//
// 字形の無い文字は画面で豆腐になる。実機で気づくしかない不具合なので、
// 表示しうる文字列をここで固定する。
func assertDrawable(t *testing.T, message string) {
	t.Helper()

	if missing := uifont.MissingGlyphs(testFace(t), message); len(missing) > 0 {
		t.Errorf("%q は同梱フォントに字形が無い文字を含む: %q", message, missing)
	}
}

// assertFitsOnScreen は1行がメッセージ枠に収まることを確かめる。
//
// **文字数ではなく実測幅で見る。** 日本語は全角と半角が混ざるので、
// 文字数もバイト数も表示幅と一致しない。
func assertFitsOnScreen(t *testing.T, message string) {
	t.Helper()

	// drawMessage が枠の左端から 10px 内側に描く。
	const inset = 10
	limit := float64(LogicalWidth - commandMargin*2 - inset*2)
	if width := textWidth(testFace(t), message); width > limit {
		t.Errorf("%q がメッセージ枠に収まらない（%.0f px > %.0f px）", message, width, limit)
	}
}
