package battle

import (
	"errors"
	"testing"
)

// testPokemon は検証を通る最小限のPokemonを返す。
// testごとに必要な値だけを書き換えて使う。
func testPokemon(species SpeciesID) Pokemon {
	return Pokemon{
		Species:   species,
		Level:     55,
		CurrentHP: 200,
		Stats:     Stats{HP: 200, Attack: 120, Defense: 110, Speed: 130, Special: 100},
		Moves: [MoveSlots]MoveSlot{
			{Move: "move-a", PP: 15, MaxPP: 15},
			{Move: "move-b", PP: 10, MaxPP: 10},
		},
	}
}

// testTeam は3体そろったチームを返す。
func testTeam() [TeamSize]Pokemon {
	return [TeamSize]Pokemon{
		testPokemon("first"),
		testPokemon("second"),
		testPokemon("third"),
	}
}

// testState は検証を通る開始状態を返す。
func testState(t *testing.T) BattleState {
	t.Helper()

	state, err := NewBattleState(testTeam(), testTeam())
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}
	return state
}

// 開始状態は1 turn目で、両者とも先頭のPokemonが場に出ている。
func TestNewBattleStateStartsFromFirstPokemon(t *testing.T) {
	t.Parallel()

	state := testState(t)

	if state.Turn != 1 {
		t.Errorf("Turn = %d, want 1", state.Turn)
	}
	if state.Status != Ongoing {
		t.Errorf("Status = %v, want %v", state.Status, Ongoing)
	}
	for _, side := range sides {
		player := state.Players[side]
		if player.Active != 0 {
			t.Errorf("%s Active = %d, want 0", side, player.Active)
		}
		if got := player.ActivePokemon().Species; got != "first" {
			t.Errorf("%s active species = %q, want %q", side, got, "first")
		}
	}
}

// 取り得ない値を含むチームでは状態を作らない。
func TestNewBattleStateRejectsInvalidTeam(t *testing.T) {
	t.Parallel()

	team := testTeam()
	team[1].Level = 0

	state, err := NewBattleState(team, testTeam())
	if !errors.Is(err, ErrInvalidState) {
		t.Fatalf("NewBattleState() error = %v, want %v", err, ErrInvalidState)
	}
	if state != (BattleState{}) {
		t.Errorf("NewBattleState() state = %+v, want zero value", state)
	}
}

// active slotはチームの範囲内だけを取る。
func TestValidateChecksActiveSlot(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		active int
		valid  bool
	}{
		{"first", 0, true},
		{"middle", 1, true},
		{"last", TeamSize - 1, true},
		{"negative", -1, false},
		{"past last", TeamSize, false},
		{"far past last", TeamSize + 10, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := testState(t)
			state.Players[Player2].Active = tt.active

			err := state.Validate()
			if tt.valid && err != nil {
				t.Fatalf("Validate() error = %v, want nil", err)
			}
			if !tt.valid && !errors.Is(err, ErrInvalidState) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidState)
			}
		})
	}
}

// 状態として成立しない値はすべて弾く。
func TestValidateRejectsImpossibleValues(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		modify func(*BattleState)
	}{
		{"turn before the first", func(s *BattleState) { s.Turn = 0 }},
		{"unknown battle status", func(s *BattleState) { s.Status = Status(99) }},
		{"level above the maximum", func(s *BattleState) { s.Players[Player1].Team[0].Level = MaxLevel + 1 }},
		{"max HP of zero", func(s *BattleState) { s.Players[Player1].Team[0].Stats.HP = 0 }},
		{"negative stat", func(s *BattleState) { s.Players[Player1].Team[0].Stats.Speed = -1 }},
		{"current HP above the maximum", func(s *BattleState) {
			s.Players[Player1].Team[0].CurrentHP = s.Players[Player1].Team[0].Stats.HP + 1
		}},
		{"negative current HP", func(s *BattleState) { s.Players[Player1].Team[0].CurrentHP = -1 }},
		{"unknown status", func(s *BattleState) { s.Players[Player1].Team[0].Status = MajorStatus(99) }},
		{"sleep turns without sleep", func(s *BattleState) { s.Players[Player1].Team[0].SleepTurns = 2 }},
		{"negative sleep turns", func(s *BattleState) { s.Players[Player1].Team[0].SleepTurns = -1 }},
		{"stage above the maximum", func(s *BattleState) { s.Players[Player1].Team[0].Stages.Attack = StageMax + 1 }},
		{"stage below the minimum", func(s *BattleState) { s.Players[Player1].Team[0].Stages.Evasion = StageMin - 1 }},
		{"PP above the maximum", func(s *BattleState) { s.Players[Player1].Team[0].Moves[0].PP = 99 }},
		{"negative PP", func(s *BattleState) { s.Players[Player1].Team[0].Moves[0].PP = -1 }},
		{"PP on an empty slot", func(s *BattleState) { s.Players[Player1].Team[0].Moves[3].PP = 5 }},
		{"negative continuing move counter", func(s *BattleState) {
			s.Players[Player1].Team[0].ContinuingMove.TurnsLeft = -1
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := testState(t)
			tt.modify(&state)

			if err := state.Validate(); !errors.Is(err, ErrInvalidState) {
				t.Fatalf("Validate() error = %v, want %v", err, ErrInvalidState)
			}
		})
	}
}

