package battle

import (
	"errors"
	"fmt"
)

// ErrInvalidAction は選んだ行動がその状態では取れないことを表す。
//
// 空き枠の技、PPの切れた技、戦闘不能な控えへの交代などが該当する。
// このerrorを返すとき状態は進めない。
var ErrInvalidAction = errors.New("battle: invalid action")

// speedTieThreshold は同速のときにPlayer1が先攻になる乱数の境界。
// 実機の 50 percent + 1 にあたる。
const speedTieThreshold = 128

// Resolver は1 turnを解決する。
//
// 状態とActionとRNGから次の状態とEvent列を決めるだけで、UIを呼ばない。
// 同じ状態・同じAction・同じ乱数列からは常に同じ結果になる。
//
// turnの進行順はBattle Rules Specificationと、YTA-18に記載したturn pipelineに従う。
// 実機はダメージ計算のあとに命中判定を行うが、乱数の消費順をROMと一致させるのは
// 非目標なので、pipelineの順序（命中 → 急所 → ダメージ）で処理する。
type Resolver struct {
	// Data は技とキャラクターの定義。
	Data Data

	// RNG は命中・急所・ダメージ・同速判定・状態異常に使う乱数。
	RNG RNG
}

// ResolveTurn は両プレイヤーのActionから1 turnを解決する。
//
// 返すBattleStateは引数を複製したもので、引数のstateは変更しない。
// Actionが取れないものであればerrorを返し、状態を進めない。
//
// 交代は技より先に解決する。技同士はmove priority、effective Speed、
// 同速なら乱数で順序を決める。
//
// 手番のあとにどちらかのactiveが戦闘不能になっていれば、そこでturnを打ち切る。
// 先攻が自分の継続ダメージで倒れた場合も同じで、後攻はそのturn行動しない。
func (r *Resolver) ResolveTurn(state BattleState, p1, p2 Action) (BattleState, []Event, error) {
	original := state
	actions := [2]Action{Player1: p1, Player2: p2}

	if state.Status != Ongoing {
		return original, nil, fmt.Errorf("%w: battle already finished (%s)", ErrInvalidAction, state.Status)
	}
	for _, side := range sides {
		if state.NeedsReplacement(side) {
			return original, nil, fmt.Errorf("%w: %s must choose a replacement", ErrInvalidAction, side)
		}
		if err := r.validateAction(state, side, actions[side]); err != nil {
			return original, nil, err
		}
	}

	var events []Event

	// voluntary switchは技より先に解決する。
	for _, side := range switchOrder(state, actions) {
		target := actions[side].(SwitchAction).Target
		events = append(events, switchIn(&state, side, target))
	}

	order, err := r.moveOrder(state, actions)
	if err != nil {
		return original, nil, err
	}
	for _, side := range order {
		turnEvents, err := r.takeTurn(&state, side, actions[side].(MoveAction))
		if err != nil {
			return original, nil, err
		}
		events = append(events, turnEvents...)

		// 実機は手番のあとに戦闘不能を見つけると、その場でfaintの処理へ移る。
		// 先攻が自分の継続ダメージで倒れた場合も含めて、後続の行動は起きない。
		if activeFainted(state) {
			break
		}
	}

	state.Turn++
	state.Status = outcome(state)
	return state, events, nil
}

// ResolveReplacement は戦闘不能になったactiveを控えと入れ替える。
//
// turnは進めない。両者がreplacementを必要とする場合は片方ずつ呼ぶ。
func (r *Resolver) ResolveReplacement(state BattleState, side Side, action SwitchAction) (BattleState, []Event, error) {
	original := state

	if state.Status != Ongoing {
		return original, nil, fmt.Errorf("%w: battle already finished (%s)", ErrInvalidAction, state.Status)
	}
	if !side.valid() {
		return original, nil, fmt.Errorf("%w: unknown side %d", ErrInvalidAction, int(side))
	}
	if !state.NeedsReplacement(side) {
		return original, nil, fmt.Errorf("%w: %s does not need a replacement", ErrInvalidAction, side)
	}
	if err := action.Validate(); err != nil {
		return original, nil, err
	}

	player := state.Players[side]
	if player.Team[action.Target].Fainted() {
		return original, nil, fmt.Errorf("%w: %s team slot %d has fainted",
			ErrInvalidAction, side, action.Target)
	}
	if _, err := r.Data.LookupSpecies(player.Team[action.Target].Species); err != nil {
		return original, nil, err
	}

	return state, []Event{switchIn(&state, side, action.Target)}, nil
}

