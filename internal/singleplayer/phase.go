package singleplayer

// Phase はsessionの進行段階。
//
// 3体の配布はNewSessionの中で終わるため、外から観測できるSetup段階は持たない。
// sessionはleadを選ぶところから始まる。
type Phase int

const (
	// PhaseLeadSelection は両者が最初に場へ出す1体を選ぶ段階。
	//
	// この段階ではまだBattleStateが無い。配布された3体はTeamで参照する。
	PhaseLeadSelection Phase = iota

	// PhaseBattle は両者のActionを受けてturnを解決する段階。
	PhaseBattle

	// PhaseReplacement は戦闘不能になったsideが次に出す控えを選ぶ段階。
	PhaseReplacement

	// PhaseFinished は勝敗が決まった段階。これ以上commandを受け付けない。
	PhaseFinished
)

// String はphase名を返す。errorやlogで使う。
func (p Phase) String() string {
	switch p {
	case PhaseLeadSelection:
		return "lead selection"
	case PhaseBattle:
		return "battle"
	case PhaseReplacement:
		return "replacement"
	case PhaseFinished:
		return "finished"
	default:
		return "unknown"
	}
}
