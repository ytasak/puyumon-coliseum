package battle

// MoveEffect は技がダメージ以外に起こすこと。
//
// 技ごとの挙動はこの値で分岐する。キャラクターや技の名前で分岐しない。
type MoveEffect int

const (
	// EffectNone は追加の効果なし。
	EffectNone MoveEffect = iota
	// EffectParalyze はまひにする。EffectChanceが0なら確定、そうでなければ追加効果。
	EffectParalyze
	// EffectFreeze はこおりにする。追加効果として使う。
	EffectFreeze
	// EffectSleep はねむりにする。
	EffectSleep
	// EffectSpecialDown は相手のSpecialのstageを1つ下げる。追加効果として使う。
	EffectSpecialDown
	// EffectHeal は最大HPの半分を回復する。
	EffectHeal
	// EffectRest は状態異常を消して全回復し、自分がねむりになる。
	EffectRest
	// EffectRecharge は使ったturnの次に行動できなくなる。
	EffectRecharge
	// EffectExplode は自爆する。相手の防御を半分にして計算し、使用者は戦闘不能になる。
	EffectExplode
	// EffectMultiHit は1回の使用で2〜5回当たる。
	EffectMultiHit
	// EffectDrain は与えたダメージの半分を回復する。
	EffectDrain
	// EffectSpeedUp2 は自分のSpeedのstageを2つ上げる。
	EffectSpeedUp2
	// EffectRecoil は与えたダメージの半分を使用者へ跳ね返す。Struggleが使う。
	EffectRecoil
)

// String はMoveEffectの名前を返す。testとlogの出力に使う。
func (e MoveEffect) String() string {
	switch e {
	case EffectNone:
		return "none"
	case EffectParalyze:
		return "paralyze"
	case EffectFreeze:
		return "freeze"
	case EffectSleep:
		return "sleep"
	case EffectSpecialDown:
		return "special_down"
	case EffectHeal:
		return "heal"
	case EffectRest:
		return "rest"
	case EffectRecharge:
		return "recharge"
	case EffectExplode:
		return "explode"
	case EffectMultiHit:
		return "multi_hit"
	case EffectDrain:
		return "drain"
	case EffectSpeedUp2:
		return "speed_up2"
	case EffectRecoil:
		return "recoil"
	default:
		return "unknown"
	}
}

// EffectChancePercent は追加効果の発生率を、百分率からGeneration Iの0〜255表現へ直す。
//
// 実機は「百分率 * 255 / 100 + 1」を閾値にし、乱数がそれ未満なら発動する。
// 10%なら26、30%なら77、33%なら85になる。
func EffectChancePercent(percent int) int {
	if percent <= 0 {
		return 0
	}
	chance := percent*MaxAccuracy/100 + 1
	if chance > randomByteValues {
		return randomByteValues
	}
	return chance
}

// Stat は能力の種類。stageの変化を表すときに使う。
type Stat int

const (
	// StatAttack はこうげき。
	StatAttack Stat = iota
	// StatDefense はぼうぎょ。
	StatDefense
	// StatSpeed はすばやさ。
	StatSpeed
	// StatSpecial はとくしゅ。
	StatSpecial
	// StatAccuracy は命中率。
	StatAccuracy
	// StatEvasion は回避率。
	StatEvasion
)

// String はStatの名前を返す。
func (s Stat) String() string {
	switch s {
	case StatAttack:
		return "attack"
	case StatDefense:
		return "defense"
	case StatSpeed:
		return "speed"
	case StatSpecial:
		return "special"
	case StatAccuracy:
		return "accuracy"
	case StatEvasion:
		return "evasion"
	default:
		return "unknown"
	}
}

// stage は指定した能力のstageへのpointerを返す。
func (s *StatStages) stage(stat Stat) *int {
	switch stat {
	case StatAttack:
		return &s.Attack
	case StatDefense:
		return &s.Defense
	case StatSpeed:
		return &s.Speed
	case StatSpecial:
		return &s.Special
	case StatAccuracy:
		return &s.Accuracy
	case StatEvasion:
		return &s.Evasion
	default:
		return nil
	}
}

// ChangeStage は能力のstageを変える。実際に変えた段階数を返す。
//
// stageは上下限で頭打ちになるので、すでに限界なら0を返す。
func ChangeStage(stages *StatStages, stat Stat, delta int) int {
	target := stages.stage(stat)
	if target == nil || delta == 0 {
		return 0
	}

	before := *target
	after := before + delta
	if after > StageMax {
		after = StageMax
	}
	if after < StageMin {
		after = StageMin
	}

	*target = after
	return after - before
}
