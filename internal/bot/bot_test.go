package bot

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// test用の技とキャラクター。実データは internal/roster が持つ。
// ここでは相性の組み合わせを作りやすい最小限だけを置く。
const (
	moveTackle = battle.MoveID("tackle") // ノーマル 85 必中
	moveSpark  = battle.MoveID("spark")  // でんき 95 必中
	moveVine   = battle.MoveID("vine")   // くさ 95 必中
	moveMend   = battle.MoveID("mend")   // 威力0 回復
	moveLull   = battle.MoveID("lull")   // ノーマル 威力0 ねむり
	moveBoom   = battle.MoveID("boom")   // ノーマル 170 自爆
	moveHaste  = battle.MoveID("haste")  // エスパー 威力0 Speed+2
)

const (
	speciesPlain = battle.SpeciesID("plain") // ノーマル
	speciesWave  = battle.SpeciesID("wave")  // みず
	speciesVolt  = battle.SpeciesID("volt")  // でんき
	speciesDune  = battle.SpeciesID("dune")  // じめん
	speciesLeaf  = battle.SpeciesID("leaf")  // くさ
)

func testData() battle.Data {
	return battle.Data{
		Moves: map[battle.MoveID]battle.Move{
			moveTackle: {ID: moveTackle, Type: battle.TypeNormal, Power: 85, Accuracy: battle.MaxAccuracy, MaxPP: 15},
			moveSpark:  {ID: moveSpark, Type: battle.TypeElectric, Power: 95, Accuracy: battle.MaxAccuracy, MaxPP: 15},
			moveVine:   {ID: moveVine, Type: battle.TypeGrass, Power: 95, Accuracy: battle.MaxAccuracy, MaxPP: 15},
			moveMend:   {ID: moveMend, Type: battle.TypeNormal, Power: 0, Accuracy: battle.MaxAccuracy, MaxPP: 20, Effect: battle.EffectHeal},
			moveLull:   {ID: moveLull, Type: battle.TypeNormal, Power: 0, Accuracy: battle.MaxAccuracy, MaxPP: 10, Effect: battle.EffectSleep},
			moveBoom:   {ID: moveBoom, Type: battle.TypeNormal, Power: 170, Accuracy: battle.MaxAccuracy, MaxPP: 5, Effect: battle.EffectExplode},
			moveHaste:  {ID: moveHaste, Type: battle.TypePsychic, Power: 0, Accuracy: battle.MaxAccuracy, MaxPP: 30, Effect: battle.EffectSpeedUp2},
		},
		Species: map[battle.SpeciesID]battle.Species{
			speciesPlain: {ID: speciesPlain, Typing: battle.SingleType(battle.TypeNormal), BaseSpeed: 100},
			speciesWave:  {ID: speciesWave, Typing: battle.SingleType(battle.TypeWater), BaseSpeed: 100},
			speciesVolt:  {ID: speciesVolt, Typing: battle.SingleType(battle.TypeElectric), BaseSpeed: 120},
			speciesDune:  {ID: speciesDune, Typing: battle.SingleType(battle.TypeGround), BaseSpeed: 90},
			speciesLeaf:  {ID: speciesLeaf, Typing: battle.SingleType(battle.TypeGrass), BaseSpeed: 95},
		},
	}
}

// 同じseedからは同じleadを選ぶ。
func TestLeadIsDeterministic(t *testing.T) {
	t.Parallel()

	// 3体とも同じなので必ず同点になり、乱数で割ることになる。
	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
	}
	opponent := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesWave, moveTackle),
		fighter(t, speciesWave, moveTackle),
		fighter(t, speciesWave, moveTackle),
	}

	picked := map[int]bool{}
	for seed := uint64(0); seed < 30; seed++ {
		first := New(testData(), seed).Lead(own, opponent)
		second := New(testData(), seed).Lead(own, opponent)
		if first != second {
			t.Fatalf("seed %d: 同じseedで違うleadを選んだ（%d と %d）", seed, first, second)
		}
		picked[first] = true
	}
	if len(picked) < 2 {
		t.Errorf("同点なのに常に同じleadを選んでいる: %v", picked)
	}
}

