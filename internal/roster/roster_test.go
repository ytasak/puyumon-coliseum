package roster

import (
	"errors"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// 6キャラがそろい、IDとRankが重複しない。
func TestAllCharacters(t *testing.T) {
	t.Parallel()

	all := All()
	if len(all) != 6 {
		t.Fatalf("All() returned %d characters, want 6", len(all))
	}

	ids := make(map[battle.SpeciesID]bool, len(all))
	ranks := make(map[int]bool, len(all))
	for _, c := range all {
		if ids[c.ID] {
			t.Errorf("duplicate character ID %q", c.ID)
		}
		ids[c.ID] = true

		if c.Rank < 1 || c.Rank > len(all) {
			t.Errorf("%q: rank %d is out of range", c.ID, c.Rank)
		}
		if ranks[c.Rank] {
			t.Errorf("duplicate rank %d", c.Rank)
		}
		ranks[c.Rank] = true

		if c.Emoji == "" {
			t.Errorf("%q has no emoji", c.ID)
		}
		// 公開表示名はまだ決めていない。決まったらこのtestを書き換える。
		if c.DisplayName != "" {
			t.Errorf("%q has a display name %q, but names are not decided yet", c.ID, c.DisplayName)
		}
	}
}

// 4技がそろい、すべて定義のある技を指している。
func TestCharacterMovesExist(t *testing.T) {
	t.Parallel()

	data := Data()
	for _, c := range All() {
		seen := make(map[battle.MoveID]bool, battle.MoveSlots)
		for slot, id := range c.Moves {
			if id == "" {
				t.Errorf("%q move slot %d is empty", c.ID, slot)
				continue
			}
			if seen[id] {
				t.Errorf("%q has %q twice", c.ID, id)
			}
			seen[id] = true

			if _, err := data.LookupMove(id); err != nil {
				t.Errorf("%q refers to %q: %v", c.ID, id, err)
			}
		}
	}
}

// 実数値がYTA-20で確定した表と一致する。
//
// 期待値はGeneration Iの式から別に求めたもので、種族値か計算が変われば落ちる。
func TestGoldenStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		id    battle.SpeciesID
		level int
		want  battle.Stats
	}{
		{SpeciesBull, 50, battle.Stats{HP: 181, Attack: 151, Defense: 146, Speed: 161, Special: 121}},
		{SpeciesStar, 50, battle.Stats{HP: 166, Attack: 126, Defense: 136, Speed: 166, Special: 151}},
		{SpeciesJolt, 50, battle.Stats{HP: 171, Attack: 116, Defense: 111, Speed: 181, Special: 161}},
		{SpeciesPalm, 50, battle.Stats{HP: 201, Attack: 146, Defense: 136, Speed: 106, Special: 176}},
		{SpeciesWhale, 50, battle.Stats{HP: 236, Attack: 136, Defense: 131, Speed: 111, Special: 146}},
		{SpeciesCharm, 50, battle.Stats{HP: 171, Attack: 101, Defense: 86, Speed: 146, Special: 146}},

		{SpeciesBull, 55, battle.Stats{HP: 198, Attack: 166, Defense: 160, Speed: 177, Special: 133}},
		{SpeciesStar, 55, battle.Stats{HP: 182, Attack: 138, Defense: 149, Speed: 182, Special: 166}},
		{SpeciesJolt, 55, battle.Stats{HP: 187, Attack: 127, Defense: 122, Speed: 199, Special: 177}},
		{SpeciesPalm, 55, battle.Stats{HP: 220, Attack: 160, Defense: 149, Speed: 116, Special: 193}},
		{SpeciesWhale, 55, battle.Stats{HP: 259, Attack: 149, Defense: 144, Speed: 122, Special: 160}},
		{SpeciesCharm, 55, battle.Stats{HP: 187, Attack: 111, Defense: 94, Speed: 160, Special: 160}},
	}

	for _, tt := range tests {
		character, ok := Lookup(tt.id)
		if !ok {
			t.Fatalf("Lookup(%q) failed", tt.id)
		}
		if got := battle.LevelStats(character.BaseStats, tt.level); got != tt.want {
			t.Errorf("%q at level %d = %+v, want %+v", tt.id, tt.level, got, tt.want)
		}
	}
}