// validateAction はActionがその状態で取れるものかを確かめる。
func (r *Resolver) validateAction(state BattleState, side Side, action Action) error {
	if action == nil {
		return fmt.Errorf("%w: %s has no action", ErrInvalidAction, side)
	}
	if err := action.Validate(); err != nil {
		return err
	}

	player := state.Players[side]
	active := player.Team[player.Active]
	if _, err := r.Data.LookupSpecies(active.Species); err != nil {
		return err
	}

	switch a := action.(type) {
	case MoveAction:
		slot := active.Moves[a.Slot]
		if slot.Empty() {
			return fmt.Errorf("%w: %s move slot %d is empty", ErrInvalidAction, side, a.Slot)
		}
		if slot.PP <= 0 {
			return fmt.Errorf("%w: %s move %q has no PP left", ErrInvalidAction, side, slot.Move)
		}
		if _, err := r.Data.LookupMove(slot.Move); err != nil {
			return err
		}
	case SwitchAction:
		if a.Target == player.Active {
			return fmt.Errorf("%w: %s is already using team slot %d", ErrInvalidAction, side, a.Target)
		}
		if player.Team[a.Target].Fainted() {
			return fmt.Errorf("%w: %s team slot %d has fainted", ErrInvalidAction, side, a.Target)
		}
		if _, err := r.Data.LookupSpecies(player.Team[a.Target].Species); err != nil {
			return err
		}
	}
	return nil
}

// switchOrder は交代を解決する順序を返す。
//
// 両者が交代する場合はeffective Speedの高い方から。Generation Iの交代には
// 順序で結果が変わる処理が無いため、同速でも乱数は引かない。
func switchOrder(state BattleState, actions [2]Action) []Side {
	var switching []Side
	for _, side := range sides {
		if _, ok := actions[side].(SwitchAction); ok {
			switching = append(switching, side)
		}
	}
	if len(switching) == 2 && effectiveSpeed(state, Player2) > effectiveSpeed(state, Player1) {
		switching[0], switching[1] = switching[1], switching[0]
	}
	return switching
}

// moveOrder は技を出す側の順序を返す。
//
// move priority（大きい方が先）→ effective Speed（大きい方が先）→ 同速なら乱数。
// 乱数を引くのは本当に同速のときだけで、順序が確定していれば消費しない。
func (r *Resolver) moveOrder(state BattleState, actions [2]Action) ([]Side, error) {
	var moving []Side
	for _, side := range sides {
		if _, ok := actions[side].(MoveAction); ok {
			moving = append(moving, side)
		}
	}
	if len(moving) < 2 {
		return moving, nil
	}

	priority := [2]int{}
	for _, side := range sides {
		move, err := r.chosenMove(state, side, actions[side].(MoveAction))
		if err != nil {
			return nil, err
		}
		priority[side] = move.Priority
	}

	first, second := Player1, Player2
	switch {
	case priority[Player1] != priority[Player2]:
		if priority[Player2] > priority[Player1] {
			first, second = Player2, Player1
		}
	default:
		speed1, speed2 := effectiveSpeed(state, Player1), effectiveSpeed(state, Player2)
		switch {
		case speed1 != speed2:
			if speed2 > speed1 {
				first, second = Player2, Player1
			}
		default:
			if r.RNG.IntN(randomByteValues) >= speedTieThreshold {
				first, second = Player2, Player1
			}
		}
	}
	return []Side{first, second}, nil
}

// chosenMove はそのturnに選んだ技の定義を返す。
func (r *Resolver) chosenMove(state BattleState, side Side, action MoveAction) (Move, error) {
	player := state.Players[side]
	return r.Data.LookupMove(player.Team[player.Active].Moves[action.Slot].Move)
}

