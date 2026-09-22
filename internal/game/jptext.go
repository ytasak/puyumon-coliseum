package game

import (
	"image"
	"image/color"

	"github.com/hajimehoshi/ebiten/v2"
	text "github.com/hajimehoshi/ebiten/v2/text/v2"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/uifont"
)

// 画面に出す日本語の表示名。
//
// **ここはUI層だけの対応表で、内部IDの言い換えに過ぎない。** SpeciesID /
// MoveID、固定roster、Battle Ruleは変えない。domain側はこの表を知らない。
//
// キャラクター名と技名は**仮の表示名で、最終的な命名ではない**（YTA-35）。
// 正式名が決まったらこの表だけを差し替える。
//
// 技名を4文字までに抑えているのは、技ボタンが「技名 タイプ 残りPP」を
// 1行に収めるため。cellWidthは192pxで、全角4文字+タイプ4文字+PP 5桁+
// 区切り2つで184pxとほぼ使い切る。長い名前は枠からはみ出す。

// speciesNames はキャラクターの表示名。
var speciesNames = map[battle.SpeciesID]string{
	roster.SpeciesBull:  "ブル",
	roster.SpeciesStar:  "スター",
	roster.SpeciesJolt:  "ジョルト",
	roster.SpeciesPalm:  "パーム",
	roster.SpeciesWhale: "ホエール",
	roster.SpeciesCharm: "チャーム",
}

// moveNames は技の表示名。ボタンへ収めるため4文字までにする。
var moveNames = map[battle.MoveID]string{
	roster.MoveSlam:      "スラム",
	roster.MoveOverdrive: "ドライブ",
	roster.MoveQuake:     "クエイク",
	roster.MoveIcestorm:  "フブキ",
	roster.MoveTide:      "タイド",
	roster.MoveMindblast: "サイコ",
	roster.MoveSpark:     "スパーク",
	roster.MoveNumb:      "シビレ",
	roster.MoveMend:      "リカバー",
	roster.MoveDoze:      "ネムル",
	roster.MoveSlumber:   "ララバイ",
	roster.MoveSpores:    "スポア",
	roster.MoveDrain:     "ドレイン",
	roster.MoveBurst:     "バースト",
	roster.MoveNeedles:   "ニードル",
	roster.MoveDash:      "ダッシュ",
}

// typeNames はタイプの表示名。
var typeNames = map[battle.Type]string{
	battle.TypeNormal:   "ノーマル",
	battle.TypeFighting: "かくとう",
	battle.TypeFlying:   "ひこう",
	battle.TypePoison:   "どく",
	battle.TypeGround:   "じめん",
	battle.TypeRock:     "いわ",
	battle.TypeBug:      "むし",
	battle.TypeGhost:    "ゴースト",
	battle.TypeFire:     "ほのお",
	battle.TypeWater:    "みず",
	battle.TypeGrass:    "くさ",
	battle.TypeElectric: "でんき",
	battle.TypePsychic:  "エスパー",
	battle.TypeIce:      "こおり",
	battle.TypeDragon:   "ドラゴン",
}

// statusNames は状態異常の表示名。情報枠にも文章にも同じ語を使う。
var statusNames = map[battleui.StatusView]string{
	battleui.StatusBurn:      "やけど",
	battleui.StatusFreeze:    "こおり",
	battleui.StatusParalysis: "まひ",
	battleui.StatusPoison:    "どく",
	battleui.StatusSleep:     "ねむり",
	battleui.StatusFainted:   "ひんし",
}

// statNames は能力の表示名。
var statNames = map[battle.Stat]string{
	battle.StatAttack:   "こうげき",
	battle.StatDefense:  "ぼうぎょ",
	battle.StatSpeed:    "すばやさ",
	battle.StatSpecial:  "とくしゅ",
	battle.StatAccuracy: "めいちゅう",
	battle.StatEvasion:  "かいひ",
}

// unknownName は対応表に無いものを指す。豆腐を出さないために置く。
const unknownName = "？"

// speciesName はキャラクターの表示名を返す。
func speciesName(species battle.SpeciesID) string {
	if name, ok := speciesNames[species]; ok {
		return name
	}
	return unknownName
}

