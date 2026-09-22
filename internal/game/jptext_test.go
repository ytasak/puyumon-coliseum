package game

import (
	"fmt"
	"image"
	"sync"
	"testing"

	text "github.com/hajimehoshi/ebiten/v2/text/v2"
	"github.com/hajimehoshi/ebiten/v2/vector"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

var (
	sharedFace     *text.GoTextFace
	sharedFaceOnce sync.Once
	sharedFaceErr  error
)

// testFace は同梱フォントを1度だけ解析して返す。
//
// 2MBのフォントをtestごとに解析すると遅いので使い回す。faceは読み取りしか
// しないので、並行に使っても問題ない。
func testFace(t *testing.T) *text.GoTextFace {
	t.Helper()

	sharedFaceOnce.Do(func() {
		sharedFace, sharedFaceErr = uifont.New()
	})
	if sharedFaceErr != nil {
		t.Fatalf("uifont.New()に失敗: %v", sharedFaceErr)
	}
	return sharedFace
}

// 表示名の対応表がrosterの全項目を覆う。
//
// rosterへキャラクターや技が増えたとき、対応表の追加を忘れると画面へ
// 「？」が出る。ここで気づけるようにする。
func TestDisplayNamesCoverTheRoster(t *testing.T) {
	t.Parallel()

	for _, character := range roster.All() {
		if speciesName(character.ID) == unknownName {
			t.Errorf("キャラクター %q の表示名が無い", character.ID)
		}
	}

	data := roster.Data()
	for id := range data.Moves {
		if moveName(id) == unknownName {
			t.Errorf("技 %q の表示名が無い", id)
		}
	}
	for id, move := range data.Moves {
		if typeName(move.Type) == unknownName {
			t.Errorf("技 %q のタイプ %v の表示名が無い", id, move.Type)
		}
	}
}

// 状態異常と能力の表示名がすべて埋まっている。
func TestStatusAndStatNamesAreComplete(t *testing.T) {
	t.Parallel()

	statuses := []battle.MajorStatus{
		battle.Burn, battle.Freeze, battle.Paralysis, battle.Poison, battle.Sleep,
	}
	for _, status := range statuses {
		if got := majorStatusName(status); got == unknownName || got == "" {
			t.Errorf("状態異常 %v の表示名が無い", status)
		}
	}
	// ひんしは major status ではないが枠に出る。
	if statusName(battleui.StatusFainted) == "" {
		t.Error("ひんしの表示名が無い")
	}

	stats := []battle.Stat{
		battle.StatAttack, battle.StatDefense, battle.StatSpeed,
		battle.StatSpecial, battle.StatAccuracy, battle.StatEvasion,
	}
	for _, stat := range stats {
		if statName(stat) == unknownName {
			t.Errorf("能力 %v の表示名が無い", stat)
		}
	}
}

// 表示しうる文字はすべて同梱フォントで描ける。
//
// 字形が無いと画面では豆腐になり、実機でしか気づけない。
func TestEveryDisplayNameIsDrawable(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	for _, group := range displayNameGroups() {
		for label, s := range group {
			if missing := uifont.MissingGlyphs(face, s); len(missing) > 0 {
				t.Errorf("%s %q に字形の無い文字がある: %q", label, s, missing)
			}
		}
	}
}

// 技ボタンのラベルが枠へ収まる。
//
// 技名は左、タイプと残りPPは右へ出す。両方が枠へ収まること。
func TestMoveLabelsFitInTheirButtons(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	// 左右に12pxずつの余白を取り、名前と補助情報が重ならないだけの間も見る。
	limit := float64(moveCellWidth() - 24)

	for id, move := range roster.Data().Moves {
		// 残りPPは最大値のときが一番長い。
		view := battleui.MoveView{Move: id, PP: move.MaxPP, MaxPP: move.MaxPP}
		name, detail := moveName(id), moveDetail(view)
		width := textWidth(face, name) + textWidth(face, detail)
		if width > limit {
			t.Errorf("技 %q の %q と %q が枠に収まらない（%.0f px > %.0f px）", id, name, detail, width, limit)
		}
	}
}

// コマンドのラベルが枠へ収まる。
func TestCommandLabelsFitInTheirButtons(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	limit := float64(moveCellWidth())

	labels := []string{"たたかう", "こうたい", "もどる", "もう一度", "わるあがき", "-"}
	for _, character := range roster.All() {
		labels = append(labels, speciesName(character.ID))
	}
	for _, label := range labels {
		if width := textWidth(face, label); width > limit {
			t.Errorf("ラベル %q が枠に収まらない（%.0f px > %.0f px）", label, width, limit)
		}
	}
}

// 情報枠のヘッダが枠へ収まる。
//
// 名前・Lv・状態を1行に並べるので、名前と状態語の長い組み合わせで確かめる。
func TestInfoHeaderFitsInThePanel(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	panel := infoPanels[viewer]
	limit := float64(panel.Dx() - hpBarInset*2)

	statuses := []battleui.StatusView{
		battleui.StatusBurn, battleui.StatusFreeze, battleui.StatusParalysis,
		battleui.StatusPoison, battleui.StatusSleep, battleui.StatusFainted,
	}
	for _, character := range roster.All() {
		for _, status := range statuses {
			header := fmt.Sprintf("%s Lv%d %s", speciesName(character.ID), 100, statusName(status))
			if width := textWidth(face, header); width > limit {
				t.Errorf("情報枠のヘッダ %q が収まらない（%.0f px > %.0f px）", header, width, limit)
			}
		}
	}
}

// 控えの略記が枠へ収まり、文字の途中で割れない。
//
// **縦も見る。** DotGothic16 の行の高さは ascent + descent = 23.2px あり、
// 18px の枠より大きい。左上を指定して描くと下へ約3pxはみ出していた。
// 実際に塗られるインクが枠の中に入っていることを、描画と同じlayoutで確かめる。
func TestShortLabelFitsInTheReserveBox(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	box := image.Rect(0, 0, reserveChipSize, reserveChipSize)

	for _, character := range roster.All() {
		label := shortLabel(character.ID)
		if label == "" {
			t.Errorf("キャラクター %q の略記が空", character.ID)
			continue
		}
		if count := len([]rune(label)); count != 1 {
			t.Errorf("キャラクター %q の略記 %q が %d 文字（1文字のはず）", character.ID, label, count)
		}
		if missing := uifont.MissingGlyphs(face, label); len(missing) > 0 {
			t.Errorf("略記 %q が文字の途中で割れている: %q", label, missing)
		}

		ink, ok := boxTextInk(face, label, box)
		if !ok {
			t.Errorf("略記 %q が何も塗らない", label)
			continue
		}
		if ink.Min.Y < float64(box.Min.Y) || ink.Max.Y > float64(box.Max.Y) {
			t.Errorf("略記 %q が枠から縦にはみ出す（y %.1f..%.1f、枠は %d..%d）",
				label, ink.Min.Y, ink.Max.Y, box.Min.Y, box.Max.Y)
		}
		if ink.Min.X < float64(box.Min.X) || ink.Max.X > float64(box.Max.X) {
			t.Errorf("略記 %q が枠から横にはみ出す（x %.1f..%.1f、枠は %d..%d）",
				label, ink.Min.X, ink.Max.X, box.Min.X, box.Max.X)
		}
	}
}

// 枠の中央へ描く指定は、左上指定より縦のはみ出しが小さい。
//
// 「中央へ寄せた」ことが効いていることを固定する。左上指定に戻すと落ちる。
func TestBoxTextIsCenteredNotTopAligned(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	box := image.Rect(0, 0, reserveChipSize, reserveChipSize)

	centered, ok := boxTextInk(face, "ホ", box)
	if !ok {
		t.Fatal("インクが取れない")
	}
	topLeft, ok := topLeftTextInk(face, "ホ", box)
	if !ok {
		t.Fatal("インクが取れない")
	}

	if topLeft.Max.Y <= float64(box.Max.Y) {
		t.Fatalf("technical: 左上指定でもはみ出していない（%.1f <= %d）。前提が変わった",
			topLeft.Max.Y, box.Max.Y)
	}
	if centered.Max.Y > topLeft.Max.Y {
		t.Errorf("中央寄せのほうが下へ出ている（%.1f > %.1f）", centered.Max.Y, topLeft.Max.Y)
	}
}

// bounds は塗られた範囲。
type bounds struct {
	Min, Max struct{ X, Y float64 }
}

// boxTextInk は drawTextInBox が実際に塗る範囲を返す。
//
// **描画と同じ boxTextLayout を使う。** 設定を書き写すと、収まると
// 確かめた位置と実際に描く位置がずれる。
func boxTextInk(face *text.GoTextFace, s string, box image.Rectangle) (bounds, bool) {
	layout := boxTextLayout()
	cx, cy := boxCenter(box)
	return inkBounds(face, s, &layout, cx, cy)
}

// topLeftTextInk は左上を指定して描いたときに塗られる範囲を返す。比較用。
func topLeftTextInk(face *text.GoTextFace, s string, box image.Rectangle) (bounds, bool) {
	layout := text.LayoutOptions{LineSpacing: uifont.LineHeight}
	return inkBounds(face, s, &layout, float64(box.Min.X), float64(box.Min.Y))
}

// inkBounds は指定のlayoutと原点で描いたときのglyphインクの範囲を返す。
func inkBounds(face *text.GoTextFace, s string, layout *text.LayoutOptions, originX, originY float64) (bounds, bool) {
	var path vector.Path
	text.AppendVectorPath(&path, s, face, layout)

	vs, _ := path.AppendVerticesAndIndicesForFilling(nil, nil)
	if len(vs) == 0 {
		return bounds{}, false
	}

	var b bounds
	b.Min.X, b.Min.Y = float64(vs[0].DstX), float64(vs[0].DstY)
	b.Max.X, b.Max.Y = b.Min.X, b.Min.Y
	for _, v := range vs {
		x, y := float64(v.DstX), float64(v.DstY)
		b.Min.X = min(b.Min.X, x)
		b.Min.Y = min(b.Min.Y, y)
		b.Max.X = max(b.Max.X, x)
		b.Max.Y = max(b.Max.Y, y)
	}
	b.Min.X += originX
	b.Max.X += originX
	b.Min.Y += originY
	b.Max.Y += originY
	return b, true
}

// 横持ちを促す文面が画面へ収まる。
func TestRotatePromptFitsTheScreen(t *testing.T) {
	t.Parallel()

	face := testFace(t)
	for _, line := range rotatePromptLines {
		if width := textWidth(face, line); width > LogicalWidth {
			t.Errorf("案内 %q が画面幅に収まらない（%.0f px）", line, width)
		}
	}
}

// displayNameGroups は画面へ出す表示名を、種類の名前つきで返す。
func displayNameGroups() []map[string]string {
	species := map[string]string{}
	for id, name := range speciesNames {
		species["キャラクター"+string(id)] = name
	}
	moves := map[string]string{}
	for id, name := range moveNames {
		moves["技"+string(id)] = name
	}
	types := map[string]string{}
	for t, name := range typeNames {
		types[fmt.Sprintf("タイプ%v", t)] = name
	}
	statuses := map[string]string{}
	for s, name := range statusNames {
		statuses[fmt.Sprintf("状態%v", s)] = name
	}
	stats := map[string]string{}
	for s, name := range statNames {
		stats[fmt.Sprintf("能力%v", s)] = name
	}
	fixed := map[string]string{}
	for i, line := range rotatePromptLines {
		fixed[fmt.Sprintf("回転案内%d", i)] = line
	}
	for i, label := range []string{"たたかう", "こうたい", "もどる", "もう一度", "わるあがき", "準備中", unknownName} {
		fixed[fmt.Sprintf("固定文言%d", i)] = label
	}
	return []map[string]string{species, moves, types, statuses, stats, fixed}
}
