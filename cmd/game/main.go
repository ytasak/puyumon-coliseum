// Command game はぷゆもんコロシアムのゲームクライアントを起動する。
//
// ここではウィンドウ設定とゲームループの起動のみを行い、
// ゲームロジックと描画は internal/game に置く。
package main

import (
	"errors"
	"log"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ytasak/puyumon-coliseum/internal/game"
)

func main() {
	g, err := game.New()
	if err != nil {
		log.Fatalf("new game: %v", err)
	}

	width, height := game.DefaultWindowSize()
	ebiten.SetWindowSize(width, height)
	ebiten.SetWindowTitle(game.WindowTitle)
	ebiten.SetWindowResizingMode(ebiten.WindowResizingModeEnabled)

	if err := ebiten.RunGame(g); err != nil && !errors.Is(err, ebiten.Termination) {
		log.Fatalf("run game: %v", err)
	}
}
