package game

import (
	"fmt"
	"image"

	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// 画面の区切り（論理座標）。640x360の横持ちを前提にする。
const (
	// fieldBottom は盤面の下端。ここまでにキャラクターと両者の状態を描く。
	fieldBottom = 216
)

// 画面下部は1つの窓として扱う。
//
// **独立したボタンを並べない。** 窓の内側を細線で仕切って選択肢にする。
// 文言と選択肢が別々の箱に見えると、初代の画面から離れる。
const (
	commandMargin = 8

	bottomTop    = 220
	bottomBottom = 352

	// bottomPadding は窓の枠から内側の余白。枠(3px) + 内線(2px) + 3px。
	bottomPadding = windowBorder + windowInset + 3

	// rootMenuWidth は通常選択時に右へ置く小窓の幅。
	rootMenuWidth = 236

	// bottomPromptHeight は選択肢の上へ残す案内文1行ぶんの高さ。
	bottomPromptHeight = 26
)

// bottomWindow は下段の窓。文言も選択肢もこの中に入る。
var bottomWindow = image.Rect(commandMargin, bottomTop, LogicalWidth-commandMargin, bottomBottom)

// bottomInner は窓の内側で、文字と選択肢を置ける範囲。
var bottomInner = bottomWindow.Inset(bottomPadding)

// listArea は lead・交代先を並べる範囲。上に案内文の1行を残す。
//
// 技の選択は4つの技名そのものが案内になるので、内側を丸ごと使う。
// 誰を出すかを選ぶ場面は、何を選んでいるのかを文で言わないと分からない。
var listArea = image.Rect(bottomInner.Min.X, bottomInner.Min.Y+bottomPromptHeight,
	bottomInner.Max.X, bottomInner.Max.Y)

// メッセージ行の範囲。cue再生中は窓の全幅を使う。
var (
	messageTop    = bottomInner.Min.Y
	messageBottom = bottomInner.Max.Y
)

// commandGrid は下段の内側を cols x rows に分けたときの、index番目の枠を返す。
//
// **描画と当たり判定はこの1か所から作る。** 別々に計算すると、見えている
// ものと押せるものがずれる。
func commandGrid(area image.Rectangle, cols, rows, index int) image.Rectangle {
	if cols <= 0 || rows <= 0 || index < 0 || index >= cols*rows {
		return image.Rectangle{}
	}
	cw, ch := area.Dx()/cols, area.Dy()/rows
	col, row := index%cols, index/cols
	x, y := area.Min.X+col*cw, area.Min.Y+row*ch
	return image.Rect(x, y, x+cw, y+ch)
}

// moveCellWidth は技の枠1つの幅。ラベルが収まるかの判断に使う。
func moveCellWidth() int {
	return commandGrid(bottomInner, moveCols, moveRows, 0).Dx()
}

// rootMenu は通常選択時に右へ出す小窓。左は対戦文言に残す。
func rootMenu() image.Rectangle {
	return image.Rect(bottomInner.Max.X-rootMenuWidth, bottomInner.Min.Y,
		bottomInner.Max.X, bottomInner.Max.Y)
}

// messageArea は文言を出せる範囲を返す。
//
// 選択肢が右の小窓に出ているあいだは、そこへ食い込まない幅にする。
func messageArea(withRootMenu bool) image.Rectangle {
	area := bottomInner
	if withRootMenu {
		area.Max.X = rootMenu().Min.X - 8
	}
	return area
}

// 下段の枠の割り方。
//
// どれも1枠が現行の192x36より広い。雰囲気のためにタップ領域を小さくしない。
const (
	// moveCols, moveRows は技の選択。4技 + わるあがき + もどる が収まる。
	moveCols = 2
	moveRows = 3

	// listCols, listRows は lead・交代先・新しい対戦。
	listCols = 3
	listRows = 2

	// rootRows は通常選択時の右の小窓。たたかう / こうたい の2つ。
	rootRows = 2
)

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
	// box は描画にも当たり判定にも使う枠。
	box image.Rectangle

	// label は左へ出す文字。detail は右へ出す補助情報（タイプと残りPP）。
	label  string
	detail string

	disabled bool
	action   buttonAction
	command  command
}

// rect はボタンの位置を返す。
func (b button) rect() image.Rectangle {
	return b.box
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
		return []button{{
			box:     commandGrid(listArea, listCols, listRows, 0),
			label:   "もう一度",
			command: command{kind: commandNewMatch},
		}}

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
			box:     commandGrid(listArea, listCols, listRows, i),
			label:   speciesName(option.Species),
			command: command{kind: commandLead, index: option.Index},
		})
	}
	return buttons
}

