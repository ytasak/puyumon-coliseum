package battle

import "testing"

// effectState は控えも同じ個体で埋めた開始状態を返す。
//
// 相手が交代しても出てくる個体のタイプと状態が変わらないので、
// 効果の条件だけを切り出してtestできる。
func effectState(t *testing.T, attacker, defender Pokemon) BattleState {
	t.Helper()

	state, err := NewBattleState(
		[TeamSize]Pokemon{attacker, attacker, attacker},
		[TeamSize]Pokemon{defender, defender, defender},
	)
	if err != nil {
		t.Fatalf("NewBattleState() error = %v", err)
	}
	return state
}

// dataWithMove は1つの技だけ差し替えたゲーム定義を返す。
func dataWithMove(move Move) Data {
	data := testData()
	moves := make(map[MoveID]Move, len(data.Moves))
	for id, m := range data.Moves {
		moves[id] = m
	}
	moves[move.ID] = move
	data.Moves = moves
	return data
}

// 追加効果は発生率の境界で切り替わる。
func TestSideEffectChanceBoundary(t *testing.T) {
	t.Parallel()

	chance := EffectChancePercent(30) // 77

	tests := []struct {
		name    string
		roll    int
		applied bool
	}{
		{"below the chance", chance - 1, true},
		{"at the chance", chance, false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := effectState(t,
				fighter(speciesRunner, 130, moveShock),
				fighter(speciesTarget, 100, moveTackle),
			)
			r := testResolver(0, 255, 255, tt.roll)

			next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
			if err != nil {
				t.Fatalf("ResolveTurn() error = %v", err)
			}

			status := next.Players[Player2].ActivePokemon().Status
			if tt.applied {
				assertEvents(t, events, []Event{
					Switched{Side: Player2, From: 0, To: 1},
					MoveUsed{Side: Player1, Slot: 0, Move: moveShock},
					Damage{Side: Player2, Amount: 50, RemainingHP: 150},
					StatusApplied{Side: Player2, Status: Paralysis},
				})
				if status != Paralysis {
					t.Errorf("status = %v, want %v", status, Paralysis)
				}
			} else if status != NoStatus {
				t.Errorf("status = %v, want %v", status, NoStatus)
			}
		})
	}
}

// 技のTypeと相手のTypeが一致していると追加効果は起きない。乱数も引かない。
func TestSideEffectSkippedForMatchingType(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		move     MoveID
		defender SpeciesID
	}{
		{"normal move against a normal type", moveShock, speciesPlain},
		{"ice move against an ice type", moveChill, speciesFrost},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := effectState(t,
				fighter(speciesRunner, 130, tt.move),
				fighter(tt.defender, 100, moveTackle),
			)
			rng := &scriptRNG{values: []int{0, 255, 255, 0}}
			r := &Resolver{Data: testData(), RNG: rng}

			next, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
			if err != nil {
				t.Fatalf("ResolveTurn() error = %v", err)
			}

			if got := next.Players[Player2].ActivePokemon().Status; got != NoStatus {
				t.Errorf("status = %v, want %v", got, NoStatus)
			}
			// 命中 / 急所 / ダメージの3回だけ。発生率の抽選は行わない。
			if rng.calls != 3 {
				t.Errorf("random draws = %d, want 3 (no chance roll)", rng.calls)
			}
		})
	}
}

// すでに状態異常なら追加効果は起きない。乱数も引かない。
func TestSideEffectSkippedWhenAlreadyStatused(t *testing.T) {
	t.Parallel()

	defender := fighter(speciesTarget, 100, moveTackle)
	defender.Status = Burn

	state := effectState(t, fighter(speciesRunner, 130, moveShock), defender)
	rng := &scriptRNG{values: []int{0, 255, 255, 0}}
	r := &Resolver{Data: testData(), RNG: rng}

	next, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	if got := next.Players[Player2].ActivePokemon().Status; got != Burn {
		t.Errorf("status = %v, want %v", got, Burn)
	}
	if rng.calls != 3 {
		t.Errorf("random draws = %d, want 3 (no chance roll)", rng.calls)
	}
}

