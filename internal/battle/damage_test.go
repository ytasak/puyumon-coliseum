package battle

import "testing"

// 物理技用の能力値。特殊側へ意図的に違う値を入れ、物理と特殊を取り違えたら
// testが落ちるようにする。
func physicalAttacker(attack int) Stats {
	return Stats{HP: 200, Attack: attack, Defense: 100, Speed: 100, Special: 1}
}

func physicalDefender(defense int) Stats {
	return Stats{HP: 200, Attack: 100, Defense: defense, Speed: 100, Special: 1}
}

func specialAttacker(special int) Stats {
	return Stats{HP: 200, Attack: 1, Defense: 100, Speed: 100, Special: special}
}

func specialDefender(special int) Stats {
	return Stats{HP: 200, Attack: 100, Defense: 1, Speed: 100, Special: special}
}

// 乱数を適用する前のダメージのgolden test。
//
// 期待値はpokeredのengine/battle/core.asmから起こした参照実装（Python）で
// 別に求めたもの。同じ仕様から独立に2つ実装して一致することを確認している。
// 丸めの順序が変わればここが落ちる。
func TestBaseDamageGoldenVectors(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		in   DamageInput
		want int
	}{
		{
			// 24 * 85 * 180 / 150 / 50 = 48、+2 で 50
			name: "physical neutral",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 50,
		},
		{
			name: "special neutral",
			in: DamageInput{
				Level: 50, Power: 95, MoveType: TypeWater,
				AttackerTyping: SingleType(TypeNormal), DefenderTyping: SingleType(TypeNormal),
				AttackerStats: specialAttacker(160), DefenderStats: specialDefender(140),
			},
			want: 49,
		},
		{
			// STABで1.5倍。50 + 25 = 75
			name: "stab",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeNormal), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 75,
		},
		{
			// かくとう技をノーマルへ。2倍
			name: "super effective",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeFighting,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeNormal),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 100,
		},
		{
			// ノーマル技をいわへ。0.5倍
			name: "not very effective",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeRock),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 25,
		},
		{
			// むし技をほのお/ひこうへ。0.5 x 0.5
			name: "quarter effective",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeBug,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: DualType(TypeFire, TypeFlying),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 12,
		},
		{
			// こおり技をくさ/ドラゴンへ。STAB込みで 1.5 x 2 x 2
			name: "stab and four times",
			in: DamageInput{
				Level: 55, Power: 95, MoveType: TypeIce,
				AttackerTyping: SingleType(TypeIce), DefenderTyping: DualType(TypeGrass, TypeDragon),
				AttackerStats: specialAttacker(170), DefenderStats: specialDefender(130),
			},
			want: 364,
		},
		{
			// ノーマル技はゴーストへ効かない
			name: "immune",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeGhost),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 0,
		},
		{
			// 急所はLevelを2倍にして計算する。50に対して95
			name: "critical",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				Critical: true,
			},
			want: 95,
		},
		{
			name: "attack raised by two stages",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				AttackerStages: StatStages{Attack: 2},
			},
			want: 101,
		},
		{
			name: "defense raised by two stages",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				DefenderStages: StatStages{Defense: 2},
			},
			want: 26,
		},
		{
			// 急所は能力変化を無視するので、上がっていても下がっていても同じ値になる
			name: "critical ignores raised stages",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				AttackerStages: StatStages{Attack: 2}, DefenderStages: StatStages{Defense: 2},
				Critical: true,
			},
			want: 95,
		},
		{
			name: "critical ignores lowered stages",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				AttackerStages: StatStages{Attack: -2}, DefenderStages: StatStages{Defense: -2},
				Critical: true,
			},
			want: 95,
		},
		{
			// 能力値が255を超えると攻守とも1/4にしてから計算する
			name: "stats scaled above 255",
			in: DamageInput{
				Level: 55, Power: 85, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
				AttackerStages: StatStages{Attack: 4},
			},
			want: 150,
		},
		{
			// 下限の2
			name: "minimum damage",
			in: DamageInput{
				Level: 5, Power: 10, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(20), DefenderStats: physicalDefender(200),
			},
			want: 2,
		},
		{
			// 上限の999
			name: "damage cap",
			in: DamageInput{
				Level: 100, Power: 250, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(999), DefenderStats: physicalDefender(1),
			},
			want: 999,
		},
		{
			name: "no power",
			in: DamageInput{
				Level: 55, Power: 0, MoveType: TypeNormal,
				AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
				AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
			},
			want: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := BaseDamage(tt.in); got != tt.want {
				t.Errorf("BaseDamage() = %d, want %d", got, tt.want)
			}
		})
	}
}

