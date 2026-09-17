package battle

import "fmt"

// MoveSlots は1体が覚えている技の数。
const MoveSlots = 4

// Levelの範囲。本作の155形式では50〜55を使う想定だが、
// engineはその配分を仕様として持たない。配分はキャラクター側のIssueで決める。
const (
	// MinLevel は取り得る最小のLevel。
	MinLevel = 1
	// MaxLevel は取り得る最大のLevel。
	MaxLevel = 100
)

// stat stageの上下限。Generation Iの範囲。
const (
	// StageMin は能力変化stageの下限。
	StageMin = -6
	// StageMax は能力変化stageの上限。
	StageMax = 6
)

// SpeciesID はキャラクター定義を参照するための識別子。
//
// engineはこの値で分岐しない。実データは別Issueで定義し、
// domain側はあくまで「どの定義を指しているか」だけを持つ。
type SpeciesID string

// MoveID は技定義を参照するための識別子。SpeciesIDと同じく分岐には使わない。
type MoveID string

// Stats はGeneration Iの5能力の実数値。
//
// Special Attack / Special Defenseへの分割は行わない。
// HPは最大HPで、現在HPはPokemon.CurrentHPが持つ。
type Stats struct {
	HP      int
	Attack  int
	Defense int
	Speed   int
	Special int
}

// StatStages は能力変化のstage。StageMinからStageMaxまでの範囲を取る。
//
// stageから実際の倍率を求める計算は別Issueの範囲で、ここでは値だけを保持する。
type StatStages struct {
	Attack   int
	Defense  int
	Speed    int
	Special  int
	Accuracy int
	Evasion  int
}

// MajorStatus はGeneration Iのmajor status。1体につき同時に1つだけ持つ。
type MajorStatus int

const (
	// NoStatus は状態異常なし。
	NoStatus MajorStatus = iota
	// Burn はやけど。
	Burn
	// Freeze はこおり。
	Freeze
	// Paralysis はまひ。
	Paralysis
	// Poison はどく。
	Poison
	// Sleep はねむり。残りturnはPokemon.SleepTurnsが持つ。
	Sleep
)

// String はMajorStatusの名前を返す。testとlogの出力に使う。
func (s MajorStatus) String() string {
	switch s {
	case NoStatus:
		return "none"
	case Burn:
		return "burn"
	case Freeze:
		return "freeze"
	case Paralysis:
		return "paralysis"
	case Poison:
		return "poison"
	case Sleep:
		return "sleep"
	default:
		return "unknown"
	}
}

// valid はMajorStatusが定義済みの値かを返す。
func (s MajorStatus) valid() bool {
	return s >= NoStatus && s <= Sleep
}

// MoveSlot は覚えている技1つ分の状態。
//
// 技の威力・タイプ・効果はここに持たない。MoveIDで定義を参照し、
// engineが持つのは残りPPのような対戦中に変化する値だけ。
type MoveSlot struct {
	// Move は技定義への参照。空なら技を覚えていない枠。
	Move MoveID

	// PP は残りPP。
	PP int

	// MaxPP は最大PP。
	MaxPP int
}

// Empty は技を覚えていない枠かを返す。
func (m MoveSlot) Empty() bool {
	return m.Move == ""
}

// Usable は残りPPの上で選べる技かを返す。
//
// 状態異常や技自体の制約による使用可否は含まない。それらはresolverが判断する。
func (m MoveSlot) Usable() bool {
	return !m.Empty() && m.PP > 0
}

// MultiTurn は複数turnにまたがる技の進行状態。
//
// Hyper Beamの反動のように次のturnを消費するものはRechargingで表し、
// こちらは「同じ技を続けて出している」状態を表す。どの技がどう進むかは
// 技側の挙動を実装するIssueで決める。
type MultiTurn struct {
	// Move は継続している技。空なら継続中の技は無い。
	Move MoveID

	// TurnsLeft は残りturn数。
	TurnsLeft int
}

// Active は継続中の技があるかを返す。
func (m MultiTurn) Active() bool {
	return m.Move != "" && m.TurnsLeft > 0
}

