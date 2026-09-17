package battle

import "testing"

// genIChartExpectation は攻撃タイプ1つ分の期待値。
// 出荷データとは別の並べ方で書き、転記と表の組み立ての両方を確かめる。
type genIChartExpectation struct {
	attack  Type
	super   []Type // 効果ばつぐん（2倍）
	notVery []Type // いまひとつ（0.5倍）
	none    []Type // 効果なし（0倍）
}

// genIChart はGeneration I（赤・緑）のタイプ相性。ここに挙げない組み合わせは等倍。
var genIChart = []genIChartExpectation{
	{
		attack: TypeNormal,
		none:   []Type{TypeGhost},
		notVery: []Type{
			TypeRock,
		},
	},
	{
		attack:  TypeFighting,
		super:   []Type{TypeNormal, TypeRock, TypeIce},
		notVery: []Type{TypePoison, TypeFlying, TypePsychic, TypeBug},
		none:    []Type{TypeGhost},
	},
	{
		attack:  TypeFlying,
		super:   []Type{TypeFighting, TypeBug, TypeGrass},
		notVery: []Type{TypeElectric, TypeRock},
	},
	{
		attack:  TypePoison,
		super:   []Type{TypeGrass, TypeBug}, // Bugへの2倍はGeneration I固有
		notVery: []Type{TypePoison, TypeGround, TypeRock, TypeGhost},
	},
	{
		attack:  TypeGround,
		super:   []Type{TypeFire, TypeElectric, TypeRock, TypePoison},
		notVery: []Type{TypeGrass, TypeBug},
		none:    []Type{TypeFlying},
	},
	{
		attack:  TypeRock,
		super:   []Type{TypeFire, TypeFlying, TypeBug, TypeIce},
		notVery: []Type{TypeFighting, TypeGround},
	},
	{
		attack:  TypeBug,
		super:   []Type{TypeGrass, TypePsychic, TypePoison}, // Poisonへの2倍はGeneration I固有
		notVery: []Type{TypeFire, TypeFighting, TypeFlying, TypeGhost},
	},
	{
		attack: TypeGhost,
		super:  []Type{TypeGhost},
		// エスパーへの0倍はGeneration I固有。
		none: []Type{TypeNormal, TypePsychic},
	},
	{
		attack:  TypeFire,
		super:   []Type{TypeGrass, TypeIce, TypeBug},
		notVery: []Type{TypeFire, TypeWater, TypeRock, TypeDragon},
	},
	{
		attack:  TypeWater,
		super:   []Type{TypeFire, TypeRock, TypeGround},
		notVery: []Type{TypeWater, TypeGrass, TypeDragon},
	},
	{
		attack:  TypeGrass,
		super:   []Type{TypeWater, TypeGround, TypeRock},
		notVery: []Type{TypeGrass, TypeFire, TypeBug, TypePoison, TypeFlying, TypeDragon},
	},
	{
		attack:  TypeElectric,
		super:   []Type{TypeWater, TypeFlying},
		notVery: []Type{TypeElectric, TypeGrass, TypeDragon},
		none:    []Type{TypeGround},
	},
	{
		attack:  TypePsychic,
		super:   []Type{TypeFighting, TypePoison},
		notVery: []Type{TypePsychic},
	},
	{
		attack:  TypeIce,
		super:   []Type{TypeGrass, TypeGround, TypeFlying, TypeDragon},
		notVery: []Type{TypeWater, TypeIce}, // ほのおへのいまひとつはGeneration II以降
	},
	{
		attack: TypeDragon,
		super:  []Type{TypeDragon},
	},
}

// 15x15すべての組み合わせを、出荷されたGeneration Iの相性と突き合わせる。
// 表に無い組み合わせが等倍であることもここで確かめる。
func TestGenIChart(t *testing.T) {
	t.Parallel()

	if len(genIChart) != 15 {
		t.Fatalf("chart covers %d attacking types, want 15", len(genIChart))
	}

	for _, row := range genIChart {
		want := make(map[Type]Effectiveness, 15)
		for _, defend := range AllTypes() {
			want[defend] = Neutral
		}
		for _, defend := range row.super {
			want[defend] = SuperEffective
		}
		for _, defend := range row.notVery {
			want[defend] = NotVeryEffective
		}
		for _, defend := range row.none {
			want[defend] = NoEffect
		}

		for _, defend := range AllTypes() {
			if got := Against(row.attack, defend); got != want[defend] {
				t.Errorf("Against(%v, %v) = %d, want %d", row.attack, defend, got, want[defend])
			}
		}
	}
}

