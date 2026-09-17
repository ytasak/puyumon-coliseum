package game

// 論理解像度。描画・座標計算はすべてこのサイズを前提とし、実ウィンドウサイズや
// ブラウザのcanvasサイズとは分離する。拡大縮小はEbitengine側のLayoutに任せる。
//
// 640x360 (16:9) は横持ちを前提とした確定値。初代準拠の対面レイアウト
// （両者のアクティブ・控え・HP・技4つ・メッセージ）を1画面へ収めるため、
// 横長を採る。変更する場合はここだけを書き換える。
//
// モバイルは縦持ちで開かれることがあるが、iOS Safariはページから画面の向きを
// ロックできない。そのため縦長で開かれたときは横持ちを促す（needsRotation）。
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
//
// 外側のサイズが分かるのはここだけなので、画面の向きの判定に使う値を控える。
// 返す論理解像度そのものは外側のサイズに影響されない。
func (g *Game) Layout(outsideWidth, outsideHeight int) (screenWidth, screenHeight int) {
	g.outsideWidth, g.outsideHeight = outsideWidth, outsideHeight
	return LogicalWidth, LogicalHeight
}

// needsRotation は外側が縦長で、横持ちを促すべきかを返す。
//
// 縦長のまま遊ばせると、16:9のゲーム画面は高さの3分の1ほどしか使えない。
// 正方形はどちらでもないため促さない。サイズが分からないうちも促さない。
func needsRotation(outsideWidth, outsideHeight int) bool {
	if outsideWidth <= 0 || outsideHeight <= 0 {
		return false
	}
	return outsideWidth < outsideHeight
}
