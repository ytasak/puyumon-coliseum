package game

import (
	"testing"
	"unicode"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
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
		assertASCII(t, message)
		assertFitsOnScreen(t, message)
	}
}

// 相手のことにはFOEを付け、自分のことには付けない。
func TestMessagesTellTheSidesApart(t *testing.T) {
	t.Parallel()

	mine := battleui.Ref{Side: viewer, Index: 0, Species: roster.SpeciesBull}
	theirs := battleui.Ref{Side: foe, Index: 0, Species: roster.SpeciesBull}

	if got := cueMessage(battleui.FaintCue{Target: mine}, viewer); got != "BULL FAINTED" {
		t.Errorf("自分側 = %q", got)
	}
	if got := cueMessage(battleui.FaintCue{Target: theirs}, viewer); got != "FOE BULL FAINTED" {
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
		{name: "勝ち", result: battleui.Result{Decided: true, Winner: viewer}, want: "YOU WIN"},
		{name: "負け", result: battleui.Result{Decided: true, Winner: foe}, want: "YOU LOSE"},
		{name: "引き分け", result: battleui.Result{Decided: true, Draw: true}, want: "DRAW"},
	}
	for _, tc := range tests {
		if got := resultMessage(tc.result, viewer); got != tc.want {
			t.Errorf("%s: %q, want %q", tc.name, got, tc.want)
		}
	}
}

// 1試合を通して出る文章はすべてASCIIで、画面に収まる。
func TestMessagesThroughAWholeMatchAreASCII(t *testing.T) {
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
		assertASCII(t, message)
		assertFitsOnScreen(t, message)
	}
}

// assertASCII は組み込みフォントで描けることを確かめる。
func assertASCII(t *testing.T, message string) {
	t.Helper()

	for _, r := range message {
		if r > unicode.MaxASCII {
			t.Errorf("%q にASCII外の文字 %q が含まれる", message, r)
		}
	}
}

// assertFitsOnScreen は1行が画面幅に収まることを確かめる。
func assertFitsOnScreen(t *testing.T, message string) {
	t.Helper()

	if width := len(message) * debugFontCharWidth; width > LogicalWidth-commandMargin*2 {
		t.Errorf("%q が画面幅に収まらない（%d px）", message, width)
	}
}
