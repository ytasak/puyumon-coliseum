package simulation

import (
	"fmt"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
)

// DefaultMaxTurns はMaxTurnsを指定しなかったときの上限。
const DefaultMaxTurns = 200

// sides は両プレイヤーを順に処理するためのSide一覧。
var sides = [...]battle.Side{battle.Player1, battle.Player2}

// Config は1対戦の設定。
type Config struct {
	// Teams は両プレイヤーへ配る3体。rosterのinternal IDで指定する。
	//
	// Levelと能力値はrosterの155の配分規則が決める。
	Teams [2][battle.TeamSize]battle.SpeciesID

	// Seed は乱数のseed。同じseedと同じ行動からは必ず同じ結果になる。
	Seed uint64

	// Choosers は各プレイヤーの行動の選び方。nilならFirstUsableを使う。
	//
	// Chooserそのものではなく作り方を持つ。こうしておくとConfigは
	// scenarioの定義のままでいられるので、同じConfigを何度でも、
	// 並列にでもRunできる。scriptedな行動列はReplayで渡す。
	Choosers [2]ChooserFactory

	// MaxTurns はsimulationを打ち切るturn数。0ならDefaultMaxTurnsを使う。
	//
	// これはsimulationが終わらなくなるのを防ぐための安全装置であって、
	// 対戦のルールではない。到達しても勝敗は決めず、ResultのTurnLimitReachedが立つ。
	MaxTurns int
}

// Result は1対戦の結果。
type Result struct {
	// Final は最後のBattleState。上限で止めた場合、StatusはOngoingのまま。
	Final battle.BattleState

	// Turns は解決したturn数。
	Turns int

	// TurnLimitReached は上限に達して決着しなかったか。
	//
	// このときも勝敗は決めない。引き分けとは別物として扱う。
	TurnLimitReached bool

	// Events は対戦中に起きたEventを順に並べたもの。
	Events []battle.Event
}

// Run は1対戦を決着または上限まで進める。
//
// 対戦のルールはすべてBattle Engineが決める。このpackageは行動を集めてresolverへ渡し、
// 戦闘不能のあとの交代を挟み、上限で止めるだけ。
func Run(cfg Config) (Result, error) {
	maxTurns := cfg.MaxTurns
	if maxTurns <= 0 {
		maxTurns = DefaultMaxTurns
	}

	var teams [2][battle.TeamSize]battle.Pokemon
	for _, side := range sides {
		team, err := roster.NewTeam(cfg.Teams[side])
		if err != nil {
			return Result{}, fmt.Errorf("%s team: %w", side, err)
		}
		teams[side] = team
	}

	state, err := battle.NewBattleState(teams[battle.Player1], teams[battle.Player2])
	if err != nil {
		return Result{}, err
	}

	choosers, err := newChoosers(cfg.Choosers)
	if err != nil {
		return Result{}, err
	}

	resolver := &battle.Resolver{Data: roster.Data(), RNG: battle.NewRand(cfg.Seed)}

	var result Result
	for result.Turns < maxTurns {
		// 戦闘不能のあとは、通常のturnより先に交代を済ませる。
		for _, side := range sides {
			if !state.NeedsReplacement(side) {
				continue
			}
			next, events, err := resolver.ResolveReplacement(state, side, choosers[side].Replacement(state, side))
			if err != nil {
				return result, fmt.Errorf("%s replacement on turn %d: %w", side, result.Turns+1, err)
			}
			state = next
			result.Events = append(result.Events, events...)
		}
		if state.Status != battle.Ongoing {
			break
		}

		var actions [2]battle.Action
		for _, side := range sides {
			actions[side] = choosers[side].Action(state, side)
		}

		next, events, err := resolver.ResolveTurn(state, actions[battle.Player1], actions[battle.Player2])
		if err != nil {
			return result, fmt.Errorf("turn %d: %w", result.Turns+1, err)
		}
		state = next
		result.Turns++
		result.Events = append(result.Events, events...)

		if state.Status != battle.Ongoing {
			break
		}
	}

	result.Final = state
	result.TurnLimitReached = state.Status == battle.Ongoing
	return result, nil
}

// newChoosers はこのRunで使うChooserを作る。
//
// Runごとに作るので、Configを使い回しても前のRunの状態を引き継がない。
func newChoosers(factories [2]ChooserFactory) ([2]Chooser, error) {
	var choosers [2]Chooser
	for _, side := range sides {
		if factories[side] == nil {
			choosers[side] = FirstUsable{}
			continue
		}
		chooser := factories[side]()
		if chooser == nil {
			return choosers, fmt.Errorf("simulation: chooser factory for %s returned nil", side)
		}
		choosers[side] = chooser
	}
	return choosers, nil
}
