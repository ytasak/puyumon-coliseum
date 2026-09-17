package battle

// StageRatio は能力変化stageの倍率。分子と分母で持つ。
//
// Generation Iは能力値へ num/den を整数演算で掛ける。小数へ直すと丸めが変わり、
// 端数の出る値で結果がずれるため、出荷されたデータと同じ分数のまま持つ。
type StageRatio struct {
	// Num は分子。
	Num int
	// Den は分母。
	Den int
}

// stageRatios はStageMinからStageMaxまでの倍率。
//
// 出荷されたROMのデータ（pokeredのdata/battle/stat_modifiers.asm）と同じ値で持つ。
// Generation Iでは命中・回避のstageもこの表を使う。別の表になるのはGeneration II以降。
var stageRatios = [StageMax - StageMin + 1]StageRatio{
	{25, 100}, // -6
	{28, 100}, // -5
	{33, 100}, // -4
	{40, 100}, // -3
	{50, 100}, // -2
	{66, 100}, // -1
	{1, 1},    //  0
	{15, 10},  // +1
	{2, 1},    // +2
	{25, 10},  // +3
	{3, 1},    // +4
	{35, 10},  // +5
	{4, 1},    // +6
}

// StageMultiplier はstageに対応する倍率を返す。
//
// stageがStageMin〜StageMaxの範囲外ならpanicする。範囲外のstageは状態として
// 成立しない値で、BattleState.Validateが弾く。
func StageMultiplier(stage int) StageRatio {
	if stage < StageMin || stage > StageMax {
		panic("battle: stat stage out of range")
	}
	return stageRatios[stage-StageMin]
}

// ApplyStage はvalueへstageの倍率を掛けた値を返す。
//
// Generation Iと同じく整数演算で計算し、端数は切り捨てる。
//
// Generation Iが能力値へ課す下限・上限（1〜999）はここでは適用しない。
// どこで頭打ちにするかは能力値とダメージの計算側で決める。
func ApplyStage(value, stage int) int {
	ratio := StageMultiplier(stage)
	return value * ratio.Num / ratio.Den
}