// こおりにすると相手の反動が解ける。
func TestFreezeClearsRecharge(t *testing.T) {
	t.Parallel()

	defender := fighter(speciesTarget, 100, moveTackle)
	defender.Recharging = true

	state := effectState(t, fighter(speciesRunner, 130, moveChill), defender)
	r := testResolver(0, 255, 255, EffectChancePercent(10)-1)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveChill},
		Damage{Side: Player2, Amount: 23, RemainingHP: 177}, // 特殊技なので双方のSpecialで計算し、みずへは0.5倍
		StatusApplied{Side: Player2, Status: Freeze},
		ActionBlocked{Side: Player2, Reason: BlockedByFreeze},
	})

	frozen := next.Players[Player2].ActivePokemon()
	if frozen.Status != Freeze {
		t.Errorf("status = %v, want %v", frozen.Status, Freeze)
	}
	if frozen.Recharging {
		t.Error("the target still needs to recharge, want it cleared by the freeze")
	}
}

// 相手を倒したturnは追加効果も反動も起きない。
func TestKnockoutSkipsSideEffectsAndRecharge(t *testing.T) {
	t.Parallel()

	weak := fighter(speciesTarget, 100, moveTackle)
	weak.CurrentHP = 10

	state := effectState(t, fighter(speciesRunner, 130, moveBeam), weak)
	r := testResolver(0, 255, 255)

	next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveBeam},
		Damage{Side: Player2, Amount: 10, RemainingHP: 0},
		Fainted{Side: Player2, Index: 0},
	})

	if next.Players[Player1].ActivePokemon().Recharging {
		t.Error("the user needs to recharge after a knockout, want it skipped")
	}
}

// 倒せなかった場合は反動が残り、次のturnに動けない。
func TestRechargeAfterTheMove(t *testing.T) {
	t.Parallel()

	state := effectState(t,
		fighter(speciesRunner, 130, moveBeam),
		fighter(speciesTarget, 100, moveTackle),
	)
	r := testResolver(0, 255, 255)

	next, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}
	if !next.Players[Player1].ActivePokemon().Recharging {
		t.Fatal("the user does not need to recharge, want it set")
	}

	// 次のturnは技を出せず、反動が解ける。
	after, events, err := r.ResolveTurn(next, MoveAction{Slot: 0}, SwitchAction{Target: 2})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}
	assertEvents(t, events, []Event{
		Switched{Side: Player2, From: 1, To: 2},
		Recharge{Side: Player1},
	})
	if after.Players[Player1].ActivePokemon().Recharging {
		t.Error("the user still needs to recharge, want it cleared")
	}
	assertPP(t, after, Player1, 0, 14) // 反動のturnはPPを消費しない
}

// 確定のまひ。相性で通らない相手には効かず、状態異常持ちには失敗する。
func TestGuaranteedParalyze(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		defender Pokemon
		want     []Event
		status   MajorStatus
	}{
		{
			name:     "applies to a normal target",
			defender: fighter(speciesTarget, 100, moveTackle),
			want: []Event{
				Switched{Side: Player2, From: 0, To: 1},
				MoveUsed{Side: Player1, Slot: 0, Move: moveZap},
				StatusApplied{Side: Player2, Status: Paralysis},
			},
			status: Paralysis,
		},
		{
			name:     "does not affect a ground type",
			defender: fighter(speciesEarth, 100, moveTackle),
			want: []Event{
				Switched{Side: Player2, From: 0, To: 1},
				MoveUsed{Side: Player1, Slot: 0, Move: moveZap},
				Unaffected{Side: Player1},
			},
			status: NoStatus,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := effectState(t, fighter(speciesRunner, 130, moveZap), tt.defender)
			r := testResolver(0)

			next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
			if err != nil {
				t.Fatalf("ResolveTurn() error = %v", err)
			}
			assertEvents(t, events, tt.want)

			if got := next.Players[Player2].ActivePokemon().Status; got != tt.status {
				t.Errorf("status = %v, want %v", got, tt.status)
			}
		})
	}
}

