package battle

import (
	"errors"
	"fmt"
)

// ErrMissingData は必要なゲーム定義が見つからないことを表す。
//
// 状態が参照している技やキャラクターの定義がDataに無い場合に返る。
var ErrMissingData = errors.New("battle: missing data")

// Move は技の定義。
//
// 対戦中に変わるのは残りPPだけで、それはPokemon側のMoveSlotが持つ。
// ここにあるのは変わらない定義で、実データは別Issueで用意する。
type Move struct {
	// ID は技の識別子。MoveSlotから参照される。
	ID MoveID

	// Type は技のタイプ。物理か特殊かもこれで決まる。
	Type Type

	// Power は威力。0なら相手のHPを直接減らさない技。
	Power int

	// Accuracy は命中率。Generation Iと同じ0〜255表現で、255が100%。
	Accuracy int

	// MaxPP は最大PP。
	MaxPP int

	// Priority は行動優先度。大きいほど先に動く。
	//
	// 実機はQuick Attack（先制）とCounter（後攻）だけを特別扱いしているが、
	// engineでは数値として一般化する。通常の技は0。
	Priority int

	// HighCritRatio は急所率の高い技か。Generation Iでは閾値が8倍になる。
	HighCritRatio bool
}

// Species はキャラクターの定義のうち、Battle Engineが必要とする分。
//
// 見た目やEmojiの構成は持たない。それはUI側の関心事。
type Species struct {
	// ID はキャラクターの識別子。Pokemonから参照される。
	ID SpeciesID

	// Typing はタイプ構成。相性とSTABの判定に使う。
	Typing Typing

	// BaseSpeed は種族ごとのSpeedの基礎値。
	//
	// Generation Iの急所率はこの値で決まる。stat stageやまひを反映した
	// effective Speedとは別物なので、対戦中に変わらない。
	BaseSpeed int
}

// Data はresolverが参照するゲーム定義。
//
// 実データは別Issueで定義し、ここへ渡す。engine側は定義の形だけを持つので、
// キャラクターや技が増えてもresolverのコードは変わらない。
type Data struct {
	// Moves は技の定義。
	Moves map[MoveID]Move

	// Species はキャラクターの定義。
	Species map[SpeciesID]Species
}

// LookupMove は技の定義を返す。無ければErrMissingDataをwrapして返す。
func (d Data) LookupMove(id MoveID) (Move, error) {
	move, ok := d.Moves[id]
	if !ok {
		return Move{}, fmt.Errorf("%w: move %q", ErrMissingData, id)
	}
	return move, nil
}

// LookupSpecies はキャラクターの定義を返す。無ければErrMissingDataをwrapして返す。
func (d Data) LookupSpecies(id SpeciesID) (Species, error) {
	species, ok := d.Species[id]
	if !ok {
		return Species{}, fmt.Errorf("%w: species %q", ErrMissingData, id)
	}
	return species, nil
}