// 3体の組み合わせは20通りで、重複しない。
func TestCombinations(t *testing.T) {
	t.Parallel()

	combinations := Combinations()
	if len(combinations) != 20 {
		t.Fatalf("Combinations() returned %d, want 20", len(combinations))
	}

	seen := make(map[[battle.TeamSize]battle.SpeciesID]bool, len(combinations))
	for _, combination := range combinations {
		if seen[combination] {
			t.Errorf("duplicate combination %v", combination)
		}
		seen[combination] = true

		members := make(map[battle.SpeciesID]bool, battle.TeamSize)
		for _, id := range combination {
			if members[id] {
				t.Errorf("%v contains %q twice", combination, id)
			}
			members[id] = true
		}
	}
}

// どの組み合わせでもLevelの合計は155で、50〜55に収まる。
func TestAssignLevelsAlwaysTotals155(t *testing.T) {
	t.Parallel()

	for _, combination := range Combinations() {
		levels, err := AssignLevels(combination)
		if err != nil {
			t.Fatalf("AssignLevels(%v) error = %v", combination, err)
		}

		total := 0
		for i, level := range levels {
			if level < 50 || level > 55 {
				t.Errorf("%v: %q got level %d, want it within 50-55", combination, combination[i], level)
			}
			total += level
		}
		if total != LevelTotal {
			t.Errorf("%v: level total = %d, want %d", combination, total, LevelTotal)
		}
	}
}

// 強い順に50 / 50 / 55を割り当てる。並びは引数のまま返す。
func TestAssignLevelsFollowsRank(t *testing.T) {
	t.Parallel()

	// 引数の並びを強さ順とは変えて、対応が崩れないことを見る。
	team := [battle.TeamSize]battle.SpeciesID{SpeciesCharm, SpeciesBull, SpeciesPalm}
	levels, err := AssignLevels(team)
	if err != nil {
		t.Fatalf("AssignLevels() error = %v", err)
	}

	want := [battle.TeamSize]int{55, 50, 50} // 💋が最下位なので55、🐂と🌴はそれぞれ50
	if levels != want {
		t.Errorf("levels = %v, want %v", levels, want)
	}
}

// 取れない配布はerrorにする。
func TestAssignLevelsRejectsInvalidTeams(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		team [battle.TeamSize]battle.SpeciesID
	}{
		{"unknown character", [battle.TeamSize]battle.SpeciesID{SpeciesBull, "ghost", SpeciesPalm}},
		{"same character twice", [battle.TeamSize]battle.SpeciesID{SpeciesBull, SpeciesBull, SpeciesPalm}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if _, err := AssignLevels(tt.team); err == nil {
				t.Error("AssignLevels() error = nil, want an error")
			}
		})
	}
}

