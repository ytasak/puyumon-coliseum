package roster

import (
	"errors"
	"fmt"
	"slices"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// LevelTotal は配布された3体のLevelの合計。
//
// 歴史的な155形式（50〜55から選出3体の合計155以下）をモチーフにした、
// 本作向けの規則。ランダム配布なので「以下」ではなく毎回ちょうどこの値にする。
const LevelTotal = 155

// levelsByRank は配布された3体へ、強い順に割り当てるLevel。
//
// 合計がLevelTotalになり、どの組み合わせでも50〜55の範囲へ収まる。
var levelsByRank = [battle.TeamSize]int{50, 50, 55}

// ErrUnknownCharacter は定義にないキャラクターを指定したことを表す。
var ErrUnknownCharacter = errors.New("roster: unknown character")

// AssignLevels は配布された3体へLevelを割り当てる。
//
// 強い順に50 / 50 / 55を割り当てるので、どの3体を配っても合計はLevelTotalになる。
// 戻り値は引数と同じ並びで、それぞれに対応するLevelが入る。
//
// 同じキャラクターを重ねて配ることは想定していないのでerrorにする。
func AssignLevels(team [battle.TeamSize]battle.SpeciesID) ([battle.TeamSize]int, error) {
	var levels [battle.TeamSize]int

	ranks := make([]int, 0, battle.TeamSize)
	order := make([]int, 0, battle.TeamSize)
	for i, id := range team {
		character, ok := Lookup(id)
		if !ok {
			return levels, fmt.Errorf("%w: %q", ErrUnknownCharacter, id)
		}
		if slices.Contains(ranks, character.Rank) {
			return levels, fmt.Errorf("roster: %q appears more than once in the team", id)
		}
		ranks = append(ranks, character.Rank)
		order = append(order, i)
	}

	// 強い順（Rankの昇順）に並べ替えてからLevelを配る。
	slices.SortFunc(order, func(a, b int) int { return ranks[a] - ranks[b] })
	for position, index := range order {
		levels[index] = levelsByRank[position]
	}
	return levels, nil
}

// Combinations は6キャラクターから3体を選ぶ組み合わせをすべて返す。20通りある。
//
// 各組み合わせはRankの昇順に並ぶ。バランス検証で全組み合わせを回すために使う。
func Combinations() [][battle.TeamSize]battle.SpeciesID {
	var combinations [][battle.TeamSize]battle.SpeciesID

	for i := 0; i < len(characters); i++ {
		for j := i + 1; j < len(characters); j++ {
			for k := j + 1; k < len(characters); k++ {
				combinations = append(combinations, [battle.TeamSize]battle.SpeciesID{
					characters[i].ID, characters[j].ID, characters[k].ID,
				})
			}
		}
	}
	return combinations
}

// NewTeam は配布された3体から、対戦開始時のPokemonを作る。
//
// Levelは155の配分規則で決まり、実数値はそのLevelから算出する。
// HPは満タン、PPは各技の最大値から始まる。
func NewTeam(team [battle.TeamSize]battle.SpeciesID) ([battle.TeamSize]battle.Pokemon, error) {
	var pokemon [battle.TeamSize]battle.Pokemon

	levels, err := AssignLevels(team)
	if err != nil {
		return pokemon, err
	}

	for i, id := range team {
		character, ok := Lookup(id)
		if !ok {
			return pokemon, fmt.Errorf("%w: %q", ErrUnknownCharacter, id)
		}

		stats := battle.LevelStats(character.BaseStats, levels[i])
		p := battle.Pokemon{
			Species:   character.ID,
			Level:     levels[i],
			CurrentHP: stats.HP,
			Stats:     stats,
		}
		for slot, moveID := range character.Moves {
			move, ok := lookupMove(moveID)
			if !ok {
				return pokemon, fmt.Errorf("roster: %q refers to unknown move %q", id, moveID)
			}
			p.Moves[slot] = battle.MoveSlot{Move: move.ID, PP: move.MaxPP, MaxPP: move.MaxPP}
		}
		pokemon[i] = p
	}
	return pokemon, nil
}

// lookupMove は技の定義を返す。
func lookupMove(id battle.MoveID) (battle.Move, bool) {
	for _, move := range moves {
		if move.ID == id {
			return move, true
		}
	}
	return battle.Move{}, false
}
