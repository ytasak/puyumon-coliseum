package battle

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

// scriptRNG は決められた値を順に返すRNG。
//
// 値はIntNの引数より大きければn-1へ丸める。255と書けば「その判定での最大値」、
// 0なら最小値になるので、pipelineのどこで何を引いたかがtestから読める。
// 値が尽きたら最後の値を返し続ける。
type scriptRNG struct {
	values []int
	calls  int
}

func (s *scriptRNG) IntN(n int) int {
	if n < 1 {
		panic("battle: IntN requires n >= 1")
	}

	value := 0
	if len(s.values) > 0 {
		if s.calls < len(s.values) {
			value = s.values[s.calls]
		} else {
			value = s.values[len(s.values)-1]
		}
	}
	s.calls++

	switch {
	case value < 0:
		return 0
	case value >= n:
		return n - 1
	default:
		return value
	}
}

// test用の技とキャラクター。実データは別Issueで定義する。
const (
	moveTackle = MoveID("tackle") // ノーマル・威力85・必中
	moveQuick  = MoveID("quick")  // ノーマル・威力40・必中・priority +1
	moveShaky  = MoveID("shaky")  // ノーマル・威力85・命中75%

	speciesRunner  = SpeciesID("runner")  // みず
	speciesTarget  = SpeciesID("target")  // みず
	speciesPhantom = SpeciesID("phantom") // ゴースト
)

func testData() Data {
	return Data{
		Moves: map[MoveID]Move{
			moveTackle: {ID: moveTackle, Type: TypeNormal, Power: 85, Accuracy: MaxAccuracy, MaxPP: 15},
			moveQuick:  {ID: moveQuick, Type: TypeNormal, Power: 40, Accuracy: MaxAccuracy, MaxPP: 30, Priority: 1},
			moveShaky:  {ID: moveShaky, Type: TypeNormal, Power: 85, Accuracy: AccuracyPercent(75), MaxPP: 10},
		},
		Species: map[SpeciesID]Species{
			speciesRunner:  {ID: speciesRunner, Typing: SingleType(TypeWater), BaseSpeed: 110},
			speciesTarget:  {ID: speciesTarget, Typing: SingleType(TypeWater), BaseSpeed: 100},
			speciesPhantom: {ID: speciesPhantom, Typing: SingleType(TypeGhost), BaseSpeed: 110},
		},
	}
}

// fighter は検証を通る個体を返す。能力値はYTA-16のgolden vectorと同じ組み合わせで、
// ノーマル技をみずタイプへ当てると威力85でダメージ50になる。
func fighter(species SpeciesID, speed int, moves ...MoveID) Pokemon {
	p := Pokemon{
		Species:   species,
		Level:     55,
		CurrentHP: 200,
		Stats:     Stats{HP: 200, Attack: 180, Defense: 150, Speed: speed, Special: 120},
	}
	for i, id := range moves {
		p.Moves[i] = MoveSlot{Move: id, PP: 15, MaxPP: 15}
	}
	return p
}

// testResolver は引く乱数を並べた順に返すresolverを返す。
func testResolver(values ...int) *Resolver {
	return &Resolver{Data: testData(), RNG: &scriptRNG{values: values}}
}

// turnState は両者の先頭を入れ替えた開始状態を返す。控えは同じ個体で埋める。
func turnState(t *testing.T, first1, first2 Pokemon) BattleState {
	t.Helper()

	bench := func() Pokemon { return fighter(speciesTarget, 100, moveTackle) }
	state, err := NewBattleState(
		[TeamSize]Pokemon{first1, bench(), bench()},
		[TeamSize]Pokemon{first2, bench(), bench()},
	)
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}
	return state
}

func formatEvents(events []Event) string {
	if len(events) == 0 {
		return "  (none)"
	}
	var b strings.Builder
	for _, e := range events {
		fmt.Fprintf(&b, "  %T%+v\n", e, e)
	}
	return b.String()
}