// 相性の良い1体をleadに選べる。先頭固定ではない。
func TestLeadPrefersTypeAdvantage(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesPlain, moveTackle), // 等倍で殴り、等倍で殴られる
		fighter(t, speciesVolt, moveSpark),   // みずへ2倍
		fighter(t, speciesDune, moveTackle),  // みずから2倍もらう
	}
	opponent := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesWave, moveTackle),
		fighter(t, speciesWave, moveTackle),
		fighter(t, speciesWave, moveTackle),
	}

	for seed := uint64(0); seed < 5; seed++ {
		if got := New(testData(), seed).Lead(own, opponent); got != 1 {
			t.Errorf("seed %d: leadが %d（みずへ強い 1 のはず）", seed, got)
		}
	}
}

// 効果ばつぐんの技を等倍の技より優先する。
func TestPrefersSuperEffectiveMove(t *testing.T) {
	t.Parallel()

	state := stateOf(t,
		[battle.TeamSize]battle.Pokemon{
			fighter(t, speciesVolt, moveTackle, moveSpark),
			fighter(t, speciesPlain, moveTackle),
			fighter(t, speciesPlain, moveTackle),
		},
		team(t, speciesWave),
	)

	action := New(testData(), 1).Action(state, battle.Player1)
	assertMove(t, action, 1)
}

// 既に状態異常の相手へねむり技を撃ち続けない。
func TestAvoidsStatusMoveOnAStatusedTarget(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesWave, moveTackle, moveLull),
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
	}

	// 相手が無傷なら、ねむり技の方が点が高い。
	healthy := stateOf(t, own, team(t, speciesPlain))
	assertMove(t, New(testData(), 1).Action(healthy, battle.Player1), 1)

	// 既に眠っているなら重ねない。
	asleep := stateOf(t, own, team(t, speciesPlain))
	asleep.Players[battle.Player2].Team[0].Status = battle.Sleep
	asleep.Players[battle.Player2].Team[0].SleepTurns = 2
	assertMove(t, New(testData(), 1).Action(asleep, battle.Player1), 0)
}

// HPが高いうちは回復技を選ばない。減ったら候補になる。
func TestHealsOnlyWhenHurt(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesWave, moveTackle, moveMend),
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
	}

	full := stateOf(t, own, team(t, speciesPlain))
	assertMove(t, New(testData(), 1).Action(full, battle.Player1), 0)

	hurt := stateOf(t, own, team(t, speciesPlain))
	active := &hurt.Players[battle.Player1].Team[0]
	active.CurrentHP = active.Stats.HP * 2 / 5
	assertMove(t, New(testData(), 1).Action(hurt, battle.Player1), 1)
}

// 無傷のうちは自爆を選ばない。追い詰められたら選ぶ。
func TestExplodesOnlyWhenNearlyDead(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesPlain, moveTackle, moveBoom),
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
	}

	full := stateOf(t, own, team(t, speciesWave))
	assertMove(t, New(testData(), 1).Action(full, battle.Player1), 0)

	dying := stateOf(t, own, team(t, speciesWave))
	active := &dying.Players[battle.Player1].Team[0]
	active.CurrentHP = active.Stats.HP / 10
	assertMove(t, New(testData(), 1).Action(dying, battle.Player1), 1)
}

// Speedが上限まで上がっていれば、もう上げる技を選ばない。
func TestDoesNotBoostSpeedAtTheCap(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesPlain, moveHaste, moveTackle),
		fighter(t, speciesPlain, moveTackle),
		fighter(t, speciesPlain, moveTackle),
	}

	capped := stateOf(t, own, team(t, speciesWave))
	capped.Players[battle.Player1].Team[0].Stages.Speed = battle.StageMax
	assertMove(t, New(testData(), 1).Action(capped, battle.Player1), 1)
}

