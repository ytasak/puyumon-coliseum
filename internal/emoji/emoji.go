// Package emoji はカラーEmojiをゲーム素材として描画するための、
// フォントの読み込みとglyph画像のcacheを提供する。
//
// EmojiはUIテキストではなくスプライト素材として扱う。そのためこのpackageは
// 文字列を毎フレーム組版するのではなく、Emoji 1文字を固定サイズの正方セルへ
// 1度だけラスタライズし、以降はその *ebiten.Image を使い回す。
//
// フォントは同梱したものだけを使い、OS側のEmojiフォールバックには依存しない。
// Desktopとブラウザで同じ絵が出ることを優先するため。
// 同梱フォントの出典とライセンスは assets/README.md を参照。
package emoji

import (
	"bytes"
	_ "embed"
	"fmt"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"
)

// twemojiMozilla はCOLRv0のカラーEmojiフォント。
// Ebitengineのtext/v2はCOLRv0のlayer listを描画できる。
//
//go:embed assets/TwemojiMozilla.ttf
var twemojiMozilla []byte

// CellSize はEmoji 1文字をラスタライズする正方セルの一辺（px）。
//
// このセルが共通の描画単位になる。表示サイズはセル画像の拡大縮小で決め、
// Emojiごとに位置やサイズを個別調整しない。
// 128pxは論理解像度640x360に対して十分大きく、拡大しても破綻しにくい値として選んだ。
const CellSize = 128

// emRatio はセルに対するem boxの比率。
//
// Emojiのglyph inkはem boxからはみ出すことがある（必須glyphでは🤪の約4%が最大）。
// セル端で切れないよう、em boxをセルより小さく取って余白で吸収する。
const emRatio = 0.875

// Set は同梱フォントと、生成済みのセル画像を保持する。
//
// Ebitengineのゲームループから使うことを前提とし、並行アクセスは想定しない。
type Set struct {
	face   *text.GoTextFace
	images map[string]*ebiten.Image
}

// New は同梱フォントを解析したSetを返す。
//
// フォントの解析はここで1度だけ行う。セル画像の生成はEbitengineの描画contextを
// 必要とするため、Imageの初回呼び出しまで遅延する。
func New() (*Set, error) {
	source, err := text.NewGoTextFaceSource(bytes.NewReader(twemojiMozilla))
	if err != nil {
		return nil, fmt.Errorf("emoji: parse bundled font: %w", err)
	}
	return &Set{
		face:   &text.GoTextFace{Source: source, Size: CellSize * emRatio},
		images: map[string]*ebiten.Image{},
	}, nil
}

// Image はemojiをセル中央へ描いた CellSize x CellSize の画像を返す。
//
// 生成は初回のみで、2回目以降は同じ *ebiten.Image を返す。
// text.Draw内部のglyph cacheはLRUで追い出されうるが、ここで保持するセル画像は
// 追い出されないため、毎フレームのfont/glyph初期化は発生しない。
//
// 描画contextを必要とするため、ゲームループ開始後（Draw内など）から呼ぶこと。
func (s *Set) Image(emoji string) *ebiten.Image {
	if img, ok := s.images[emoji]; ok {
		return img
	}

	img := ebiten.NewImage(CellSize, CellSize)

	// em boxの中心をセルの中心へ合わせる。glyphのinkの形ではなくem boxを基準に
	// するので、どのEmojiでもアンカーが揃う。
	op := &text.DrawOptions{}
	op.GeoM.Translate(CellSize/2, CellSize/2)
	op.PrimaryAlign = text.AlignCenter
	op.SecondaryAlign = text.AlignCenter
	text.Draw(img, emoji, s.face, op)

	s.images[emoji] = img
	return img
}

// IsColorGlyph はemojiが同梱フォントの単一のカラーglyphへ解決できるかを返す。
//
// 同梱フォントに無いEmojiは描画してもカラーにならないため、素材として使う前の
// 確認に用いる。複数glyphに分かれる場合も1つの素材として扱えないためfalseを返す。
// 描画contextを必要としないのでtestからも呼べる。
func (s *Set) IsColorGlyph(emoji string) bool {
	glyphs := text.AppendGlyphs(nil, emoji, s.face, nil)
	if len(glyphs) != 1 {
		return false
	}
	return glyphs[0].Colored
}
