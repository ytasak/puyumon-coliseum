package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ytasak/puyumon-coliseum/internal/anim"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// nassy はPoCキャラクター「ナッシー型」。body 🌴 に faces 🥺 😫 🤪 を載せる。
//
// 座標・倍率は見た目で調整した暫定値。元キャラクターの再現ではなく、
// 複数Emojiを合体させるvisual languageの検証が目的なので、
// キャラクターデザインとしては未確定。
var nassy = sprite.NewCharacter(
	sprite.Part{Emoji: "🌴", X: 0, Y: 0, Scale: 1, Z: 0},
	sprite.Part{Emoji: "🥺", X: -0.23, Y: -0.20, Scale: 0.32, Rotation: -0.15, Z: 1},
	sprite.Part{Emoji: "😫", X: 0.01, Y: -0.30, Scale: 0.34, Z: 1},
	sprite.Part{Emoji: "🤪", X: 0.25, Y: -0.17, Scale: 0.32, Rotation: 0.15, Z: 1},
)

// actions は画面下のタップ領域。タップでanimationとparticleを発火する。
//
// Emojiが欠損しないかを実機で見分けられるよう、particleは種類を分けている。
var actions = []struct {
	label    string
	rect     image.Rectangle
	motion   anim.Motion
	particle string
}{
	{label: "ATTACK", rect: image.Rect(24, 272, 208, 336), motion: anim.Attack, particle: "⚡"},
	{label: "HIT", rect: image.Rect(228, 272, 412, 336), motion: anim.Hit, particle: "💥"},
	{label: "EMPHASIS", rect: image.Rect(432, 272, 616, 336), motion: anim.Emphasis, particle: "❄️"},
}

// idleParticle は待機中に自動で出るEmoji。操作しなくても描画が動いていることが分かる。
const idleParticle = "💤"

// PoC sceneのレイアウト（論理座標 / tick）。
const (
	characterX     = 320
	characterY     = 158
	characterScale = 104

	particleSpawnY = 112
	particleSize   = 30

	// idleParticlePeriod は待機中にparticleを出す間隔。
	idleParticlePeriod = anim.TPS * 2

	// tapMarkerRadius はタップ位置に出す印の大きさ。
	tapMarkerRadius = 10
	// tapMarkerReach は印から伸ばす十字線の長さ。
	tapMarkerReach = tapMarkerRadius + 5

	// debugFontCharWidth はebitenutil.DebugPrintAtが使う組み込みフォントの
	// 1文字分の幅と高さ。ラベルを中央揃えするために使う。
	debugFontCharWidth  = 6
	debugFontCharHeight = 16
)

var (
	buttonFillColor   = color.RGBA{R: 0x2c, G: 0x3a, B: 0x52, A: 0xff}
	buttonBorderColor = color.RGBA{R: 0x5a, G: 0x6e, B: 0x90, A: 0xff}
	// tapMarkerColor はタップ位置の印。背景ともキャラクターとも紛れない色にする。
	tapMarkerColor = color.RGBA{R: 0xff, G: 0xd0, B: 0x4a, A: 0xff}
)

// actionAt はpに重なるタップ領域を返す。どこにも当たらなければokがfalse。
func actionAt(p image.Point) (int, bool) {
	for i, action := range actions {
		if p.In(action.rect) {
			return i, true
		}
	}
	return 0, false
}

// updateSpriteDemo は入力を受けてanimationの状態を進める。描画は行わない。
func (g *Game) updateSpriteDemo(ticks uint64) {
	for _, tap := range g.taps() {
		g.tapCount++
		g.lastTap = tap

		if i, ok := actionAt(tap); ok {
			g.player.Play(actions[i].motion)
			g.particles.Spawn(actions[i].particle, characterX, particleSpawnY, particleSize)
		}
	}

	if ticks%idleParticlePeriod == 0 {
		g.particles.Spawn(idleParticle, characterX, particleSpawnY, particleSize)
	}

	g.player.Update()
	g.particles.Update()
}

// drawSpritePoC はキャラクター・particle・操作用のボタン・タップ位置を描く。
func (g *Game) drawSpritePoC(screen *ebiten.Image) {
	base := sprite.Transform{X: characterX, Y: characterY, Scale: characterScale}
	g.sprites.Draw(screen, nassy, g.player.Transform(base))

	// particleはキャラクターより手前に出す。
	for i := range g.particles.Len() {
		character, transform := g.particles.At(i)
		g.sprites.Draw(screen, character, transform)
	}

	g.drawActions(screen)
	g.drawTapMarker(screen)
}

// drawActions はタップ領域を枠付きで描く。
// 枠が見えていれば、どこを押したつもりかとどこが反応したかを突き合わせられる。
func (g *Game) drawActions(screen *ebiten.Image) {
	for _, action := range actions {
		r := action.rect
		x, y := float32(r.Min.X), float32(r.Min.Y)
		w, h := float32(r.Dx()), float32(r.Dy())

		vector.DrawFilledRect(screen, x, y, w, h, buttonFillColor, false)
		vector.StrokeRect(screen, x, y, w, h, 1, buttonBorderColor, false)
		drawCenteredLabel(screen, action.label,
			float64(r.Min.X+r.Dx()/2), r.Min.Y+r.Dy()/2-debugFontCharHeight/2)
	}
}

// drawTapMarker は最後にタップした位置へ印を出す。
//
// 指を置いた場所と印がずれていれば、論理座標への変換が合っていない。
// iframeやdevicePixelRatioの影響を実機で切り分けるための手掛かりになる。
func (g *Game) drawTapMarker(screen *ebiten.Image) {
	if g.tapCount == 0 {
		return
	}

	x, y := float32(g.lastTap.X), float32(g.lastTap.Y)
	vector.StrokeCircle(screen, x, y, tapMarkerRadius, 2, tapMarkerColor, true)
	vector.StrokeLine(screen, x-tapMarkerReach, y, x+tapMarkerReach, y, 1, tapMarkerColor, true)
	vector.StrokeLine(screen, x, y-tapMarkerReach, x, y+tapMarkerReach, 1, tapMarkerColor, true)
}

// drawCenteredLabel はASCIIラベルをcenterXの中央揃えで描く。
func drawCenteredLabel(screen *ebiten.Image, label string, centerX float64, y int) {
	ebitenutil.DebugPrintAt(screen, label, int(centerX)-len(label)*debugFontCharWidth/2, y)
}
