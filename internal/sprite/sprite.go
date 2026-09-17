// Package sprite は複数のEmojiを組み合わせて1体のキャラクターとして
// 定義・描画するComposite Sprite層を提供する。
//
// キャラクターは「データ定義」であり、描画側にキャラクター固有の分岐を持たせない。
// 新しいキャラクターの追加は Character の定義を増やすだけで済み、
// 専用の描画コードを必要としない。
//
// このfileはEbitengineに依存しない定義と座標計算だけを持つ。
// 実際の描画は renderer.go にある Renderer が担当する。
package sprite

import (
	"cmp"
	"math"
	"slices"
)

// Part はキャラクターを構成する部品ひとつ分の定義。
//
// 位置・大きさはscreen座標ではなく character-local 座標で保持する。
// 1.0 が Emoji セル1個分の辺の長さにあたり、原点はキャラクターの中心。
// X は右、Y は下を正とする（screenと同じ向き）。
type Part struct {
	// Emoji はこの部品に使うEmoji。
	Emoji string

	// X, Y はキャラクター中心から見た部品の中心位置（character-local座標）。
	X, Y float64

	// Scale は部品の大きさ。1.0 で Emoji セル1個分。
	Scale float64

	// Rotation は部品自身の回転（ラジアン）。正の値で時計回り。
	Rotation float64

	// Z は描画順。小さいものが先に（奥に）描かれる。
	Z int
}

// Character は複数のPartで構成された1体のキャラクター定義。
//
// Parts は描画順に並んでいる前提で、Renderer は先頭から順に描画する。
// NewCharacter を使えばZの昇順へ並べ替えられる。
type Character struct {
	Parts []Part
}

// NewCharacter はpartsをZの昇順へ並べたCharacterを返す。
//
// Zが同じ部品は引数の順序を保つ。並べ替えはここで1度だけ行い、
// 描画のたびにsortしない。
func NewCharacter(parts ...Part) Character {
	sorted := slices.Clone(parts)
	slices.SortStableFunc(sorted, func(a, b Part) int {
		return cmp.Compare(a.Z, b.Z)
	})
	return Character{Parts: sorted}
}

// Transform はキャラクター全体の配置。単位はscreen座標。
//
// Partのlocal transformとは分離されており、ここを変えると全部品が
// 一体として移動・拡大・回転する。
type Transform struct {
	// X, Y はキャラクター中心のscreen座標。
	X, Y float64

	// Scale は character-local 座標1.0あたりのpixel数。
	// 例えば100なら Scale 1.0 の部品が100px四方で描かれる。
	Scale float64

	// Rotation はキャラクター全体の回転（ラジアン）。正の値で時計回り。
	Rotation float64
}

// Placement は1つのPartが最終的にどこへどの大きさ・角度で描かれるかを表す。
// 単位はscreen座標。
type Placement struct {
	// CenterX, CenterY は部品の中心のscreen座標。
	CenterX, CenterY float64

	// Size は描画されるEmojiセルの一辺（px）。
	Size float64

	// Rotation は最終的な回転（ラジアン）。
	Rotation float64
}

// Place はPartをTransformのもとへ配置したときの描画パラメータを返す。
//
// 部品のlocal transformを先に適用し、その上からキャラクター全体の
// transformを重ねる。Ebitengineに依存しない純粋な計算なので、
// 描画contextなしでも検証できる。
func (p Part) Place(t Transform) Placement {
	sin, cos := math.Sincos(t.Rotation)
	return Placement{
		CenterX:  t.X + (p.X*cos-p.Y*sin)*t.Scale,
		CenterY:  t.Y + (p.X*sin+p.Y*cos)*t.Scale,
		Size:     p.Scale * t.Scale,
		Rotation: p.Rotation + t.Rotation,
	}
}