// 後の世代で変わった相性を、Generation Iの値のまま保っているかを個別に押さえる。
// 現代世代の知識で「直して」しまう事故を防ぐためのtest。
func TestGenIChartKeepsGenerationISpecificMatchups(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		attack Type
		defend Type
		want   Effectiveness
	}{
		{"poison hits bug hard in gen 1", TypePoison, TypeBug, SuperEffective},
		{"bug hits poison hard in gen 1", TypeBug, TypePoison, SuperEffective},
		{"bug is weak against ghost in gen 1", TypeBug, TypeGhost, NotVeryEffective},
		{"ghost does not affect psychic in gen 1", TypeGhost, TypePsychic, NoEffect},
		{"ice is not resisted by fire in gen 1", TypeIce, TypeFire, Neutral},
		{"normal does not affect ghost", TypeNormal, TypeGhost, NoEffect},
		{"fighting does not affect ghost", TypeFighting, TypeGhost, NoEffect},
		{"ground does not affect flying", TypeGround, TypeFlying, NoEffect},
		{"electric does not affect ground", TypeElectric, TypeGround, NoEffect},
		{"psychic is neutral against ghost", TypePsychic, TypeGhost, Neutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := Against(tt.attack, tt.defend); got != tt.want {
				t.Errorf("Against(%v, %v) = %d, want %d", tt.attack, tt.defend, got, tt.want)
			}
		})
	}
}

// 複合タイプでは各タイプの相性を合成する。
func TestAgainstTypingComposesEachType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		attack   Type
		defender Typing
		want     Effectiveness
	}{
		{"single type", TypeElectric, SingleType(TypeWater), SuperEffective},
		{"both weak", TypeElectric, DualType(TypeWater, TypeFlying), 400},
		{"both resist", TypeIce, DualType(TypeWater, TypeIce), 25},
		{"weak and resistant cancel out", TypeFighting, DualType(TypeNormal, TypeFlying), Neutral},
		{"immunity wins over weakness", TypeGround, DualType(TypeFire, TypeFlying), NoEffect},
		{"immunity wins over resistance", TypeElectric, DualType(TypeGrass, TypeGround), NoEffect},
		{"neutral against both", TypeNormal, DualType(TypeWater, TypePsychic), Neutral},
		{"no types at all", TypeNormal, Typing{}, Neutral},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if got := AgainstTyping(tt.attack, tt.defender); got != tt.want {
				t.Errorf("AgainstTyping(%v, %+v) = %d, want %d", tt.attack, tt.defender, got, tt.want)
			}
		})
	}
}

// タイプが決まっていない組み合わせは等倍として扱い、参照表の外を引かない。
func TestAgainstUnknownTypeIsNeutral(t *testing.T) {
	t.Parallel()

	for _, tt := range []struct {
		attack Type
		defend Type
	}{
		{NoType, TypeWater},
		{TypeWater, NoType},
		{Type(99), TypeWater},
		{TypeWater, Type(-1)},
	} {
		if got := Against(tt.attack, tt.defend); got != Neutral {
			t.Errorf("Against(%v, %v) = %d, want %d", tt.attack, tt.defend, got, Neutral)
		}
	}
}

// 相性の合成は倍率どうしの掛け算になる。
func TestEffectivenessTimes(t *testing.T) {
	t.Parallel()

	tests := []struct {
		a, b Effectiveness
		want Effectiveness
	}{
		{Neutral, Neutral, Neutral},
		{SuperEffective, SuperEffective, 400},
		{NotVeryEffective, NotVeryEffective, 25},
		{SuperEffective, NotVeryEffective, Neutral},
		{NoEffect, SuperEffective, NoEffect},
		{SuperEffective, NoEffect, NoEffect},
	}

	for _, tt := range tests {
		if got := tt.a.Times(tt.b); got != tt.want {
			t.Errorf("%d.Times(%d) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}