func assertEvents(t *testing.T, got, want []Event) {
	t.Helper()

	if !reflect.DeepEqual(got, want) {
		t.Errorf("events =\n%swant\n%s", formatEvents(got), formatEvents(want))
	}
}

func assertPP(t *testing.T, state BattleState, side Side, slot, want int) {
	t.Helper()

	if got := state.Players[side].ActivePokemon().Moves[slot].PP; got != want {
		t.Errorf("%s move slot %d PP = %d, want %d", side, slot, got, want)
	}
}

// 速い側から順に技が出て、両者のHPが減る。
func TestResolveTurnMoveVsMove(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		fighter(speciesRunner, 130, moveTackle),
		fighter(speciesTarget, 100, moveTackle),
	)
	r := testResolver(
		0, 255, 255, // Player1: 命中 / 急所なし / ダメージ乱数は最大
		0, 255, 255, // Player2: 同じ
	)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveTackle},
		Damage{Side: Player2, Amount: 50, RemainingHP: 150},
		MoveUsed{Side: Player2, Slot: 0, Move: moveTackle},
		Damage{Side: Player1, Amount: 50, RemainingHP: 150},
	})

	if next.Turn != 2 {
		t.Errorf("Turn = %d, want 2", next.Turn)
	}
	if next.Status != Ongoing {
		t.Errorf("Status = %v, want %v", next.Status, Ongoing)
	}
	assertPP(t, next, Player1, 0, 14)
	assertPP(t, next, Player2, 0, 14)
}

// 同速のときは乱数で先攻が決まる。境界は128。
func TestResolveTurnSpeedTie(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		roll  int
		first Side
	}{
		{"below the threshold", speedTieThreshold - 1, Player1},
		{"at the threshold", speedTieThreshold, Player2},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := turnState(t,
				fighter(speciesRunner, 120, moveTackle),
				fighter(speciesTarget, 120, moveTackle),
			)
			r := testResolver(tt.roll, 0, 255, 255, 0, 255, 255)

			_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
			if err != nil {
				t.Fatalf("ResolveTurn() error = %v", err)
			}

			used, ok := events[0].(MoveUsed)
			if !ok {
				t.Fatalf("first event = %T, want MoveUsed", events[0])
			}
			if used.Side != tt.first {
				t.Errorf("first to move = %v, want %v", used.Side, tt.first)
			}
		})
	}
}

// priorityはSpeedより先に比べる。遅い側でも先制技なら先に動く。
func TestResolveTurnPriorityBeatsSpeed(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		fighter(speciesRunner, 100, moveQuick),  // 遅いが priority +1
		fighter(speciesTarget, 130, moveTackle), // 速いが priority 0
	)
	r := testResolver(0, 255, 255, 0, 255, 255)

	_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveQuick},
		Damage{Side: Player2, Amount: 25, RemainingHP: 175},
		MoveUsed{Side: Player2, Slot: 0, Move: moveTackle},
		Damage{Side: Player1, Amount: 50, RemainingHP: 150},
	})
}

// まひはeffective Speedを4分の1にするので、行動順が入れ替わる。
func TestResolveTurnParalysisChangesOrder(t *testing.T) {
	t.Parallel()

	fast := fighter(speciesRunner, 130, moveTackle)
	fast.Status = Paralysis // 130 -> 32

	state := turnState(t, fast, fighter(speciesTarget, 100, moveTackle))
	r := testResolver(
		0, 255, 255, // Player2（まひしていない側）が先
		255,         // Player1 のまひ判定は行動できる値
		0, 255, 255, // Player1
	)

	_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	used, ok := events[0].(MoveUsed)
	if !ok {
		t.Fatalf("first event = %T, want MoveUsed", events[0])
	}
	if used.Side != Player2 {
		t.Errorf("first to move = %v, want %v", used.Side, Player2)
	}
}

