package battle

// 状態異常に関するGeneration Iの定数。
const (
	// MinSleepTurns / MaxSleepTurns はねむりの継続turn数の範囲。
	MinSleepTurns = 1
	MaxSleepTurns = 7

	// FullParalysisThreshold はまひで行動できなくなる乱数の上限。
	// 実機の25%（25 * 255 / 100）にあたる。
	FullParalysisThreshold = 63

	// residualDivisor は毒・やけどの継続ダメージを求めるときに最大HPを割る数。
	residualDivisor = 16

	// randomByteValues は実機が1 byteの乱数で判定する箇所の、取り得る値の数。
	randomByteValues = 256
)

// StatusBlock はturn開始時に行動を妨げた理由。
//
// 妨げた理由ごとに見せ方が変わるため、真偽値ではなく理由として返す。
// Eventの組み立てはturn resolverが行う。
type StatusBlock int

const (
	// NotBlocked は状態異常に妨げられていない。
	NotBlocked StatusBlock = iota
	// BlockedBySleep はねむったままで動けない。
	BlockedBySleep
	// BlockedByWakeUp は目を覚ましたが、そのturnは動けない。
	BlockedByWakeUp
	// BlockedByFreeze はこおって動けない。
	BlockedByFreeze
	// BlockedByParalysis はまひで動けない。
	BlockedByParalysis
)

// String はStatusBlockの名前を返す。testとlogの出力に使う。
func (b StatusBlock) String() string {
	switch b {
	case NotBlocked:
		return "not_blocked"
	case BlockedBySleep:
		return "sleep"
	case BlockedByWakeUp:
		return "wake_up"
	case BlockedByFreeze:
		return "freeze"
	case BlockedByParalysis:
		return "paralysis"
	default:
		return "unknown"
	}
}

// SleepDuration はねむりの継続turn数を1つ引く。1〜7の一様分布。
//
// 実機は乱数の下位3bitを取り、0なら引き直して1〜7にしている。
// こちらは同じ分布を直接引く。
func SleepDuration(rng RNG) int {
	return MinSleepTurns + rng.IntN(MaxSleepTurns-MinSleepTurns+1)
}

// IsFullyParalyzed はまひで行動できないかを判定する。
func IsFullyParalyzed(rng RNG) bool {
	return rng.IntN(randomByteValues) < FullParalysisThreshold
}

// ParalyzedSpeed はまひ中のSpeedを返す。Generation Iでは4分の1になる。
//
// 下限は設けない。実機も端数を切り捨てるだけで、1へ引き上げない。
func ParalyzedSpeed(speed int) int {
	if speed < 0 {
		return 0
	}
	return speed / 4
}

// BurnedAttack はやけど中のAttackを返す。Generation Iでは半分になる。
func BurnedAttack(attack int) int {
	if attack < 0 {
		return 0
	}
	return attack / 2
}

// ResidualDamage は毒・やけどでturnごとに受けるダメージを返す。
//
// 最大HPの16分の1で、切り捨てた結果が0でも1は受ける。
func ResidualDamage(maxHP int) int {
	damage := maxHP / residualDivisor
	if damage < 1 {
		return 1
	}
	return damage
}

// ApplyStatus は状態異常を付与する。付与できたらtrueを返す。
//
// Generation Iではmajor statusを同時に1つしか持てないため、すでに何か持っていれば
// 何も起きない。ねむりの継続turnもここで決める。
//
// 付与する確率は技ごとのデータで決まるもので、ここでは扱わない。
// Eventの組み立てはturn resolverが行う。
func ApplyStatus(rng RNG, p *Pokemon, status MajorStatus) bool {
	if status == NoStatus || !status.valid() {
		return false
	}
	if p.Fainted() || p.Status != NoStatus {
		return false
	}

	p.Status = status
	if status == Sleep {
		p.SleepTurns = SleepDuration(rng)
	}
	return true
}

// CheckStatus はturn開始時の状態異常を処理し、行動を妨げる理由を返す。
//
// ねむりはcounterを1減らし、0になったら起床させる。**起床したturnは行動できない**
// のがGeneration Iの挙動で、現代世代とここが違う。
//
// こおりは解けない。Generation Iには自然解凍が無く、解除は技の側から
// Statusを消して行う。
//
// 乱数を引くのはまひのときだけ。ほかの状態異常は乱数を使わない。
func CheckStatus(rng RNG, p *Pokemon) StatusBlock {
	switch p.Status {
	case Sleep:
		if p.SleepTurns > 0 {
			p.SleepTurns--
		}
		if p.SleepTurns == 0 {
			p.Status = NoStatus
			return BlockedByWakeUp
		}
		return BlockedBySleep
	case Freeze:
		return BlockedByFreeze
	case Paralysis:
		if IsFullyParalyzed(rng) {
			return BlockedByParalysis
		}
	}
	return NotBlocked
}

// ApplyResidualDamage は毒・やけどの継続ダメージを与え、与えたダメージを返す。
//
// 毒でもやけどでもなければ0を返す。HPは0より下がらない。
//
// Generation Iはこのダメージを「先攻が動く → 先攻の継続ダメージ →
// 後攻が動く → 後攻の継続ダメージ」の順で処理する。turnの終わりへ一括しない。
// 呼ぶ順序はturn resolverが決める。
func ApplyResidualDamage(p *Pokemon) int {
	if p.Status != Poison && p.Status != Burn {
		return 0
	}
	if p.Fainted() {
		return 0
	}

	damage := ResidualDamage(p.Stats.HP)
	if damage > p.CurrentHP {
		damage = p.CurrentHP
	}
	p.CurrentHP -= damage
	return damage
}
