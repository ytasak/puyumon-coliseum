package battle

// Effectiveness はタイプ相性の倍率。100を等倍とする整数で持つ。
//
// Generation Iは相性を10倍の整数（0 / 5 / 10 / 20）で保持し、ダメージへ
// 防御側のタイプごとに順番に掛けて都度切り捨てる。合成した倍率を一度に掛けた場合と
// 結果が変わることがあるため、ダメージ計算では AgainstTyping ではなく
// Against をタイプごとに使う（damage formulaのIssueの範囲）。
type Effectiveness int

const (
	// NoEffect は効果がない（0倍）。
	NoEffect Effectiveness = 0
	// NotVeryEffective は効果がいまひとつ（0.5倍）。
	NotVeryEffective Effectiveness = 50
	// Neutral は等倍。
	Neutral Effectiveness = 100
	// SuperEffective は効果ばつぐん（2倍）。
	SuperEffective Effectiveness = 200
)

// Times は2つの相性を合成した倍率を返す。複合タイプの合成に使う。
func (e Effectiveness) Times(other Effectiveness) Effectiveness {
	return e * other / 100
}

// Percent は倍率を百分率で返す。100が等倍。
func (e Effectiveness) Percent() int {
	return int(e)
}

// typeMatchup は等倍にならない組み合わせ1つ分。
type typeMatchup struct {
	attack Type
	defend Type
	effect Effectiveness
}

// genIMatchups はGeneration I（赤・緑）のタイプ相性。
//
// 出荷されたROMのデータ（pokeredのdata/types/type_matchups.asm）と同じ並び・同じ値で持つ。
// ここに無い組み合わせはすべて等倍になる。後の世代で変わった相性を混ぜない。
//
// 後の世代と異なるものには個別に印をつけている。
var genIMatchups = []typeMatchup{
	{TypeWater, TypeFire, SuperEffective},
	{TypeFire, TypeGrass, SuperEffective},
	{TypeFire, TypeIce, SuperEffective},
	{TypeGrass, TypeWater, SuperEffective},
	{TypeElectric, TypeWater, SuperEffective},
	{TypeWater, TypeRock, SuperEffective},
	{TypeGround, TypeFlying, NoEffect},
	{TypeWater, TypeWater, NotVeryEffective},
	{TypeFire, TypeFire, NotVeryEffective},
	{TypeElectric, TypeElectric, NotVeryEffective},
	{TypeIce, TypeIce, NotVeryEffective},
	{TypeGrass, TypeGrass, NotVeryEffective},
	{TypePsychic, TypePsychic, NotVeryEffective},
	{TypeFire, TypeWater, NotVeryEffective},
	{TypeGrass, TypeFire, NotVeryEffective},
	{TypeWater, TypeGrass, NotVeryEffective},
	{TypeElectric, TypeGrass, NotVeryEffective},
	{TypeNormal, TypeRock, NotVeryEffective},
	{TypeNormal, TypeGhost, NoEffect},
	{TypeGhost, TypeGhost, SuperEffective},
	{TypeFire, TypeBug, SuperEffective},
	{TypeFire, TypeRock, NotVeryEffective},
	{TypeWater, TypeGround, SuperEffective},
	{TypeElectric, TypeGround, NoEffect},
	{TypeElectric, TypeFlying, SuperEffective},
	{TypeGrass, TypeGround, SuperEffective},
	{TypeGrass, TypeBug, NotVeryEffective},
	{TypeGrass, TypePoison, NotVeryEffective},
	{TypeGrass, TypeRock, SuperEffective},
	{TypeGrass, TypeFlying, NotVeryEffective},
	{TypeIce, TypeWater, NotVeryEffective},
	{TypeIce, TypeGrass, SuperEffective},
	{TypeIce, TypeGround, SuperEffective},
	{TypeIce, TypeFlying, SuperEffective},
	{TypeFighting, TypeNormal, SuperEffective},
	{TypeFighting, TypePoison, NotVeryEffective},
	{TypeFighting, TypeFlying, NotVeryEffective},
	{TypeFighting, TypePsychic, NotVeryEffective},
	{TypeFighting, TypeBug, NotVeryEffective},
	{TypeFighting, TypeRock, SuperEffective},
	{TypeFighting, TypeIce, SuperEffective},
	{TypeFighting, TypeGhost, NoEffect},
	{TypePoison, TypeGrass, SuperEffective},
	{TypePoison, TypePoison, NotVeryEffective},
	{TypePoison, TypeGround, NotVeryEffective},
	{TypePoison, TypeBug, SuperEffective}, // Generation I固有。Gen II以降は等倍
	{TypePoison, TypeRock, NotVeryEffective},
	{TypePoison, TypeGhost, NotVeryEffective},
	{TypeGround, TypeFire, SuperEffective},
	{TypeGround, TypeElectric, SuperEffective},
	{TypeGround, TypeGrass, NotVeryEffective},
	{TypeGround, TypeBug, NotVeryEffective},
	{TypeGround, TypeRock, SuperEffective},
	{TypeGround, TypePoison, SuperEffective},
	{TypeFlying, TypeElectric, NotVeryEffective},
	{TypeFlying, TypeFighting, SuperEffective},
	{TypeFlying, TypeBug, SuperEffective},
	{TypeFlying, TypeGrass, SuperEffective},
	{TypeFlying, TypeRock, NotVeryEffective},
	{TypePsychic, TypeFighting, SuperEffective},
	{TypePsychic, TypePoison, SuperEffective},
	{TypeBug, TypeFire, NotVeryEffective},
	{TypeBug, TypeGrass, SuperEffective},
	{TypeBug, TypeFighting, NotVeryEffective},
	{TypeBug, TypeFlying, NotVeryEffective},
	{TypeBug, TypePsychic, SuperEffective},
	{TypeBug, TypeGhost, NotVeryEffective},
	{TypeBug, TypePoison, SuperEffective}, // Generation I固有。Gen II以降はいまひとつ
	{TypeRock, TypeFire, SuperEffective},
	{TypeRock, TypeFighting, NotVeryEffective},
	{TypeRock, TypeGround, NotVeryEffective},
	{TypeRock, TypeFlying, SuperEffective},
	{TypeRock, TypeBug, SuperEffective},
	{TypeRock, TypeIce, SuperEffective},
	{TypeGhost, TypeNormal, NoEffect},
	// Generation I固有。ゴースト技はエスパーに効かない。
	// 本来は効果ばつぐんの意図だったとされる実装ミスだが、初代の対戦を
	// 決定づけた挙動なので出荷されたとおりに持つ（Product Owner判断）。
	{TypeGhost, TypePsychic, NoEffect},
	{TypeFire, TypeDragon, NotVeryEffective},
	{TypeWater, TypeDragon, NotVeryEffective},
	{TypeElectric, TypeDragon, NotVeryEffective},
	{TypeGrass, TypeDragon, NotVeryEffective},
	{TypeIce, TypeDragon, SuperEffective},
	{TypeDragon, TypeDragon, SuperEffective},
	// こおり技がほのおへいまひとつになるのはGeneration II以降。ここでは等倍。
}

