package battle

import (
	"errors"
	"reflect"
	"testing"
)

// drained は4技すべてのPPを0にした個体を返す。Struggleが使える条件そのもの。
func drained(p Pokemon) Pokemon {
	for i := range p.Moves {
		p.Moves[i].PP = 0
	}
	return p
}

// frozen はこおり状態の個体を返す。
//
// こおりはその手番をActionBlockedだけで終わらせ、乱数も引かない。
// 相手の手番を固定しておくと、StruggleのEventだけを読み取れる。
func frozen(p Pokemon) Pokemon {
	p.Status = Freeze
	return p
}

// struggleValues はStruggleが1回引く乱数。命中 / 急所なし / ダメージ乱数は最大。
var struggleValues = []int{0, 255, 255}

// blocked は相手がこおりで動けなかったことを表すEvent。
var blocked = ActionBlocked{Side: Player2, Reason: BlockedByFreeze}

// Struggleの威力は50。Level 55・Attack 180・Defense 150で、
// ノーマル技をみずタイプへ当てると等倍で30になる（(24*50*180/150)/50+2 = 30）。
// ノーマルタイプが使えばSTABが乗って45。
const (
	struggleDamage     = 30
	struggleRecoil     = struggleDamage / 2 // 15
	struggleSTABDamage = 45
	struggleSTABRecoil = struggleSTABDamage / 2 // 22。奇数は切り捨て
)

// TestStruggleIsLegalWithoutUsableMoves は4技すべてPP 0ならStruggleを使えることと、
// 通常のdamage pipelineを通って反動が返ることを確かめる。
func TestStruggleIsLegalWithoutUsableMoves(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		drained(fighter(speciesRunner, 130, moveTackle, moveQuick)),
		frozen(fighter(speciesTarget, 100, moveTackle)),
	)
	r := testResolver(struggleValues...)

	next, events, err := r.ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Damage{Side: Player2, Amount: struggleDamage, RemainingHP: 200 - struggleDamage},
		Damage{Side: Player1, Amount: struggleRecoil, RemainingHP: 200 - struggleRecoil},
		blocked,
	})
	if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 200-struggleRecoil {
		t.Errorf("使用者のHP = %d, want %d", got, 200-struggleRecoil)
	}
}

// TestStruggleGetsSTAB はノーマルタイプが使えばSTABが乗ることと、
// 反動が奇数ダメージで切り捨てになることを確かめる。
func TestStruggleGetsSTAB(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		drained(fighter(speciesPlain, 130, moveTackle)),
		frozen(fighter(speciesTarget, 100, moveTackle)),
	)
	r := testResolver(struggleValues...)

	_, events, err := r.ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Damage{Side: Player2, Amount: struggleSTABDamage, RemainingHP: 200 - struggleSTABDamage},
		Damage{Side: Player1, Amount: struggleSTABRecoil, RemainingHP: 200 - struggleSTABRecoil},
		blocked,
	})
}

// TestStruggleAgainstGhost はゴーストへノーマルが通らないことと、
// それでも最低1の反動が返ることを確かめる。
func TestStruggleAgainstGhost(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		drained(fighter(speciesRunner, 130, moveTackle)),
		frozen(fighter(speciesPhantom, 100, moveTackle)),
	)
	r := testResolver(struggleValues...)

	_, events, err := r.ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Unaffected{Side: Player1},
		Damage{Side: Player1, Amount: 1, RemainingHP: 199},
		blocked,
	})
}

// TestStruggleOverkillRecoilsFromDealtDamage は相手を倒したときの反動が、
// 残HPまでcapされた実ダメージを基準にすることを確かめる。
//
// ダメージ計算上は30でも、残り7しか削れないので反動は3になる。
func TestStruggleOverkillRecoilsFromDealtDamage(t *testing.T) {
	t.Parallel()

	defender := frozen(fighter(speciesTarget, 100, moveTackle))
	defender.CurrentHP = 7

	state := turnState(t, drained(fighter(speciesRunner, 130, moveTackle)), defender)
	r := testResolver(struggleValues...)

	next, events, err := r.ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	// 相手を倒したturnでも反動は起きる。
	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Damage{Side: Player2, Amount: 7, RemainingHP: 0},
		Damage{Side: Player1, Amount: 3, RemainingHP: 197},
		Fainted{Side: Player2, Index: 0},
	})
	if !next.NeedsReplacement(Player2) {
		t.Error("倒された側がreplacementを求めていない")
	}
}