// takeTurn は片側の手番を処理する。
func (r *Resolver) takeTurn(state *BattleState, side Side, action MoveAction) ([]Event, error) {
	attacker := state.Players[side].ActivePokemon()

	// 手番が来る前に倒されていれば何も起きない。
	if attacker.Fainted() {
		return nil, nil
	}

	// 反動で動けないturnは技を出さず、PPも消費しない。
	if attacker.Recharging {
		attacker.Recharging = false
		return []Event{Recharge{Side: side}}, nil
	}

	switch block := CheckStatus(r.RNG, attacker); block {
	case NotBlocked:
	case BlockedByWakeUp:
		// 目を覚ましたturnは行動できない（Generation Iの挙動）。
		return []Event{
			StatusRecovered{Side: side, Status: Sleep},
			ActionBlocked{Side: side, Reason: block},
		}, nil
	default:
		return []Event{ActionBlocked{Side: side, Reason: block}}, nil
	}

	move, err := r.chosenMove(*state, side, action)
	if err != nil {
		return nil, err
	}
	defenderTyping, err := r.typing(*state, side.Opponent())
	if err != nil {
		return nil, err
	}

	// PPは命中判定より前に消費する。技が外れても減る。
	attacker.Moves[action.Slot].PP--
	events := []Event{MoveUsed{Side: side, Slot: action.Slot, Move: move.ID}}

	// 自爆は命中判定より前に使用者を戦闘不能にする。外れても倒れる（実機の挙動）。
	exploded := move.Effect == EffectExplode
	if exploded {
		selfDestruct(attacker)
	}

	if move.Power > 0 {
		if !r.hits(state, side, move) {
			events = append(events, MoveMissed{Side: side})
			return r.endTurn(state, side, events, exploded), nil
		}

		hit, damage, fainted, err := r.strike(state, side, move)
		if err != nil {
			return nil, err
		}
		events = append(events, hit...)

		// 吸収は相手を倒したturnでも起きる。実機もこの効果だけは最後まで処理する。
		if move.Effect == EffectDrain {
			events = append(events, applyDrain(state, side, damage)...)
		}
		if fainted {
			defenderSide := side.Opponent()
			events = append(events, Fainted{Side: defenderSide, Index: state.Players[defenderSide].Active})
		} else {
			events = append(events, r.applySideEffect(state, side, move, defenderTyping)...)
		}
	} else {
		// 威力0の技は命中判定も効果の側で行う。実機は効果ごとに判定の有無と
		// 順序が違い、反動中の相手を眠らせる場合のように判定しないものもある。
		events = append(events, r.applyMoveEffect(state, side, move, defenderTyping)...)
	}

	return r.endTurn(state, side, events, exploded), nil
}

// endTurn は手番の後始末をする。
//
// 自爆した使用者をここで戦闘不能にし、そうでなければ継続ダメージを処理する。
// 相手を倒したturnは継続ダメージが起きない。
func (r *Resolver) endTurn(state *BattleState, side Side, events []Event, exploded bool) []Event {
	if exploded {
		return append(events, Fainted{Side: side, Index: state.Players[side].Active})
	}
	if state.Players[side].ActivePokemon().Fainted() {
		return events
	}
	if state.Players[side.Opponent()].ActivePokemon().Fainted() {
		return events
	}
	return append(events, r.residual(state, side)...)
}

// hits は命中判定を行う。
func (r *Resolver) hits(state *BattleState, side Side, move Move) bool {
	attacker := state.Players[side].ActivePokemon()
	defender := state.Players[side.Opponent()].ActivePokemon()

	return Hits(r.RNG, move.Accuracy, attacker.Stages.Accuracy, defender.Stages.Evasion)
}

// typing は場に出ているPokemonのタイプ構成を返す。
func (r *Resolver) typing(state BattleState, side Side) (Typing, error) {
	player := state.Players[side]

	species, err := r.Data.LookupSpecies(player.Team[player.Active].Species)
	if err != nil {
		return Typing{}, err
	}
	return species.Typing, nil
}

