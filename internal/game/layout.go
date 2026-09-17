package game

// 論理解像度。描画・座標計算はすべてこのサイズを前提とし、実ウィンドウサイズや
// ブラウザのcanvasサイズとは分離する。拡大縮小はEbitengine側のLayoutに任せる。
//
// 640x360 (16:9) はiframe埋め込みとモバイル横持ちを想定した暫定値。
// 変更する場合はここだけを書き換える。
const (
	LogicalWidth  = 640
	LogicalHeight = 360
)

// DefaultWindowScale はDesktop起動時の初期ウィンドウ倍率。
// 論理解像度そのものには影響しない。
const DefaultWindowScale = 2

// DefaultWindowSize はDesktop起動時の初期ウィンドウサイズを返す。
func DefaultWindowSize() (width, height int) {
	return LogicalWidth * DefaultWindowScale, LogicalHeight * DefaultWindowScale
}

// Layout は ebiten.Game の実装。外側のサイズに関係なく論理解像度を固定で返すため、
// ウィンドウリサイズやブラウザのviewport変化でゲーム側の座標系は変化しない。
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	return LogicalWidth, LogicalHeight
}
