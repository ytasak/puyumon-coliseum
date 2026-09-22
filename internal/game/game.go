// Package game はぷゆもんコロシアムのゲームクライアント本体を提供する。
//
// このpackageはEbitengineのゲームループ（Update / Draw / Layout）と描画を担当する。
// 対戦のルールは持たない。1試合の進行は internal/singleplayer のsessionが持ち、
// 表示に必要な形への変換は internal/battleui が行う。この層はそれを描き、
// 押された場所をcommandへ変えて渡すだけにとどめる。
package game

import (
	"fmt"
	"image"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
	"github.com/ytasak/puyumon-coliseum/internal/sprite"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// WindowTitle はDesktop起動時のウィンドウタイトル。
// OSが描画するためマルチバイト文字を使用できる。
const WindowTitle = "ぷゆもんコロシアム 155 Battle"

// backgroundColor は盤面の背景。4階調のうち最も明るい階調。
var backgroundColor = toneLightest

// Game は ebiten.Game の実装。
//
// 持つのは対戦の進行状態・描画に必要な資源・入力用のbufferだけで、
// 対戦のルールは持たない。
type Game struct {
	// ticks は Update が呼ばれた回数。ゲームループが継続動作していることを
	// 画面とtestの双方から観測できるようにするために保持する。
	ticks uint64

	// sprites はCharacterを描画する。Emoji素材のセル画像を内部で使い回すため、
	// Gameと同じ寿命で1つだけ持つ。
	sprites *sprite.Renderer

	// face は画面テキストを描く同梱フォント。解析は起動時の1度だけで済ませる。
	face *text.GoTextFace

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

	face, err := uifont.New()
	if err != nil {
		return nil, fmt.Errorf("game: %w", err)
	}

	scene, err := newBattleScene(seed, face)
	if err != nil {
		return nil, err
	}
	return &Game{seed: seed, sprites: sprite.NewRenderer(emojis), face: face, battle: scene}, nil
}

// Update はEbitengineのtickごとに呼ばれる。
//
// 押された場所をcommandへ変えて対戦へ渡し、進行とanimationを1 tick進める。
// 対戦のルールはここでは判断しない。
func (g *Game) Update() error {
	for _, tap := range g.taps() {
		if _, err := g.battle.tap(tap); err != nil {
			return err
		}
	}

	if err := g.battle.update(); err != nil {
		return err
	}

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

	g.battle.draw(screen, g.sprites)
}