// 物理技はAttackとDefense、特殊技は双方のSpecialを使う。
// 攻撃側はみずタイプにしてSTABが乗らないようにし、防御側はノーマルで相性を等倍にしている。
func TestBaseDamageUsesCategoryStats(t *testing.T) {
	t.Parallel()

	in := DamageInput{
		Level: 55, Power: 85,
		AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeNormal),
		AttackerStats: Stats{HP: 200, Attack: 180, Defense: 100, Speed: 100, Special: 90},
		DefenderStats: Stats{HP: 200, Attack: 100, Defense: 150, Speed: 100, Special: 200},
	}

	// じめんは物理なのでAttack 180 と Defense 150 を使う。
	in.MoveType = TypeGround
	if got := BaseDamage(in); got != 50 {
		t.Errorf("physical damage = %d, want 50", got)
	}

	// でんきは特殊なので双方のSpecial（90と200）を使う。
	in.MoveType = TypeElectric
	if got := BaseDamage(in); got != 20 {
		t.Errorf("special damage = %d, want 20", got)
	}
}

// 乱数の適用。ダメージが1以下なら変化しない。
func TestApplyDamageRoll(t *testing.T) {
	t.Parallel()

	tests := []struct {
		damage int
		roll   int
		want   int
	}{
		{50, 255, 50}, // 最大の乱数では減らない
		{50, 217, 42}, // 最小の乱数で約85%
		{364, 217, 309},
		{364, 255, 364},
		{2, 217, 1},
		{1, 217, 1}, // 1以下は乱数を適用しない
		{0, 217, 0},
	}

	for _, tt := range tests {
		if got := ApplyDamageRoll(tt.damage, tt.roll); got != tt.want {
			t.Errorf("ApplyDamageRoll(%d, %d) = %d, want %d", tt.damage, tt.roll, got, tt.want)
		}
	}
}

// ダメージ乱数は217〜255に収まり、その範囲を実際に使い切る。
func TestDamageRollRange(t *testing.T) {
	t.Parallel()

	seen := make(map[int]bool, maxDamageRoll-minDamageRoll+1)
	rng := NewRand(155)
	for i := 0; i < 5000; i++ {
		roll := DamageRoll(rng)
		if roll < minDamageRoll || roll > maxDamageRoll {
			t.Fatalf("DamageRoll() = %d, want [%d,%d]", roll, minDamageRoll, maxDamageRoll)
		}
		seen[roll] = true
	}
	for roll := minDamageRoll; roll <= maxDamageRoll; roll++ {
		if !seen[roll] {
			t.Errorf("DamageRoll() never returned %d", roll)
		}
	}
}

// 同じseedからは同じダメージ列になる。
func TestCalculateDamageIsReproducible(t *testing.T) {
	t.Parallel()

	in := DamageInput{
		Level: 55, Power: 85, MoveType: TypeNormal,
		AttackerTyping: SingleType(TypeWater), DefenderTyping: SingleType(TypeWater),
		AttackerStats: physicalAttacker(180), DefenderStats: physicalDefender(150),
	}

	first, second := NewRand(20260918), NewRand(20260918)
	for i := 0; i < 50; i++ {
		a, b := CalculateDamage(first, in), CalculateDamage(second, in)
		if a != b {
			t.Fatalf("damage %d: %d != %d", i, a, b)
		}
		if a < 42 || a > 50 {
			t.Fatalf("damage %d = %d, want [42,50]", i, a)
		}
	}
}