// 交代は技より先に解決し、出てきた個体が技を受ける。能力変化は交代で戻る。
func TestResolveTurnSwitchResolvesBeforeMoves(t *testing.T) {
	t.Parallel()

	boosted := fighter(speciesRunner, 130, moveTackle)
	boosted.Stages = StatStages{Attack: 2, Speed: 1}

	state := turnState(t, boosted, fighter(speciesTarget, 100, moveTackle))
	r := testResolver(0, 255, 255) // Player2 の命中 / 急所 / ダメージのみ

	next, events, err := r.ResolveTurn(state, SwitchAction{Target: 1}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		Switched{Side: Player1, From: 0, To: 1},
		MoveUsed{Side: Player2, Slot: 0, Move: moveTackle},
		Damage{Side: Player1, Amount: 50, RemainingHP: 150},
	})

	player1 := next.Players[Player1]
	if player1.Active != 1 {
		t.Errorf("active = %d, want 1", player1.Active)
	}
	if got := player1.Team[0].Stages; got != (StatStages{}) {
		t.Errorf("stages of the withdrawn pokemon = %+v, want zero", got)
	}
	if got := player1.Team[0].CurrentHP; got != 200 {
		t.Errorf("withdrawn pokemon HP = %d, want 200 (it left the field)", got)
	}
}

// 技が外れてもPPは減る。実機もPPを命中判定より前に消費する。
func TestResolveTurnMissStillConsumesPP(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		fighter(speciesRunner, 130, moveShaky),
		fighter(speciesTarget, 100, moveTackle),
	)
	r := testResolver(255) // Player1 の命中判定を外す

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		Switched{Side: Player2, From: 0, To: 1},
		MoveUsed{Side: Player1, Slot: 0, Move: moveShaky},
		MoveMissed{Side: Player1},
	})
	assertPP(t, next, Player1, 0, 14)
	if got := next.Players[Player2].ActivePokemon().CurrentHP; got != 200 {
		t.Errorf("defender HP = %d, want 200", got)
	}
}

// 状態異常で動けなかったturnはPPを消費しない。
func TestResolveTurnBlockedTurnKeepsPP(t *testing.T) {
	t.Parallel()

	paralyzed := fighter(speciesRunner, 130, moveTackle)
	paralyzed.Status = Paralysis

	state := turnState(t, paralyzed, fighter(speciesTarget, 100, moveTackle))
	r := testResolver(FullParalysisThreshold - 1) // 行動不能になる値

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		Switched{Side: Player2, From: 0, To: 1},
		ActionBlocked{Side: Player1, Reason: BlockedByParalysis},
	})
	assertPP(t, next, Player1, 0, 15)
}

// 目を覚ましたturnは行動できない。
func TestResolveTurnWakeUpBlocksTheTurn(t *testing.T) {
	t.Parallel()

	sleeping := fighter(speciesRunner, 130, moveTackle)
	sleeping.Status = Sleep
	sleeping.SleepTurns = 1

	state := turnState(t, sleeping, fighter(speciesTarget, 100, moveTackle))
	r := testResolver()

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		Switched{Side: Player2, From: 0, To: 1},
		StatusRecovered{Side: Player1, Status: Sleep},
		ActionBlocked{Side: Player1, Reason: BlockedByWakeUp},
	})
	assertPP(t, next, Player1, 0, 15)
	if got := next.Players[Player1].ActivePokemon().Status; got != NoStatus {
		t.Errorf("status = %v, want %v", got, NoStatus)
	}
}

// タイプ相性で通らない技は、当たっていても効果がない。PPは減る。
func TestResolveTurnImmuneDefender(t *testing.T) {
	t.Parallel()

	// ゴーストへノーマル技は通らない。逆にゴースト側のノーマル技はみずへ通る。
	state := turnState(t,
		fighter(speciesRunner, 130, moveTackle),
		fighter(speciesPhantom, 100, moveTackle),
	)
	r := testResolver(
		0, 255, 255, // Player1: 命中するが相性で通らない
		0, 255, 255, // Player2
	)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveTackle},
		Unaffected{Side: Player1},
		MoveUsed{Side: Player2, Slot: 0, Move: moveTackle},
		Damage{Side: Player1, Amount: 50, RemainingHP: 150},
	})
	assertPP(t, next, Player1, 0, 14)
	if got := next.Players[Player2].ActivePokemon().CurrentHP; got != 200 {
		t.Errorf("immune defender HP = %d, want 200", got)
	}
}

