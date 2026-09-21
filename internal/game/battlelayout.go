package game

import (
	"fmt"
	"image"
	"strings"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// 画面の区切り（論理座標）。640x360の横持ちを前提にする。
const (
	// fieldBottom は盤面の下端。ここまでにキャラクターと両者の状態を描く。
	fieldBottom = 216

	// messageTop, messageBottom は対戦の経過を出す行の範囲。
	messageTop    = 220
	messageBottom = 258
)

// コマンドのボタンの並び。3列2段の6枠を、画面ごとに使い分ける。
const (
	commandCols   = 3
	commandRows   = 2
	commandCells  = commandCols * commandRows
	commandMargin = 20
	commandGap    = 12
	commandTop    = 266
	commandHeight = 36

	cellWidth = (LogicalWidth - commandMargin*2 - commandGap*(commandCols-1)) / commandCols
)

// commandRects は6枠の位置。左上から右へ、次の段へと並ぶ。
var commandRects = func() [commandCells]image.Rectangle {
	var rects [commandCells]image.Rectangle
	for i := range rects {
		col, row := i%commandCols, i/commandCols
		x := commandMargin + col*(cellWidth+commandGap)
		y := commandTop + row*(commandHeight+commandGap)
		rects[i] = image.Rect(x, y, x+cellWidth, y+commandHeight)
	}
	return rects
}()

// menuState はコマンドのどの階層を開いているか。
type menuState int

const (
	// menuRoot はFight / Switchを選ぶ階層。
	menuRoot menuState = iota
	// menuFight は技を選ぶ階層。
	menuFight
	// menuSwitch は交代先を選ぶ階層。
	menuSwitch
)

// buttonAction は押したときに起きること。
type buttonAction int

const (
	// actionSubmit はcommandをsessionへ送る。
	actionSubmit buttonAction = iota
	// actionOpenFight は技の階層を開く。
	actionOpenFight
	// actionOpenSwitch は交代先の階層を開く。
	actionOpenSwitch
	// actionBack は1つ上の階層へ戻る。
	actionBack
)

// button は画面に出す1つのボタン。
//
// **描画と当たり判定はこの1か所から作る。** 別々に持つと、見えているものと
// 押せるものがずれる。
type button struct {
	cell     int
	label    string
	disabled bool
	action   buttonAction
	command  command
}

// rect はボタンの位置を返す。
func (b button) rect() image.Rectangle {
	return commandRects[b.cell]
}

// buttons はいま画面に出すボタンを返す。
//
// 出す中身はSnapshotのCommandsが決める。ここでは並べ方とラベルだけを決め、
// 何が選べるかを自前で判断しない。
func (s *battleScene) buttons() []button {
	commands := s.view.Commands

	switch commands.Kind {
	case battleui.CommandChooseLead:
		return leadButtons(commands.Leads)

	case battleui.CommandChooseAction:
		switch s.menu {
		case menuFight:
			return fightButtons(commands)
		case menuSwitch:
			return append(switchButtons(commands.Switches), backButton())
		default:
			return rootButtons(commands)
		}

	case battleui.CommandChooseReplacement:
		// 戻る先が無い。控えを出すまで進めない。
		return switchButtons(commands.Switches)

	case battleui.CommandFinished:
		if !commands.NewMatch {
			return nil
		}
		return []button{{cell: 0, label: "NEW MATCH", command: command{kind: commandNewMatch}}}

	default:
		// 相手待ち。押せるものは無い。
		return nil
	}
}

// leadButtons は最初に出す1体の候補を並べる。
func leadButtons(options []battleui.TeamOption) []button {
	buttons := make([]button, 0, len(options))
	for i, option := range options {
		buttons = append(buttons, button{
			cell:    i,
			label:   speciesLabel(option.Species),
			command: command{kind: commandLead, index: option.Index},
		})
	}
	return buttons
}

// rootButtons はFightとSwitchを並べる。
func rootButtons(commands battleui.Commands) []button {
	buttons := []button{{cell: 0, label: "FIGHT", action: actionOpenFight}}
	buttons = append(buttons, button{
		cell:     1,
		label:    "SWITCH",
		disabled: len(commands.Switches) == 0,
		action:   actionOpenSwitch,
	})
	return buttons
}

// fightButtons は4つの技と、Struggleと、戻るを並べる。
func fightButtons(commands battleui.Commands) []button {
	buttons := make([]button, 0, commandCells)
	for slot, move := range commands.Moves {
		buttons = append(buttons, button{
			cell:     slot,
			label:    moveLabel(move),
			disabled: move.Disabled,
			command:  command{kind: commandMove, index: slot},
		})
	}
	if commands.Struggle {
		buttons = append(buttons, button{
			cell:    commandCells - 2,
			label:   "STRUGGLE",
			command: command{kind: commandStruggle},
		})
	}
	return append(buttons, backButton())
}

// switchButtons は交代先を並べる。
func switchButtons(options []battleui.TeamOption) []button {
	buttons := make([]button, 0, len(options))
	for i, option := range options {
		buttons = append(buttons, button{
			cell:    i,
			label:   speciesLabel(option.Species),
			command: command{kind: commandSwitch, index: option.Index},
		})
	}
	return buttons
}

// backButton は1つ上の階層へ戻るボタン。いつも右下に置く。
func backButton() button {
	return button{cell: commandCells - 1, label: "BACK", action: actionBack}
}

// visibleButtons は画面に出すボタンを返す。
//
// **cueを消化しているあいだは何も出さない。** 押せないボタンを見せ続けると、
// 反応しない画面に見える。何が起きているかはメッセージ行が伝える。
// viewのCommandsは書き換えないので、消化しきればそのまま元の選択肢へ戻る。
func (s *battleScene) visibleButtons() []button {
	if s.busy() {
		return nil
	}
	return s.buttons()
}

// tap は押された位置を操作へ変える。受理したらtrueを返す。
//
// 階層を開く・戻るだけの操作ではsessionを呼ばない。
func (s *battleScene) tap(p image.Point) (bool, error) {
	if s.busy() {
		return false, nil
	}

	for _, b := range s.buttons() {
		if !p.In(b.rect()) {
			continue
		}
		if b.disabled {
			return false, nil
		}

		switch b.action {
		case actionOpenFight:
			s.menu = menuFight
			return true, nil
		case actionOpenSwitch:
			s.menu = menuSwitch
			return true, nil
		case actionBack:
			s.menu = menuRoot
			return true, nil
		default:
			return s.submit(b.command)
		}
	}
	return false, nil
}

// speciesLabel は一覧に出す名前。表示名が未確定なのでinternal IDを大文字で使う。
func speciesLabel(species battle.SpeciesID) string {
	return strings.ToUpper(string(species))
}

// moveLabel は技のボタンに出す文字列。
//
// 技名・タイプ・残りPPを1行に収める。タイプは定義から引くだけで、
// 相性の判断はしない。
func moveLabel(move battleui.MoveView) string {
	if move.Move == "" {
		return "-"
	}

	name := strings.ToUpper(string(move.Move))
	label := fmt.Sprintf("%s %d/%d", name, move.PP, move.MaxPP)
	if definition, err := roster.Data().LookupMove(move.Move); err == nil {
		label = fmt.Sprintf("%s %s %d/%d", name, strings.ToUpper(definition.Type.String()), move.PP, move.MaxPP)
	}
	return label
}

// 盤面の配置（論理座標）。手前を大きく、奥を小さくして向きを分かるようにする。
var (
	spriteAnchors = [2]sprite.Transform{
		viewer: {X: 172, Y: 156, Scale: 92},
		foe:    {X: 472, Y: 78, Scale: 70},
	}

	// infoPanels は名前・Level・HP・状態・控えを出す枠。
	infoPanels = [2]image.Rectangle{
		viewer: image.Rect(332, 118, 624, 200),
		foe:    image.Rect(16, 16, 308, 98),
	}
)

const (
	// debugFontCharWidth, debugFontCharHeight は ebitenutil.DebugPrintAt が使う
	// 組み込みフォントの1文字分の大きさ。中央揃えと折り返しの判断に使う。
	debugFontCharWidth  = 6
	debugFontCharHeight = 16

	// particleSize はEmoji particleの大きさ。
	particleSize = 28
)
