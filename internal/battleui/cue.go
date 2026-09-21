package battleui

import "github.com/ytasak/puyumon-coliseum/internal/battle"

// Cue は画面が順に再生する1つ分の出来事。
//
// Battle Eventをそのまま渡すのではなく、「誰の出来事か」「HPがいくつになるか」を
// この層で解決してから渡す。UIがmechanicsを読み解かなくてよいようにするのが目的。
//
// 実装できるのはこのpackage内の型だけで、外から別のcueを足せない。
type Cue interface {
	// cue はこのinterfaceをpackage外から実装させないための印。
	cue()
}

// Ref はcueが指す1体。
//
// EventはSideしか持たないことが多いので、Event列を辿って解決したteam indexと
// SpeciesIDをここに載せる。indexは配布順で、交代しても変わらない。
type Ref struct {
	Side    battle.Side
	Index   int
	Species battle.SpeciesID
}

// MoveUsedCue は技を出したこと。命中したかは含まない。
//
// StruggleではSlotがbattle.NoMoveSlotになる。
type MoveUsedCue struct {
	Actor Ref
	Move  battle.MoveID
	Slot  int
}

// DamageCue はHPが減ったこと。
//
// HPはこの時点の値で、turnが終わったあとの値ではない。cueを順に再生すれば
// HPバーが途中経過をたどる。
type DamageCue struct {
	Target Ref
	Amount int
	HP     int
	MaxHP  int

	// Critical は急所に当たったか。直前のCriticalHitをここへ畳んでいる。
	Critical bool
}

// HealCue はHPが回復したこと。HPの意味はDamageCueと同じ。
type HealCue struct {
	Target Ref
	Amount int
	HP     int
	MaxHP  int
}

// StatusCue はmajor statusが付いた、または解けたこと。
type StatusCue struct {
	Target Ref
	Status battle.MajorStatus

	// Applied はついたのか解けたのか。falseなら解けた。
	Applied bool
}

// StatStageCue は能力変化のstageが動いたこと。
type StatStageCue struct {
	Target Ref
	Stat   battle.Stat
	Delta  int
}

// MultiHitCue は1回の使用で複数回当たったこと。内訳は直前のDamageCue。
type MultiHitCue struct {
	Actor Ref
	Hits  int
}

// MissCue は技が外れたこと。
type MissCue struct {
	Actor  Ref
	Target Ref
}

// UnaffectedCue はタイプ相性で技が通らなかったこと。
//
// 外れたのとは区別する。Eventが持つのは使用者のSideだけなので、
// 「効果がない」側であるTargetもここで解決して渡す。
type UnaffectedCue struct {
	Actor  Ref
	Target Ref
}

// FailedCue は技は当たったが効果が起きなかったこと。
type FailedCue struct {
	Actor Ref
}

// BlockedCue は行動できなかったこと。
type BlockedCue struct {
	Target Ref
	Reason BlockReason
}

// SwitchOutCue は場から下がったこと。
type SwitchOutCue struct {
	Target Ref
}

// SwitchInCue は場に出たこと。
type SwitchInCue struct {
	Target Ref
}

// FaintCue は戦闘不能になったこと。
//
// **このcueは対象をHP 0・ひんし表示へ移す意味を持つ。** 自爆や、自爆が外れた
// 場合のように、直前にDamageCueが無いまま倒れることがある。UIは技の効果から
// 推測せず、このcueだけでHPを0にしてひんしへ切り替えてよい。
type FaintCue struct {
	Target Ref
}

func (MoveUsedCue) cue()   {}
func (DamageCue) cue()     {}
func (HealCue) cue()       {}
func (StatusCue) cue()     {}
func (StatStageCue) cue()  {}
func (MultiHitCue) cue()   {}
func (MissCue) cue()       {}
func (UnaffectedCue) cue() {}
func (FailedCue) cue()     {}
func (BlockedCue) cue()    {}
func (SwitchOutCue) cue()  {}
func (SwitchInCue) cue()   {}
func (FaintCue) cue()      {}

// BlockReason は行動できなかった理由。
//
// 状態異常による行動不能と、反動による行動不能をUIから見て同じ扱いにできるよう、
// battle.StatusBlockとRechargeをここで1つにまとめている。
type BlockReason int

const (
	// BlockUnknown は理由を判別できない場合。
	BlockUnknown BlockReason = iota
	BlockSleep
	BlockWakeUp
	BlockFreeze
	BlockParalysis
	// BlockRecharge は反動で動けないこと。
	BlockRecharge
)

// String はBlockReasonの名前を返す。testとlogの出力に使う。
func (r BlockReason) String() string {
	switch r {
	case BlockSleep:
		return "sleep"
	case BlockWakeUp:
		return "wake up"
	case BlockFreeze:
		return "freeze"
	case BlockParalysis:
		return "paralysis"
	case BlockRecharge:
		return "recharge"
	default:
		return "unknown"
	}
}

