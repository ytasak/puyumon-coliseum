package simulation

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// testState は Chooser を試すためのBattleStateを作る。
func testState(t *testing.T) battle.BattleState {
	t.Helper()

	first, err := roster.NewTeam([battle.TeamSize]battle.SpeciesID{
		roster.SpeciesBull, roster.SpeciesStar, roster.SpeciesJolt,
	})
	if err != nil {
		t.Fatalf("roster.NewTeam() error = %v", err)
	}
	second, err := roster.NewTeam([battle.TeamSize]battle.SpeciesID{
		roster.SpeciesPalm, roster.SpeciesCharm, roster.SpeciesWhale,
	})
	if err != nil {
		t.Fatalf("roster.NewTeam() error = %v", err)
	}

	state, err := battle.NewBattleState(first, second)
	if err != nil {
		t.Fatalf("battle.NewBattleState() error = %v", err)
	}
	return state
}

// TestScriptReturnsActionsInOrder はScriptが指定した順に行動を返すことを確かめる。
func TestScriptReturnsActionsInOrder(t *testing.T) {
	state := testState(t)
	script := &Script{Actions: []battle.Action{
		battle.MoveAction{Slot: 2},
		battle.SwitchAction{Target: 1},
		battle.MoveAction{Slot: 3},
	}}

	for i, want := range script.Actions {
		if got := script.Action(state, battle.Player1); got != want {
			t.Errorf("%d回目のAction() = %v, want %v", i+1, got, want)
		}
	}
}

// TestScriptFallsBackWhenExhausted はActionsが尽きたらFallbackへ委ねることを確かめる。
func TestScriptFallsBackWhenExhausted(t *testing.T) {
	state := testState(t)
	script := &Script{Actions: []battle.Action{battle.MoveAction{Slot: 2}}}

	if got := script.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 2}) {
		t.Fatalf("1回目のAction() = %v, want slot 2", got)
	}
	// FallbackはnilなのでFirstUsableが引き継ぎ、使える先頭の技を選ぶ。
	if got := script.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 0}) {
		t.Errorf("2回目のAction() = %v, want slot 0", got)
	}
}

// TestScriptUsesGivenFallback は指定したFallbackを使うことを確かめる。
func TestScriptUsesGivenFallback(t *testing.T) {
	state := testState(t)
	script := &Script{Fallback: fixedChooser{slot: 3, target: 2}}

	if got := script.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 3}) {
		t.Errorf("Action() = %v, want slot 3", got)
	}
	if got := script.Replacement(state, battle.Player1); got != (battle.SwitchAction{Target: 2}) {
		t.Errorf("Replacement() = %v, want target 2", got)
	}
}

// TestScriptReplacementsRunOutIndependently はActionsとReplacementsが
// 別々に消費されることを確かめる。
func TestScriptReplacementsRunOutIndependently(t *testing.T) {
	state := testState(t)
	script := &Script{
		Actions:      []battle.Action{battle.MoveAction{Slot: 1}, battle.MoveAction{Slot: 2}},
		Replacements: []battle.SwitchAction{{Target: 2}},
	}

	script.Action(state, battle.Player1)
	if got := script.Replacement(state, battle.Player1); got != (battle.SwitchAction{Target: 2}) {
		t.Errorf("Replacement() = %v, want target 2", got)
	}
	if got := script.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 2}) {
		t.Errorf("Action() = %v, want slot 2（Replacementに消費されていない）", got)
	}
}

// TestFirstUsableSkipsEmptyPP はPPの無い技を飛ばすことを確かめる。
func TestFirstUsableSkipsEmptyPP(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	player.Team[player.Active].Moves[0].PP = 0

	if got := (FirstUsable{}).Action(state, battle.Player1); got != (battle.MoveAction{Slot: 1}) {
		t.Errorf("Action() = %v, want slot 1", got)
	}
}

// TestFirstUsableSwitchesWithoutUsableMove は使える技が無ければ交代することを確かめる。
func TestFirstUsableSwitchesWithoutUsableMove(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	for slot := range player.Team[player.Active].Moves {
		player.Team[player.Active].Moves[slot].PP = 0
	}

	if got := (FirstUsable{}).Action(state, battle.Player1); got != (battle.SwitchAction{Target: 1}) {
		t.Errorf("Action() = %v, want target 1", got)
	}
}

// TestFirstUsableReplacementSkipsFainted は戦闘不能の控えを選ばないことを確かめる。
func TestFirstUsableReplacementSkipsFainted(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	player.Team[player.Active].CurrentHP = 0
	player.Team[1].CurrentHP = 0

	if got := (FirstUsable{}).Replacement(state, battle.Player1); got != (battle.SwitchAction{Target: 2}) {
		t.Errorf("Replacement() = %v, want target 2", got)
	}
}

// fixedChooser は常に同じ行動を返すtest用のChooser。
type fixedChooser struct {
	slot   int
	target int
}

func (c fixedChooser) Action(battle.BattleState, battle.Side) battle.Action {
	return battle.MoveAction{Slot: c.slot}
}

func (c fixedChooser) Replacement(battle.BattleState, battle.Side) battle.SwitchAction {
	return battle.SwitchAction{Target: c.target}
}