// すでに状態異常なら、確定のまひでも失敗する。
func TestGuaranteedParalyzeFailsOnStatusedTarget(t *testing.T) {
	t.Parallel()

	defender := fighter(speciesTarget, 100, moveTackle)
	defender.Status = Burn

	state := effectState(t, fighter(speciesRunner, 130, moveZap), defender)
	r := testResolver(0)

	_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		Switched{Side: Player2, From: 0, To: 1},
		MoveUsed{Side: Player1, Slot: 0, Move: moveZap},
		MoveFailed{Side: Player1},
	})
}

// ねむりは継続turnも決まる。相手が反動中なら状態異常を上書きして眠らせる。
func TestSleepMove(t *testing.T) {
	t.Parallel()

	t.Run("applies with a duration", func(t *testing.T) {
		t.Parallel()

		state := effectState(t,
			fighter(speciesRunner, 130, moveLull),
			fighter(speciesTarget, 100, moveTackle),
		)
		r := testResolver(0, 3) // 命中 / ねむりの長さ（1 + 3）

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveLull},
			StatusApplied{Side: Player2, Status: Sleep},
		})

		asleep := next.Players[Player2].ActivePokemon()
		if asleep.Status != Sleep {
			t.Errorf("status = %v, want %v", asleep.Status, Sleep)
		}
		if asleep.SleepTurns != 4 {
			t.Errorf("sleep turns = %d, want 4", asleep.SleepTurns)
		}
	})

	t.Run("overrides the status of a recharging target", func(t *testing.T) {
		t.Parallel()

		defender := fighter(speciesTarget, 100, moveTackle)
		defender.Status = Paralysis
		defender.Recharging = true

		state := effectState(t, fighter(speciesRunner, 130, moveLull), defender)
		r := testResolver(0, 2)

		next, _, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}

		target := next.Players[Player2].ActivePokemon()
		if target.Status != Sleep {
			t.Errorf("status = %v, want %v", target.Status, Sleep)
		}
		if target.Recharging {
			t.Error("the target still needs to recharge, want it cleared")
		}
	})

	t.Run("fails against a statused target", func(t *testing.T) {
		t.Parallel()

		defender := fighter(speciesTarget, 100, moveTackle)
		defender.Status = Burn

		state := effectState(t, fighter(speciesRunner, 130, moveLull), defender)
		r := testResolver(0)

		_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveLull},
			MoveFailed{Side: Player1},
		})
	})
}

// 回復は最大HPの半分。満タンなら失敗する。
func TestHealMove(t *testing.T) {
	t.Parallel()

	t.Run("heals half of the maximum HP", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveRelief)
		attacker.CurrentHP = 60

		state := effectState(t, attacker, fighter(speciesTarget, 100, moveTackle))
		r := testResolver(0)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveRelief},
			Healed{Side: Player1, Amount: 100, RemainingHP: 160},
		})
		if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 160 {
			t.Errorf("current HP = %d, want 160", got)
		}
	})

	t.Run("fails at full HP", func(t *testing.T) {
		t.Parallel()

		state := effectState(t,
			fighter(speciesRunner, 130, moveRelief),
			fighter(speciesTarget, 100, moveTackle),
		)
		r := testResolver(0)

		_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveRelief},
			MoveFailed{Side: Player1},
		})
	})
}

// Restは状態異常を消して全回復し、2turn眠る。
func TestRestMove(t *testing.T) {
	t.Parallel()

	t.Run("clears the status and sleeps for two turns", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveNap)
		attacker.CurrentHP = 60
		attacker.Status = Poison

		state := effectState(t, attacker, fighter(speciesTarget, 100, moveTackle))
		r := testResolver(0)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveNap},
			StatusRecovered{Side: Player1, Status: Poison},
			Healed{Side: Player1, Amount: 140, RemainingHP: 200},
			StatusApplied{Side: Player1, Status: Sleep},
		})

		user := next.Players[Player1].ActivePokemon()
		if user.CurrentHP != 200 {
			t.Errorf("current HP = %d, want 200", user.CurrentHP)
		}
		if user.Status != Sleep {
			t.Errorf("status = %v, want %v", user.Status, Sleep)
		}
		if user.SleepTurns != restSleepTurns {
			t.Errorf("sleep turns = %d, want %d", user.SleepTurns, restSleepTurns)
		}
	})

	t.Run("fails at full HP and keeps the status", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveNap)
		attacker.Status = Poison

		state := effectState(t, attacker, fighter(speciesTarget, 100, moveTackle))
		r := testResolver(0)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveNap},
			MoveFailed{Side: Player1},
			Damage{Side: Player1, Amount: 12, RemainingHP: 188}, // どくの継続ダメージ
		})
		if got := next.Players[Player1].ActivePokemon().Status; got != Poison {
			t.Errorf("status = %v, want %v", got, Poison)
		}
	})
}

