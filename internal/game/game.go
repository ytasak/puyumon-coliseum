// Package game はぷゆもんコロシアムのゲームクライアント本体を提供する。
//
// このpackageはEbitengineのゲームループ（Update / Draw / Layout）と描画のみを担当する。
// Battle Engineなどのgame domain logicは後続Issueで別packageとして追加し、
// この層からはUI非依存の状態として参照する構成を維持する。
package game

import (
	"fmt"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"
)

// WindowTitle はDesktop起動時のウィンドウタイトル。
// OSが描画するためマルチバイト文字を使用できる。
const WindowTitle = "ぷゆもんコロシアム 155 Battle (PoC)"

// backgroundColor はPoCの単色背景。
var backgroundColor = color.RGBA{R: 0x1b, G: 0x24, B: 0x38, A: 0xff}

// overlayTextOriginX, overlayTextOriginY は識別用テキストの描画開始位置（論理座標）。
const (
	overlayTextOriginX = 16
	overlayTextOriginY = 16
)

// Game は ebiten.Game の実装。
//
// 現時点では起動確認用の最小状態しか持たない。
type Game struct {
	// ticks は Update が呼ばれた回数。ゲームループが継続動作していることを
	// 画面とtestの双方から観測できるようにするために保持する。
	ticks uint64
}

// 実装漏れをコンパイル時に検出する。
var _ ebiten.Game = (*Game)(nil)

// New は初期状態のGameを返す。
func New() *Game {
	return &Game{}
}

// Update はEbitengineのtickごとに呼ばれる。
func (g *Game) Update() error {
	g.ticks++
	return nil
}

// Ticks はこれまでに処理したtick数を返す。
func (g *Game) Ticks() uint64 {
	return g.ticks
}

// Draw は1フレーム分の描画を行う。
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(backgroundColor)
	ebitenutil.DebugPrintAt(screen, g.overlayText(), overlayTextOriginX, overlayTextOriginY)
}

// overlayText はPoC識別用テキストを返す。
//
// ebitenutil.DebugPrint は組み込みのASCIIフォントで描画するため、
// ここでは日本語やEmojiを含めない。フォント同梱とEmoji描画はYTA-7で扱う。
func (g *Game) overlayText() string {
	return fmt.Sprintf(
		"PUYUMON COLISEUM 155 BATTLE\nEbitengine PoC (YTA-5)\nlogical resolution: %dx%d\nticks: %d",
		LogicalWidth, LogicalHeight, g.ticks,
	)
}
