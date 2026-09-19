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

// ChooserFactory は1回のRunで使うChooserを作る。
//
// Chooserは行動を選ぶ過程で状態を持つことがある（Scriptがどこまで使ったか、など）。
// Configがそのinstanceを直接持つと、同じConfigで2回Runしたときに
// 1回目で進んだ状態から始まってしまい、同じ初期状態・同じ行動列にならない。
//
// そのためConfigはinstanceではなく作り方を持つ。Runは開始時にこれを呼び、
// そのRun専用のChooserを得る。並列にRunしてもChooserを共有しない。
//
// 呼ばれるたびに新しいChooserを返すこと。同じinstanceを返すと上の前提が崩れる。
type ChooserFactory func() Chooser

// Replay はScriptの定義から、Runごとに同じ行動列を再現するChooserFactoryを返す。
//
// 引数は定義として扱い、呼ばれるたびに複製する。使いかけのScriptを渡すと
// その途中から始まる複製ができるので、定義そのものを渡すこと。
func Replay(script Script) ChooserFactory {
	return func() Chooser {
		// 値として複製することで、どこまで使ったかがRunごとに分かれる。
		fresh := script
		return &fresh
	}
}

// Script は決められたAction列を順に返すChooser。
//
// Actionが尽きたらFallbackへ委ねる。scripted actionsで特定の流れを再現するために使う。
//
// どこまで使ったかを持つので、1つのinstanceを複数のRunで使い回さない。
// ConfigへはReplayを通して渡す。
type Script struct {
	// Actions は使う順に並べた行動。
	Actions []battle.Action

	// Replacements は戦闘不能のたびに使う交代先。尽きたらFallbackへ委ねる。
	Replacements []battle.SwitchAction

	// Fallback はActionsやReplacementsが尽きたときに使うChooserの作り方。
	// nilならFirstUsableを使う。
	//
	// instanceではなく作り方を持つのはConfigと同じ理由で、
	// Scriptを複製したときにFallbackの状態まで共有してしまうのを防ぐため。
	Fallback ChooserFactory

	actionIndex      int
	replacementIndex int

	// fallback は初回に作ったFallbackのinstance。Script1つにつき1つ。
	fallback Chooser
}

// Action は次の行動を返す。
func (s *Script) Action(state battle.BattleState, side battle.Side) battle.Action {
	if s.actionIndex < len(s.Actions) {
		action := s.Actions[s.actionIndex]
		s.actionIndex++
		return action
	}
	return s.fallbackChooser().Action(state, side)
}

// Replacement は次の交代先を返す。
func (s *Script) Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction {
	if s.replacementIndex < len(s.Replacements) {
		action := s.Replacements[s.replacementIndex]
		s.replacementIndex++
		return action
	}
	return s.fallbackChooser().Replacement(state, side)
}

// fallbackChooser はFallbackのinstanceを返す。初回だけ作る。
func (s *Script) fallbackChooser() Chooser {
	if s.fallback != nil {
		return s.fallback
	}
	if s.Fallback != nil {
		s.fallback = s.Fallback()
	}
	if s.fallback == nil {
		s.fallback = FirstUsable{}
	}
	return s.fallback
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

// UniformUsable は使える技から一様ランダムに1つ選ぶChooser。
//
// FirstUsableはPPが切れるまで同じ技を撃ち続けるため、大量simulationの結果が
// 「slot 0の質」に寄る。その偏りを崩すための対照用の方針で、勝ち筋を考えることはしない。
//
// 乱数はBattle Engineのものと分けて持つ。行動選択でengineの乱数列を消費すると、
// 同じseedでもChooserを替えただけでダメージや命中の乱数がずれてしまう。
//
// RNGは必須。Runごとに新しいものを渡すため、ConfigへはUniformUsablePolicyで渡す。
type UniformUsable struct {
	// RNG は行動選択にだけ使う乱数。
	RNG battle.RNG
}

// UniformUsablePolicy はseedから、Runごとに同じ行動列を再現するChooserFactoryを返す。
//
// Runのたびに新しいRNGを作るので、同じConfigを何度Runしても、並列に回しても同じ結果になる。
func UniformUsablePolicy(seed uint64) ChooserFactory {
	return func() Chooser {
		return UniformUsable{RNG: battle.NewRand(seed)}
	}
}

// Action は使える技から一様ランダムに1つ返す。
//
// 使える技が1つしかない場合も乱数を1つ引く。行動を選ぶたびに必ず1つ消費することで、
// 乱数列の位置が「何回選んだか」だけで決まり、その時点で何技使えたかに左右されない。
//
// 使える技が無ければFirstUsableと同じ扱いにする。ここでは乱数を引かない。
func (u UniformUsable) Action(state battle.BattleState, side battle.Side) battle.Action {
	player := state.Players[side]
	active := player.Team[player.Active]

	usable := make([]int, 0, battle.MoveSlots)
	for slot, move := range active.Moves {
		if move.Usable() {
			usable = append(usable, slot)
		}
	}
	if len(usable) == 0 {
		return FirstUsable{}.Action(state, side)
	}
	return battle.MoveAction{Slot: usable[u.RNG.IntN(len(usable))]}
}

// Replacement は戦える控えのうちいちばん上を返す。
//
// 交代先は決定論的でよい。乱数を引かないので、戦闘不能の起きた回数で
// 行動選択の乱数列がずれることもない。
func (UniformUsable) Replacement(state battle.BattleState, side battle.Side) battle.SwitchAction {
	return FirstUsable{}.Replacement(state, side)
}