// Speedを2段階上げる。上限なら失敗する。
func TestSpeedUpMove(t *testing.T) {
	t.Parallel()

	t.Run("raises the stage by two", func(t *testing.T) {
		t.Parallel()

		state := effectState(t,
			fighter(speciesRunner, 130, moveHaste),
			fighter(speciesTarget, 100, moveTackle),
		)
		r := testResolver(0)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveHaste},
			StatStageChanged{Side: Player1, Stat: StatSpeed, Delta: 2},
		})
		if got := next.Players[Player1].ActivePokemon().Stages.Speed; got != 2 {
			t.Errorf("speed stage = %d, want 2", got)
		}
	})

	t.Run("fails at the maximum stage", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveHaste)
		attacker.Stages.Speed = StageMax

		state := effectState(t, attacker, fighter(speciesTarget, 100, moveTackle))
		r := testResolver(0)

		_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveHaste},
			MoveFailed{Side: Player1},
		})
	})
}

// 自爆は相手の防御を半分にして計算し、使用者は倒れる。
func TestExplodeMove(t *testing.T) {
	t.Parallel()

	t.Run("hits with halved defense and faints the user", func(t *testing.T) {
		t.Parallel()

		state := effectState(t,
			fighter(speciesRunner, 130, moveBoom),
			fighter(speciesTarget, 100, moveTackle),
		)
		r := testResolver(0, 255, 255)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}

		// 防御150を半分の75として計算するので、通常より大きく入る。
		assertEvents(t, events, []Event{
			MoveUsed{Side: Player1, Slot: 0, Move: moveBoom},
			Damage{Side: Player2, Amount: 197, RemainingHP: 3},
			Fainted{Side: Player1, Index: 0},
		})
		if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 0 {
			t.Errorf("user HP = %d, want 0", got)
		}
		if !next.NeedsReplacement(Player1) {
			t.Error("NeedsReplacement(player1) = false, want true")
		}
	})

	t.Run("faints the user even when it misses", func(t *testing.T) {
		t.Parallel()

		state := effectState(t,
			fighter(speciesRunner, 130, moveBoom),
			fighter(speciesTarget, 100, moveTackle),
		)
		// 外れる命中率の自爆技に差し替える。
		inaccurate := dataWithMove(Move{
			ID: moveBoom, Type: TypeNormal, Power: 170, Accuracy: AccuracyPercent(75), MaxPP: 5,
			Effect: EffectExplode,
		})
		r := &Resolver{Data: inaccurate, RNG: &scriptRNG{values: []int{255}}}

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			MoveUsed{Side: Player1, Slot: 0, Move: moveBoom},
			MoveMissed{Side: Player1},
			Fainted{Side: Player1, Index: 0},
		})
		if got := next.Players[Player2].ActivePokemon().CurrentHP; got != 200 {
			t.Errorf("target HP = %d, want 200", got)
		}
	})

	t.Run("can end the battle in a draw", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveBoom)
		defender := fighter(speciesTarget, 100, moveTackle)
		defender.CurrentHP = 10

		fainted := fighter(speciesTarget, 100, moveTackle)
		fainted.CurrentHP = 0

		state, err := NewBattleState(
			[TeamSize]Pokemon{attacker, fainted, fainted},
			[TeamSize]Pokemon{defender, fainted, fainted},
		)
		if err != nil {
			t.Fatalf("NewBattleState() error = %v", err)
		}

		r := testResolver(0, 255, 255)
		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}

		assertEvents(t, events, []Event{
			MoveUsed{Side: Player1, Slot: 0, Move: moveBoom},
			Damage{Side: Player2, Amount: 10, RemainingHP: 0},
			Fainted{Side: Player2, Index: 0},
			Fainted{Side: Player1, Index: 0},
		})
		if next.Status != Draw {
			t.Errorf("Status = %v, want %v", next.Status, Draw)
		}
	})
}

