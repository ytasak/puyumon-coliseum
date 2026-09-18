// Package roster は本作で使う6キャラクターと固定move setの定義を持つ。
//
// 値はGeneration Iの実データ（pokeredの base_stats / moves）に合わせている。
// Battle Engineが解釈できる形（battle.Data）へ変換して渡すのがこのpackageの役目で、
// 対戦のルールそのものは持たない。
//
// 公開表示名は未定。internal IDだけを確定させ、表示名はあとから入れられるよう
// 分離してある。仕様の正はLinearのYTA-20。
package roster

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// 技のinternal ID。公開表示名とは分けて持つ。
const (
	MoveSlam      battle.MoveID = "slam"      // ノーマル物理。まひ付与
	MoveOverdrive battle.MoveID = "overdrive" // 高威力。反動あり
	MoveQuake     battle.MoveID = "quake"     // じめん
	MoveIcestorm  battle.MoveID = "icestorm"  // こおり。こおり付与
	MoveTide      battle.MoveID = "tide"      // みず
	MoveMindblast battle.MoveID = "mindblast" // エスパー。Special下降
	MoveSpark     battle.MoveID = "spark"     // でんき。まひ付与
	MoveNumb      battle.MoveID = "numb"      // まひ付与のみ
	MoveMend      battle.MoveID = "mend"      // 最大HPの半分回復
	MoveDoze      battle.MoveID = "doze"      // 全回復して自分がねむり
	MoveSlumber   battle.MoveID = "slumber"   // ねむり付与
	MoveSpores    battle.MoveID = "spores"    // ねむり付与
	MoveDrain     battle.MoveID = "drain"     // ダメージの半分を吸収
	MoveBurst     battle.MoveID = "burst"     // 自爆
	MoveNeedles   battle.MoveID = "needles"   // 2〜5回ヒット
	MoveDash      battle.MoveID = "dash"      // 自分のSpeedを2段階上げる
)

// moves は技の定義。
//
// 効果（まひ付与・回復・自爆など）はまだ表現していない。battle.Moveが効果を
// 持つようになるのは技の特殊挙動を実装するIssueで、それまで威力0の技は
// 何も起きない技として扱われる。
var moves = []battle.Move{
	{ID: MoveSlam, Type: battle.TypeNormal, Power: 85, Accuracy: battle.AccuracyPercent(100), MaxPP: 15},
	{ID: MoveOverdrive, Type: battle.TypeNormal, Power: 150, Accuracy: battle.AccuracyPercent(90), MaxPP: 5},
	{ID: MoveQuake, Type: battle.TypeGround, Power: 100, Accuracy: battle.AccuracyPercent(100), MaxPP: 10},
	{ID: MoveIcestorm, Type: battle.TypeIce, Power: 120, Accuracy: battle.AccuracyPercent(90), MaxPP: 5},
	{ID: MoveTide, Type: battle.TypeWater, Power: 95, Accuracy: battle.AccuracyPercent(100), MaxPP: 15},
	{ID: MoveMindblast, Type: battle.TypePsychic, Power: 90, Accuracy: battle.AccuracyPercent(100), MaxPP: 10},
	{ID: MoveSpark, Type: battle.TypeElectric, Power: 95, Accuracy: battle.AccuracyPercent(100), MaxPP: 15},
	{ID: MoveNumb, Type: battle.TypeElectric, Power: 0, Accuracy: battle.AccuracyPercent(100), MaxPP: 20},
	{ID: MoveMend, Type: battle.TypeNormal, Power: 0, Accuracy: battle.AccuracyPercent(100), MaxPP: 20},
	{ID: MoveDoze, Type: battle.TypePsychic, Power: 0, Accuracy: battle.AccuracyPercent(100), MaxPP: 10},
	{ID: MoveSlumber, Type: battle.TypeNormal, Power: 0, Accuracy: battle.AccuracyPercent(75), MaxPP: 10},
	{ID: MoveSpores, Type: battle.TypeGrass, Power: 0, Accuracy: battle.AccuracyPercent(75), MaxPP: 15},
	{ID: MoveDrain, Type: battle.TypeGrass, Power: 40, Accuracy: battle.AccuracyPercent(100), MaxPP: 10},
	{ID: MoveBurst, Type: battle.TypeNormal, Power: 170, Accuracy: battle.AccuracyPercent(100), MaxPP: 5},
	{ID: MoveNeedles, Type: battle.TypeBug, Power: 14, Accuracy: battle.AccuracyPercent(85), MaxPP: 20},
	{ID: MoveDash, Type: battle.TypePsychic, Power: 0, Accuracy: battle.AccuracyPercent(100), MaxPP: 30},
}