// 継続ダメージは各手番の直後に入る。turnの終わりへ一括しない。
func TestResolveTurnResidualDamageFollowsEachTurn(t *testing.T) {
	t.Parallel()

	poisoned := fighter(speciesRunner, 130, moveTackle)
	poisoned.Status = Poison

	state := turnState(t, poisoned, fighter(speciesTarget, 100, moveTackle))
	r := testResolver(
		0, 255, 255, // Player1
		0, 255, 255, // Player2
	)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveTackle},
		Damage{Side: Player2, Amount: 50, RemainingHP: 150},
		Damage{Side: Player1, Amount: 12, RemainingHP: 188}, // どくの継続ダメージ
		MoveUsed{Side: Player2, Slot: 0, Move: moveTackle},
		Damage{Side: Player1, Amount: 50, RemainingHP: 138},
	})

	if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 138 {
		t.Errorf("player1 HP = %d, want 138", got)
	}
}

// 自分の技で相手を倒したturnは、自分の継続ダメージが発生しない。
func TestResolveTurnKnockoutSkipsOwnResidualDamage(t *testing.T) {
	t.Parallel()

	poisoned := fighter(speciesRunner, 130, moveTackle)
	poisoned.Status = Poison
	weak := fighter(speciesTarget, 100, moveTackle)
	weak.CurrentHP = 10

	state := turnState(t, poisoned, weak)
	r := testResolver(0, 255, 255)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveTackle},
		Damage{Side: Player2, Amount: 10, RemainingHP: 0},
		Fainted{Side: Player2, Index: 0},
	})

	if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 200 {
		t.Errorf("player1 HP = %d, want 200 (residual damage is skipped)", got)
	}
	if !next.NeedsReplacement(Player2) {
		t.Error("NeedsReplacement(player2) = false, want true")
	}
	if next.NeedsReplacement(Player1) {
		t.Error("NeedsReplacement(player1) = true, want false")
	}
	if next.Status != Ongoing {
		t.Errorf("Status = %v, want %v", next.Status, Ongoing)
	}
}

// 戦闘不能のあとは交代だけを受け付ける。
func TestResolveReplacementAfterFaint(t *testing.T) {
	t.Parallel()

	weak := fighter(speciesTarget, 100, moveTackle)
	weak.CurrentHP = 10
	state := turnState(t, fighter(speciesRunner, 130, moveTackle), weak)

	r := testResolver(0, 255, 255)
	state, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	// replacementが要る状態では通常のturnを進められない。
	if _, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0}); !errors.Is(err, ErrInvalidAction) {
		t.Fatalf("ResolveTurn() error = %v, want %v", err, ErrInvalidAction)
	}

	// 交代なら受け付ける。能力変化は戻り、turnは進まない。
	before := state
	next, events, err := r.ResolveReplacement(state, Player2, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveReplacement() error = %v", err)
	}
	assertEvents(t, events, []Event{Switched{Side: Player2, From: 0, To: 1}})

	if next.Players[Player2].Active != 1 {
		t.Errorf("active = %d, want 1", next.Players[Player2].Active)
	}
	if next.Turn != before.Turn {
		t.Errorf("Turn = %d, want %d (replacement does not advance the turn)", next.Turn, before.Turn)
	}
	if next.NeedsReplacement(Player2) {
		t.Error("NeedsReplacement(player2) = true, want false")
	}

	// 交代が済めば通常のturnへ戻れる。
	if _, _, err := r.ResolveTurn(next, MoveAction{Slot: 0}, MoveAction{Slot: 0}); err != nil {
		t.Errorf("ResolveTurn() after replacement error = %v", err)
	}
}

