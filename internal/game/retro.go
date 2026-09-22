package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"
)

// 画面は4階調のグリーングレーだけで描く。
//
// 1996年の携帯機RPGの見た目へ寄せるための配色で、参照するのは「階調が少ない」
// という視覚文法だけ。原作の画像・ロゴ・UIは使わない。
//
// **階調が4つしかないので、色相で意味を分けられない。** 残りHP・押せない選択肢・
// 戦闘不能のように「状態の違い」を示す場所では、明度に加えて線やパターンでも
// 区別する。色だけに頼ると、隣り合う階調では見分けがつかなくなる。
var (
	toneLightest = color.RGBA{R: 0xcd, G: 0xd6, B: 0xc0, A: 0xff}
	toneLight    = color.RGBA{R: 0x94, G: 0xa3, B: 0x8c, A: 0xff}
	toneDark     = color.RGBA{R: 0x4d, G: 0x5a, B: 0x4a, A: 0xff}
	toneDarkest  = color.RGBA{R: 0x20, G: 0x29, B: 0x20, A: 0xff}
)

// 枠の太さ（論理座標）。
const (
	// windowBorder は窓の外周。太くして「枠のある窓」に見せる。
	windowBorder = 3

	// windowInset は外周の内側へ引く細線までの距離。
	windowInset = 2
)

// drawWindow は枠付きの窓を描く。
//
// 太い外周と、その内側の細線の2重にする。背景と同じ明度で塗り、枠線だけで
// 区切るので、濃い色のカードのように浮かない。角は丸めない。
func drawWindow(dst *ebiten.Image, rect image.Rectangle) {
	x, y := float32(rect.Min.X), float32(rect.Min.Y)
	w, h := float32(rect.Dx()), float32(rect.Dy())

	vector.DrawFilledRect(dst, x, y, w, h, toneDarkest, false)
	vector.DrawFilledRect(dst, x+windowBorder, y+windowBorder,
		w-windowBorder*2, h-windowBorder*2, toneLightest, false)
	vector.StrokeRect(dst, x+windowBorder+windowInset, y+windowBorder+windowInset,
		w-(windowBorder+windowInset)*2, h-(windowBorder+windowInset)*2, 1, toneDark, false)
}

// drawDivider は窓の中を仕切る細線を引く。
//
// セルを独立したボタンにせず、1つの窓を分けたものとして見せるために使う。
func drawDivider(dst *ebiten.Image, x, y, w, h int) {
	vector.DrawFilledRect(dst, float32(x), float32(y), float32(w), float32(h), toneDark, false)
}

// drawHatch は斜線を引いて「押せない」ことを示す。
//
// 4階調では明度を1段落としただけだと隣の階調と紛れる。線が入っているかどうかは
// 階調に関係なく分かるので、色と併せて使う。
func drawHatch(dst *ebiten.Image, rect image.Rectangle) {
	const gap = 6

	// 枠の外へはみ出さないよう、線の両端を矩形の内側へ収めてから引く。
	for offset := -rect.Dy(); offset < rect.Dx(); offset += gap {
		x0, y0 := rect.Min.X+offset, rect.Max.Y
		x1, y1 := x0+rect.Dy(), rect.Min.Y
		if x0 < rect.Min.X {
			y0 -= rect.Min.X - x0
			x0 = rect.Min.X
		}
		if x1 > rect.Max.X {
			y1 += x1 - rect.Max.X
			x1 = rect.Max.X
		}
		if x0 >= x1 {
			continue
		}
		vector.StrokeLine(dst, float32(x0), float32(y0), float32(x1), float32(y1), 1, toneLight, false)
	}
}

// drawStripes は塗りの上へ等間隔の隙間を空けて縞にする。
//
// HPの残量を色ではなく密度で示すために使う。間隔が狭いほど危ない。
func drawStripes(dst *ebiten.Image, rect image.Rectangle, gap int) {
	if gap <= 0 {
		return
	}
	for x := rect.Min.X; x < rect.Max.X; x += gap {
		vector.DrawFilledRect(dst, float32(x), float32(rect.Min.Y), 2, float32(rect.Dy()),
			toneLightest, false)
	}
}

// drawChipCross は控えの枠へ×を引く。戦闘不能を明度に頼らず示す。
func drawChipCross(dst *ebiten.Image, box image.Rectangle) {
	const inset = 3

	x0, y0 := float32(box.Min.X+inset), float32(box.Min.Y+inset)
	x1, y1 := float32(box.Max.X-inset), float32(box.Max.Y-inset)
	vector.StrokeLine(dst, x0, y0, x1, y1, 2, toneDarkest, false)
	vector.StrokeLine(dst, x1, y0, x0, y1, 2, toneDarkest, false)
}