// Pokemon は1体分の対戦中の状態。
type Pokemon struct {
	// Species はキャラクター定義への参照。
	Species SpeciesID

	// Level はこの個体のLevel。155形式のレベル配分はengineの外で決める。
	Level int

	// CurrentHP は現在HP。0で戦闘不能。上限はStats.HP。
	CurrentHP int

	// Stats は5能力の実数値。Stats.HPは最大HP。
	Stats Stats

	// Status はmajor status。
	Status MajorStatus

	// SleepTurns はねむりの残りturn。StatusがSleepのときだけ意味を持つ。
	SleepTurns int

	// Stages は能力変化のstage。交代でresetされる範囲は交代処理のIssueで決める。
	Stages StatStages

	// Moves は覚えている技。
	Moves [MoveSlots]MoveSlot

	// Recharging は反動で次のturnに行動できない状態か。
	Recharging bool

	// MultiTurn は複数turnにまたがる技の進行状態。
	MultiTurn MultiTurn
}

// Fainted は戦闘不能かを返す。
func (p *Pokemon) Fainted() bool {
	return p.CurrentHP <= 0
}

// validate はPokemonが取り得ない値を含んでいないかを確かめる。
func (p *Pokemon) validate() error {
	if p.Level < MinLevel || p.Level > MaxLevel {
		return fmt.Errorf("%w: level %d is out of range [%d,%d]",
			ErrInvalidState, p.Level, MinLevel, MaxLevel)
	}
	if p.Stats.HP < 1 {
		return fmt.Errorf("%w: max HP %d is less than 1", ErrInvalidState, p.Stats.HP)
	}
	for _, stat := range []struct {
		name  string
		value int
	}{
		{"attack", p.Stats.Attack},
		{"defense", p.Stats.Defense},
		{"speed", p.Stats.Speed},
		{"special", p.Stats.Special},
	} {
		if stat.value < 0 {
			return fmt.Errorf("%w: %s %d is negative", ErrInvalidState, stat.name, stat.value)
		}
	}
	if p.CurrentHP < 0 || p.CurrentHP > p.Stats.HP {
		return fmt.Errorf("%w: current HP %d is out of range [0,%d]",
			ErrInvalidState, p.CurrentHP, p.Stats.HP)
	}
	if !p.Status.valid() {
		return fmt.Errorf("%w: unknown status %d", ErrInvalidState, int(p.Status))
	}
	if p.SleepTurns < 0 {
		return fmt.Errorf("%w: sleep turns %d is negative", ErrInvalidState, p.SleepTurns)
	}
	if p.SleepTurns > 0 && p.Status != Sleep {
		return fmt.Errorf("%w: sleep turns %d with status %s",
			ErrInvalidState, p.SleepTurns, p.Status)
	}
	if err := p.Stages.validate(); err != nil {
		return err
	}
	for i, move := range p.Moves {
		if err := move.validate(); err != nil {
			return fmt.Errorf("move[%d]: %w", i, err)
		}
	}
	if p.MultiTurn.TurnsLeft < 0 {
		return fmt.Errorf("%w: multi-turn turns left %d is negative",
			ErrInvalidState, p.MultiTurn.TurnsLeft)
	}
	return nil
}

// validate はstageが上下限に収まっているかを確かめる。
func (s StatStages) validate() error {
	for _, stage := range []struct {
		name  string
		value int
	}{
		{"attack", s.Attack},
		{"defense", s.Defense},
		{"speed", s.Speed},
		{"special", s.Special},
		{"accuracy", s.Accuracy},
		{"evasion", s.Evasion},
	} {
		if stage.value < StageMin || stage.value > StageMax {
			return fmt.Errorf("%w: %s stage %d is out of range [%d,%d]",
				ErrInvalidState, stage.name, stage.value, StageMin, StageMax)
		}
	}
	return nil
}

// validate は技枠が取り得ない値を含んでいないかを確かめる。
func (m MoveSlot) validate() error {
	if m.MaxPP < 0 {
		return fmt.Errorf("%w: max PP %d is negative", ErrInvalidState, m.MaxPP)
	}
	if m.PP < 0 || m.PP > m.MaxPP {
		return fmt.Errorf("%w: PP %d is out of range [0,%d]", ErrInvalidState, m.PP, m.MaxPP)
	}
	if m.Empty() && (m.PP != 0 || m.MaxPP != 0) {
		return fmt.Errorf("%w: empty move slot has PP %d/%d", ErrInvalidState, m.PP, m.MaxPP)
	}
	return nil
}