// 配布された3体から、そのまま対戦を始められるPokemonができる。
func TestNewTeam(t *testing.T) {
	t.Parallel()

	for _, combination := range Combinations() {
		team, err := NewTeam(combination)
		if err != nil {
			t.Fatalf("NewTeam(%v) error = %v", combination, err)
		}

		levels, err := AssignLevels(combination)
		if err != nil {
			t.Fatalf("AssignLevels(%v) error = %v", combination, err)
		}

		for i, p := range team {
			character, _ := Lookup(combination[i])

			if p.Species != combination[i] {
				t.Errorf("%v: species = %q, want %q", combination, p.Species, combination[i])
			}
			if p.Level != levels[i] {
				t.Errorf("%v: %q level = %d, want %d", combination, p.Species, p.Level, levels[i])
			}
			if want := battle.LevelStats(character.BaseStats, levels[i]); p.Stats != want {
				t.Errorf("%v: %q stats = %+v, want %+v", combination, p.Species, p.Stats, want)
			}
			if p.CurrentHP != p.Stats.HP {
				t.Errorf("%v: %q starts at %d HP, want full %d", combination, p.Species, p.CurrentHP, p.Stats.HP)
			}
			for slot, move := range p.Moves {
				if move.Move != character.Moves[slot] {
					t.Errorf("%v: %q slot %d = %q, want %q",
						combination, p.Species, slot, move.Move, character.Moves[slot])
				}
				if move.PP != move.MaxPP || move.MaxPP == 0 {
					t.Errorf("%v: %q slot %d PP = %d/%d, want full", combination, p.Species, slot, move.PP, move.MaxPP)
				}
			}
		}
	}
}

// 定義にないキャラクターを配ろうとしたらerrorにする。
func TestNewTeamRejectsUnknownCharacter(t *testing.T) {
	t.Parallel()

	team := [battle.TeamSize]battle.SpeciesID{SpeciesBull, SpeciesStar, "ghost"}
	if _, err := NewTeam(team); !errors.Is(err, ErrUnknownCharacter) {
		t.Errorf("NewTeam() error = %v, want %v", err, ErrUnknownCharacter)
	}
}

// Battle Engineへ渡すデータが6キャラと全技をそろえている。
func TestDataCoversTheRoster(t *testing.T) {
	t.Parallel()

	data := Data()
	if len(data.Species) != 6 {
		t.Errorf("Data() has %d species, want 6", len(data.Species))
	}
	if len(data.Moves) != len(moves) {
		t.Errorf("Data() has %d moves, want %d", len(data.Moves), len(moves))
	}

	for _, c := range All() {
		species, err := data.LookupSpecies(c.ID)
		if err != nil {
			t.Fatalf("LookupSpecies(%q) error = %v", c.ID, err)
		}
		if species.Typing != c.Typing {
			t.Errorf("%q typing = %+v, want %+v", c.ID, species.Typing, c.Typing)
		}
		// 急所率はbase Speedで決まるので、種族値のSpeedをそのまま渡す。
		if species.BaseSpeed != c.BaseStats.Speed {
			t.Errorf("%q base speed = %d, want %d", c.ID, species.BaseSpeed, c.BaseStats.Speed)
		}
	}
}

// 配布したデータでturnを解決できる。engineとデータがかみ合っていることの確認。
func TestRosterDataDrivesABattle(t *testing.T) {
	t.Parallel()

	team1, err := NewTeam([battle.TeamSize]battle.SpeciesID{SpeciesBull, SpeciesStar, SpeciesJolt})
	if err != nil {
		t.Fatalf("NewTeam() error = %v", err)
	}
	team2, err := NewTeam([battle.TeamSize]battle.SpeciesID{SpeciesPalm, SpeciesWhale, SpeciesCharm})
	if err != nil {
		t.Fatalf("NewTeam() error = %v", err)
	}

	state, err := battle.NewBattleState(team1, team2)
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}

	resolver := &battle.Resolver{Data: Data(), RNG: battle.NewRand(20260918)}
	next, events, err := resolver.ResolveTurn(state, battle.MoveAction{Slot: 0}, battle.MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}
	if len(events) == 0 {
		t.Error("no events were produced")
	}
	if next.Turn != 2 {
		t.Errorf("Turn = %d, want 2", next.Turn)
	}

	// 🐂 の方が速いので先に動く。
	used, ok := events[0].(battle.MoveUsed)
	if !ok {
		t.Fatalf("first event = %T, want MoveUsed", events[0])
	}
	if used.Side != battle.Player1 {
		t.Errorf("first to move = %v, want %v", used.Side, battle.Player1)
	}
}
