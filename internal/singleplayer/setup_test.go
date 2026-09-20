package singleplayer

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// 用途ごとのseedはroot seedから導出する。導出方法を変えると同じseedから
// 別の試合になるため、値を固定して不用意に変わらないようにする。
func TestDerivedSeedsAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		root uint64
		want seeds
	}{
		{root: 0, want: seeds{setup: 16294208416658607535, battle: 7960286522194355700, bot: 487617019471545679}},
		{root: 1, want: seeds{setup: 10451216379200822465, battle: 13757245211066428519, bot: 17911839290282890590}},
		{root: 42, want: seeds{setup: 13679457532755275413, battle: 2949826092126892291, bot: 5139283748462763858}},
	}
	for _, tc := range cases {
		if got := deriveSeeds(tc.root); got != tc.want {
			t.Errorf("deriveSeeds(%d) = %+v, want %+v", tc.root, got, tc.want)
		}
	}
}

// 用途ごとのseedは互いに異なる。同じ値だと片方の消費がもう片方へ響く。
func TestDerivedSeedsDifferPerStream(t *testing.T) {
	t.Parallel()

	for _, root := range []uint64{0, 1, 42, 12345} {
		derived := deriveSeeds(root)
		if derived.setup == derived.battle || derived.setup == derived.bot || derived.battle == derived.bot {
			t.Errorf("root %d: seedが重複している: %+v", root, derived)
		}
	}
}

// 同じseedからは同じ配布になる。
func TestSameSeedDealsTheSameTrios(t *testing.T) {
	t.Parallel()

	first := newSessionOrFatal(t, Config{Seed: 7})
	second := newSessionOrFatal(t, Config{Seed: 7})

	for _, side := range sides {
		want := speciesOf(teamOrFatal(t, first, side))
		got := speciesOf(teamOrFatal(t, second, side))
		if got != want {
			t.Errorf("%s: 同じseedで配布が違う: %v と %v", side, got, want)
		}
	}
}

// seedが違えば配布も変わる。1つでも変われば配布がseedに依存していることになる。
func TestDifferentSeedsChangeTheDeal(t *testing.T) {
	t.Parallel()

	base := speciesOf(teamOrFatal(t, newSessionOrFatal(t, Config{Seed: 1}), Player))
	for seed := uint64(2); seed < 40; seed++ {
		if speciesOf(teamOrFatal(t, newSessionOrFatal(t, Config{Seed: seed}), Player)) != base {
			return
		}
	}
	t.Error("seedを変えても配布が変わらない")
}

// 6体を重複なく3+3へ分ける。
func TestDealUsesEveryCharacterOnce(t *testing.T) {
	t.Parallel()

	all := roster.All()
	for seed := uint64(0); seed < 50; seed++ {
		session := newSessionOrFatal(t, Config{Seed: seed})

		seen := map[battle.SpeciesID]int{}
		for _, side := range sides {
			for _, pokemon := range teamOrFatal(t, session, side) {
				seen[pokemon.Species]++
			}
		}
		if len(seen) != len(all) {
			t.Fatalf("seed %d: %d種類しか配られていない（%d体いる）", seed, len(seen), len(all))
		}
		for _, character := range all {
			if seen[character.ID] != 1 {
				t.Errorf("seed %d: %s が %d 回配られた", seed, character.ID, seen[character.ID])
			}
		}
	}
}

// 配布された3体のLevelはv1の155配分に従う。
func TestDealtTeamsFollowTheLevelRule(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 20; seed++ {
		session := newSessionOrFatal(t, Config{Seed: seed})
		for _, side := range sides {
			team := teamOrFatal(t, session, side)

			total := 0
			weakest, weakestRank := battle.SpeciesID(""), 0
			for _, pokemon := range team {
				total += pokemon.Level

				character, ok := roster.Lookup(pokemon.Species)
				if !ok {
					t.Fatalf("seed %d: %s の定義が無い", seed, pokemon.Species)
				}
				// Rankは1が最強なので、値が大きいほど弱い。
				if character.Rank > weakestRank {
					weakest, weakestRank = pokemon.Species, character.Rank
				}
			}
			if total != roster.LevelTotal {
				t.Errorf("seed %d %s: Level合計が %d（%d のはず）", seed, side, total, roster.LevelTotal)
			}
			for _, pokemon := range team {
				want := 50
				if pokemon.Species == weakest {
					want = 55
				}
				if pokemon.Level != want {
					t.Errorf("seed %d %s: %s のLevelが %d（%d のはず）", seed, side, pokemon.Species, pokemon.Level, want)
				}
			}
		}
	}
}

// speciesOf は3体のSpeciesIDを配布順に返す。
func speciesOf(team [battle.TeamSize]battle.Pokemon) [battle.TeamSize]battle.SpeciesID {
	var ids [battle.TeamSize]battle.SpeciesID
	for i, pokemon := range team {
		ids[i] = pokemon.Species
	}
	return ids
}

// teamOrFatal は配布された3体を取り出す。
func teamOrFatal(t *testing.T, s *Session, side battle.Side) [battle.TeamSize]battle.Pokemon {
	t.Helper()

	team, ok := s.Team(side)
	if !ok {
		t.Fatalf("%s のteamを取得できない", side)
	}
	return team
}

// newSessionOrFatal はsessionを作る。
func newSessionOrFatal(t *testing.T, cfg Config) *Session {
	t.Helper()

	session, err := NewSession(cfg)
	if err != nil {
		t.Fatalf("NewSession(%+v)に失敗: %v", cfg, err)
	}
	return session
}