// replacementの要求そのものが不正な場合。
func TestResolveReplacementRejectsInvalidRequests(t *testing.T) {
	t.Parallel()

	fainted := fighter(speciesTarget, 100, moveTackle)
	fainted.CurrentHP = 0

	state := turnState(t, fighter(speciesRunner, 130, moveTackle), fighter(speciesTarget, 100, moveTackle))
	state.Players[Player2].Team[0].CurrentHP = 0 // activeが戦闘不能
	state.Players[Player2].Team[1].CurrentHP = 0 // 控えの1体も戦闘不能

	r := testResolver()

	tests := []struct {
		name   string
		side   Side
		action SwitchAction
	}{
		{"not needed", Player1, SwitchAction{Target: 1}},
		{"target has fainted", Player2, SwitchAction{Target: 1}},
		{"target out of range", Player2, SwitchAction{Target: TeamSize}},
		{"unknown side", Side(9), SwitchAction{Target: 1}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			next, events, err := r.ResolveReplacement(state, tt.side, tt.action)
			if err == nil {
				t.Fatal("ResolveReplacement() error = nil, want an error")
			}
			if events != nil {
				t.Errorf("events = %v, want none", events)
			}
			if !reflect.DeepEqual(next, state) {
				t.Error("state changed on error, want it untouched")
			}
		})
	}
}

