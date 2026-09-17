package game

import (
	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

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

// whale は2体目のPoCキャラクター。
//
// 新しいキャラクターの追加が定義だけで済み、専用の描画コードもanimation側の
// 分岐も必要としないことを画面で確かめるために置いている。デザインは未確定。
var whale = sprite.NewCharacter(
	sprite.Part{Emoji: "🐋", X: 0, Y: 0, Scale: 1, Z: 0},
	sprite.Part{Emoji: "😫", X: -0.20, Y: 0.04, Scale: 0.24, Z: 1},
	sprite.Part{Emoji: "⭐", X: 0.20, Y: -0.22, Scale: 0.26, Rotation: 0.3, Z: 1},
)

// spriteDemos は画面へ並べるキャラクターと、そこで繰り返し見せるanimation。
//
// animation primitiveがキャラクターに依存しないことを見比べられるよう、
// ナッシー型とクジラ型を混ぜている。
var spriteDemos = []struct {
	label     string
	character sprite.Character
	// motion は周期的に再生する単発animation。
	// playMotionがfalseなら待機の動きだけを見せる。
	motion     anim.Motion
	playMotion bool
	// particle はanimationに合わせて出すEmoji。
	particle string
	x        float64
}{
	{label: "idle", character: nassy, particle: "💤", x: 80},
	{label: "attack", character: whale, motion: anim.Attack, playMotion: true, particle: "⚡", x: 240},
	{label: "hit", character: nassy, motion: anim.Hit, playMotion: true, particle: "💥", x: 400},
	{label: "emphasis", character: whale, motion: anim.Emphasis, playMotion: true, particle: "❄️", x: 560},
}

// PoC sceneのレイアウトと進行（論理座標 / tick）。
const (
	// characterScale は character-local 座標1.0あたりのpixel数。
	characterScale  = 76
	charactersY     = 190
	characterLabelY = 246

	particleSpawnY = 150
	particleSize   = 30

	// demoPeriod は1体が次にanimationを始めるまでのtick数。
	demoPeriod = 90
	// demoStagger は隣どうしが同時に動かないようずらすtick数。
	demoStagger = 22

	// debugFontCharWidth はebitenutil.DebugPrintAtが使う組み込みフォントの
	// 1文字分の幅。ラベルを中央揃えするために使う。
	debugFontCharWidth = 6
)

// demoTriggers はindex番目のdemoがこのtickでanimationを始めるかを返す。
func demoTriggers(index int, ticks uint64) bool {
	return ticks%demoPeriod == uint64(index)*demoStagger%demoPeriod
}

// updateSpriteDemo はanimationの状態だけを進める。描画は行わない。
func (g *Game) updateSpriteDemo(ticks uint64) {
	for i := range spriteDemos {
		demo := &spriteDemos[i]

		if demoTriggers(i, ticks) {
			if demo.playMotion {
				g.players[i].Play(demo.motion)
			}
			g.particles.Spawn(demo.particle, demo.x, particleSpawnY, particleSize)
		}
		g.players[i].Update()
	}
	g.particles.Update()
}

// drawSpritePoC はキャラクターとparticleを描く。状態は変更しない。
func (g *Game) drawSpritePoC(screen *ebiten.Image) {
	for i, demo := range spriteDemos {
		base := sprite.Transform{X: demo.x, Y: charactersY, Scale: characterScale}
		g.sprites.Draw(screen, demo.character, g.players[i].Transform(base))
		drawCenteredLabel(screen, demo.label, demo.x, characterLabelY)
	}

	// particleはキャラクターより手前に出す。
	for i := range g.particles.Len() {
		character, transform := g.particles.At(i)
		g.sprites.Draw(screen, character, transform)
	}
}

// drawCenteredLabel はASCIIラベルをcenterXの中央揃えで描く。
func drawCenteredLabel(screen *ebiten.Image, label string, centerX float64, y int) {
	ebitenutil.DebugPrintAt(screen, label, int(centerX)-len(label)*debugFontCharWidth/2, y)
}
