package battle

import "fmt"

// Action は1 turnにプレイヤーが選ぶ行動。
//
// 実装できるのはこのpackage内のMoveActionとSwitchActionだけで、
// 外から別の行動を足せない。resolverはtype switchで両者を漏れなく扱える。
//
// 技の名前をMoveと呼ぶ余地を残すため、行動側の型名はActionで終える。
type Action interface {
	// Validate は行動そのものが取り得る値かを確かめる。
	//
	// 「その技が今使えるか」「その控えが出せるか」のような状態に依存する判定は
	// ここでは行わない。状態と突き合わせる検証はresolverの仕事になる。
	Validate() error

	// action はこのinterfaceをpackage外から実装させないための印。
	action()
}

// MoveAction は覚えている技を使う行動。
type MoveAction struct {
	// Slot は使う技のMoves内index。
	Slot int
}

func (MoveAction) action() {}

// Validate はslotが技枠の範囲に収まっているかを確かめる。
func (a MoveAction) Validate() error {
	if a.Slot < 0 || a.Slot >= MoveSlots {
		return fmt.Errorf("%w: move slot %d is out of range [0,%d)",
			ErrInvalidState, a.Slot, MoveSlots)
	}
	return nil
}

// SwitchAction は控えのPokemonと交代する行動。
type SwitchAction struct {
	// Target は場に出すPokemonのTeam内index。
	Target int
}

func (SwitchAction) action() {}

// Validate はtargetがteamの範囲に収まっているかを確かめる。
func (a SwitchAction) Validate() error {
	if a.Target < 0 || a.Target >= TeamSize {
		return fmt.Errorf("%w: switch target %d is out of range [0,%d)",
			ErrInvalidState, a.Target, TeamSize)
	}
	return nil
}