// 取れないActionはerrorにし、状態を進めない。
func TestResolveTurnRejectsInvalidActions(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		prepare func(*BattleState)
		p1      Action
		p2      Action
		wantErr error
	}{
		{
			name:    "no action",
			p1:      nil,
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
		{
			name:    "move slot out of range",
			p1:      MoveAction{Slot: MoveSlots},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidState,
		},
		{
			name:    "empty move slot",
			p1:      MoveAction{Slot: 1},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
		{
			name: "move without PP",
			prepare: func(s *BattleState) {
				s.Players[Player1].Team[0].Moves[0].PP = 0
			},
			p1:      MoveAction{Slot: 0},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
		{
			name:    "switch to the active pokemon",
			p1:      SwitchAction{Target: 0},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
		{
			name: "switch to a fainted pokemon",
			prepare: func(s *BattleState) {
				s.Players[Player1].Team[1].CurrentHP = 0
			},
			p1:      SwitchAction{Target: 1},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
		{
			name:    "switch out of range",
			p1:      SwitchAction{Target: TeamSize},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidState,
		},
		{
			name: "unknown move",
			prepare: func(s *BattleState) {
				s.Players[Player1].Team[0].Moves[0].Move = "unknown"
			},
			p1:      MoveAction{Slot: 0},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrMissingData,
		},
		{
			name: "unknown species",
			prepare: func(s *BattleState) {
				s.Players[Player2].Team[0].Species = "unknown"
			},
			p1:      MoveAction{Slot: 0},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrMissingData,
		},
		{
			name: "battle already finished",
			prepare: func(s *BattleState) {
				s.Status = Player1Won
			},
			p1:      MoveAction{Slot: 0},
			p2:      MoveAction{Slot: 0},
			wantErr: ErrInvalidAction,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := turnState(t,
				fighter(speciesRunner, 130, moveTackle),
				fighter(speciesTarget, 100, moveTackle),
			)
			if tt.prepare != nil {
				tt.prepare(&state)
			}
			before := state

			r := testResolver(0, 255, 255, 0, 255, 255)
			next, events, err := r.ResolveTurn(state, tt.p1, tt.p2)
			if !errors.Is(err, tt.wantErr) {
				t.Fatalf("ResolveTurn() error = %v, want %v", err, tt.wantErr)
			}
			if events != nil {
				t.Errorf("events = %v, want none", events)
			}
			if !reflect.DeepEqual(next, before) {
				t.Error("state changed on error, want it untouched")
			}
		})
	}
}

// 引数のstateは変更しない。呼び出し側は前後の状態を並べて持てる。
func TestResolveTurnDoesNotMutateItsInput(t *testing.T) {
	t.Parallel()

	state := turnState(t,
		fighter(speciesRunner, 130, moveTackle),
		fighter(speciesTarget, 100, moveTackle),
	)
	before := state

	r := testResolver(0, 255, 255, 0, 255, 255)
	if _, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0}); err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	if !reflect.DeepEqual(state, before) {
		t.Error("the input state was modified, want it untouched")
	}
}

// 同じ状態・同じAction・同じseedからは同じ結果になる。
func TestResolveTurnIsReproducible(t *testing.T) {
	t.Parallel()

	run := func() (BattleState, []Event) {
		state := turnState(t,
			fighter(speciesRunner, 120, moveTackle),
			fighter(speciesTarget, 120, moveShaky),
		)
		r := &Resolver{Data: testData(), RNG: NewRand(20260918)}

		var events []Event
		for turn := 0; turn < 5; turn++ {
			next, turnEvents, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
			if err != nil {
				t.Fatalf("turn %d: ResolveTurn() error = %v", turn, err)
			}
			state = next
			events = append(events, turnEvents...)
			if state.Status != Ongoing || state.NeedsReplacement(Player1) || state.NeedsReplacement(Player2) {
				break
			}
		}
		return state, events
	}

	firstState, firstEvents := run()
	secondState, secondEvents := run()

	if !reflect.DeepEqual(firstState, secondState) {
		t.Error("final state differs between runs with the same seed")
	}
	if !reflect.DeepEqual(firstEvents, secondEvents) {
		t.Errorf("events differ between runs with the same seed:\n%s\n%s",
			formatEvents(firstEvents), formatEvents(secondEvents))
	}
	if len(firstEvents) == 0 {
		t.Error("no events were produced")
	}
}

// 最後の1体が倒れたら勝敗が決まる。
func TestResolveTurnDecidesTheWinner(t *testing.T) {
	t.Parallel()

	last := fighter(speciesTarget, 100, moveTackle)
	last.CurrentHP = 10

	state := turnState(t, fighter(speciesRunner, 130, moveTackle), last)
	state.Players[Player2].Team[1].CurrentHP = 0
	state.Players[Player2].Team[2].CurrentHP = 0

	r := testResolver(0, 255, 255)
	next, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	if next.Status != Player1Won {
		t.Errorf("Status = %v, want %v", next.Status, Player1Won)
	}
	if next.NeedsReplacement(Player2) {
		t.Error("NeedsReplacement(player2) = true, want false once the battle is over")
	}
}

// 両者の最後の1体が同じturnに倒れたら引き分け。
func TestResolveTurnDraw(t *testing.T) {
	t.Parallel()

	dying := func() Pokemon {
		p := fighter(speciesTarget, 100, moveShaky)
		p.Status = Poison
		p.CurrentHP = 12 // 継続ダメージでちょうど倒れる
		return p
	}

	state := turnState(t, dying(), dying())
	state.Players[Player1].Team[0].Stats.Speed = 130 // 先攻を固定する
	for _, side := range sides {
		state.Players[side].Team[1].CurrentHP = 0
		state.Players[side].Team[2].CurrentHP = 0
	}

	r := testResolver(255) // 両者とも技を外し、継続ダメージだけが入る
	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveShaky},
		MoveMissed{Side: Player1},
		Damage{Side: Player1, Amount: 12, RemainingHP: 0},
		Fainted{Side: Player1, Index: 0},
		MoveUsed{Side: Player2, Slot: 0, Move: moveShaky},
		MoveMissed{Side: Player2},
		Damage{Side: Player2, Amount: 12, RemainingHP: 0},
		Fainted{Side: Player2, Index: 0},
	})
	if next.Status != Draw {
		t.Errorf("Status = %v, want %v", next.Status, Draw)
	}
}
