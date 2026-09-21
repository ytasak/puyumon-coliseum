package game

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// 画面の色。背景と紛れず、押せるものと押せないものが見分けられる組み合わせにする。
var (
	panelFillColor   = color.RGBA{R: 0x22, G: 0x2d, B: 0x42, A: 0xff}
	panelBorderColor = color.RGBA{R: 0x4a, G: 0x5c, B: 0x7a, A: 0xff}

	buttonFillColor     = color.RGBA{R: 0x2c, G: 0x3a, B: 0x52, A: 0xff}
	buttonBorderColor   = color.RGBA{R: 0x5a, G: 0x6e, B: 0x90, A: 0xff}
	buttonDisabledColor = color.RGBA{R: 0x16, G: 0x1d, B: 0x2b, A: 0xff}

	// disabledVeilColor は押せない枠へかける半透明。文字ごと沈ませる。
	disabledVeilColor = color.RGBA{R: 0x1b, G: 0x24, B: 0x38, A: 0xb0}

	hpTrackColor = color.RGBA{R: 0x14, G: 0x1b, B: 0x28, A: 0xff}
	hpHighColor  = color.RGBA{R: 0x4a, G: 0xd0, B: 0x6a, A: 0xff}
	hpMidColor   = color.RGBA{R: 0xf0, G: 0xc0, B: 0x40, A: 0xff}
	hpLowColor   = color.RGBA{R: 0xe8, G: 0x5a, B: 0x4a, A: 0xff}
)

// HPバーの形（論理座標）。
const (
	hpBarHeight = 8
	hpBarInset  = 10

	// hpMidPercent, hpLowPercent は色が変わる残量。
	hpMidPercent = 50
	hpLowPercent = 20
)

// draw は対戦画面を描く。状態は変えない。
func (s *battleScene) draw(screen *ebiten.Image, sprites *sprite.Renderer) {
	for _, side := range battleSides {
		s.drawActive(screen, sprites, side)
	}
	for i := range s.particles.Len() {
		character, transform := s.particles.At(i)
		sprites.Draw(screen, character, transform)
	}
	for _, side := range battleSides {
		s.drawInfo(screen, side)
	}

	s.drawMessage(screen)
	s.drawCommands(screen)
}

// drawActive は場に出ている1体を描く。
//
// lead選択中はまだ誰も出ていない。倒れた個体は描かず、枠の表示で伝える。
func (s *battleScene) drawActive(screen *ebiten.Image, sprites *sprite.Renderer, side battle.Side) {
	index, visible := s.activeSprite(side)
	if !visible {
		return
	}

	pokemon, ok := s.pokemonAt(side, index)
	if !ok {
		return
	}
	sprites.Draw(screen, characterFor(pokemon.Species), s.players[side].Transform(spriteAnchors[side]))
}

// activeSprite は場に描く1体のteam indexを返す。
//
// 交代で下がってから次が出るまでのあいだと、倒れたあとは誰も描かない。
// 枠に出す情報は残すので、描くかどうかだけをここで決める。
func (s *battleScene) activeSprite(side battle.Side) (int, bool) {
	if side != viewer && side != foe {
		return 0, false
	}
	index := s.shown.active[side]
	if !inTeam(index) || !s.shown.onField[side] || s.shown.fainted[side][index] {
		return 0, false
	}
	return index, true
}

// drawInfo は名前・Level・HP・状態・控えを枠に描く。
func (s *battleScene) drawInfo(screen *ebiten.Image, side battle.Side) {
	panel := infoPanels[side]
	drawPanel(screen, panel, panelFillColor, panelBorderColor)

	index := s.shown.active[side]
	if !inTeam(index) {
		// lead選択中。まだ出ていないので控えだけを並べる。
		drawText(screen, s.face, "準備中", panel.Min.X+hpBarInset, panel.Min.Y+6)
		s.drawReserves(screen, side, panel, -1)
		return
	}

	pokemon, ok := s.pokemonAt(side, index)
	if !ok {
		return
	}

	header := fmt.Sprintf("%s Lv%d", speciesName(pokemon.Species), pokemon.Level)
	if tag := statusName(s.statusOf(side, index)); tag != "" {
		header += " " + tag
	}
	drawText(screen, s.face, header, panel.Min.X+hpBarInset, panel.Min.Y+6)

	hp := s.shown.hp[side][index]
	barTop := panel.Min.Y + 6 + uifont.Size + 4
	drawHPBar(screen, image.Rect(panel.Min.X+hpBarInset, barTop, panel.Max.X-hpBarInset, barTop+hpBarHeight), hp, pokemon.MaxHP)
	drawText(screen, s.face, fmt.Sprintf("%d/%d", hp, pokemon.MaxHP), panel.Min.X+hpBarInset, barTop+hpBarHeight+2)

	s.drawReserves(screen, side, panel, index)
}

// drawReserves は控えの簡易状態を右下へ並べる。
//
// 代表Emojiを小さく出し、倒れていれば枠の色で示す。
func (s *battleScene) drawReserves(screen *ebiten.Image, side battle.Side, panel image.Rectangle, active int) {
	const (
		dotSize = 18
		dotGap  = 6
	)

	x := panel.Max.X - hpBarInset - dotSize
	y := panel.Max.Y - hpBarInset - dotSize
	for index := battle.TeamSize - 1; index >= 0; index-- {
		if index == active {
			continue
		}
		pokemon, ok := s.pokemonAt(side, index)
		if !ok {
			continue
		}

		box := image.Rect(x, y, x+dotSize, y+dotSize)
		fill := buttonFillColor
		if s.shown.fainted[side][index] {
			fill = buttonDisabledColor
		}
		drawPanel(screen, box, fill, panelBorderColor)
		drawText(screen, s.face, shortLabel(pokemon.Species), box.Min.X+1, box.Min.Y+1)

		x -= dotSize + dotGap
	}
}