// TestStruggleRecoilCanFaintTheUser は反動で使用者が倒れることを確かめる。
func TestStruggleRecoilCanFaintTheUser(t *testing.T) {
	t.Parallel()

	attacker := drained(fighter(speciesRunner, 130, moveTackle))
	attacker.CurrentHP = 10

	state := turnState(t, attacker, frozen(fighter(speciesTarget, 100, moveTackle)))
	r := testResolver(struggleValues...)

	next, events, err := r.ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Damage{Side: Player2, Amount: struggleDamage, RemainingHP: 200 - struggleDamage},
		Damage{Side: Player1, Amount: 10, RemainingHP: 0},
		Fainted{Side: Player1, Index: 0},
	})
	if next.Status != Ongoing {
		t.Errorf("Status = %v, want %v（控えが残っている）", next.Status, Ongoing)
	}
	if !next.NeedsReplacement(Player1) {
		t.Error("反動で倒れた側がreplacementを求めていない")
	}
}

// TestStruggleMutualKnockOutIsADraw は同じActionで双方が倒れ、
// どちらも最後の1体ならDrawになることを確かめる。
func TestStruggleMutualKnockOutIsADraw(t *testing.T) {
	t.Parallel()

	down := func() Pokemon {
		p := fighter(speciesTarget, 100, moveTackle)
		p.CurrentHP = 0
		return p
	}

	attacker := drained(fighter(speciesRunner, 130, moveTackle))
	attacker.CurrentHP = 2
	defender := frozen(fighter(speciesTarget, 100, moveTackle))
	defender.CurrentHP = 7

	state, err := NewBattleState(
		[TeamSize]Pokemon{attacker, down(), down()},
		[TeamSize]Pokemon{defender, down(), down()},
	)
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}

	next, events, err := testResolver(struggleValues...).ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: NoMoveSlot, Move: MoveStruggle},
		Damage{Side: Player2, Amount: 7, RemainingHP: 0},
		Damage{Side: Player1, Amount: 2, RemainingHP: 0},
		Fainted{Side: Player2, Index: 0},
		Fainted{Side: Player1, Index: 0},
	})
	if next.Status != Draw {
		t.Errorf("Status = %v, want %v", next.Status, Draw)
	}
}

// TestStruggleDoesNotSpendPP は技枠のPPを動かさないことを確かめる。
//
// 技枠を消費する実装になっていれば、PP 0の枠が負の値になる。
func TestStruggleDoesNotSpendPP(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		drained(fighter(speciesRunner, 130, moveTackle, moveQuick, moveShaky)),
		frozen(fighter(speciesTarget, 100, moveTackle)),
	)

	next, _, err := testResolver(struggleValues...).ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	for slot := range next.Players[Player1].ActivePokemon().Moves {
		assertPP(t, next, Player1, slot, 0)
	}
}

// TestStruggleIsRejectedWithAUsableMove は使える技が残っていればStruggleを選べないことを確かめる。
func TestStruggleIsRejectedWithAUsableMove(t *testing.T) {
	t.Parallel()

	attacker := drained(fighter(speciesRunner, 130, moveTackle, moveQuick))
	attacker.Moves[1].PP = 1 // 1つでも残っていれば使えない

	state := turnState(t, attacker, frozen(fighter(speciesTarget, 100, moveTackle)))

	if _, _, err := testResolver().ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0}); !errors.Is(err, ErrInvalidAction) {
		t.Errorf("ResolveTurn() error = %v, want %v", err, ErrInvalidAction)
	}
}

// TestMoveActionWithoutPPStaysInvalid はPP 0の技をMoveActionで選ぶのは
// これまでどおり通らないことを確かめる。
//
// Struggleを足したことで、PP切れの技がそのまま出せるようになっていないか。
func TestMoveActionWithoutPPStaysInvalid(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		drained(fighter(speciesRunner, 130, moveTackle)),
		frozen(fighter(speciesTarget, 100, moveTackle)),
	)

	if _, _, err := testResolver().ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0}); !errors.Is(err, ErrInvalidAction) {
		t.Errorf("ResolveTurn() error = %v, want %v", err, ErrInvalidAction)
	}
}

