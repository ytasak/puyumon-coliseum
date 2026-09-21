// Package battleui はBattle Engineとsessionの状態を、画面がそのまま並べられる形へ変換する。
//
// 対戦のルールも進行も持たない。resolverを呼ばず、Botの判断もせず、描画もしない。
// 「今どうなっているか」をUIが解釈しなくてよい形に直すだけの層。
//
// Ebitengineには依存しない。`internal/anim` と `internal/sprite` は推移的に
// Ebitengineへ届くため、こちらからはimportしない。cueは抽象のまま返し、
// どのanimationに対応させるかはUI側が決める。
//
// # 見せてよい情報
//
// 相手については画面から読める情報だけを返す。技構成と残りPPを持つのは
// OwnPokemonだけで、相手側のPokemonViewにはそのfieldが無い。
// HPは両者とも数値で返し、バーだけ描くか数値も出すかはUI側が決める。
//
// # Snapshotとcueの役割
//
// Snapshotは「落ち着いたあとの値」、cueは「途中経過」。UIはcommandが返した
// Event列からcueを作って順に再生し、queueを消化し終えた時点でSnapshotと一致する。
// Snapshotを読むこと自体はcueを生まないので、毎frame呼んでよい。
package battleui

import (
	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// NoActive はまだ場に出る1体が決まっていないことを表すindex。
//
// lead選択中はどちらのactiveも決まっていない。
const NoActive = -1

// Input はSnapshotを作るのに必要な値。
//
// Sessionそのものは受け取らない。値で受ければ、後のnetwork clientも
// 同じ層を使える。
type Input struct {
	// Phase はsessionの進行段階。
	Phase singleplayer.Phase

	// Teams は配布された3体。lead選択中はこれだけが真。
	Teams [2][battle.TeamSize]battle.Pokemon

	// State は対戦の現在の状態。Startedがfalseのあいだは読まない。
	State battle.BattleState

	// Started はStateが生成済みか。Session.State()の2つ目の戻り値をそのまま渡す。
	Started bool

	// Pending はviewer自身が入力を出して相手を待っている状態。
	Pending Pending
}

// Pending はviewer自身が出した入力の待ち状態。
//
// 「相手が先に出した」ことは含まない。Sessionは誰がsubmit済みかを公開して
// いないので、受理された自分の入力だけをUI側が立てて渡す。
//
// phaseごとに対応するfieldだけを見るため、phaseが変わったあとに古い値が
// 残っていても表示には影響しない。
type Pending struct {
	// Lead はlead選択を出して相手を待っているか。Battleへ移れば意味を持たない。
	Lead bool

	// Action はそのturnの行動を出して相手を待っているか。turnが解決すれば意味を持たない。
	Action bool
}

// View は画面が1 frameで必要とするものをまとめたもの。
type View struct {
	// Phase はsessionの進行段階。
	Phase singleplayer.Phase

	// You はviewer自身の側。技と残りPPを持つ。
	You OwnSide

	// Foe は相手側。技と残りPPを持たない。
	Foe FoeSide

	// Commands はいま出せる選択肢。
	Commands Commands

	// Result は決着の結果。Decidedがfalseなら未決着。
	Result Result
}

// OwnSide はviewer自身の3体。
type OwnSide struct {
	Side battle.Side

	// Active は場に出ている1体のTeam内index。lead選択中はNoActive。
	Active int

	// Team は配布順の3体。交代してもこの並びは変わらない。
	Team [battle.TeamSize]OwnPokemon
}

// FoeSide は相手の3体。
type FoeSide struct {
	Side battle.Side

	// Active は場に出ている1体のTeam内index。lead選択中はNoActive。
	Active int

	Team [battle.TeamSize]PokemonView
}

// PokemonView は両者について画面に出してよい情報。
//
// 技と残りPPを持たない。相手側はこの型で返すため、
// 未公開の情報を載せるコードを書けない。
type PokemonView struct {
	// Species はキャラクター定義への参照。Composite Spriteを引くときのキーでもある。
	Species battle.SpeciesID

	// Emoji は一覧やlogで見分けるための代表Emoji。最終的な見た目ではない。
	Emoji string

	Level int

	// HP と MaxHP は現在HPと最大HP。バーだけ描くか数値も出すかはUIが決める。
	HP    int
	MaxHP int

	// Status は表示用の状態。ひんしを状態異常より優先する。
	Status StatusView

	// Fainted は戦闘不能か。
	Fainted bool

	// Active は場に出ているか。
	Active bool
}

// OwnPokemon はviewer自身の1体。技と残りPPを持つ。
type OwnPokemon struct {
	PokemonView

	// Moves は4つの技枠。空き枠はMoveが空でDisabledになる。
	Moves [battle.MoveSlots]MoveView
}

// MoveView は技枠1つ分の表示。
type MoveView struct {
	// Move は技の識別子。空き枠では空。
	Move battle.MoveID

	PP    int
	MaxPP int

	// Disabled はいま選べないか。PPが尽きた技と空き枠が該当する。
	Disabled bool
}

// StatusView は表示用の状態。
//
// Generation Iではfaintしてもmajor statusは消えないが、画面ではひんしを
// 優先して見せる（Battle Rules Specificationのdecision）。その優先を
// ここで解決しておき、UIに判断させない。
type StatusView int

const (
	// StatusNone は状態異常なし。
	StatusNone StatusView = iota
	StatusBurn
	StatusFreeze
	StatusParalysis
	StatusPoison
	StatusSleep
	// StatusFainted はひんし。major statusより優先する。
	StatusFainted
)

// String はStatusViewの名前を返す。testとlogの出力に使う。
// 画面へ出す文言ではない。
func (s StatusView) String() string {
	switch s {
	case StatusNone:
		return "none"
	case StatusBurn:
		return "burn"
	case StatusFreeze:
		return "freeze"
	case StatusParalysis:
		return "paralysis"
	case StatusPoison:
		return "poison"
	case StatusSleep:
		return "sleep"
	case StatusFainted:
		return "fainted"
	default:
		return "unknown"
	}
}

// Result は決着の結果。
type Result struct {
	// Decided は決着したか。
	Decided bool

	// Draw は引き分けか。
	Draw bool

	// Winner は勝った側。DecidedかつDrawでないときだけ意味を持つ。
	Winner battle.Side
}

// Snapshot はviewerから見た画面の状態を返す。
//
// viewerが両者のどちらでもなければfalseを返す。
// 何度呼んでも同じ値を返すだけで、cueは生まない。
func Snapshot(in Input, viewer battle.Side) (View, bool) {
	if viewer != battle.Player1 && viewer != battle.Player2 {
		return View{}, false
	}

	return View{
		Phase:    in.Phase,
		You:      ownSide(in, viewer),
		Foe:      foeSide(in, viewer.Opponent()),
		Commands: commands(in, viewer),
		Result:   result(in),
	}, true
}

// ownSide はviewer自身の側を作る。
func ownSide(in Input, side battle.Side) OwnSide {
	team := teamOf(in, side)
	active := activeOf(in, side)

	own := OwnSide{Side: side, Active: active}
	for i, pokemon := range team {
		own.Team[i] = OwnPokemon{
			PokemonView: pokemonView(pokemon, i == active),
			Moves:       moveViews(pokemon),
		}
	}
	return own
}

// foeSide は相手側を作る。技と残りPPは載せない。
func foeSide(in Input, side battle.Side) FoeSide {
	team := teamOf(in, side)
	active := activeOf(in, side)

	foe := FoeSide{Side: side, Active: active}
	for i, pokemon := range team {
		foe.Team[i] = pokemonView(pokemon, i == active)
	}
	return foe
}

// teamOf はその時点で正しい3体を返す。
//
// 対戦が始まる前は配布された3体、始まったあとはBattleStateが持つ現在の3体。
func teamOf(in Input, side battle.Side) [battle.TeamSize]battle.Pokemon {
	if in.Started {
		return in.State.Players[side].Team
	}
	return in.Teams[side]
}

// activeOf は場に出ている1体のindexを返す。lead選択中はNoActive。
func activeOf(in Input, side battle.Side) int {
	if !in.Started {
		return NoActive
	}
	return in.State.Players[side].Active
}

// pokemonView は1体分の表示を作る。
func pokemonView(p battle.Pokemon, active bool) PokemonView {
	return PokemonView{
		Species: p.Species,
		Emoji:   emojiOf(p.Species),
		Level:   p.Level,
		HP:      p.CurrentHP,
		MaxHP:   p.Stats.HP,
		Status:  statusView(p),
		Fainted: p.Fainted(),
		Active:  active,
	}
}

// moveViews は4つの技枠の表示を作る。
func moveViews(p battle.Pokemon) [battle.MoveSlots]MoveView {
	var views [battle.MoveSlots]MoveView
	for i := range p.Moves {
		slot := p.Moves[i]
		views[i] = MoveView{
			Move:     slot.Move,
			PP:       slot.PP,
			MaxPP:    slot.MaxPP,
			Disabled: !slot.Usable(),
		}
	}
	return views
}

// statusView はひんしを優先した表示用の状態を返す。
func statusView(p battle.Pokemon) StatusView {
	if p.Fainted() {
		return StatusFainted
	}
	switch p.Status {
	case battle.Burn:
		return StatusBurn
	case battle.Freeze:
		return StatusFreeze
	case battle.Paralysis:
		return StatusParalysis
	case battle.Poison:
		return StatusPoison
	case battle.Sleep:
		return StatusSleep
	default:
		return StatusNone
	}
}

// result は決着の結果を返す。
func result(in Input) Result {
	if !in.Started {
		return Result{}
	}
	switch in.State.Status {
	case battle.Player1Won:
		return Result{Decided: true, Winner: battle.Player1}
	case battle.Player2Won:
		return Result{Decided: true, Winner: battle.Player2}
	case battle.Draw:
		return Result{Decided: true, Draw: true}
	default:
		return Result{}
	}
}

// emojiOf は代表Emojiを返す。定義が無ければ空を返す。
func emojiOf(id battle.SpeciesID) string {
	character, ok := roster.Lookup(id)
	if !ok {
		return ""
	}
	return character.Emoji
}
