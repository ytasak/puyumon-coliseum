package roster

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// キャラクターのinternal ID。公開表示名とは分けて持つ。
const (
	SpeciesBull  battle.SpeciesID = "bull"  // 🐂 高速物理
	SpeciesStar  battle.SpeciesID = "star"  // ⭐ 高速特殊と自己回復
	SpeciesJolt  battle.SpeciesID = "jolt"  // ⚡ 最速。まひ撒き
	SpeciesPalm  battle.SpeciesID = "palm"  // 🌴 耐久と催眠と自爆
	SpeciesWhale battle.SpeciesID = "whale" // 🐋 耐久
	SpeciesCharm battle.SpeciesID = "charm" // 💋 眠らせ役
)

// Character は1キャラクターの定義。
//
// Battle Engineが使うのはTyping・BaseStats・Movesで、残りは配布や表示のための情報。
type Character struct {
	// ID はinternal ID。dataとmechanicsはこの値で参照する。
	ID battle.SpeciesID

	// DisplayName は公開表示名。まだ決めていないので空のままにする。
	DisplayName string

	// Emoji は一覧やlogでキャラクターを見分けるための代表Emoji。
	// 実際の見た目はComposite Sprite定義で、同じinternal IDから引く。
	Emoji string

	// Typing はタイプ構成。
	Typing battle.Typing

	// BaseStats は種族値。実数値はLevelが決まってから算出する。
	BaseStats battle.Stats

	// Moves は固定の4技。
	Moves [battle.MoveSlots]battle.MoveID

	// Rank は強さ順位。1が最強で、Levelの割り当てに使う。
	Rank int

	// Role は役割の説明。バランス検証で参照する。
	Role string
}

// characters は6キャラクターの定義。Rankの昇順に並べる。
var characters = []Character{
	{
		ID:        SpeciesBull,
		Emoji:     "🐂",
		Typing:    battle.SingleType(battle.TypeNormal),
		BaseStats: battle.Stats{HP: 75, Attack: 100, Defense: 95, Speed: 110, Special: 70},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveSlam, MoveOverdrive, MoveQuake, MoveIcestorm},
		Rank:      1,
		Role:      "高速物理。base Speedが高く急所率も高い",
	},
	{
		ID:        SpeciesStar,
		Emoji:     "⭐",
		Typing:    battle.DualType(battle.TypeWater, battle.TypePsychic),
		BaseStats: battle.Stats{HP: 60, Attack: 75, Defense: 85, Speed: 115, Special: 100},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveTide, MoveMindblast, MoveSpark, MoveMend},
		Rank:      2,
		Role:      "高速特殊と自己回復",
	},
	{
		ID:        SpeciesJolt,
		Emoji:     "⚡",
		Typing:    battle.SingleType(battle.TypeElectric),
		BaseStats: battle.Stats{HP: 65, Attack: 65, Defense: 60, Speed: 130, Special: 110},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveSpark, MoveNumb, MoveNeedles, MoveDash},
		Rank:      3,
		Role:      "最速。まひ撒き",
	},
	{
		ID:        SpeciesPalm,
		Emoji:     "🌴",
		Typing:    battle.DualType(battle.TypeGrass, battle.TypePsychic),
		BaseStats: battle.Stats{HP: 95, Attack: 95, Defense: 85, Speed: 55, Special: 125},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveSpores, MoveMindblast, MoveDrain, MoveBurst},
		Rank:      4,
		Role:      "耐久と催眠と自爆",
	},
	{
		ID:        SpeciesWhale,
		Emoji:     "🐋",
		Typing:    battle.DualType(battle.TypeWater, battle.TypeIce),
		BaseStats: battle.Stats{HP: 130, Attack: 85, Defense: 80, Speed: 60, Special: 95},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveTide, MoveIcestorm, MoveSlam, MoveDoze},
		Rank:      5,
		Role:      "耐久。氷技の起点",
	},
	{
		ID:        SpeciesCharm,
		Emoji:     "💋",
		Typing:    battle.DualType(battle.TypeIce, battle.TypePsychic),
		BaseStats: battle.Stats{HP: 65, Attack: 50, Defense: 35, Speed: 95, Special: 95},
		Moves:     [battle.MoveSlots]battle.MoveID{MoveSlumber, MoveIcestorm, MoveMindblast, MoveDoze},
		Rank:      6,
		Role:      "眠らせ役。紙耐久",
	},
}

// All は6キャラクターの定義をRankの昇順で返す。
func All() []Character {
	all := make([]Character, len(characters))
	copy(all, characters)
	return all
}

// Lookup はinternal IDからキャラクターの定義を返す。
func Lookup(id battle.SpeciesID) (Character, bool) {
	for _, c := range characters {
		if c.ID == id {
			return c, true
		}
	}
	return Character{}, false
}

// Data はBattle Engineが参照するゲーム定義を返す。
func Data() battle.Data {
	data := battle.Data{
		Moves:   make(map[battle.MoveID]battle.Move, len(moves)),
		Species: make(map[battle.SpeciesID]battle.Species, len(characters)),
	}
	for _, move := range moves {
		data.Moves[move.ID] = move
	}
	for _, c := range characters {
		data.Species[c.ID] = battle.Species{
			ID:        c.ID,
			Typing:    c.Typing,
			BaseSpeed: c.BaseStats.Speed,
		}
	}
	return data
}
