// Package game はぷゆもんコロシアムのゲームクライアント本体を提供する。
//
// このpackageはEbitengineのゲームループ（Update / Draw / Layout）と描画のみを担当する。
// Battle Engineなどのgame domain logicは後続Issueで別packageとして追加し、
// この層からはUI非依存の状態として参照する構成を維持する。
package game

import (
	"fmt"
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/ebitenutil"

	"github.com/ytasak/puyumon-coliseum/internal/anim"
	"github.com/ytasak/puyumon-coliseum/internal/emoji"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
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
// 現時点ではComposite Sprite animation PoCの状態しか持たない。
type Game struct {
	// ticks は Update が呼ばれた回数。ゲームループが継続動作していることを
	// 画面とtestの双方から観測できるようにするために保持する。
	ticks uint64

	// sprites はCharacterを描画する。Emoji素材のセル画像を内部で使い回すため、
	// Gameと同じ寿命で1つだけ持つ。
	sprites *sprite.Renderer

	// player はキャラクターのanimation状態。キャラクター定義とは分けて持つ。
	player anim.Player

	// particles は表示中のEmoji particle。
	particles anim.Particles

	// tapCount はこれまでに受け取ったタップの数。
	// 実機で入力が拾えているかを画面から確認するために数える。
	tapCount int
	// lastTap は最後にタップされた論理座標。
	lastTap image.Point

	// touchIDs, tapped は入力の取得に使い回すbuffer。
	// 毎tick確保しないために保持する。
	touchIDs []ebiten.TouchID
	tapped   []image.Point

	// outsideWidth, outsideHeight はLayoutが受け取った外側のサイズ。
	// 画面の向きの判定にだけ使う。論理解像度には影響しない。
	outsideWidth, outsideHeight int

	// seed はこの起動ぶんの乱数の素。新しい対戦のseedもここから導く。
	seed uint64

	// battle は対戦の進行状態。Battle Ruleは持たず、sessionを介してのみ対戦を進める。
	//
	// 画面への接続（入力の割り当てと描画）は、画面レイアウトと一緒に入れる。
	battle *battleScene
}

// 実装漏れをコンパイル時に検出する。
var _ ebiten.Game = (*Game)(nil)

// New は初期状態のGameを返す。
//
// seedはこの起動ぶんの対戦を決める。同じseedからは同じ試合列になるので、
// 再現したい場合は同じ値を渡す。時刻のようなその場の値から決めるのは
// 呼び出し側（entry point）の責務で、ここではグローバルな乱数状態に触らない。
//
// 同梱フォントの読み込みに失敗した場合はerrorを返す。
func New(seed uint64) (*Game, error) {
	emojis, err := emoji.New()
	if err != nil {
		return nil, fmt.Errorf("game: %w", err)
	}

	scene, err := newBattleScene(seed)
	if err != nil {
		return nil, err
	}
	return &Game{seed: seed, sprites: sprite.NewRenderer(emojis), battle: scene}, nil
}

// Update はEbitengineのtickごとに呼ばれる。
//
// ここではanimationの状態だけを進め、描画は行わない。
func (g *Game) Update() error {
	if err := g.battle.update(); err != nil {
		return err
	}

	g.updateSpriteDemo(g.ticks)
	g.ticks++
	return nil
}

// Ticks はこれまでに処理したtick数を返す。
func (g *Game) Ticks() uint64 {
	return g.ticks
}

// Draw は1フレーム分の描画を行う。
//
// 縦長で開かれているときはゲーム画面の代わりに横持ちを促す。
// Updateは止めないので、横にすればそのまま続きが見える。
func (g *Game) Draw(screen *ebiten.Image) {
	screen.Fill(backgroundColor)

	if needsRotation(g.outsideWidth, g.outsideHeight) {
		g.drawRotatePrompt(screen)
		return
	}

	g.drawSpritePoC(screen)
	ebitenutil.DebugPrintAt(screen, g.overlayText(), overlayTextOriginX, overlayTextOriginY)
}

// overlayText はPoC識別用テキストを返す。
//
// ebitenutil.DebugPrint は組み込みのASCIIフォントで描画するため、
// ここでは日本語やEmojiを含めない。画面のEmojiは同梱フォントで描いており、
// この識別用テキストとは描画経路が別になっている。
func (g *Game) overlayText() string {
	return fmt.Sprintf(
		"PUYUMON COLISEUM 155 BATTLE\nYTA-10 mobile safari PoC\n"+
			"logical %dx%d / fps %.1f tps %.1f / ticks: %d\ntaps: %d / last: (%d, %d)",
		LogicalWidth, LogicalHeight, ebiten.ActualFPS(), ebiten.ActualTPS(), g.ticks,
		g.tapCount, g.lastTap.X, g.lastTap.Y,
	)
}
