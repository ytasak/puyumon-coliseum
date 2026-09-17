package game

import (
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

// pocEmoji はYTA-7が必須とするEmoji。
// 同一画面へまとめて描画し、カラー描画とDesktop / WASM間の差を目視で比較する。
var pocEmoji = []string{"🌴", "🥺", "😫", "🤪", "🐂", "⭐", "⚡", "🐋", "💋", "❄️", "💥", "💤"}

// scaleFactors は拡大縮小の確認に使う倍率。
// Composite Sprite（YTA-8）では部品ごとに倍率が変わるため、同じ素材が
// どの倍率でも成立するかを1画面で見比べられるようにする。
var scaleFactors = []float64{0.5, 1, 1.5, 2}

// scaleSampleEmoji は倍率確認に使うEmoji。細部の多い素材の方が劣化を判断しやすい。
const scaleSampleEmoji = "🌴"

// PoC sceneのレイアウト（論理座標）。
const (
	// emojiSize はgrid 1マスのEmoji表示サイズ。倍率1.0の基準でもある。
	emojiSize = 48

	gridColumns = 6
	gridStepX   = 88
	gridStepY   = 68
	// gridTopY は1行目の中心Y。
	gridTopY = 116

	// scaleRowCenterY は倍率サンプルの中心Y。倍率が変わっても中心線は揃える。
	scaleRowCenterY = 268
	scaleRowGap     = 28
	scaleLabelY     = 324
	// scaleLabelHalfWidth はDebugPrintAtで描くラベル（"0.5x"）のおおよその半幅。
	scaleLabelHalfWidth = 12
)

// drawEmojiPoC は必須Emoji一覧と倍率サンプルを描く。
func (g *Game) drawEmojiPoC(screen *ebiten.Image) {
	g.drawEmojiGrid(screen)
	g.drawScaleSamples(screen)
}

// drawEmojiGrid は必須Emojiを等間隔のgridへ並べる。
func (g *Game) drawEmojiGrid(screen *ebiten.Image) {
	left := (LogicalWidth - float64(gridColumns*gridStepX)) / 2
	for i, e := range pocEmoji {
		col, row := i%gridColumns, i/gridColumns
		x := left + gridStepX*float64(col) + gridStepX/2
		y := float64(gridTopY + gridStepY*row)
		g.drawEmoji(screen, e, x, y, emojiSize)
	}
}

// drawScaleSamples は同じEmojiを複数の倍率で横に並べる。
func (g *Game) drawScaleSamples(screen *ebiten.Image) {
	total := scaleRowGap * float64(len(scaleFactors)-1)
	for _, f := range scaleFactors {
		total += emojiSize * f
	}

	x := (LogicalWidth - total) / 2
	for _, f := range scaleFactors {
		size := emojiSize * f
		centerX := x + size/2
		g.drawEmoji(screen, scaleSampleEmoji, centerX, scaleRowCenterY, size)
		ebitenutil.DebugPrintAt(screen, fmt.Sprintf("%.1fx", f), int(centerX)-scaleLabelHalfWidth, scaleLabelY)
		x += size + scaleRowGap
	}
}

// drawEmoji はemojiのセル画像を(centerX, centerY)中心にsize px四方で描く。
//
// 表示サイズの違いは共通の描画単位であるセルの拡大縮小だけで表現し、
// Emojiごとの位置やサイズの補正は行わない。
func (g *Game) drawEmoji(screen *ebiten.Image, e string, centerX, centerY, size float64) {
	scale := size / emoji.CellSize

	op := &ebiten.DrawImageOptions{}
	// セル中心を原点へ移してから拡大縮小するので、倍率を変えても中心がずれない。
	op.GeoM.Translate(-emoji.CellSize/2, -emoji.CellSize/2)
	op.GeoM.Scale(scale, scale)
	op.GeoM.Translate(centerX, centerY)
	// 縮小・拡大どちらでも輪郭のジャギーを抑える。
	op.Filter = ebiten.FilterLinear

	screen.DrawImage(g.emojis.Image(e), op)
}