// rootButtons はFightとSwitchを並べる。
func rootButtons(commands battleui.Commands) []button {
	menu := rootMenu()
	return []button{
		{box: commandGrid(menu, 1, rootRows, 0), label: "たたかう", action: actionOpenFight},
		{
			box:      commandGrid(menu, 1, rootRows, 1),
			label:    "こうたい",
			disabled: len(commands.Switches) == 0,
			action:   actionOpenSwitch,
		},
	}
}

// fightButtons は4つの技と、Struggleと、戻るを並べる。
func fightButtons(commands battleui.Commands) []button {
	buttons := make([]button, 0, moveCols*moveRows)
	for slot, move := range commands.Moves {
		buttons = append(buttons, button{
			box:      commandGrid(bottomInner, moveCols, moveRows, slot),
			label:    moveName(move.Move),
			detail:   moveDetail(move),
			disabled: move.Disabled,
			command:  command{kind: commandMove, index: slot},
		})
	}
	if commands.Struggle {
		buttons = append(buttons, button{
			box:     commandGrid(bottomInner, moveCols, moveRows, len(commands.Moves)),
			label:   "わるあがき",
			command: command{kind: commandStruggle},
		})
	}
	// もどるは最後の枠。技が4つでも わるあがき が出ても、位置が動かない。
	return append(buttons, button{
		box:    commandGrid(bottomInner, moveCols, moveRows, moveCols*moveRows-1),
		label:  "もどる",
		action: actionBack,
	})
}

// switchButtons は交代先を並べる。
func switchButtons(options []battleui.TeamOption) []button {
	buttons := make([]button, 0, len(options))
	for i, option := range options {
		buttons = append(buttons, button{
			box:     commandGrid(listArea, listCols, listRows, i),
			label:   speciesName(option.Species),
			command: command{kind: commandSwitch, index: option.Index},
		})
	}
	return buttons
}

// backButton は1つ上の階層へ戻るボタン。いつも右下に置く。
func backButton() button {
	return button{
		box:    commandGrid(listArea, listCols, listRows, listCols*listRows-1),
		label:  "もどる",
		action: actionBack,
	}
}

// showsRootMenu は右の小窓（たたかう / こうたい）を出す場面かを返す。
func (s *battleScene) showsRootMenu() bool {
	if s.busy() {
		return false
	}
	return s.view.Commands.Kind == battleui.CommandChooseAction && s.menu == menuRoot
}

// showsList は誰を出すかを選ぶ場面かを返す。案内文の1行を残す。
func (s *battleScene) showsList() bool {
	if s.busy() {
		return false
	}
	switch s.view.Commands.Kind {
	case battleui.CommandChooseLead, battleui.CommandChooseReplacement, battleui.CommandFinished:
		return true
	case battleui.CommandChooseAction:
		return s.menu == menuSwitch
	default:
		return false
	}
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

// moveDetail は技の枠の右へ出すタイプと残りPPを返す。
//
// 技名は左、これは右へ出す。名前の長さが変わっても右端が揃う。
// タイプは定義から引くだけで、相性の判断はしない。
func moveDetail(move battleui.MoveView) string {
	if move.Move == "" {
		return ""
	}

	detail := fmt.Sprintf("%d/%d", move.PP, move.MaxPP)
	if definition, err := roster.Data().LookupMove(move.Move); err == nil {
		detail = fmt.Sprintf("%s %s", typeName(definition.Type), detail)
	}
	return detail
}

// 盤面の配置（論理座標）。手前を大きく、奥を小さくして向きを分かるようにする。
var (
	spriteAnchors = [2]sprite.Transform{
		viewer: {X: 150, Y: 158, Scale: 104},
		foe:    {X: 516, Y: 84, Scale: 72},
	}

	// infoPanels は名前・Level・HP・状態・控えを出す枠。
	//
	// 高さは中の4段（名前 / HPバー / 数値・状態 / 控え）が収まるように取る。
	// 控えの枠は18pxあるので、infoChipTop + 18 が枠の高さを超えないこと。
	infoPanels = [2]image.Rectangle{
		viewer: image.Rect(348, 118, 624, 206),
		foe:    image.Rect(16, 14, 292, 102),
	}

	// groundLine は奥と手前を分ける地面の線。対面の奥行きを出す。
	groundLine = image.Rect(0, 138, LogicalWidth, 140)
)

const (
	// particleSize はEmoji particleの大きさ。
	particleSize = 28
)
