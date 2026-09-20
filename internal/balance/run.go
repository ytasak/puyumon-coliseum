package balance

import (
	"fmt"
	"slices"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/simulation"
)

// sides は両プレイヤーを順に処理するためのSide一覧。
var sides = [...]battle.Side{battle.Player1, battle.Player2}

// Run はroster combinationの総当たりを複数seedで実行し、結果を集計する。
//
// 回すのは順序付きの組み合わせなので、combinationがN通りならN×N×Config.Trials対戦になる。
// 同じteam同士（matrixの対角）も回す。
//
// 対戦は internal/simulation が進める。行動の選び方はConfig.Policyで選ぶ。
// どちらも賢いBotではなく、FirstUsableはslot 0を撃ち続ける方針、
// UniformUsableはその偏りを崩すための対照でしかない。
//
// 同じConfigからは必ず同じReportが返る。実行順も集計順もConfigだけで決まり、
// 並列化もしていないので、結果が走らせ方に左右されることはない。
func Run(cfg Config) (Report, error) {
	cfg, err := cfg.normalize()
	if err != nil {
		return Report{}, err
	}

	combinations := roster.Combinations()
	rec, err := newRecorder(cfg, combinations)
	if err != nil {
		return Report{}, err
	}

	for first := range combinations {
		for second := range combinations {
			for trial := 0; trial < cfg.Trials; trial++ {
				seed := cfg.seed(trial)
				result, err := simulation.Run(simulation.Config{
					Teams:    [2][battle.TeamSize]battle.SpeciesID{combinations[first], combinations[second]},
					Seed:     seed,
					Choosers: cfg.Policy.choosers(seed),
					MaxTurns: cfg.MaxTurns,
				})
				if err != nil {
					return Report{}, matchupError(combinations, first, second, seed, err)
				}
				if err := rec.record(first, second, result); err != nil {
					return Report{}, matchupError(combinations, first, second, seed, err)
				}
			}
		}
	}
	return rec.report, nil
}

// matchupError はどのmatchupのどのseedで起きたかを添えたerrorを返す。
func matchupError(combinations [][battle.TeamSize]battle.SpeciesID, first, second int, seed uint64, err error) error {
	return fmt.Errorf("%v vs %v (seed %d): %w", combinations[first], combinations[second], seed, err)
}

// recorder は対戦結果を1件ずつ数える。
//
// 集計の形はConfigとrosterだけで決まるので、対戦を始める前に全部作っておく。
// 数えるときに枠が増えないため、結果が出てくる順序に左右されない。
type recorder struct {
	// report は数えている途中のReport。
	report Report

	// contains は[combinationの添字][speciesの添字]で、そのteamがそのキャラクターを含むか。
	contains [][]bool
}

// newRecorder は空のReportを組み立てる。
func newRecorder(cfg Config, combinations [][battle.TeamSize]battle.SpeciesID) (*recorder, error) {
	characters := roster.All()

	report := Report{
		Config:       cfg,
		Turns:        newTurnStats(cfg.MaxTurns),
		Combinations: make([]CombinationSummary, len(combinations)),
		Species:      make([]SpeciesSummary, len(characters)),
		Matchups:     make([][]MatchupSummary, len(combinations)),
	}
	contains := make([][]bool, len(combinations))

	for i, team := range combinations {
		levels, err := roster.AssignLevels(team)
		if err != nil {
			return nil, fmt.Errorf("balance: %v: %w", team, err)
		}
		report.Combinations[i] = CombinationSummary{
			Team:   team,
			Levels: levels,
			Turns:  newTurnStats(cfg.MaxTurns),
		}

		report.Matchups[i] = make([]MatchupSummary, len(combinations))
		for j := range report.Matchups[i] {
			report.Matchups[i][j] = MatchupSummary{First: i, Second: j}
		}

		contains[i] = make([]bool, len(characters))
		for s, character := range characters {
			contains[i][s] = slices.Contains(team[:], character.ID)
		}
	}

	for s, character := range characters {
		summary := SpeciesSummary{Species: character.ID}
		for i := range combinations {
			if contains[i][s] {
				summary.Teams++
			}
		}
		report.Species[s] = summary
	}

	return &recorder{report: report, contains: contains}, nil
}

// record は1対戦の結果を、全体・side・combination・speciesのそれぞれへ数える。
func (r *recorder) record(first, second int, result simulation.Result) error {
	firstView, err := outcomeFor(result)
	if err != nil {
		return err
	}
	views := [2]outcome{battle.Player1: firstView, battle.Player2: firstView.flip()}
	teams := [2]int{battle.Player1: first, battle.Player2: second}
	resolved := firstView != unresolved

	r.report.Battles++
	r.report.Matchups[first][second].Outcomes.add(firstView)
	if resolved {
		r.report.Turns.add(result.Turns)
	}

	for _, side := range sides {
		view := views[side]
		team := teams[side]

		r.report.Sides[side].add(view)

		summary := &r.report.Combinations[team]
		summary.Total.add(view)
		summary.Sides[side].add(view)
		if resolved {
			summary.Turns.add(result.Turns)
		}

		for s := range r.report.Species {
			species := &r.report.Species[s]
			if r.contains[team][s] {
				species.With.add(view)
			} else {
				species.Without.add(view)
			}
		}
	}
	return nil
}

// outcomeFor は1対戦の結果をPlayer1側から見て返す。
//
// 上限に達した対戦は勝敗を付けずunresolvedにする。simulationが上限で止めたときも
// 決着したときもStatusを見れば分かるが、判断はTurnLimitReachedを正とする。
func outcomeFor(result simulation.Result) (outcome, error) {
	if result.TurnLimitReached {
		return unresolved, nil
	}
	switch result.Final.Status {
	case battle.Player1Won:
		return win, nil
	case battle.Player2Won:
		return loss, nil
	case battle.Draw:
		return draw, nil
	default:
		return 0, fmt.Errorf("balance: battle stopped with status %v before the turn limit", result.Final.Status)
	}
}