// 多段ヒットは2〜5回。回数の抽選はGeneration Iの2段構えに従う。
func TestMultiHitMove(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		rolls []int
		hits  int
	}{
		{"two hits", []int{0}, 2},
		{"three hits", []int{1}, 3},
		{"four hits after the second roll", []int{2, 2}, 4},
		{"five hits after the second roll", []int{3, 3}, 5},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			state := effectState(t,
				fighter(speciesRunner, 130, moveJab),
				fighter(speciesTarget, 100, moveTackle),
			)
			r := testResolver(append([]int{0, 255, 255}, tt.rolls...)...)

			next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
			if err != nil {
				t.Fatalf("ResolveTurn() error = %v", err)
			}

			want := []Event{
				Switched{Side: Player2, From: 0, To: 1},
				MoveUsed{Side: Player1, Slot: 0, Move: moveJab},
			}
			hp := 200
			for i := 0; i < tt.hits; i++ {
				hp -= 10
				want = append(want, Damage{Side: Player2, Amount: 10, RemainingHP: hp})
			}
			want = append(want, MultiHit{Side: Player1, Hits: tt.hits})
			assertEvents(t, events, want)

			if got := next.Players[Player2].ActivePokemon().CurrentHP; got != hp {
				t.Errorf("target HP = %d, want %d", got, hp)
			}
		})
	}
}

// 相手が倒れたら、残りの回数は当たらない。
func TestMultiHitStopsWhenTheTargetFaints(t *testing.T) {
	t.Parallel()

	weak := fighter(speciesTarget, 100, moveTackle)
	weak.CurrentHP = 15

	state := effectState(t, fighter(speciesRunner, 130, moveJab), weak)
	r := testResolver(0, 255, 255, 3, 3) // 本来は5回

	_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("ResolveTurn() error = %v", err)
	}

	assertEvents(t, events, []Event{
		MoveUsed{Side: Player1, Slot: 0, Move: moveJab},
		Damage{Side: Player2, Amount: 10, RemainingHP: 5},
		Damage{Side: Player2, Amount: 5, RemainingHP: 0},
		MultiHit{Side: Player1, Hits: 2},
		Fainted{Side: Player2, Index: 0},
	})
}

// 吸収は与えたダメージの半分を回復する。相手を倒しても回復する。
func TestDrainMove(t *testing.T) {
	t.Parallel()

	t.Run("heals half of the damage", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveSap)
		attacker.CurrentHP = 100

		state := effectState(t, attacker, fighter(speciesTarget, 100, moveTackle))
		r := testResolver(0, 255, 255)

		next, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, SwitchAction{Target: 1})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			Switched{Side: Player2, From: 0, To: 1},
			MoveUsed{Side: Player1, Slot: 0, Move: moveSap},
			Damage{Side: Player2, Amount: 42, RemainingHP: 158},
			Healed{Side: Player1, Amount: 21, RemainingHP: 121},
		})
		if got := next.Players[Player1].ActivePokemon().CurrentHP; got != 121 {
			t.Errorf("user HP = %d, want 121", got)
		}
	})

	t.Run("heals even when it knocks the target out", func(t *testing.T) {
		t.Parallel()

		attacker := fighter(speciesRunner, 130, moveSap)
		attacker.CurrentHP = 100
		weak := fighter(speciesTarget, 100, moveTackle)
		weak.CurrentHP = 10

		state := effectState(t, attacker, weak)
		r := testResolver(0, 255, 255)

		_, events, err := r.ResolveTurn(state, MoveAction{Slot: 0}, MoveAction{Slot: 0})
		if err != nil {
			t.Fatalf("ResolveTurn() error = %v", err)
		}
		assertEvents(t, events, []Event{
			MoveUsed{Side: Player1, Slot: 0, Move: moveSap},
			Damage{Side: Player2, Amount: 10, RemainingHP: 0},
			Healed{Side: Player1, Amount: 5, RemainingHP: 105},
			Fainted{Side: Player2, Index: 0},
		})
	})
}
