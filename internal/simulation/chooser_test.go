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
	script := &Script{Fallback: func() Chooser { return fixedChooser{slot: 3, target: 2} }}

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

// TestReplayStartsFromTheDefinition はReplayが呼ばれるたびに
// 使いかけでないScriptを返すことを確かめる。
//
// Configを使い回しても、前のRunで進んだ分を引き継がないことの土台になる。
func TestReplayStartsFromTheDefinition(t *testing.T) {
	state := testState(t)
	factory := Replay(Script{Actions: []battle.Action{
		battle.MoveAction{Slot: 2},
		battle.MoveAction{Slot: 3},
	}})

	first := factory()
	first.Action(state, battle.Player1)
	first.Action(state, battle.Player1)

	second := factory()
	if got := second.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 2}) {
		t.Errorf("2つ目のChooserのAction() = %v, want slot 2（先頭から始まる）", got)
	}
	if first == second {
		t.Error("Replayが同じinstanceを返している")
	}
}

// TestReplayDoesNotShareFallback は複製したScript同士が
// Fallbackのinstanceを共有しないことを確かめる。
func TestReplayDoesNotShareFallback(t *testing.T) {
	state := testState(t)
	created := 0
	factory := Replay(Script{Fallback: func() Chooser {
		created++
		return fixedChooser{slot: 1, target: 1}
	}})

	factory().Action(state, battle.Player1)
	factory().Action(state, battle.Player1)

	if created != 2 {
		t.Errorf("Fallbackを作った回数 = %d, want 2", created)
	}
}

// TestScriptCreatesFallbackOnce は1つのScriptがFallbackを1度だけ作ることを確かめる。
func TestScriptCreatesFallbackOnce(t *testing.T) {
	state := testState(t)
	created := 0
	script := &Script{Fallback: func() Chooser {
		created++
		return fixedChooser{slot: 1, target: 1}
	}}

	script.Action(state, battle.Player1)
	script.Action(state, battle.Player1)
	script.Replacement(state, battle.Player1)

	if created != 1 {
		t.Errorf("Fallbackを作った回数 = %d, want 1", created)
	}
}

// stubRNG は決められた値を順に返すtest用のRNG。
//
// 尽きたら先頭へ戻る。引かれた回数も数える。
type stubRNG struct {
	values []int
	calls  int
}

// IntN は次の値をnで丸めて返す。
func (r *stubRNG) IntN(n int) int {
	value := r.values[r.calls%len(r.values)]
	r.calls++
	return value % n
}

// TestUniformUsablePicksFromUsableMoves は使える技の中から乱数で選ぶことを確かめる。
func TestUniformUsablePicksFromUsableMoves(t *testing.T) {
	state := testState(t)
	rng := &stubRNG{values: []int{0, 1, 2, 3}}
	chooser := UniformUsable{RNG: rng}

	for i := 0; i < battle.MoveSlots; i++ {
		want := battle.MoveAction{Slot: i}
		if got := chooser.Action(state, battle.Player1); got != want {
			t.Errorf("%d回目のAction() = %v, want %v", i+1, got, want)
		}
	}
	if rng.calls != battle.MoveSlots {
		t.Errorf("引いた乱数の数 = %d, want %d", rng.calls, battle.MoveSlots)
	}
}

// TestUniformUsableSkipsEmptyPP はPPの無い技を候補から外すことを確かめる。
//
// 乱数が0を返しても、PPの切れたslot 0は選ばない。
func TestUniformUsableSkipsEmptyPP(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	player.Team[player.Active].Moves[0].PP = 0
	player.Team[player.Active].Moves[1].PP = 0

	chooser := UniformUsable{RNG: &stubRNG{values: []int{0}}}
	if got := chooser.Action(state, battle.Player1); got != (battle.MoveAction{Slot: 2}) {
		t.Errorf("Action() = %v, want slot 2", got)
	}
}

// TestUniformUsableDrawsEvenWithOneChoice は候補が1つでも乱数を1つ引くことを確かめる。
//
// 行動を選ぶたびに必ず1つ消費することで、乱数列の位置が
// 「その時点で何技使えたか」に左右されなくなる。
func TestUniformUsableDrawsEvenWithOneChoice(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	for slot := 1; slot < battle.MoveSlots; slot++ {
		player.Team[player.Active].Moves[slot].PP = 0
	}

	rng := &stubRNG{values: []int{0}}
	if got := (UniformUsable{RNG: rng}).Action(state, battle.Player1); got != (battle.MoveAction{Slot: 0}) {
		t.Errorf("Action() = %v, want slot 0", got)
	}
	if rng.calls != 1 {
		t.Errorf("引いた乱数の数 = %d, want 1", rng.calls)
	}
}

