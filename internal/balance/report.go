package balance

import (
	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// TurnBucketWidth はturn分布の区間の幅。
const TurnBucketWidth = 10

// Report は1回のbatch実行の集計結果。
//
// 数えた結果だけを持ち、balanceの合否は判定しない。
//
// 行動は simulation.FirstUsable（使える先頭の技）で固定なので、ここに出る勝率は
// 人間同士の勝率ではない。roster自体の構造的な偏りと、この行動方針が生む偏りは
// 分けて読む必要がある。
//
// 総当たりには自分自身との対戦（matrixの対角）も含む。同じteam同士の結果は
// roster差が消えてside差だけが残るので、先攻・後攻の非対称性を見るのに使える。
type Report struct {
	// Config は実行に使った設定。省略値を埋めたあとの値なので、そのまま渡せば再現できる。
	Config Config

	// Battles は実行した対戦数。combinationの数の2乗 × Config.Trials。
	Battles int

	// Sides はside別の結果。battle.Sideをそのまま添字に使える。
	//
	// Sides[battle.Player1].WinsはPlayer1側が勝った対戦数。総当たりは順序付きなので
	// rosterの強弱は打ち消し合い、残るのはsideの非対称性になる。
	Sides [2]Outcomes

	// Turns は決着した全対戦のturn分布。
	Turns TurnStats

	// Combinations はcombination別の結果。roster.Combinations()と同じ並び。
	Combinations []CombinationSummary

	// Species はキャラクター別の結果。roster.All()と同じ並び（Rankの昇順）。
	Species []SpeciesSummary

	// Matchups は[Player1側][Player2側]のmatchup matrix。添字はCombinationsの添字。
	Matchups [][]MatchupSummary
}

// CombinationSummary は1つのroster combinationから見た結果。
//
// 1対戦につき、そのcombinationが立っていたside分だけ数える。自分自身との対戦では
// 両sideに立つので2件数えることになる。
type CombinationSummary struct {
	// Team は配られる3体。roster.Combinations()と同じ並び（Rankの昇順）。
	Team [battle.TeamSize]battle.SpeciesID

	// Levels はTeamと同じ並びのLevel。155の配分規則が決めた実際の値。
	Levels [battle.TeamSize]int

	// Total は両sideを合わせた結果。
	Total Outcomes

	// Sides はside別の結果。battle.Sideをそのまま添字に使える。
	Sides [2]Outcomes

	// Turns はこのcombinationが出た対戦のうち、決着したもののturn分布。
	Turns TurnStats
}

// SpeciesSummary は1キャラクターを含むteamと、含まないteamの結果。
//
// 数えるのは対戦ではなく「対戦ごとの各side」で、1対戦につき2件になる。
// そのためWithとWithoutのBattlesの合計は Report.Battles の2倍になる。
//
// 両teamが同じキャラクターを含む対戦では、Withに勝ちと負けが1件ずつ入る。
// 含む側と含まない側が当たった対戦だけがWithとWithoutの差になる。
type SpeciesSummary struct {
	// Species はキャラクターのinternal ID。
	Species battle.SpeciesID

	// Teams はこのキャラクターを含むcombinationの数。
	Teams int

	// With はこのキャラクターを含むteam側から見た結果。
	With Outcomes

	// Without は含まないteam側から見た結果。
	Without Outcomes
}

// MatchupSummary はordered matchup 1つ分の結果。
//
// 数はすべてPlayer1側（First）から見たもの。逆順の対戦は別のMatchupSummaryになる。
type MatchupSummary struct {
	// First はPlayer1側のcombination。Report.Combinationsの添字。
	First int

	// Second はPlayer2側のcombination。Report.Combinationsの添字。
	Second int

	// Outcomes はFirstから見た結果。Config.Trials回分。
	Outcomes Outcomes
}

// Outcomes は対戦結果の内訳。ある視点（全体・side・combination・species）から数えたもの。
type Outcomes struct {
	// Battles は数えた対戦数。Wins + Losses + Draws + Unresolved と一致する。
	Battles int

	// Wins は勝った数。
	Wins int

	// Losses は負けた数。
	Losses int

	// Draws は両者に戦えるPokemonが残らず引き分けた数。
	Draws int

	// Unresolved はturn上限に達して決着しなかった数。引き分けとは別に数える。
	Unresolved int
}

// WinRate は数えた全対戦に対する勝率。引き分けも未決着も分母に含む。
func (o Outcomes) WinRate() float64 {
	return ratio(o.Wins, o.Battles)
}

// ResolvedWinRate は決着した対戦に対する勝率。未決着を分母から除く。
//
// 引き分けは決着なので分母に残る。
func (o Outcomes) ResolvedWinRate() float64 {
	return ratio(o.Wins, o.Battles-o.Unresolved)
}

// UnresolvedRate は未決着だった割合。
func (o Outcomes) UnresolvedRate() float64 {
	return ratio(o.Unresolved, o.Battles)
}

// add は1対戦分を数える。
func (o *Outcomes) add(result outcome) {
	o.Battles++
	switch result {
	case win:
		o.Wins++
	case loss:
		o.Losses++
	case draw:
		o.Draws++
	case unresolved:
		o.Unresolved++
	}
}

// TurnStats は決着した対戦のturn数の分布。
//
// 未決着の対戦は含めない。turn数が必ず上限と同じ値になり、分布の末尾へ実態のない
// 山を作ってしまうため。未決着の数は Outcomes.Unresolved で数える。
type TurnStats struct {
	// Battles は数えた対戦数。
	Battles int

	// Min は最短のturn数。1対戦も数えていなければ0。
	Min int

	// Max は最長のturn数。1対戦も数えていなければ0。
	Max int

	// Total はturn数の合計。平均を出すために持つ。
	Total int

	// Buckets はturn数の区間ごとの対戦数。区間はturn 1から上限までを
	// TurnBucketWidthずつ区切ったもので、数えた対戦が無くても並びは変わらない。
	Buckets []TurnBucket
}

// TurnBucket はturn数の区間と、そこへ入った対戦数。
type TurnBucket struct {
	// Min は区間に含む最小のturn数。
	Min int

	// Max は区間に含む最大のturn数。
	Max int

	// Battles はこの区間に入った対戦数。
	Battles int
}

// Mean は平均turn数。1対戦も数えていなければ0。
func (t TurnStats) Mean() float64 {
	return ratio(t.Total, t.Battles)
}

// add は1対戦分のturn数を数える。
func (t *TurnStats) add(turns int) {
	if t.Battles == 0 || turns < t.Min {
		t.Min = turns
	}
	if turns > t.Max {
		t.Max = turns
	}
	t.Battles++
	t.Total += turns

	index := (turns - 1) / TurnBucketWidth
	if index < 0 {
		index = 0
	}
	if index >= len(t.Buckets) {
		index = len(t.Buckets) - 1
	}
	if index >= 0 {
		t.Buckets[index].Battles++
	}
}

// newTurnStats はturn 1から上限までの区間を並べた、まだ何も数えていないTurnStatsを作る。
func newTurnStats(maxTurns int) TurnStats {
	var stats TurnStats
	for low := 1; low <= maxTurns; low += TurnBucketWidth {
		high := low + TurnBucketWidth - 1
		if high > maxTurns {
			high = maxTurns
		}
		stats.Buckets = append(stats.Buckets, TurnBucket{Min: low, Max: high})
	}
	return stats
}

// outcome は1対戦をある側から見た結果。
type outcome int

const (
	win outcome = iota
	loss
	draw
	unresolved
)

// flip は反対側から見た結果を返す。
func (o outcome) flip() outcome {
	switch o {
	case win:
		return loss
	case loss:
		return win
	default:
		return o
	}
}

// ratio は割合を返す。分母が0なら0を返す。
func ratio(part, whole int) float64 {
	if whole <= 0 {
		return 0
	}
	return float64(part) / float64(whole)
}
