package battleui

import (
	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// CommandKind はいま画面が出すべき選択肢の種類。
type CommandKind int

const (
	// CommandWaiting は相手待ちで、viewerの入力を受け付けない状態。
	//
	// 自分は出したが相手がまだのとき、相手だけが交代を選んでいるときが該当する。
	CommandWaiting CommandKind = iota

	// CommandChooseLead は最初に出す1体を選ぶ。
	CommandChooseLead

	// CommandChooseAction は技か交代を選ぶ。
	CommandChooseAction

	// CommandChooseReplacement は戦闘不能のあとに出す控えを選ぶ。
	CommandChooseReplacement

	// CommandFinished は決着後。新しい対戦を始める以外の選択肢は無い。
	CommandFinished
)

// String はCommandKindの名前を返す。testとlogの出力に使う。
func (k CommandKind) String() string {
	switch k {
	case CommandWaiting:
		return "waiting"
	case CommandChooseLead:
		return "choose lead"
	case CommandChooseAction:
		return "choose action"
	case CommandChooseReplacement:
		return "choose replacement"
	case CommandFinished:
		return "finished"
	default:
		return "unknown"
	}
}

// Commands はいま出せる選択肢。
//
// Kindに対応するfieldだけが埋まる。
type Commands struct {
	Kind CommandKind

	// Leads はlead候補の3体。CommandChooseLeadのときだけ埋まる。
	Leads []TeamOption

	// Moves は場に出ている1体の4つの技枠。CommandChooseActionのときだけ埋まる。
	// PPの尽きた技はDisabledで、選択肢から消さずに無効として見せる。
	Moves []MoveView

	// Struggle は技が1つも選べず、Struggleだけが残っているか。
	// CommandChooseActionのときだけ意味を持つ。
	Struggle bool

	// Switches は交代先の候補。CommandChooseActionとCommandChooseReplacementで埋まる。
	Switches []TeamOption

	// NewMatch は新しい対戦を始められるか。CommandFinishedのときにtrueになる。
	//
	// 新しいSessionを作るのはUI側（YTA-31）の仕事で、この層は
	// 「その選択肢を出してよい」ことだけを伝える。対戦履歴の再生ではない。
	NewMatch bool
}

// TeamOption は3体のうち1体を指す選択肢。
type TeamOption struct {
	// Index はTeam内index。配布順で、対戦中も変わらない。
	Index int

	// Species は選択肢の見た目を引くためのキー。
	Species battle.SpeciesID
}

// commands はphaseとviewerの状況から選択肢を決める。
//
// 出してよいかどうかだけを決め、合法性の最終判断はしない。
// それはsessionとresolverの責務で、ここでルールを複製しない。
func commands(in Input, viewer battle.Side) Commands {
	switch in.Phase {
	case singleplayer.PhaseLeadSelection:
		// 出したあとは相手待ち。
		if in.Pending.Lead {
			return Commands{Kind: CommandWaiting}
		}
		return Commands{Kind: CommandChooseLead, Leads: teamOptions(in.Teams[viewer])}

	case singleplayer.PhaseBattle:
		if !in.Started || in.Pending.Action {
			return Commands{Kind: CommandWaiting}
		}
		player := in.State.Players[viewer]
		active := player.Team[player.Active]
		return Commands{
			Kind:     CommandChooseAction,
			Moves:    movesOf(active),
			Struggle: !active.HasUsableMove(),
			Switches: reserveOptions(player),
		}

	case singleplayer.PhaseReplacement:
		// 倒れたのが相手だけなら、こちらは待つ。
		if !in.Started || !in.State.NeedsReplacement(viewer) {
			return Commands{Kind: CommandWaiting}
		}
		return Commands{Kind: CommandChooseReplacement, Switches: reserveOptions(in.State.Players[viewer])}

	case singleplayer.PhaseFinished:
		return Commands{Kind: CommandFinished, NewMatch: true}

	default:
		return Commands{Kind: CommandWaiting}
	}
}

// teamOptions は3体すべてを選択肢にする。
func teamOptions(team [battle.TeamSize]battle.Pokemon) []TeamOption {
	options := make([]TeamOption, 0, battle.TeamSize)
	for i, pokemon := range team {
		options = append(options, TeamOption{Index: i, Species: pokemon.Species})
	}
	return options
}

// reserveOptions は交代先になれる控えを選択肢にする。
//
// 誰が交代先になれるかはbattle.Player.Reserveが決める。
func reserveOptions(player battle.Player) []TeamOption {
	reserve := player.Reserve()
	options := make([]TeamOption, 0, len(reserve))
	for _, index := range reserve {
		options = append(options, TeamOption{Index: index, Species: player.Team[index].Species})
	}
	return options
}

// movesOf は4つの技枠をsliceで返す。
func movesOf(p battle.Pokemon) []MoveView {
	views := moveViews(p)
	return views[:]
}
