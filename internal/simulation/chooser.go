// Package simulation はBattle EngineをUIなしで回すための入口を提供する。
//
// 対戦のルールは持たない。行動順もダメージも状態異常も internal/battle のresolverが決め、
// キャラクターと技の定義は internal/roster から取る。このpackageの責務は
// 「scenarioを進める」「結果をまとめる」「上限で止める」の3つだけ。
//
// 後続のバランス検証がそのまま呼べるように、testではなくproductionのコードとして置いている。
package simulation

import (
	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// Chooser は各手番の行動を選ぶ。
//
// 賢い判断はしない。testやバランス検証のための決まった方針だけを置く。
type Chooser interface {
	// Action は通常のturnの行動を選ぶ。
	Action(state battle.BattleState, side battle.Side) battle.Action

	// Replacement は戦闘不能になったあとに場へ出す控えを選ぶ。
	Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction
}

// Script は決められたAction列を順に返すChooser。
//
// Actionが尽きたらFallbackへ委ねる。scripted actionsで特定の流れを再現するために使う。
type Script struct {
	// Actions は使う順に並べた行動。
	Actions []battle.Action

	// Replacements は戦闘不能のたびに使う交代先。尽きたらFallbackへ委ねる。
	Replacements []battle.SwitchAction

	// Fallback はActionsやReplacementsが尽きたときに使う。nilならFirstUsableを使う。
	Fallback Chooser

	actionIndex      int
	replacementIndex int
}

// Action は次の行動を返す。
func (s *Script) Action(state battle.BattleState, side battle.Side) battle.Action {
	if s.actionIndex < len(s.Actions) {
		action := s.Actions[s.actionIndex]
		s.actionIndex++
		return action
	}
	return s.fallback().Action(state, side)
}

// Replacement は次の交代先を返す。
func (s *Script) Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction {
	if s.replacementIndex < len(s.Replacements) {
		action := s.Replacements[s.replacementIndex]
		s.replacementIndex++
		return action
	}
	return s.fallback().Replacement(state, side)
}

func (s *Script) fallback() Chooser {
	if s.Fallback != nil {
		return s.Fallback
	}
	return FirstUsable{}
}

// FirstUsable は使えるいちばん上の技を選ぶChooser。
//
// 大量のsimulationを回すための単純な方針で、勝ち筋を考えることはしない。
// 使える技が無ければ、戦える控えのうちいちばん上へ交代する。
type FirstUsable struct{}

// Action は使えるいちばん上の技を返す。
func (FirstUsable) Action(state battle.BattleState, side battle.Side) battle.Action {
	player := state.Players[side]
	active := player.Team[player.Active]

	for slot, move := range active.Moves {
		if move.Usable() {
			return battle.MoveAction{Slot: slot}
		}
	}
	// 使える技が無ければ交代する。控えも居なければ先頭の技を返し、resolverの検証に委ねる。
	if reserve := player.Reserve(); len(reserve) > 0 {
		return battle.SwitchAction{Target: reserve[0]}
	}
	return battle.MoveAction{Slot: 0}
}

// Replacement は戦える控えのうちいちばん上を返す。
func (FirstUsable) Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction {
	if reserve := state.Players[side].Reserve(); len(reserve) > 0 {
		return battle.SwitchAction{Target: reserve[0]}
	}
	return battle.SwitchAction{Target: state.Players[side].Active}
}