// Cues はcommandが返したEvent列を、順番を保ったままcueへ変換する。
//
// beforeはそのEvent列を適用する前のBattleState。Event列の途中でactiveが
// 入れ替わるため、誰の出来事かを解決するには開始点が要る。
//
// 決着そのもののcueは返さない。outcomeにEventが無く、afterの状態も受け取らない
// ので、この関数からは結果を発行できない。結果はSnapshotのResultで見る。
func Cues(before battle.BattleState, events []battle.Event) []Cue {
	resolver := newCueResolver(before)
	cues := make([]Cue, 0, len(events))

	for i := 0; i < len(events); i++ {
		switch event := events[i].(type) {
		case battle.CriticalHit:
			// 実機と同じく直後のDamageに対応する。隣接する1件だけへ畳み、
			// 反動や継続ダメージへ急所が漏れないようにする。
			// 直後がそのDamageでなければ、cueを出さずに捨てる。
			if damage, ok := nextDamage(events, i, event.Side); ok {
				cues = append(cues, resolver.damage(damage, true))
				i++
			}

		case battle.Damage:
			cues = append(cues, resolver.damage(event, false))

		case battle.Healed:
			target := resolver.activeRef(event.Side)
			cues = append(cues, HealCue{
				Target: target,
				Amount: event.Amount,
				HP:     event.RemainingHP,
				MaxHP:  resolver.maxHP(target),
			})

		case battle.MoveUsed:
			cues = append(cues, MoveUsedCue{
				Actor: resolver.activeRef(event.Side),
				Move:  event.Move,
				Slot:  event.Slot,
			})

		case battle.Switched:
			cues = append(cues,
				SwitchOutCue{Target: resolver.ref(event.Side, event.From)},
				SwitchInCue{Target: resolver.ref(event.Side, event.To)},
			)
			resolver.setActive(event.Side, event.To)

		case battle.StatusApplied:
			cues = append(cues, StatusCue{Target: resolver.activeRef(event.Side), Status: event.Status, Applied: true})

		case battle.StatusRecovered:
			cues = append(cues, StatusCue{Target: resolver.activeRef(event.Side), Status: event.Status})

		case battle.StatStageChanged:
			cues = append(cues, StatStageCue{Target: resolver.activeRef(event.Side), Stat: event.Stat, Delta: event.Delta})

		case battle.MultiHit:
			cues = append(cues, MultiHitCue{Actor: resolver.activeRef(event.Side), Hits: event.Hits})

		case battle.MoveMissed:
			cues = append(cues, MissCue{
				Actor:  resolver.activeRef(event.Side),
				Target: resolver.activeRef(event.Side.Opponent()),
			})

		case battle.Unaffected:
			cues = append(cues, UnaffectedCue{
				Actor:  resolver.activeRef(event.Side),
				Target: resolver.activeRef(event.Side.Opponent()),
			})

		case battle.MoveFailed:
			cues = append(cues, FailedCue{Actor: resolver.activeRef(event.Side)})

		case battle.ActionBlocked:
			cues = append(cues, BlockedCue{Target: resolver.activeRef(event.Side), Reason: blockReason(event.Reason)})

		case battle.Recharge:
			cues = append(cues, BlockedCue{Target: resolver.activeRef(event.Side), Reason: BlockRecharge})

		case battle.Fainted:
			// 倒れた個体はEventのindexで指す。交代前はactiveのままだが、
			// 自爆のように使用者が倒れる場合もあるため、Event側を信じる。
			cues = append(cues, FaintCue{Target: resolver.ref(event.Side, event.Index)})
		}
	}
	return cues
}

// nextDamage はCriticalHitの直後が同じ側のDamageかを見る。
func nextDamage(events []battle.Event, i int, side battle.Side) (battle.Damage, bool) {
	if i+1 >= len(events) {
		return battle.Damage{}, false
	}
	damage, ok := events[i+1].(battle.Damage)
	if !ok || damage.Side != side {
		return battle.Damage{}, false
	}
	return damage, true
}

// blockReason はengineの理由を表示用の理由へ移す。
func blockReason(reason battle.StatusBlock) BlockReason {
	switch reason {
	case battle.BlockedBySleep:
		return BlockSleep
	case battle.BlockedByWakeUp:
		return BlockWakeUp
	case battle.BlockedByFreeze:
		return BlockFreeze
	case battle.BlockedByParalysis:
		return BlockParalysis
	default:
		return BlockUnknown
	}
}

// cueResolver はEvent列を辿りながら「誰の出来事か」を解決する。
type cueResolver struct {
	active [2]int
	teams  [2][battle.TeamSize]battle.Pokemon
}

func newCueResolver(before battle.BattleState) *cueResolver {
	r := &cueResolver{}
	for _, side := range [...]battle.Side{battle.Player1, battle.Player2} {
		r.active[side] = before.Players[side].Active
		r.teams[side] = before.Players[side].Team
	}
	return r
}

// ref はteam indexから1体を指す参照を作る。
func (r *cueResolver) ref(side battle.Side, index int) Ref {
	var species battle.SpeciesID
	if index >= 0 && index < battle.TeamSize {
		species = r.teams[side][index].Species
	}
	return Ref{Side: side, Index: index, Species: species}
}

// activeRef はその時点で場に出ている1体を指す。
func (r *cueResolver) activeRef(side battle.Side) Ref {
	return r.ref(side, r.active[side])
}

// setActive は交代を反映する。
func (r *cueResolver) setActive(side battle.Side, index int) {
	r.active[side] = index
}

// maxHP は最大HPを返す。対戦中に変わらないのでbefore時点の値でよい。
func (r *cueResolver) maxHP(ref Ref) int {
	if ref.Index < 0 || ref.Index >= battle.TeamSize {
		return 0
	}
	return r.teams[ref.Side][ref.Index].Stats.HP
}

// damage はDamage Eventからcueを作る。
func (r *cueResolver) damage(event battle.Damage, critical bool) DamageCue {
	target := r.activeRef(event.Side)
	return DamageCue{
		Target:   target,
		Amount:   event.Amount,
		HP:       event.RemainingHP,
		MaxHP:    r.maxHP(target),
		Critical: critical,
	}
}