// 明らかに相性が悪く、改善する控えがいるなら交代する。
func TestSwitchesAwayFromABadMatchup(t *testing.T) {
	t.Parallel()

	state := badMatchupState(t)

	action := New(testData(), 1).Action(state, battle.Player1)
	switchAction, ok := action.(battle.SwitchAction)
	if !ok {
		t.Fatalf("交代しなかった: %T %+v", action, action)
	}
	if switchAction.Target != 1 {
		t.Errorf("交代先が %d（じめんへ2倍を取れる 1 のはず）", switchAction.Target)
	}
}

// 交代した直後のturnはまた交代しない。
func TestDoesNotBounceBackAndForth(t *testing.T) {
	t.Parallel()

	state := badMatchupState(t)
	bot := New(testData(), 1)

	if _, ok := bot.Action(state, battle.Player1).(battle.SwitchAction); !ok {
		t.Fatal("1回目で交代しなかった")
	}
	// 状態は変えずにもう一度聞く。出た直後の扱いなので交代してはいけない。
	if action := bot.Action(state, battle.Player1); !isMove(action) {
		t.Errorf("交代直後にまた交代した: %T %+v", action, action)
	}
	// cooldownが解けたので、また交代できる。
	if _, ok := bot.Action(state, battle.Player1).(battle.SwitchAction); !ok {
		t.Error("cooldownが解けても交代しない")
	}
}

// 相性が悪くても、改善する控えがいなければ交代しない。
func TestStaysWhenNoReserveImproves(t *testing.T) {
	t.Parallel()

	state := stateOf(t,
		[battle.TeamSize]battle.Pokemon{
			fighter(t, speciesVolt, moveSpark),   // じめんへ無効
			fighter(t, speciesDune, moveTackle),  // じめんを等倍で受けて等倍で殴る
			fighter(t, speciesPlain, moveTackle), // 同上
		},
		team(t, speciesDune),
	)

	if action := New(testData(), 1).Action(state, battle.Player1); !isMove(action) {
		t.Errorf("改善しない控えへ交代した: %T %+v", action, action)
	}
}

// 戦闘不能のあとの交代先は必ず合法。
func TestReplacementIsLegal(t *testing.T) {
	t.Parallel()

	state := stateOf(t,
		[battle.TeamSize]battle.Pokemon{
			fighter(t, speciesPlain, moveTackle),
			fighter(t, speciesVolt, moveSpark),
			fighter(t, speciesLeaf, moveVine),
		},
		team(t, speciesWave),
	)
	state.Players[battle.Player1].Team[0].CurrentHP = 0

	action := New(testData(), 1).Replacement(state, battle.Player1)

	player := state.Players[battle.Player1]
	legal := false
	for _, index := range player.Reserve() {
		if index == action.Target {
			legal = true
		}
	}
	if !legal {
		t.Fatalf("交代先 %d が控えに居ない（控え: %v）", action.Target, player.Reserve())
	}
	// でんきもくさもみずへ2倍だが、くさはみずを半減で受けられるぶん上。
	if action.Target != 2 {
		t.Errorf("交代先が %d（みずへ最も強い 2 のはず）", action.Target)
	}
}

// 使える技が1つも無ければStruggleを選ぶ。
func TestStruggleWhenNoPPIsLeft(t *testing.T) {
	t.Parallel()

	state := stateOf(t,
		[battle.TeamSize]battle.Pokemon{
			fighter(t, speciesPlain, moveTackle, moveMend),
			fighter(t, speciesPlain, moveTackle),
			fighter(t, speciesPlain, moveTackle),
		},
		team(t, speciesWave),
	)
	for i := range state.Players[battle.Player1].Team[0].Moves {
		state.Players[battle.Player1].Team[0].Moves[i].PP = 0
	}

	if _, ok := New(testData(), 1).Action(state, battle.Player1).(battle.StruggleAction); !ok {
		t.Error("PPが尽きてもStruggleを選ばない")
	}
}