// TestStruggleIsDeterministic は同じ状態・同じAction・同じseedから
// 同じ結果になることを確かめる。
func TestStruggleIsDeterministic(t *testing.T) {
	t.Parallel()

	run := func() (BattleState, []Event) {
		state := turnState(t,
			drained(fighter(speciesRunner, 130, moveTackle)),
			frozen(fighter(speciesTarget, 100, moveTackle)),
		)
		next, events, err := testResolver(struggleValues...).ResolveTurn(state, StruggleAction{}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		return next, events
	}

	firstState, firstEvents := run()
	secondState, secondEvents := run()

	if !reflect.DeepEqual(firstState, secondState) {
		t.Error("2回目のBattleStateが一致しない")
	}
	if !reflect.DeepEqual(firstEvents, secondEvents) {
		t.Error("2回目のEvent列が一致しない")
	}
}

// TestStruggleMoveDefinition はsynthetic moveの定義を固定する。
//
// ノーマルなのでGeneration Iでは物理として扱われ、必中になる。
func TestStruggleMoveDefinition(t *testing.T) {
	t.Parallel()

	want := Move{
		ID:       MoveStruggle,
		Type:     TypeNormal,
		Power:    50,
		Accuracy: MaxAccuracy,
		Effect:   EffectRecoil,
	}
	if struggleMove != want {
		t.Errorf("struggleMove = %+v, want %+v", struggleMove, want)
	}
	if struggleMove.Type.Category() != Physical {
		t.Errorf("Struggleの分類 = %v, want %v", struggleMove.Type.Category(), Physical)
	}
}

// TestHasUsableMove はStruggleの利用条件の判定を確かめる。
func TestHasUsableMove(t *testing.T) {
	t.Parallel()

	full := fighter(speciesRunner, 100, moveTackle, moveQuick)
	if !full.HasUsableMove() {
		t.Error("PPが残っているのにHasUsableMove() = false")
	}

	empty := drained(full)
	if empty.HasUsableMove() {
		t.Error("全技PP 0なのにHasUsableMove() = true")
	}

	// 技を覚えていない枠も使える技には数えない。
	none := fighter(speciesRunner, 100)
	if none.HasUsableMove() {
		t.Error("技が無いのにHasUsableMove() = true")
	}
}

// TestStruggleRunsABattleToTheEnd は全員が技を撃ち尽くした状態でも
// 対戦が進んで決着することを確かめる。
//
// Struggleが無かったときは、この状態でresolverがinvalid actionを返し、
// 対戦を続けられなかった。
func TestStruggleRunsABattleToTheEnd(t *testing.T) {
	t.Parallel()

	team := func() [TeamSize]Pokemon {
		return [TeamSize]Pokemon{
			drained(fighter(speciesRunner, 130, moveTackle)),
			drained(fighter(speciesTarget, 100, moveTackle)),
			drained(fighter(speciesPlain, 90, moveTackle)),
		}
	}
	state, err := NewBattleState(team(), team())
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}

	r := &Resolver{Data: testData(), RNG: NewRand(7)}
	const maxTurns = 200
	turns := 0
	for state.Status == Ongoing && turns < maxTurns {
		for _, side := range sides {
			if !state.NeedsReplacement(side) {
				continue
			}
			player := state.Players[side]
			reserve := player.Reserve()
			next, _, err := r.ResolveReplacement(state, side, SwitchAction{Target: reserve[0]})
			if err != nil {
				t.Fatalf("turn %d: ResolveReplacement() error = %v", turns+1, err)
			}
			state = next
		}
		if state.Status != Ongoing {
			break
		}

		next, _, err := r.ResolveTurn(state, StruggleAction{}, StruggleAction{})
		if err != nil {
			t.Fatalf("turn %d: ResolveTurn() error = %v", turns+1, err)
		}
		state = next
		turns++
	}

	if state.Status == Ongoing {
		t.Fatalf("%d turnかけても決着しない", maxTurns)
	}
	t.Logf("%d turnで決着した: %v", turns, state.Status)
}
