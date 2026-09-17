package battle

// Event はturnの解決結果を、UI・animation・replayが解釈できる形で表したもの。
//
// 描画に関する型（座標・色・Emoji・Ebitengineの型）を持たない。
// 見せ方はEventを受け取る側が決める。
//
// 実装できるのはこのpackage内の型だけで、外から別のEventを足せない。
type Event interface {
	// event はこのinterfaceをpackage外から実装させないための印。
	event()
}

// MoveUsed は技を使ったこと。命中したかどうかは含まない。
type MoveUsed struct {
	// Side は技を使った側。
	Side Side
	// Slot は使った技のMoves内index。
	Slot int
	// Move は使った技の識別子。
	Move MoveID
}

func (MoveUsed) event() {}

// Switched は場に出ているPokemonが入れ替わったこと。
type Switched struct {
	// Side は交代した側。
	Side Side
	// From は下がったPokemonのTeam内index。
	From int
	// To は場に出たPokemonのTeam内index。
	To int
}

func (Switched) event() {}

// CriticalHit は急所に当たったこと。直後のDamageに対応する。
type CriticalHit struct {
	// Side は急所を受けた側。
	Side Side
}

func (CriticalHit) event() {}

// Damage はHPが減ったこと。技によるものか継続ダメージかは区別しない。
type Damage struct {
	// Side はダメージを受けた側。
	Side Side
	// Amount は減ったHP。
	Amount int
	// RemainingHP は減ったあとの現在HP。HPバーの表示に使う。
	RemainingHP int
}

func (Damage) event() {}

// StatusApplied はmajor statusになったこと。
type StatusApplied struct {
	// Side は状態異常になった側。
	Side Side
	// Status になった状態異常。
	Status MajorStatus
}

func (StatusApplied) event() {}

// StatusRecovered はmajor statusが解けたこと。
type StatusRecovered struct {
	// Side は状態異常が解けた側。
	Side Side
	// Status は解けた状態異常。
	Status MajorStatus
}

func (StatusRecovered) event() {}

// Fainted は戦闘不能になったこと。
type Fainted struct {
	// Side は戦闘不能になった側。
	Side Side
	// Index は戦闘不能になったPokemonのTeam内index。
	Index int
}

func (Fainted) event() {}

// Recharge は反動でそのturnに行動できなかったこと。
type Recharge struct {
	// Side は行動できなかった側。
	Side Side
}

func (Recharge) event() {}