// drawMessage は対戦の経過を1行で出す。
func (s *battleScene) drawMessage(screen *ebiten.Image) {
	box := image.Rect(commandMargin, messageTop, LogicalWidth-commandMargin, messageBottom)
	drawPanel(screen, box, panelFillColor, panelBorderColor)
	drawText(screen, s.face, s.message, box.Min.X+10, box.Min.Y+(box.Dy()-uifont.Size)/2)
}

// drawCommands は選択肢を描く。
func (s *battleScene) drawCommands(screen *ebiten.Image) {
	for _, b := range s.visibleButtons() {
		rect := b.rect()
		fill := buttonFillColor
		if b.disabled {
			fill = buttonDisabledColor
		}
		drawPanel(screen, rect, fill, buttonBorderColor)
		drawCenteredText(screen, s.face, b.label, float64(rect.Min.X+rect.Dx()/2), rect.Min.Y+(rect.Dy()-uifont.Size)/2)

		if b.disabled {
			// 組み込みフォントは色を選べないので、文字の上から半透明をかけて
			// まとめて沈ませる。枠の色だけでは押せないことが分かりにくかった。
			dimPanel(screen, rect)
		}
	}
}

// pokemonAt は表示に使う1体を返す。
//
// 自分側だけが技を持つので型が違う。ここでは共通部分だけを見る。
func (s *battleScene) pokemonAt(side battle.Side, index int) (battleui.PokemonView, bool) {
	if !inTeam(index) {
		return battleui.PokemonView{}, false
	}
	switch side {
	case s.view.You.Side:
		return s.view.You.Team[index].PokemonView, true
	case s.view.Foe.Side:
		return s.view.Foe.Team[index], true
	default:
		return battleui.PokemonView{}, false
	}
}

// statusOf は見せている状態を返す。
//
// cueを消化しながら更新した値を使う。Snapshotから読むと、消化しきるまで
// 古い状態が残る。ひんしは状態異常より優先する。
func (s *battleScene) statusOf(side battle.Side, index int) battleui.StatusView {
	if !inTeam(index) {
		return battleui.StatusNone
	}
	if s.shown.fainted[side][index] {
		return battleui.StatusFainted
	}
	return s.shown.status[side][index]
}

// statusViewOf は状態異常を表示用の値へ移す。
func statusViewOf(status battle.MajorStatus) battleui.StatusView {
	switch status {
	case battle.Burn:
		return battleui.StatusBurn
	case battle.Freeze:
		return battleui.StatusFreeze
	case battle.Paralysis:
		return battleui.StatusParalysis
	case battle.Poison:
		return battleui.StatusPoison
	case battle.Sleep:
		return battleui.StatusSleep
	default:
		return battleui.StatusNone
	}
}

// shortLabel は控えの枠に収まる略記を返す。
//
// **rune単位で切る。** byteで切ると日本語が文字の途中で割れて豆腐になる。
// 枠は18pxしかないので全角1文字だけ入る。
func shortLabel(species battle.SpeciesID) string {
	for _, r := range speciesName(species) {
		return string(r)
	}
	return ""
}

// drawPanel は枠を塗って縁を描く。
func drawPanel(screen *ebiten.Image, rect image.Rectangle, fill, border color.Color) {
	x, y := float32(rect.Min.X), float32(rect.Min.Y)
	w, h := float32(rect.Dx()), float32(rect.Dy())

	vector.DrawFilledRect(screen, x, y, w, h, fill, false)
	vector.StrokeRect(screen, x, y, w, h, 1, border, false)
}

// dimPanel は枠の上に半透明をかけて沈ませる。
func dimPanel(screen *ebiten.Image, rect image.Rectangle) {
	vector.DrawFilledRect(screen,
		float32(rect.Min.X), float32(rect.Min.Y), float32(rect.Dx()), float32(rect.Dy()),
		disabledVeilColor, false)
}

// drawHPBar は残量に応じて色の変わるHPバーを描く。
func drawHPBar(screen *ebiten.Image, rect image.Rectangle, hp, maxHP int) {
	x, y := float32(rect.Min.X), float32(rect.Min.Y)
	w, h := float32(rect.Dx()), float32(rect.Dy())
	vector.DrawFilledRect(screen, x, y, w, h, hpTrackColor, false)

	if maxHP <= 0 || hp <= 0 {
		vector.StrokeRect(screen, x, y, w, h, 1, panelBorderColor, false)
		return
	}

	percent := hp * 100 / maxHP
	fill := hpHighColor
	switch {
	case percent <= hpLowPercent:
		fill = hpLowColor
	case percent <= hpMidPercent:
		fill = hpMidColor
	}

	filled := w * float32(hp) / float32(maxHP)
	vector.DrawFilledRect(screen, x, y, filled, h, fill, false)
	vector.StrokeRect(screen, x, y, w, h, 1, panelBorderColor, false)
}

// faintParticle は倒れたときに出すEmoji。
const faintParticle = "💥"

// statusParticle は状態異常に合わせたEmojiを返す。無ければ空。
func statusParticle(status battle.MajorStatus) string {
	switch status {
	case battle.Freeze:
		return "❄️"
	case battle.Paralysis:
		return "⚡"
	case battle.Sleep:
		return "💤"
	default:
		return ""
	}
}
