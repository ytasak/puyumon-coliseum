package battle

// ダメージ計算で使うGeneration Iの定数。
const (
	// MinDamage は相性で0にならない限りのダメージの下限。
	MinDamage = 2
	// MaxDamage は乱数を適用する前のダメージの上限。
	MaxDamage = 999
	// MaxStat は能力変化を適用したあとの能力値の上限。
	MaxStat = 999

	// statScaleLimit を超える能力値があると、攻守の能力値を1/4にしてから計算する。
	statScaleLimit = 255
	// minDamageRoll / maxDamageRoll はダメージ乱数の範囲。
	minDamageRoll = 217
	maxDamageRoll = 255
)

// DamageInput はGeneration Iのダメージ計算に必要な値。
//
// 能力値は stat stage を適用する前の実数値を渡す。stageの適用と、
// 急所のときにstageを無視する判断はこのpackage側で行う。
type DamageInput struct {
	// Level は攻撃側のLevel。急所のときは計算の中で2倍にする。
	Level int

	// Power は技の威力。0以下ならダメージは0になる。
	Power int

	// MoveType は技のタイプ。物理か特殊かもここで決まる。
	MoveType Type

	// AttackerTyping はSTAB判定に使う攻撃側のタイプ。
	AttackerTyping Typing
	// DefenderTyping は相性に使う防御側のタイプ。
	DefenderTyping Typing

	// AttackerStats / DefenderStats は stat stage を適用する前の実数値。
	AttackerStats Stats
	DefenderStats Stats

	// AttackerStages / DefenderStages は現在のstat stage。
	// 急所のときはGeneration Iに従って無視する。
	AttackerStages StatStages
	DefenderStages StatStages

	// Critical は急所かどうか。
	Critical bool
}

// BaseDamage は乱数を適用する前のダメージを返す。
//
// Generation Iの整数演算をそのまま行う。途中の割り算はすべて切り捨てで、
// 掛ける順序と切り捨ての位置が変わると結果が変わる。
//
//	damage = floor(2 * Level / 5) + 2      急所ならLevelを2倍にしてから
//	damage = damage * Power
//	damage = damage * Attack
//	damage = floor(damage / Defense)
//	damage = floor(damage / 50)
//	damage = min(damage, 997) + 2          上限999
//	STABがあれば damage = damage + floor(damage / 2)
//	防御側のタイプごとに damage = floor(damage * 相性 / 100)
//
// 相性を合成してから1度に掛けると結果が変わることがあるため、
// 防御側のタイプごとに順に適用する。
func BaseDamage(in DamageInput) int {
	if in.Power <= 0 {
		return 0
	}

	level := in.Level
	if in.Critical {
		// 急所はダメージを倍にするのではなく、Levelを倍にして計算する。
		level *= 2
	}

	attack, defense := in.offensiveStats()

	damage := 2*level/5 + 2
	damage *= in.Power
	damage *= attack
	damage /= defense
	damage /= 50

	if damage > MaxDamage-MinDamage {
		damage = MaxDamage - MinDamage
	}
	damage += MinDamage

	if HasSTAB(in.MoveType, in.AttackerTyping) {
		damage += damage / 2
	}
	for _, defending := range in.DefenderTyping.List() {
		damage = damage * int(Against(in.MoveType, defending)) / 100
	}
	return damage
}

// offensiveStats は計算に使う攻撃側と防御側の能力値を返す。
//
// 技のタイプが物理ならAttackとDefense、特殊なら双方のSpecialを使う。
// 急所のときは能力変化を無視した素の値を使う（Generation Iの挙動）。
func (in DamageInput) offensiveStats() (attack, defense int) {
	if in.MoveType.Category() == Special {
		attack, defense = in.AttackerStats.Special, in.DefenderStats.Special
		if !in.Critical {
			attack = battleStat(attack, in.AttackerStages.Special)
			defense = battleStat(defense, in.DefenderStages.Special)
		}
	} else {
		attack, defense = in.AttackerStats.Attack, in.DefenderStats.Defense
		if !in.Critical {
			attack = battleStat(attack, in.AttackerStages.Attack)
			defense = battleStat(defense, in.DefenderStages.Defense)
		}
	}

	// どちらかが255を超えると、Generation Iは両方を1/4にしてから計算する。
	// 1 byteへ収めるための処理で、能力を上げたときの結果に影響する。
	if attack > statScaleLimit || defense > statScaleLimit {
		attack /= 4
		defense /= 4
		if attack < 1 {
			attack = 1
		}
	}
	// 実機は防御側が0になると0除算で停止する。再現できないので1として扱う。
	if defense < 1 {
		defense = 1
	}
	return attack, defense
}

// battleStat はstat stageを適用した能力値を返す。上限は999。
func battleStat(stat, stage int) int {
	modified := ApplyStage(stat, stage)
	if modified > MaxStat {
		return MaxStat
	}
	return modified
}

// DamageRoll はダメージ乱数を1つ引く。217〜255の一様分布。
func DamageRoll(rng RNG) int {
	return minDamageRoll + rng.IntN(maxDamageRoll-minDamageRoll+1)
}

// ApplyDamageRoll は乱数をダメージへ適用する。
//
// ダメージが1以下のときは変化しない。Generation Iも同じ条件で乱数を適用しない。
func ApplyDamageRoll(damage, roll int) int {
	if damage <= 1 {
		return damage
	}
	return damage * roll / maxDamageRoll
}

// CalculateDamage は乱数まで含めたダメージを返す。
//
// Eventの Damage と名前がぶつからないよう、実機のroutine名に合わせている。
//
// 乱数は常に1つ消費する。実機はダメージが1以下のとき乱数を引かないが、
// こちらはROMとbit互換ではないので、消費数を一定にして追いやすさを優先した。
func CalculateDamage(rng RNG, in DamageInput) int {
	damage := BaseDamage(in)
	return ApplyDamageRoll(damage, DamageRoll(rng))
}