// chart は genIMatchups から作る参照表。[攻撃タイプ][防御タイプ]で引く。
//
// 表に無い組み合わせを等倍で埋めてから上書きするので、
// 相性の定義はgenIMatchupsだけを見ればよい。
var chart = buildChart(genIMatchups)

// buildChart は相性の一覧から参照表を作る。
func buildChart(matchups []typeMatchup) [typeCount][typeCount]Effectiveness {
	var built [typeCount][typeCount]Effectiveness
	for attack := range built {
		for defend := range built[attack] {
			built[attack][defend] = Neutral
		}
	}
	for _, m := range matchups {
		built[m.attack][m.defend] = m.effect
	}
	return built
}

// Against は攻撃タイプと防御側のタイプ1つ分の相性を返す。
//
// 定義外のタイプを渡した場合は等倍を返す。
func Against(attack, defend Type) Effectiveness {
	if !attack.valid() || !defend.valid() {
		return Neutral
	}
	return chart[attack][defend]
}

// AgainstTyping は防御側のタイプ構成全体に対する相性を返す。
//
// 複合タイプでは各タイプの相性を合成する。片方が無効なら全体も無効になる。
// ダメージ計算ではGeneration Iの丸め順序に合わせて Against をタイプごとに
// 使う必要があるため、この関数は表示や判定に使う。
func AgainstTyping(attack Type, defender Typing) Effectiveness {
	effect := Neutral
	for _, defend := range defender.List() {
		effect = effect.Times(Against(attack, defend))
	}
	return effect
}

// valid は参照表を引けるタイプかを返す。NoTypeは含まない。
func (t Type) valid() bool {
	return t >= TypeNormal && t <= TypeDragon
}
