package battle

// Generation Iの能力値計算で固定している値。
//
// 本作のキャラクターはテンプレートなので、個体値（DV）と努力値（stat exp）を
// 最大で固定する。DVは15、stat expを上限まで振ったときのbonusは63になる
// （実機は ceil(sqrt(stat exp)) を255で頭打ちにし、それを4で割る）。
const (
	maxDV           = 15
	maxStatExpBonus = 63
)

// LevelStats は種族値とLevelから実数値を求める。
//
// Generation Iの式をそのまま使う。
//
//	stat = floor(((Base + 15) * 2 + 63) * Level / 100) + 5
//	HP   = floor(((Base + 15) * 2 + 63) * Level / 100) + Level + 10
//
// DVとstat expを固定しているため、同じ種族値とLevelからは必ず同じ実数値になる。
// 個体差を入れる場合はここへ引数を足すことになる。
func LevelStats(base Stats, level int) Stats {
	return Stats{
		HP:      scaleStat(base.HP, level) + level + 10,
		Attack:  scaleStat(base.Attack, level) + 5,
		Defense: scaleStat(base.Defense, level) + 5,
		Speed:   scaleStat(base.Speed, level) + 5,
		Special: scaleStat(base.Special, level) + 5,
	}
}

// scaleStat は種族値へDV・stat exp・Levelを反映した共通部分を返す。
func scaleStat(base, level int) int {
	if base < 0 || level < 0 {
		return 0
	}
	return ((base+maxDV)*2 + maxStatExpBonus) * level / 100
}
