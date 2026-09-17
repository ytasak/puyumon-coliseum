package game

import (
	"math"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ytasak/puyumon-coliseum/internal/sprite"
)

// nassy はYTA-8のPoCキャラクター「ナッシー型」。body 🌴 に faces 🥺 😫 🤪 を載せる。
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
// 新しいキャラクターの追加が定義だけで済み、専用の描画コードを必要としないことを
// 画面で確かめるために置いている。こちらもデザインとしては未確定。
var whale = sprite.NewCharacter(
	sprite.Part{Emoji: "🐋", X: 0, Y: 0, Scale: 1, Z: 0},
	sprite.Part{Emoji: "😫", X: -0.20, Y: 0.04, Scale: 0.24, Z: 1},
	sprite.Part{Emoji: "⭐", X: 0.20, Y: -0.22, Scale: 0.26, Rotation: 0.3, Z: 1},
)

// pocCharacters は上段へ並べるキャラクター。定義が違っても描画は同じコードを通る。
var pocCharacters = []struct {
	label     string
	character sprite.Character
	x         float64
}{
	{label: "nassy type", character: nassy, x: 190},
	{label: "whale type", character: whale, x: 452},
}

// transformVariants は下段へ並べるCharacter全体のtransform。
// 同じ定義でも position / scale / rotation を変えれば一体として変換される。
var transformVariants = []struct {
	label    string
	x        float64
	scale    float64
	rotation float64
}{
	{label: "0.6x", x: 140, scale: characterScale * 0.6},
	{label: "rot 15deg", x: 320, scale: characterScale * 0.85, rotation: 15 * math.Pi / 180},
	{label: "1.2x", x: 505, scale: characterScale * 1.2},
}

// PoC sceneのレイアウト（論理座標）。
const (
	// characterScale は character-local 座標1.0あたりのpixel数の基準値。
	characterScale = 104

	charactersY     = 122
	characterLabelY = 186
	variantsY       = 262
	variantLabelY   = 332

	// debugFontCharWidth はebitenutil.DebugPrintAtが使う組み込みフォントの
	// 1文字分の幅。ラベルを中央揃えするために使う。
	debugFontCharWidth = 6
)

// drawSpritePoC はComposite Sprite PoCの画面を描く。
func (g *Game) drawSpritePoC(screen *ebiten.Image) {
	g.drawCharacters(screen)
	g.drawTransformVariants(screen)
}

// drawCharacters は定義の異なるキャラクターを同じ描画コードで並べる。
func (g *Game) drawCharacters(screen *ebiten.Image) {
	for _, c := range pocCharacters {
		g.sprites.Draw(screen, c.character, sprite.Transform{X: c.x, Y: charactersY, Scale: characterScale})
		drawCenteredLabel(screen, c.label, c.x, characterLabelY)
	}
}

// drawTransformVariants は同じCharacter定義をtransform違いで並べる。
func (g *Game) drawTransformVariants(screen *ebiten.Image) {
	for _, v := range transformVariants {
		g.sprites.Draw(screen, nassy, sprite.Transform{X: v.x, Y: variantsY, Scale: v.scale, Rotation: v.rotation})
		drawCenteredLabel(screen, v.label, v.x, variantLabelY)
	}
}

// drawCenteredLabel はASCIIラベルをcenterXの中央揃えで描く。
func drawCenteredLabel(screen *ebiten.Image, label string, centerX float64, y int) {
	ebitenutil.DebugPrintAt(screen, label, int(centerX)-len(label)*debugFontCharWidth/2, y)
}
