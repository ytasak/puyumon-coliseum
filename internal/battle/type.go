package battle

// Type はGeneration Iに存在する15タイプ。
//
// Dark / Steel / Fairyは後の世代で追加されたものなので持たない。
type Type int

const (
	// NoType はタイプが無いことを表す。単タイプのSecondaryに使う。
	NoType Type = iota
	// TypeNormal はノーマル。
	TypeNormal
	// TypeFighting はかくとう。
	TypeFighting
	// TypeFlying はひこう。
	TypeFlying
	// TypePoison はどく。
	TypePoison
	// TypeGround はじめん。
	TypeGround
	// TypeRock はいわ。
	TypeRock
	// TypeBug はむし。
	TypeBug
	// TypeGhost はゴースト。
	TypeGhost
	// TypeFire はほのお。
	TypeFire
	// TypeWater はみず。
	TypeWater
	// TypeGrass はくさ。
	TypeGrass
	// TypeElectric はでんき。
	TypeElectric
	// TypePsychic はエスパー。
	TypePsychic
	// TypeIce はこおり。
	TypeIce
	// TypeDragon はドラゴン。
	TypeDragon
)

// typeCount は参照表を引くための要素数。NoTypeを含む。
const typeCount = int(TypeDragon) + 1

// AllTypes はGeneration Iの15タイプを定義順に返す。
//
// 表の網羅性をtestで確かめるために使う。NoTypeは含まない。
func AllTypes() []Type {
	all := make([]Type, 0, typeCount-1)
	for t := TypeNormal; t <= TypeDragon; t++ {
		all = append(all, t)
	}
	return all
}

// String はTypeの名前を返す。testとlogの出力に使う。
func (t Type) String() string {
	switch t {
	case NoType:
		return "none"
	case TypeNormal:
		return "normal"
	case TypeFighting:
		return "fighting"
	case TypeFlying:
		return "flying"
	case TypePoison:
		return "poison"
	case TypeGround:
		return "ground"
	case TypeRock:
		return "rock"
	case TypeBug:
		return "bug"
	case TypeGhost:
		return "ghost"
	case TypeFire:
		return "fire"
	case TypeWater:
		return "water"
	case TypeGrass:
		return "grass"
	case TypeElectric:
		return "electric"
	case TypePsychic:
		return "psychic"
	case TypeIce:
		return "ice"
	case TypeDragon:
		return "dragon"
	default:
		return "unknown"
	}
}

// Category はダメージ計算にどの能力値を使うかの分類。
//
// Generation Iでは技ごとではなくタイプで決まる。物理ならAttack/Defense、
// 特殊ならSpecial同士を使う。
type Category int

const (
	// Physical は物理。Attackと Defenseを使う。
	Physical Category = iota
	// Special は特殊。攻守ともSpecialを使う。
	Special
)

// String はCategoryの名前を返す。
func (c Category) String() string {
	switch c {
	case Physical:
		return "physical"
	case Special:
		return "special"
	default:
		return "unknown"
	}
}

// Category はそのタイプの技が物理か特殊かを返す。
//
// 分類はBattle Rules Specificationのとおり。NoTypeはPhysicalを返すが、
// タイプの無い技は存在しないため意味を持たない。
func (t Type) Category() Category {
	switch t {
	case TypeFire, TypeWater, TypeGrass, TypeElectric, TypePsychic, TypeIce, TypeDragon:
		return Special
	default:
		return Physical
	}
}

// Typing はPokemonのタイプ構成。
//
// Generation Iでは1体につき1つか2つのタイプを持つ。単タイプならSecondaryはNoType。
type Typing struct {
	// Primary は第1タイプ。
	Primary Type
	// Secondary は第2タイプ。単タイプならNoType。
	Secondary Type
}

// SingleType は単タイプのTypingを返す。
func SingleType(t Type) Typing {
	return Typing{Primary: t, Secondary: NoType}
}

// DualType は複合タイプのTypingを返す。
func DualType(primary, secondary Type) Typing {
	return Typing{Primary: primary, Secondary: secondary}
}

// List はタイプを第1・第2の順で返す。NoTypeは含まない。
func (t Typing) List() []Type {
	types := make([]Type, 0, 2)
	if t.Primary != NoType {
		types = append(types, t.Primary)
	}
	if t.Secondary != NoType && t.Secondary != t.Primary {
		types = append(types, t.Secondary)
	}
	return types
}

// Has は指定のタイプを持っているかを返す。
func (t Typing) Has(target Type) bool {
	if target == NoType {
		return false
	}
	return t.Primary == target || t.Secondary == target
}

// HasSTAB は技のタイプが使用者自身のタイプと一致するか（STABが乗るか）を返す。
//
// 一致したときにダメージへ1.5倍相当を掛けるのはGeneration Iのダメージ計算の側で、
// 整数演算の順序を含めてdamage formulaのIssueで扱う。
func HasSTAB(move Type, user Typing) bool {
	return user.Has(move)
}