// moveName は技の表示名を返す。
func moveName(move battle.MoveID) string {
	if name, ok := moveNames[move]; ok {
		return name
	}
	return unknownName
}

// typeName はタイプの表示名を返す。
func typeName(t battle.Type) string {
	if name, ok := typeNames[t]; ok {
		return name
	}
	return unknownName
}

// statusName は状態異常の表示名を返す。状態が無ければ空。
func statusName(status battleui.StatusView) string {
	return statusNames[status]
}

// statName は能力の表示名を返す。
func statName(stat battle.Stat) string {
	if name, ok := statNames[stat]; ok {
		return name
	}
	return unknownName
}

// textColor は画面テキストの色。4階調のうち最も暗い階調を使う。
var textColor = toneDarkest

// drawText は文字列の左上を (x, y) に置いて描く。
//
// 座標の意味を ebitenutil.DebugPrintAt と揃えてあるので、置き換えても
// 既存のレイアウト定数がそのまま使える。
func drawText(dst *ebiten.Image, face *text.GoTextFace, s string, x, y int) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(textColor)
	op.LineSpacing = uifont.LineHeight
	text.Draw(dst, s, face, op)
}

// drawCenteredText は文字列を centerX で中央揃えして描く。
//
// **文字数ではなく実測幅で中央を出す。** 日本語は全角と半角が混ざり、
// バイト数とも文字数とも表示幅が一致しない。
func drawCenteredText(dst *ebiten.Image, face *text.GoTextFace, s string, centerX float64, y int) {
	if s == "" {
		return
	}
	width, _ := text.Measure(s, face, uifont.LineHeight)
	drawText(dst, face, s, int(centerX-width/2), y)
}

// drawTextWithColor は色を指定して文字列を描く。
func drawTextWithColor(dst *ebiten.Image, face *text.GoTextFace, s string, x, y int, c color.Color) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.GeoM.Translate(float64(x), float64(y))
	op.ColorScale.ScaleWithColor(c)
	op.LineSpacing = uifont.LineHeight
	text.Draw(dst, s, face, op)
}

// drawRightText は右端を揃えて文字列を描く。
//
// Lvや残りPPのように、値の桁数が変わっても右端を保ちたい場所で使う。
func drawRightText(dst *ebiten.Image, face *text.GoTextFace, s string, right, y int) {
	if s == "" {
		return
	}
	drawText(dst, face, s, right-int(textWidth(face, s)), y)
}

// textWidth は文字列の表示幅を返す。枠へ収まるかの判断に使う。
func textWidth(face *text.GoTextFace, s string) float64 {
	width, _ := text.Measure(s, face, uifont.LineHeight)
	return width
}

// boxTextLayout は枠の中心へ文字を置くためのlayout設定。
//
// **描画とtestの両方がこれを使う。** 別々に設定を書くと、枠に収まるか
// 確かめたつもりの位置と実際に描く位置がずれる。
func boxTextLayout() text.LayoutOptions {
	return text.LayoutOptions{
		LineSpacing:    uifont.LineHeight,
		PrimaryAlign:   text.AlignCenter,
		SecondaryAlign: text.AlignCenter,
	}
}

// drawTextInBox は枠の中心へ文字を描く。
//
// 行の高さ（ascent + descent = 23.2px）は小さな枠より大きいことがあるが、
// 実際に塗られるのはglyphのインクだけで、そちらは枠へ収まる。
// 左上を指定して描くとdescentのぶん下へずれ、枠から出る。
func drawTextInBox(dst *ebiten.Image, face *text.GoTextFace, s string, box image.Rectangle) {
	if s == "" {
		return
	}
	op := &text.DrawOptions{}
	op.LayoutOptions = boxTextLayout()
	op.GeoM.Translate(boxCenter(box))
	op.ColorScale.ScaleWithColor(textColor)
	text.Draw(dst, s, face, op)
}

// boxCenter は枠の中心を返す。
func boxCenter(box image.Rectangle) (x, y float64) {
	return float64(box.Min.X) + float64(box.Dx())/2, float64(box.Min.Y) + float64(box.Dy())/2
}
