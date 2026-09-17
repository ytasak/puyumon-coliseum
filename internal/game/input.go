package game

import (
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/inpututil"
)

// taps はこのtickで新しく押された位置を論理座標で返す。
//
// マウスとtouchの両方を拾う。Desktopとブラウザ、さらにiframeの中でも
// 同じコードで扱えるようにしておき、座標変換はEbitengineへ任せる。
// iframe内で座標がずれていないかは、画面のタップマーカーで確認する。
//
// 戻り値は次のtickで上書きされる。保持したい場合は呼び出し側で複製する。
func (g *Game) taps() []image.Point {
	g.tapped = g.tapped[:0]

	if inpututil.IsMouseButtonJustPressed(ebiten.MouseButtonLeft) {
		x, y := ebiten.CursorPosition()
		g.tapped = append(g.tapped, image.Pt(x, y))
	}

	// AppendJustPressedTouchIDs は渡したsliceへ追記する。
	// 毎tick呼ばれるので、bufferを使い回して確保を避ける。
	g.touchIDs = inpututil.AppendJustPressedTouchIDs(g.touchIDs[:0])
	for _, id := range g.touchIDs {
		x, y := ebiten.TouchPosition(id)
		g.tapped = append(g.tapped, image.Pt(x, y))
	}

	return g.tapped
}