// strike は命中した技のダメージを適用する。
//
// 戻り値は順に、起きたEvent、与えた合計ダメージ、相手が戦闘不能になったかどうか。
func (r *Resolver) strike(state *BattleState, side Side, move Move) ([]Event, int, bool, error) {
	defenderSide := side.Opponent()
	attacker := state.Players[side].ActivePokemon()
	defender := state.Players[defenderSide].ActivePokemon()

	attackerSpecies, err := r.Data.LookupSpecies(attacker.Species)
	if err != nil {
		return nil, 0, false, err
	}
	defenderSpecies, err := r.Data.LookupSpecies(defender.Species)
	if err != nil {
		return nil, 0, false, err
	}

	critical := IsCritical(r.RNG, attackerSpecies.BaseSpeed, move.HighCritRatio)
	damage := CalculateDamage(r.RNG, DamageInput{
		Level:          attacker.Level,
		Power:          move.Power,
		MoveType:       move.Type,
		AttackerTyping: attackerSpecies.Typing,
		DefenderTyping: defenderSpecies.Typing,
		AttackerStats:  attacker.Stats,
		DefenderStats:  defender.Stats,
		AttackerStages: attacker.Stages,
		DefenderStages: defender.Stages,
		Critical:       critical,
		HalveDefense:   move.Effect == EffectExplode,
	})

	// 相性で通らなかった場合は急所も報告しない。
	if damage == 0 {
		return []Event{Unaffected{Side: side}}, 0, false, nil
	}

	var events []Event
	if critical {
		events = append(events, CriticalHit{Side: defenderSide})
	}

	// 多段ヒットはダメージを1度だけ求め、同じ値を当たった回数ぶん与える。
	// 途中で相手が倒れたらそこで止まる（実機の挙動）。
	hits := 1
	if move.Effect == EffectMultiHit {
		hits = multiHitCount(r.RNG)
	}

	total, landed := 0, 0
	for i := 0; i < hits; i++ {
		dealt := damage
		if dealt > defender.CurrentHP {
			dealt = defender.CurrentHP
		}
		defender.CurrentHP -= dealt
		total += dealt
		landed++

		events = append(events, Damage{Side: defenderSide, Amount: dealt, RemainingHP: defender.CurrentHP})
		if defender.Fainted() {
			break
		}
	}
	if move.Effect == EffectMultiHit {
		events = append(events, MultiHit{Side: side, Hits: landed})
	}

	// 戦闘不能のEventは、吸収のように相手を倒しても起きる効果のあとで出す。
	return events, total, defender.Fainted(), nil
}

// residual は毒・やけどの継続ダメージを適用する。
func (r *Resolver) residual(state *BattleState, side Side) []Event {
	active := state.Players[side].ActivePokemon()

	damage := ApplyResidualDamage(active)
	if damage == 0 {
		return nil
	}

	events := []Event{Damage{Side: side, Amount: damage, RemainingHP: active.CurrentHP}}
	if active.Fainted() {
		events = append(events, Fainted{Side: side, Index: state.Players[side].Active})
	}
	return events
}

// switchIn は交代を適用する。
//
// Generation Iでは能力変化や反動が場の側に紐づくため、交代でstat stagesが
// 元へ戻り、反動と継続中の技も解除される。major statusと残りPPは引き継ぐ。
func switchIn(state *BattleState, side Side, target int) Event {
	player := &state.Players[side]
	from := player.Active

	for _, index := range [2]int{from, target} {
		p := &player.Team[index]
		p.Stages = StatStages{}
		p.Recharging = false
		p.ContinuingMove = ContinuingMove{}
	}
	player.Active = target

	return Switched{Side: side, From: from, To: target}
}

// effectiveSpeed は行動順の判定に使うSpeedを返す。
//
// stat stageを適用したあとに、まひなら4分の1にする（この順序）。
func effectiveSpeed(state BattleState, side Side) int {
	player := state.Players[side]
	active := player.Team[player.Active]

	speed := battleStat(active.Stats.Speed, active.Stages.Speed)
	if active.Status == Paralysis {
		speed = ParalyzedSpeed(speed)
	}
	return speed
}

// activeFainted は場に出ているPokemonのどちらかが戦闘不能かを返す。
func activeFainted(state BattleState) bool {
	for _, side := range sides {
		player := state.Players[side]
		if player.Team[player.Active].Fainted() {
			return true
		}
	}
	return false
}

// outcome は両者に戦えるPokemonが残っているかから対戦の進行状況を決める。
func outcome(state BattleState) Status {
	player1, player2 := canFight(state.Players[Player1]), canFight(state.Players[Player2])
	switch {
	case !player1 && !player2:
		return Draw
	case !player1:
		return Player2Won
	case !player2:
		return Player1Won
	default:
		return Ongoing
	}
}

// canFight は戦えるPokemonが残っているかを返す。
func canFight(player Player) bool {
	for i := range player.Team {
		if !player.Team[i].Fainted() {
			return true
		}
	}
	return false
}