// 相手の技構成は判断に使わない。相手の技だけを差し替えても行動は変わらない。
func TestIgnoresTheOpponentMoveSet(t *testing.T) {
	t.Parallel()

	own := [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesVolt, moveTackle, moveSpark),
		fighter(t, speciesLeaf, moveVine),
		fighter(t, speciesPlain, moveTackle),
	}

	// 相手は同じ個体で、持っている技だけが違う。
	harmless := stateOf(t, own, [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesDune, moveMend),
		fighter(t, speciesWave, moveMend),
		fighter(t, speciesWave, moveMend),
	})
	dangerous := stateOf(t, own, [battle.TeamSize]battle.Pokemon{
		fighter(t, speciesDune, moveTackle, moveBoom),
		fighter(t, speciesWave, moveVine),
		fighter(t, speciesWave, moveSpark),
	})

	first := New(testData(), 1).Action(harmless, battle.Player1)
	second := New(testData(), 1).Action(dangerous, battle.Player1)
	if first != second {
		t.Errorf("相手の技構成で行動が変わった: %T %+v と %T %+v", first, first, second, second)
	}
}

// badMatchupState はでんきがじめんと向き合い、くさの控えがいる状態を返す。
func badMatchupState(t *testing.T) battle.BattleState {
	t.Helper()

	return stateOf(t,
		[battle.TeamSize]battle.Pokemon{
			fighter(t, speciesVolt, moveSpark),   // でんきはじめんへ無効、じめんから2倍
			fighter(t, speciesLeaf, moveVine),    // くさはじめんへ2倍、じめんから半減
			fighter(t, speciesPlain, moveTackle), // 等倍どうし
		},
		team(t, speciesDune),
	)
}

// fighter は検証を通る個体を返す。PPは定義の最大値で埋める。
func fighter(t *testing.T, species battle.SpeciesID, moves ...battle.MoveID) battle.Pokemon {
	t.Helper()

	data := testData()
	p := battle.Pokemon{
		Species:   species,
		Level:     50,
		CurrentHP: 200,
		Stats:     battle.Stats{HP: 200, Attack: 120, Defense: 120, Speed: 100, Special: 120},
	}
	for i, id := range moves {
		move, err := data.LookupMove(id)
		if err != nil {
			t.Fatalf("technical: 未定義の技 %q", id)
		}
		p.Moves[i] = battle.MoveSlot{Move: move.ID, PP: move.MaxPP, MaxPP: move.MaxPP}
	}
	return p
}

// team は同じ個体3体のteamを返す。相手側の中身を問わない場面で使う。
func team(t *testing.T, species battle.SpeciesID) [battle.TeamSize]battle.Pokemon {
	t.Helper()

	return [battle.TeamSize]battle.Pokemon{
		fighter(t, species, moveTackle),
		fighter(t, species, moveTackle),
		fighter(t, species, moveTackle),
	}
}

// stateOf は両teamから開始状態を作る。
func stateOf(t *testing.T, own, opponent [battle.TeamSize]battle.Pokemon) battle.BattleState {
	t.Helper()

	state, err := battle.NewBattleState(own, opponent)
	if err != nil {
		t.Fatalf("NewBattleState()に失敗: %v", err)
	}
	return state
}

// assertMove は指定のslotの技を選んだことを確かめる。
func assertMove(t *testing.T, action battle.Action, slot int) {
	t.Helper()

	move, ok := action.(battle.MoveAction)
	if !ok {
		t.Fatalf("技を選ばなかった: %T %+v", action, action)
	}
	if move.Slot != slot {
		t.Errorf("slot %d を選んだ（%d のはず）", move.Slot, slot)
	}
}

// isMove は技かStruggleを選んだかを返す。
func isMove(action battle.Action) bool {
	switch action.(type) {
	case battle.MoveAction, battle.StruggleAction:
		return true
	default:
		return false
	}
}
