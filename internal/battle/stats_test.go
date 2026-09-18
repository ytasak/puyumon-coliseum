package battle

import "testing"

// 実数値の計算。Generation Iの式どおりになる。
//
// base Speed 110 のLv100が318になるのは既知の最大値で、式の検算に使える。
func TestLevelStats(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		base  Stats
		level int
		want  Stats
	}{
		{
			// 検算用。base Speed 110 のLv100は318。
			name:  "level 100",
			base:  Stats{HP: 75, Attack: 100, Defense: 95, Speed: 110, Special: 70},
			level: 100,
			want:  Stats{HP: 353, Attack: 298, Defense: 288, Speed: 318, Special: 238},
		},
		{
			name:  "level 50",
			base:  Stats{HP: 75, Attack: 100, Defense: 95, Speed: 110, Special: 70},
			level: 50,
			want:  Stats{HP: 181, Attack: 151, Defense: 146, Speed: 161, Special: 121},
		},
		{
			name:  "level 55",
			base:  Stats{HP: 130, Attack: 85, Defense: 80, Speed: 60, Special: 95},
			level: 55,
			want:  Stats{HP: 259, Attack: 149, Defense: 144, Speed: 122, Special: 160},
		},
		{
			name:  "level 1",
			base:  Stats{HP: 75, Attack: 100, Defense: 95, Speed: 110, Special: 70},
			level: 1,
			want:  Stats{HP: 13, Attack: 7, Defense: 7, Speed: 8, Special: 7},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := LevelStats(tt.base, tt.level); got != tt.want {
				t.Errorf("LevelStats() = %+v, want %+v", got, tt.want)
			}
		})
	}
}

// Levelが上がれば実数値も上がる。
func TestLevelStatsRiseWithLevel(t *testing.T) {
	t.Parallel()

	base := Stats{HP: 95, Attack: 95, Defense: 85, Speed: 55, Special: 125}

	previous := LevelStats(base, 50)
	for level := 51; level <= 55; level++ {
		current := LevelStats(base, level)
		if current.HP <= previous.HP || current.Special <= previous.Special {
			t.Fatalf("level %d: stats did not rise: %+v then %+v", level, previous, current)
		}
		previous = current
	}
}

// 実数値は状態としても成立する（BattleStateの検証を通る）。
func TestLevelStatsProduceValidPokemon(t *testing.T) {
	t.Parallel()

	stats := LevelStats(Stats{HP: 65, Attack: 50, Defense: 35, Speed: 95, Special: 95}, 55)
	p := Pokemon{
		Species:   "charm",
		Level:     55,
		CurrentHP: stats.HP,
		Stats:     stats,
		Moves:     [MoveSlots]MoveSlot{{Move: "slumber", PP: 10, MaxPP: 10}},
	}

	if err := p.validate(); err != nil {
		t.Errorf("validate() error = %v", err)
	}
}
