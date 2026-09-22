package game

import (
	"image"
	"image/color"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// 画面に使う階調は4つで、明るさの順に並んでいる。
//
// 階調が近すぎると隣同士を見分けられない。実際に描いた結果は headless では
// 読めないので、色の定義そのものを固定する。
func TestPaletteHasFourDistinctTones(t *testing.T) {
	t.Parallel()

	tones := []struct {
		name string
		c    color.RGBA
	}{
		{"最明", toneLightest},
		{"明", toneLight},
		{"暗", toneDark},
		{"最暗", toneDarkest},
	}

	luminance := func(c color.RGBA) float64 {
		return 0.299*float64(c.R) + 0.587*float64(c.G) + 0.114*float64(c.B)
	}

	for i := 1; i < len(tones); i++ {
		prev, cur := tones[i-1], tones[i]
		lp, lc := luminance(prev.c), luminance(cur.c)
		if lc >= lp {
			t.Errorf("%s(%.0f) が %s(%.0f) より暗くない", cur.name, lc, prev.name, lp)
		}
		// 隣り合う階調が近すぎると見分けられない。
		if diff := lp - lc; diff < 24 {
			t.Errorf("%s と %s の明度差が %.0f しかない", prev.name, cur.name, diff)
		}
	}

	for _, tone := range tones {
		if tone.c.A != 0xff {
			t.Errorf("%s が不透明でない", tone.name)
		}
	}
}

// 情報枠の4段が重ならず、枠の中に収まる。
//
// **段の位置は定数で決めている。** 名前が長いときや状態が付いたときに
// 段どうしが重なる作りにすると、実機で初めて気づくことになる。
func TestInfoRowsFitInThePanel(t *testing.T) {
	t.Parallel()

	rows := []struct {
		name   string
		top    int
		height int
	}{
		{"名前", infoNameTop, uifont.Size},
		{"HPバー", infoBarTop, hpBarHeight},
		{"数値・状態", infoNumberTop, uifont.Size + 2},
		{"控え", infoChipTop, reserveChipSize},
	}

	for i := 1; i < len(rows); i++ {
		prev, cur := rows[i-1], rows[i]
		if bottom := prev.top + prev.height; bottom > cur.top {
			t.Errorf("%s(下端 %d) が %s(上端 %d) に重なる", prev.name, bottom, cur.name, cur.top)
		}
	}

	last := rows[len(rows)-1]
	for _, side := range battleSides {
		panel := infoPanels[side]
		if bottom := last.top + last.height; bottom > panel.Dy() {
			t.Errorf("%v: %s の下端 %d が枠の高さ %d を超える", side, last.name, bottom, panel.Dy())
		}
		// 控え3体とHPバーが横幅に収まること。
		width := reserveChipSize*3 + reserveChipGap*2
		if width > panel.Dx()-hpBarInset*2 {
			t.Errorf("%v: 控え3体の幅 %d が枠に収まらない", side, width)
		}
	}
}

// 情報枠と下段の窓、盤面が重ならない。
func TestPanelsDoNotOverlapTheCommandArea(t *testing.T) {
	t.Parallel()

	message := image.Rect(commandMargin, messageTop, LogicalWidth-commandMargin, messageBottom)
	for _, side := range battleSides {
		panel := infoPanels[side]
		if !panel.In(image.Rect(0, 0, LogicalWidth, LogicalHeight)) {
			t.Errorf("%v: 情報枠 %v が画面からはみ出す", side, panel)
		}
		if panel.Overlaps(message) {
			t.Errorf("%v: 情報枠 %v がメッセージ枠 %v に重なる", side, panel, message)
		}
		for _, rect := range commandRects {
			if panel.Overlaps(rect) {
				t.Errorf("%v: 情報枠 %v がコマンド枠 %v に重なる", side, panel, rect)
			}
		}
	}

	if infoPanels[viewer].Overlaps(infoPanels[foe]) {
		t.Error("両者の情報枠が重なる")
	}
}
