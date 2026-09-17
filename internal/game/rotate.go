package game

import (
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// rotatePromptLines は横持ちを促す文面。
//
// ebitenutil.DebugPrintAt の組み込みフォントで描くためASCIIのみで構成する。
var rotatePromptLines = []string{
	"ROTATE YOUR DEVICE",
	"",
	"this game is played in landscape",
}

// 回転を促す画面のレイアウト（論理座標）。
// 縦長の枠から横長の枠へ、という図を中央に置く。
const (
	rotateIconY = 155

	portraitIconLeft   = 225
	portraitIconTop    = 120
	portraitIconWidth  = 40
	portraitIconHeight = 70

	landscapeIconLeft   = 345
	landscapeIconTop    = 135
	landscapeIconWidth  = 70
	landscapeIconHeight = 40

	rotateArrowLeft  = 283
	rotateArrowRight = 327
	rotateArrowHead  = 9

	rotatePromptTextTop = 224
	rotatePromptLineGap = 16
)

var (
	// dimColor は今の向き（縦）。bright側との対比で、どちらへ向かうかを示す。
	rotateDimColor    = color.RGBA{R: 0x4a, G: 0x58, B: 0x70, A: 0xff}
	rotateBrightColor = color.RGBA{R: 0xff, G: 0xd0, B: 0x4a, A: 0xff}
)

// drawRotatePrompt は横持ちを促す画面を描く。
//
// iOS Safariはページから画面の向きをロックできないため、促すことしかできない。
// ゲーム画面を縮めて出すより、横にしてもらうほうが最終的に見やすい。
func (g *Game) drawRotatePrompt(screen *ebiten.Image) {
	// 今の向き（縦長）。
	vector.StrokeRect(screen,
		portraitIconLeft, portraitIconTop, portraitIconWidth, portraitIconHeight,
		2, rotateDimColor, true)

	// 促したい向き（横長）。
	vector.StrokeRect(screen,
		landscapeIconLeft, landscapeIconTop, landscapeIconWidth, landscapeIconHeight,
		2, rotateBrightColor, true)

	// 2つをつなぐ矢印。
	vector.StrokeLine(screen, rotateArrowLeft, rotateIconY, rotateArrowRight, rotateIconY,
		2, rotateBrightColor, true)
	vector.StrokeLine(screen, rotateArrowRight-rotateArrowHead, rotateIconY-rotateArrowHead,
		rotateArrowRight, rotateIconY, 2, rotateBrightColor, true)
	vector.StrokeLine(screen, rotateArrowRight-rotateArrowHead, rotateIconY+rotateArrowHead,
		rotateArrowRight, rotateIconY, 2, rotateBrightColor, true)

	for i, line := range rotatePromptLines {
		drawCenteredLabel(screen, line, LogicalWidth/2, rotatePromptTextTop+i*rotatePromptLineGap)
	}
}
