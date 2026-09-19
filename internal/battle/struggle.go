package battle

// MoveStruggle はStruggleの識別子。
//
// 固定move setには含めない。Dataにも載せず、Battle Engineだけが持つ。
const MoveStruggle MoveID = "struggle"

// struggleMove はStruggleの定義。
//
// 使える技が1つも無いときの代替行動で、通常の技と同じdamage pipelineを通る。
// Power 50・ノーマル（したがって物理）・必中で、STAB・タイプ相性・急所・
// ダメージ乱数はすべて通常どおり適用される。PPは持たない。
//
// 仕様の正はBattle Rules Specificationの Struggle section。
var struggleMove = Move{
	ID:       MoveStruggle,
	Type:     TypeNormal,
	Power:    50,
	Accuracy: MaxAccuracy,
	Effect:   EffectRecoil,
}
