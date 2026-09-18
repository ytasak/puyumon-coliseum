package battle

// restSleepTurns はRestで眠る長さ。Generation Iは2に固定する。
const restSleepTurns = 2

// multiHitCount は多段ヒットの回数を1つ引く。
//
// Generation Iは 2 → 3/8、3 → 3/8、4 → 1/8、5 → 1/8 になる。
// 乱数の下位2bitを取り、2未満ならその値を、そうでなければもう一度引き直した値を使う。
func multiHitCount(rng RNG) int {
	if first := rng.IntN(4); first < 2 {
		return first + 2
	}
	return rng.IntN(4) + 2
}

// applySideEffect はダメージのあとに起きる追加効果を適用する。
//
// 相手を倒したturnには呼ばない。実機は相手が戦闘不能になった時点で処理を終えるため、
// 追加効果も反動も起きない。
func (r *Resolver) applySideEffect(state *BattleState, side Side, move Move, defenderTyping Typing) []Event {
	defenderSide := side.Opponent()

	switch move.Effect {
	case EffectRecharge:
		state.Players[side].ActivePokemon().Recharging = true
		return nil

	case EffectParalyze, EffectFreeze:
		defender := state.Players[defenderSide].ActivePokemon()

		// 実機は条件を先に見て、通らない場合は乱数を引かない。
		// すでに状態異常なら付与しない。技のTypeと相手のTypeが一致していても付与しない
		// （こおり技はこおりタイプを凍らせず、ノーマル技はノーマルを麻痺させない）。
		if defender.Status != NoStatus || defenderTyping.Has(move.Type) {
			return nil
		}
		if !r.sideEffectHappens(move) {
			return nil
		}

		status := Paralysis
		if move.Effect == EffectFreeze {
			status = Freeze
		}
		if !ApplyStatus(r.RNG, defender, status) {
			return nil
		}
		if status == Freeze {
			// こおらせると相手の反動が解ける。
			defender.Recharging = false
		}
		return []Event{StatusApplied{Side: defenderSide, Status: status}}

	case EffectSpecialDown:
		if !r.sideEffectHappens(move) {
			return nil
		}
		defender := state.Players[defenderSide].ActivePokemon()
		delta := ChangeStage(&defender.Stages, StatSpecial, -1)
		if delta == 0 {
			return nil
		}
		return []Event{StatStageChanged{Side: defenderSide, Stat: StatSpecial, Delta: delta}}
	}
	return nil
}

// sideEffectHappens は追加効果の抽選を行う。発生率が0なら常に起こる。
func (r *Resolver) sideEffectHappens(move Move) bool {
	if move.EffectChance <= 0 {
		return true
	}
	return r.RNG.IntN(randomByteValues) < move.EffectChance
}

// applyMoveEffect は威力0の技の効果を適用する。
//
// ダメージを与えない技は、この効果そのものが技の内容になる。
func (r *Resolver) applyMoveEffect(state *BattleState, side Side, move Move, defenderTyping Typing) []Event {
	defenderSide := side.Opponent()

	switch move.Effect {
	case EffectParalyze:
		// 確定のまひ。タイプ相性で通らない相手には効かない（でんき技とじめんタイプ）。
		if AgainstTyping(move.Type, defenderTyping) == NoEffect {
			return []Event{Unaffected{Side: side}}
		}
		defender := state.Players[defenderSide].ActivePokemon()
		if !ApplyStatus(r.RNG, defender, Paralysis) {
			return []Event{MoveFailed{Side: side}}
		}
		return []Event{StatusApplied{Side: defenderSide, Status: Paralysis}}

	case EffectSleep:
		defender := state.Players[defenderSide].ActivePokemon()

		// Generation Iは、相手が反動で動けない状態のとき、すでに状態異常を持っていても
		// 上書きして眠らせる。
		if defender.Recharging {
			defender.Recharging = false
			defender.Status = NoStatus
			defender.SleepTurns = 0
		}
		if !ApplyStatus(r.RNG, defender, Sleep) {
			return []Event{MoveFailed{Side: side}}
		}
		return []Event{StatusApplied{Side: defenderSide, Status: Sleep}}

	case EffectHeal:
		user := state.Players[side].ActivePokemon()
		healed := heal(user, user.Stats.HP/2)
		if healed == 0 {
			// HPが満タンなら失敗する。
			return []Event{MoveFailed{Side: side}}
		}
		return []Event{Healed{Side: side, Amount: healed, RemainingHP: user.CurrentHP}}

	case EffectRest:
		user := state.Players[side].ActivePokemon()
		if user.CurrentHP >= user.Stats.HP {
			// HPが満タンなら失敗する。状態異常も消えない。
			return []Event{MoveFailed{Side: side}}
		}

		var events []Event
		if user.Status != NoStatus {
			events = append(events, StatusRecovered{Side: side, Status: user.Status})
		}
		user.Status = NoStatus
		user.SleepTurns = 0

		healed := heal(user, user.Stats.HP)
		user.Status = Sleep
		user.SleepTurns = restSleepTurns

		events = append(events,
			Healed{Side: side, Amount: healed, RemainingHP: user.CurrentHP},
			StatusApplied{Side: side, Status: Sleep},
		)
		return events

	case EffectSpeedUp2:
		user := state.Players[side].ActivePokemon()
		delta := ChangeStage(&user.Stages, StatSpeed, 2)
		if delta == 0 {
			// すでに上限なら何も起きない。
			return []Event{MoveFailed{Side: side}}
		}
		return []Event{StatStageChanged{Side: side, Stat: StatSpeed, Delta: delta}}
	}
	return nil
}

// applyDrain は与えたダメージの半分を使用者へ回復する。
//
// 実機では相手を倒したturnでも吸収が起きるため、戦闘不能の判定より先に呼ぶ。
func applyDrain(state *BattleState, side Side, damage int) []Event {
	if damage <= 0 {
		return nil
	}

	amount := damage / 2
	if amount < 1 {
		amount = 1
	}

	user := state.Players[side].ActivePokemon()
	healed := heal(user, amount)
	if healed == 0 {
		return nil
	}
	return []Event{Healed{Side: side, Amount: healed, RemainingHP: user.CurrentHP}}
}

// selfDestruct は使用者を戦闘不能にする。
//
// Generation Iの自爆系は命中判定より前にこれを行うため、技が外れても倒れる。
func selfDestruct(p *Pokemon) {
	p.CurrentHP = 0
	p.Status = NoStatus
	p.SleepTurns = 0
	p.Recharging = false
	p.ContinuingMove = ContinuingMove{}
}

// heal はHPを回復し、実際に回復した量を返す。最大HPは超えない。
func heal(p *Pokemon, amount int) int {
	if amount <= 0 {
		return 0
	}

	room := p.Stats.HP - p.CurrentHP
	if room <= 0 {
		return 0
	}
	if amount > room {
		amount = room
	}

	p.CurrentHP += amount
	return amount
}