// TestUniformUsableSwitchesWithoutUsableMove は使える技が無ければ交代し、
// そのときは乱数を引かないことを確かめる。
func TestUniformUsableSwitchesWithoutUsableMove(t *testing.T) {
	state := testState(t)
	player := &state.Players[battle.Player1]
	for slot := range player.Team[player.Active].Moves {
		player.Team[player.Active].Moves[slot].PP = 0
	}

	rng := &stubRNG{values: []int{0}}
	if got := (UniformUsable{RNG: rng}).Action(state, battle.Player1); got != (battle.SwitchAction{Target: 1}) {
		t.Errorf("Action() = %v, want target 1", got)
	}
	if rng.calls != 0 {
		t.Errorf("引いた乱数の数 = %d, want 0", rng.calls)
	}
}

// TestUniformUsableReplacementIsDeterministic は交代先が乱数に依らないことを確かめる。
func TestUniformUsableReplacementIsDeterministic(t *testing.T) {
	state := testState(t)
	state.Players[battle.Player1].Team[0].CurrentHP = 0

	rng := &stubRNG{values: []int{1}}
	if got := (UniformUsable{RNG: rng}).Replacement(state, battle.Player1); got != (battle.SwitchAction{Target: 1}) {
		t.Errorf("Replacement() = %v, want target 1", got)
	}
	if rng.calls != 0 {
		t.Errorf("引いた乱数の数 = %d, want 0", rng.calls)
	}
}

// TestUniformUsablePolicyStartsFromTheSeed はfactoryが呼ばれるたびに
// 同じseedから始まるChooserを返すことを確かめる。
func TestUniformUsablePolicyStartsFromTheSeed(t *testing.T) {
	state := testState(t)
	factory := UniformUsablePolicy(42)

	first := factory()
	firstActions := []battle.Action{first.Action(state, battle.Player1), first.Action(state, battle.Player1)}

	second := factory()
	for i, want := range firstActions {
		if got := second.Action(state, battle.Player1); got != want {
			t.Errorf("2つ目のChooserの%d回目 = %v, want %v", i+1, got, want)
		}
	}
}

// TestUniformUsablePolicySeedsDiffer は別のseedなら別の行動列になることを確かめる。
func TestUniformUsablePolicySeedsDiffer(t *testing.T) {
	state := testState(t)

	const draws = 20
	same := true
	first, second := UniformUsablePolicy(1)(), UniformUsablePolicy(2)()
	for i := 0; i < draws; i++ {
		if first.Action(state, battle.Player1) != second.Action(state, battle.Player1) {
			same = false
			break
		}
	}
	if same {
		t.Errorf("seedを変えても%d回とも同じ行動を選んでいる", draws)
	}
}

// TestFirstUsableStrugglesWithoutMovesOrReserve は使える技も控えも無ければ
// Struggleを選ぶことを確かめる。
//
// ここで合法でない行動を返すと、resolverがinvalid actionを返して対戦が止まる。
func TestFirstUsableStrugglesWithoutMovesOrReserve(t *testing.T) {
	state := testState(t)
	strandThe(&state, battle.Player1)

	if got := (FirstUsable{}).Action(state, battle.Player1); got != (battle.StruggleAction{}) {
		t.Errorf("Action() = %v, want StruggleAction", got)
	}
}

// TestUniformUsableStrugglesWithoutMovesOrReserve はUniformUsableも
// 同じ状況でStruggleを選び、そのとき乱数を引かないことを確かめる。
func TestUniformUsableStrugglesWithoutMovesOrReserve(t *testing.T) {
	state := testState(t)
	strandThe(&state, battle.Player1)

	rng := &stubRNG{values: []int{0}}
	if got := (UniformUsable{RNG: rng}).Action(state, battle.Player1); got != (battle.StruggleAction{}) {
		t.Errorf("Action() = %v, want StruggleAction", got)
	}
	if rng.calls != 0 {
		t.Errorf("引いた乱数の数 = %d, want 0", rng.calls)
	}
}

// strandThe は使える技も戦える控えも無い状態にする。Struggleしか残らない。
func strandThe(state *battle.BattleState, side battle.Side) {
	player := &state.Players[side]
	for slot := range player.Team[player.Active].Moves {
		player.Team[player.Active].Moves[slot].PP = 0
	}
	for i := range player.Team {
		if i != player.Active {
			player.Team[i].CurrentHP = 0
		}
	}
}
