// Command game はぷゆもんコロシアムのゲームクライアントを起動する。
//
// ここではウィンドウ設定とゲームループの起動のみを行い、
// ゲームロジックと描画は internal/game に置く。
package main

import (
	"errors"
	"log"
	"time"

	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ytasak/puyumon-coliseum/internal/game"
)

func main() {
	// 起動ごとに別の対戦になるよう、seedはここで決める。
	// その場の値に触れるのはentry pointだけにして、internal/game以下は
	// 渡されたseedしか使わない。グローバルな乱数状態は使わない。
	g, err := game.New(uint64(time.Now().UnixNano()))
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
