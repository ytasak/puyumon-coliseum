package game

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// HPバーの形（論理座標）。
const (
	hpBarHeight = 8
	hpBarInset  = 12

	// hpMidPercent, hpLowPercent は縞の細かさが変わる残量。
	//
	// 4階調では色で危険度を示せないので、塗りの密度で示す。
	hpMidPercent = 50
	hpLowPercent = 20

	// reserveChipSize, reserveChipGap は控えを並べる小さな枠の大きさと間隔。
	//
	// 枠には略記を全角1文字だけ入れる。**行の高さ（ascent + descent）は
	// この枠より大きい**ので、左上を指定して描くと下へはみ出す。
	// drawTextInBox でインクを枠の中央へ寄せて収める。
	reserveChipSize = 18
	reserveChipGap  = 6
)

// draw は対戦画面を描く。状態は変えない。
func (s *battleScene) draw(screen *ebiten.Image, sprites *sprite.Renderer) {
	// 地面の線。奥（相手）と手前（自分）を分けて対面の奥行きを出す。
	vector.DrawFilledRect(screen, float32(groundLine.Min.X), float32(groundLine.Min.Y),
		float32(groundLine.Dx()), float32(groundLine.Dy()), toneLight, false)

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

	s.drawBottom(screen)
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

// 情報枠の中の段（枠の上端からの位置）。
//
// **段のyをここで一度に決める。** 描く場所ごとに足し算していくと、名前が長い
// ときや状態が付いたときに段どうしが重なる。
const (
	infoNameTop   = 8
	infoBarTop    = 34
	infoNumberTop = 46
	infoChipTop   = 66
)

// drawInfo は名前・Level・HP・状態・控えを枠に描く。
//
// HPの数値は**両sideに出す**。相手の残りHPは対戦中ずっと公開されている情報で、
// 見た目を変えるついでに減らさない。
func (s *battleScene) drawInfo(screen *ebiten.Image, side battle.Side) {
	panel := infoPanels[side]
	drawWindow(screen, panel)

	left := panel.Min.X + hpBarInset
	right := panel.Max.X - hpBarInset

	index := s.shown.active[side]
	if !inTeam(index) {
		// lead選択中。まだ出ていないので控えだけを並べる。
		drawText(screen, s.face, "準備中", left, panel.Min.Y+infoNameTop)
		s.drawReserves(screen, side, panel, -1)
		return
	}

	pokemon, ok := s.pokemonAt(side, index)
	if !ok {
		return
	}

	drawText(screen, s.face, speciesName(pokemon.Species), left, panel.Min.Y+infoNameTop)
	drawRightText(screen, s.face, fmt.Sprintf("Lv%d", pokemon.Level), right, panel.Min.Y+infoNameTop)

	hp := s.shown.hp[side][index]
	drawHPBar(screen, image.Rect(left, panel.Min.Y+infoBarTop, right, panel.Min.Y+infoBarTop+hpBarHeight), hp, pokemon.MaxHP)
	drawText(screen, s.face, fmt.Sprintf("%d/%d", hp, pokemon.MaxHP), left, panel.Min.Y+infoNumberTop)

	// 状態は数値と同じ段の右端へ。名前の長さに影響されない。
	if tag := statusName(s.statusOf(side, index)); tag != "" {
		drawStatusTag(screen, s.face, tag, right, panel.Min.Y+infoNumberTop)
	}

	s.drawReserves(screen, side, panel, index)
}

// drawStatusTag は状態を反転したラベルで描く。
//
// 文字色だけを変えても4階調では目立たないので、地と文字を入れ替える。
func drawStatusTag(screen *ebiten.Image, face *text.GoTextFace, tag string, right, top int) {
	const padding = 6

	width := int(textWidth(face, tag)) + padding*2
	box := image.Rect(right-width, top, right, top+uifont.Size+2)
	vector.DrawFilledRect(screen, float32(box.Min.X), float32(box.Min.Y),
		float32(box.Dx()), float32(box.Dy()), toneDarkest, false)
	drawTextWithColor(screen, face, tag, box.Min.X+padding, top, toneLightest)
}

// drawReserves は控えの簡易状態を右下へ並べる。
//
// 代表Emojiを小さく出し、倒れていれば枠の色で示す。
func (s *battleScene) drawReserves(screen *ebiten.Image, side battle.Side, panel image.Rectangle, active int) {
	x := panel.Max.X - hpBarInset - reserveChipSize
	y := panel.Max.Y - hpBarInset - reserveChipSize
	for index := battle.TeamSize - 1; index >= 0; index-- {
		if index == active {
			continue
		}
		pokemon, ok := s.pokemonAt(side, index)
		if !ok {
			continue
		}

		box := image.Rect(x, y, x+reserveChipSize, y+reserveChipSize)
		vector.DrawFilledRect(screen, float32(box.Min.X), float32(box.Min.Y),
			float32(box.Dx()), float32(box.Dy()), toneDarkest, false)
		vector.DrawFilledRect(screen, float32(box.Min.X+1), float32(box.Min.Y+1),
			float32(box.Dx()-2), float32(box.Dy()-2), toneLightest, false)
		drawTextInBox(screen, s.face, shortLabel(pokemon.Species), box)

		// **ひんしは×で示す。** 明度を落とすだけだと状態異常の表示と紛れる。
		if s.shown.fainted[side][index] {
			drawChipCross(screen, box)
		}

		x -= reserveChipSize + reserveChipGap
	}
}

// drawBottom は画面下部の窓と、その中の文言・選択肢を描く。
//
// **文言と選択肢を1つの窓へ入れる。** 別々の箱に見えると初代の画面から離れる。
func (s *battleScene) drawBottom(screen *ebiten.Image) {
	drawWindow(screen, bottomWindow)

	buttons := s.visibleButtons()
	root := s.showsRootMenu()

	if area, ok := s.messageBox(); ok {
		drawText(screen, s.face, s.message, area.Min.X+4, area.Min.Y+4)
	}
	if root {
		drawWindow(screen, rootMenu())
	}
	for _, b := range buttons {
		s.drawCommand(screen, b)
	}
}

// messageBox は文言を出す範囲を返す。出さない場面ではfalseを返す。
//
// cueを消化しているあいだは窓の全幅を使う。長い経過文を選択肢の脇の
// 狭い幅へ押し込むと切れる。
func (s *battleScene) messageBox() (image.Rectangle, bool) {
	if s.message == "" {
		return image.Rectangle{}, false
	}
	if s.busy() {
		return bottomInner, true
	}
	switch {
	case s.showsRootMenu():
		return messageArea(true), true
	case s.showsList():
		return image.Rect(bottomInner.Min.X, bottomInner.Min.Y, bottomInner.Max.X, listArea.Min.Y), true
	default:
		// 技の選択。技名そのものが案内になるので文言は出さない。
		return image.Rectangle{}, false
	}
}

// drawCommand は選択肢を1つ描く。
//
// 独立したボタンにせず、窓を細線で仕切ったものとして見せる。
func (s *battleScene) drawCommand(screen *ebiten.Image, b button) {
	rect := b.rect()

	// 枠線ではなく仕切り線で区切る。右端と下端が窓の縁に接する枠には引かない。
	if rect.Max.X < bottomInner.Max.X {
		drawDivider(screen, rect.Max.X-1, rect.Min.Y+4, 1, rect.Dy()-8)
	}
	if rect.Max.Y < bottomInner.Max.Y {
		drawDivider(screen, rect.Min.X+6, rect.Max.Y-1, rect.Dx()-12, 1)
	}

	top := rect.Min.Y + (rect.Dy()-uifont.Size)/2
	drawText(screen, s.face, b.label, rect.Min.X+12, top)
	if b.detail != "" {
		drawRightText(screen, s.face, b.detail, rect.Max.X-12, top)
	}

	// **押せないことは斜線で示す。** 4階調では明度を1段落としても
	// 隣の階調と紛れる。線が入っているかどうかは階調に関係なく分かる。
	if b.disabled {
		drawHatch(screen, rect)
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

// drawHPBar は残量を塗りの長さと縞の細かさで示す。
//
// **色は変えない。** 4階調では残量ごとに色相を割り当てられないので、危ないほど
// 縞を細かくする。塗りの長さだけでも読めるが、少なくなったことに気づきやすくする。
func drawHPBar(screen *ebiten.Image, rect image.Rectangle, hp, maxHP int) {
	x, y := float32(rect.Min.X), float32(rect.Min.Y)
	w, h := float32(rect.Dx()), float32(rect.Dy())

	vector.DrawFilledRect(screen, x-1, y-1, w+2, h+2, toneDarkest, false)
	vector.DrawFilledRect(screen, x, y, w, h, toneLightest, false)
	if maxHP <= 0 || hp <= 0 {
		return
	}

	filled := int(float32(rect.Dx()) * float32(hp) / float32(maxHP))
	if filled <= 0 {
		return
	}
	bar := image.Rect(rect.Min.X, rect.Min.Y, rect.Min.X+filled, rect.Max.Y)
	vector.DrawFilledRect(screen, float32(bar.Min.X), float32(bar.Min.Y),
		float32(bar.Dx()), float32(bar.Dy()), toneDark, false)

	switch percent := hp * 100 / maxHP; {
	case percent <= hpLowPercent:
		drawStripes(screen, bar, 4)
	case percent <= hpMidPercent:
		drawStripes(screen, bar, 8)
	}
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
