package sprite

import (
	"github.com/hajimehoshi/ebiten/v2"

	"github.com/ytasak/puyumon-coliseum/internal/emoji"
)

// Renderer はCharacterを描画する。
//
// キャラクター固有の状態も分岐も持たず、渡されたCharacterの定義を
// そのまま解釈する。そのため新しいキャラクターを追加しても
// このpackageへ手を入れる必要はない。
type Renderer struct {
	emojis *emoji.Set
}

// NewRenderer はEmoji素材を供給するSetを使うRendererを返す。
func NewRenderer(emojis *emoji.Set) *Renderer {
	return &Renderer{emojis: emojis}
}

// Draw はcharacterをtransformの位置・大きさ・角度でdstへ描く。
//
// 部品は Character.Parts の順（NewCharacterならZの昇順）に描画される。
// Rendererは状態を持たないため、Updateの責務はここへ入らない。
//
// dstは任意の *ebiten.Image でよい。静的なキャラクターをoffscreenの
// imageへ一度だけ描いて使い回すこともできる。
func (r *Renderer) Draw(dst *ebiten.Image, character Character, transform Transform) {
	// 部品ごとにDrawImageOptionsを作らず、GeoMだけ作り直して使い回す。
	op := &ebiten.DrawImageOptions{}
	// 拡大縮小・回転どちらでも輪郭のジャギーを抑える。
	op.Filter = ebiten.FilterLinear

	for _, part := range character.Parts {
		placement := part.Place(transform)

		op.GeoM.Reset()
		// セル中心を原点へ移してから拡大縮小・回転するので、
		// どの部品も中心を基準に変換される。
		op.GeoM.Translate(-emoji.CellSize/2, -emoji.CellSize/2)
		scale := placement.Size / emoji.CellSize
		op.GeoM.Scale(scale, scale)
		op.GeoM.Rotate(placement.Rotation)
		op.GeoM.Translate(placement.CenterX, placement.CenterY)

		dst.DrawImage(r.emojis.Image(part.Emoji), op)
	}
}