// ねむり状態なら残りturnを持てる。
func TestValidateAllowsSleepTurnsWhileAsleep(t *testing.T) {
	t.Parallel()

	state := testState(t)
	state.Players[Player1].Team[0].Status = Sleep
	state.Players[Player1].Team[0].SleepTurns = 3

	if err := state.Validate(); err != nil {
		t.Fatalf("Validate() error = %v, want nil", err)
	}
}

// 場に出ているPokemonはactive slotに従う。
func TestActivePokemonFollowsActiveSlot(t *testing.T) {
	t.Parallel()

	state := testState(t)
	state.Players[Player1].Active = 2

	if got := state.Players[Player1].ActivePokemon().Species; got != "third" {
		t.Errorf("active species = %q, want %q", got, "third")
	}
}

// BattleStateは値として複製でき、複製を書き換えても元へ波及しない。
func TestBattleStateCopiesIndependently(t *testing.T) {
	t.Parallel()

	state := testState(t)
	copied := state
	copied.Players[Player1].ActivePokemon().CurrentHP = 1
	copied.Turn = 2

	if got := state.Players[Player1].ActivePokemon().CurrentHP; got != 200 {
		t.Errorf("original current HP = %d, want 200", got)
	}
	if state.Turn != 1 {
		t.Errorf("original turn = %d, want 1", state.Turn)
	}
}

// reserveは控えのうち、まだ戦える個体だけになる。
// 戦闘不能の個体は交代先にならないため含めない（Glossaryのreserveの定義）。
func TestReserveExcludesActiveAndFainted(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		active  int
		fainted []int
		want    []int
	}{
		{"all healthy, first active", 0, nil, []int{1, 2}},
		{"all healthy, middle active", 1, nil, []int{0, 2}},
		{"all healthy, last active", 2, nil, []int{0, 1}},
		{"one reserve fainted", 0, []int{1}, []int{2}},
		{"every reserve fainted", 0, []int{1, 2}, []int{}},
		{"active fainted too", 0, []int{0, 1}, []int{2}},
	}

	for _, tt := range tests {
		team := testTeam()
		for _, i := range tt.fainted {
			team[i].CurrentHP = 0
		}
		player := Player{Team: team, Active: tt.active}

		got := player.Reserve()
		if len(got) != len(tt.want) {
			t.Fatalf("%s: Reserve() = %v, want %v", tt.name, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("%s: Reserve() = %v, want %v", tt.name, got, tt.want)
			}
		}
	}
}

// 現在HPが0なら戦闘不能。
func TestFaintedAtZeroHP(t *testing.T) {
	t.Parallel()

	p := testPokemon("first")
	if p.Fainted() {
		t.Error("Fainted() = true at full HP, want false")
	}

	p.CurrentHP = 0
	if !p.Fainted() {
		t.Error("Fainted() = false at 0 HP, want true")
	}
}

// Opponentは常に反対側を返す。
func TestSideOpponent(t *testing.T) {
	t.Parallel()

	if got := Player1.Opponent(); got != Player2 {
		t.Errorf("Player1.Opponent() = %v, want %v", got, Player2)
	}
	if got := Player2.Opponent(); got != Player1 {
		t.Errorf("Player2.Opponent() = %v, want %v", got, Player1)
	}
}

// Sideはそのまま Players の添字として使える。
func TestSideIndexesPlayers(t *testing.T) {
	t.Parallel()

	state := testState(t)
	state.Players[Player2].Team[0].Species = "opponent"

	if got := state.Players[Player2].ActivePokemon().Species; got != "opponent" {
		t.Errorf("player2 active species = %q, want %q", got, "opponent")
	}
	if got := state.Players[Player1].ActivePokemon().Species; got != "first" {
		t.Errorf("player1 active species = %q, want %q", got, "first")
	}
}

// 技枠の使用可否は残りPPと技の有無で決まる。
func TestMoveSlotUsable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		slot MoveSlot
		want bool
	}{
		{"has PP", MoveSlot{Move: "move-a", PP: 1, MaxPP: 10}, true},
		{"out of PP", MoveSlot{Move: "move-a", PP: 0, MaxPP: 10}, false},
		{"empty slot", MoveSlot{}, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := tt.slot.Usable(); got != tt.want {
				t.Errorf("Usable() = %v, want %v", got, tt.want)
			}
		})
	}
}
