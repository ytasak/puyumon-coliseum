package singleplayer_test

import (
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/bot"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// 1試合の流れを通しで確かめる統合test。
//
// UIには依存しない。sessionへcommandを送り、相手の判断は YTA-29 のBotへ委ねる。
// 対戦のルールそのものは検証しない。それは internal/battle のtestの範囲で、
// ここで見るのは「配布からresultまで通るか」と「層の境界が壊れていないか」。

// testMaxTurns は1試合を打ち切るturn数。
//
// **対戦のルールではなくtestの安全装置。** 決着しなくなったことを検出するために置く。
// gameplayのturn capではないので、この値を根拠に仕様を決めない。
const testMaxTurns = 500

// 代表scenarioに使うseed。探して選んだ固定値で、状態異常・戦闘不能・交代・決着を含む。
const scenarioSeed = 1

// struggleSeed はplayerのactiveが4技すべてPP 0になるseed。
//
// 0..2999 を「残りPPが最も多い技を選ぶ」方針で回して見つけた1件。
// Struggleは4技すべてのPPが尽きてはじめて合法なので、通常のsessionで
// そこへ到達できることを示すために固定している。
const struggleSeed = 247

// sides は両sideを順に処理するための一覧。
var sides = [...]battle.Side{singleplayer.Player, singleplayer.Opponent}

// playerPolicy はplayer側の行動を決める。相手はBotに委ねる。
type playerPolicy func(state battle.BattleState) battle.Action

// record は1試合を通して観測したこと。
//
// Eventの全snapshotは残さない。主要な遷移と件数だけを見る。
type record struct {
	phases       []singleplayer.Phase
	eventKinds   map[string]int
	statuses     int
	faints       int
	switches     int
	replacements int
	struggles    int
	turns        int
	outcome      battle.Status
	history      []battle.Event
	finalState   battle.BattleState
}

// playMatch は1試合を決着まで進め、観測したことを返す。
func playMatch(t *testing.T, seed uint64, policy playerPolicy) record {
	t.Helper()

	session, err := singleplayer.NewSession(singleplayer.Config{Seed: seed})
	if err != nil {
		t.Fatalf("seed %d: NewSession()に失敗: %v", seed, err)
	}
	opponent := bot.New(roster.Data(), session.BotSeed())

	for _, side := range sides {
		own, ok := session.Team(side)
		if !ok {
			t.Fatalf("seed %d: %s のteamを取得できない", seed, side)
		}
		foe, _ := session.Team(side.Opponent())

		lead := 0
		if side == singleplayer.Opponent {
			lead = opponent.Lead(own, foe)
		}
		if err := session.SubmitLead(side, lead); err != nil {
			t.Fatalf("seed %d: %s のleadが拒否された: %v", seed, side, err)
		}
	}

	got := record{eventKinds: map[string]int{}}
	got.phases = append(got.phases, singleplayer.PhaseLeadSelection)

	for session.Phase() != singleplayer.PhaseFinished {
		if got.turns >= testMaxTurns {
			t.Fatalf("seed %d: %d turnで決着しなかった", seed, testMaxTurns)
		}
		got.notePhase(session.Phase())

		state, ok := session.State()
		if !ok {
			t.Fatalf("seed %d: stateが取れない", seed)
		}

		if session.Phase() == singleplayer.PhaseReplacement {
			for _, side := range sides {
				if !state.NeedsReplacement(side) {
					continue
				}
				action := opponent.Replacement(state, side)
				if side == singleplayer.Player {
					action = battle.SwitchAction{Target: state.Players[side].Reserve()[0]}
					got.replacements++
				}
				events, err := session.SubmitReplacement(side, action)
				if err != nil {
					t.Fatalf("seed %d: %s のreplacementが拒否された: %v", seed, side, err)
				}
				got.count(events)
				state, _ = session.State()
			}
			continue
		}

		action := policy(state)
		switch action.(type) {
		case battle.SwitchAction:
			got.switches++
		case battle.StruggleAction:
			got.struggles++
		}
		if _, err := session.SubmitAction(singleplayer.Player, action); err != nil {
			t.Fatalf("seed %d turn %d: playerのActionが拒否された: %v", seed, state.Turn, err)
		}

		result, err := session.SubmitAction(singleplayer.Opponent, opponent.Action(state, singleplayer.Opponent))
		if err != nil {
			t.Fatalf("seed %d turn %d: 相手のActionが拒否された: %v", seed, state.Turn, err)
		}
		got.count(result.Events)
		got.turns++
	}

	got.notePhase(singleplayer.PhaseFinished)
	got.outcome, _ = session.Outcome()
	got.history = session.History()
	got.finalState, _ = session.State()
	return got
}

// notePhase はphaseの遷移を控える。同じphaseが続く分はまとめる。
func (r *record) notePhase(phase singleplayer.Phase) {
	if len(r.phases) > 0 && r.phases[len(r.phases)-1] == phase {
		return
	}
	r.phases = append(r.phases, phase)
}

// count はEventの種別ごとの件数を数える。
func (r *record) count(events []battle.Event) {
	for _, event := range events {
		name := fmt.Sprintf("%T", event)
		r.eventKinds[name[strings.LastIndex(name, ".")+1:]]++

		switch event.(type) {
		case battle.StatusApplied:
			r.statuses++
		case battle.Fainted:
			r.faints++
		}
	}
}

// firstUsable は使える技の先頭を選ぶ。使える技が無ければStruggle。
func firstUsable(state battle.BattleState) battle.Action {
	player := state.Players[singleplayer.Player]
	active := player.Team[player.Active]
	for slot := range active.Moves {
		if active.Moves[slot].Usable() {
			return battle.MoveAction{Slot: slot}
		}
	}
	return battle.StruggleAction{}
}

// mostPP は残りPPが最も多い技を選ぶ。4技を均等に減らして枯らしにいく。
func mostPP(state battle.BattleState) battle.Action {
	player := state.Players[singleplayer.Player]
	active := player.Team[player.Active]

	best, bestPP := -1, 0
	for slot := range active.Moves {
		if move := active.Moves[slot]; move.Usable() && move.PP > bestPP {
			best, bestPP = slot, move.PP
		}
	}
	if best < 0 {
		return battle.StruggleAction{}
	}
	return battle.MoveAction{Slot: best}
}

// 1. 配布はseedで決まり、6体を重複なく3+3へ分ける。
func TestSeededTrioAssignment(t *testing.T) {
	t.Parallel()

	first := newSessionOrFatal(t, scenarioSeed)
	again := newSessionOrFatal(t, scenarioSeed)
	other := newSessionOrFatal(t, scenarioSeed+1)

	seen := map[battle.SpeciesID]int{}
	for _, side := range sides {
		a, _ := first.Team(side)
		b, _ := again.Team(side)
		if speciesOf(a) != speciesOf(b) {
			t.Errorf("%s: 同じseedで配布が違う", side)
		}
		for _, pokemon := range a {
			seen[pokemon.Species]++
		}
	}
	if len(seen) != len(roster.All()) {
		t.Errorf("配られたのが %d 種類（%d 体いる）", len(seen), len(roster.All()))
	}

	same := true
	for _, side := range sides {
		a, _ := first.Team(side)
		c, _ := other.Team(side)
		if speciesOf(a) != speciesOf(c) {
			same = false
		}
	}
	if same {
		t.Error("seedを変えても配布が変わらない")
	}
}

// 2, 3. 両者がleadを選ぶと対戦が始まる。相手のleadはBotが決める。
func TestLeadSelectionStartsTheBattle(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, scenarioSeed)
	opponent := bot.New(roster.Data(), session.BotSeed())

	if _, started := session.State(); started {
		t.Error("lead選択前にBattleStateがある")
	}

	if err := session.SubmitLead(singleplayer.Player, 2); err != nil {
		t.Fatalf("playerのleadが拒否された: %v", err)
	}
	if session.Phase() != singleplayer.PhaseLeadSelection {
		t.Error("片側だけで対戦が始まった")
	}

	own, _ := session.Team(singleplayer.Opponent)
	foe, _ := session.Team(singleplayer.Player)
	botLead := opponent.Lead(own, foe)
	if err := session.SubmitLead(singleplayer.Opponent, botLead); err != nil {
		t.Fatalf("BotのleadがSessionに拒否された: %v", err)
	}

	if session.Phase() != singleplayer.PhaseBattle {
		t.Fatalf("phase = %s, want %s", session.Phase(), singleplayer.PhaseBattle)
	}
	state, _ := session.State()
	if got := state.Players[singleplayer.Player].Active; got != 2 {
		t.Errorf("playerのactive = %d, want 2", got)
	}
	if got := state.Players[singleplayer.Opponent].Active; got != botLead {
		t.Errorf("相手のactive = %d, want %d", got, botLead)
	}
}

// 4. playerのMoveActionとBotのActionで1 turnが解決する。
func TestTurnResolvesWithBothActions(t *testing.T) {
	t.Parallel()

	session, opponent := startedMatch(t, scenarioSeed)
	before, _ := session.State()

	result, err := session.SubmitAction(singleplayer.Player, firstUsable(before))
	if err != nil {
		t.Fatalf("playerのActionが拒否された: %v", err)
	}
	if result.Resolved {
		t.Error("片側だけでturnが解決した")
	}

	result, err = session.SubmitAction(singleplayer.Opponent, opponent.Action(before, singleplayer.Opponent))
	if err != nil {
		t.Fatalf("相手のActionが拒否された: %v", err)
	}
	if !result.Resolved || len(result.Events) == 0 {
		t.Fatalf("turnが解決していない: %+v", result)
	}

	after, _ := session.State()
	if after.Turn != before.Turn+1 && after.Status == battle.Ongoing {
		t.Errorf("turnが %d から %d へ進んでいない", before.Turn, after.Turn)
	}
}

// 5. 自発的な交代で場に出ている1体が変わる。
func TestVoluntarySwitchChangesTheActive(t *testing.T) {
	t.Parallel()

	session, opponent := startedMatch(t, scenarioSeed)
	before, _ := session.State()

	player := before.Players[singleplayer.Player]
	target := player.Reserve()[0]
	if target == player.Active {
		t.Fatal("technical: 交代先が今のactiveと同じ")
	}

	if _, err := session.SubmitAction(singleplayer.Player, battle.SwitchAction{Target: target}); err != nil {
		t.Fatalf("交代が拒否された: %v", err)
	}
	if _, err := session.SubmitAction(singleplayer.Opponent, opponent.Action(before, singleplayer.Opponent)); err != nil {
		t.Fatalf("相手のActionが拒否された: %v", err)
	}

	after, _ := session.State()
	if got := after.Players[singleplayer.Player].Active; got != target {
		t.Errorf("交代後のactive = %d, want %d", got, target)
	}
}

// 6, 7, 8, 10. 代表scenarioで状態異常・戦闘不能・交代を通り、決着する。
func TestRepresentativeMatchCoversTheLifecycle(t *testing.T) {
	t.Parallel()

	got := playMatch(t, scenarioSeed, firstUsable)

	if got.statuses == 0 {
		t.Error("状態異常が1度も起きていない")
	}
	if got.faints == 0 {
		t.Error("戦闘不能が1度も起きていない")
	}
	if got.replacements == 0 {
		t.Error("playerの交代が1度も起きていない")
	}
	if got.outcome == battle.Ongoing {
		t.Errorf("決着していない: %s", got.outcome)
	}
	for _, kind := range []string{"MoveUsed", "Damage", "StatusApplied", "Fainted", "Switched"} {
		if got.eventKinds[kind] == 0 {
			t.Errorf("%s のEventが出ていない", kind)
		}
	}

	// 主要な遷移を固定する。Event列そのものは残さない。
	if first, last := got.phases[0], got.phases[len(got.phases)-1]; first != singleplayer.PhaseLeadSelection || last != singleplayer.PhaseFinished {
		t.Errorf("phaseの端が %s -> %s", first, last)
	}
	if !containsPhase(got.phases, singleplayer.PhaseReplacement) {
		t.Error("replacementを通っていない")
	}
	assertPhaseTransitions(t, got.phases)
}

// 9. 4技すべてのPPが尽きるとStruggleが合法になる。
//
// 1技だけ尽きてもStruggleにはならない。通常のsessionで4技すべてを
// 枯らせることを、固定したseedで示す。
func TestStruggleBecomesLegalWhenEveryMoveIsSpent(t *testing.T) {
	t.Parallel()

	session, opponent := startedMatch(t, struggleSeed)

	for turns := 0; turns < testMaxTurns; turns++ {
		if session.Phase() == singleplayer.PhaseFinished {
			t.Fatal("全技のPPが尽きる前に決着した")
		}

		state, _ := session.State()
		if session.Phase() == singleplayer.PhaseReplacement {
			for _, side := range sides {
				if !state.NeedsReplacement(side) {
					continue
				}
				action := opponent.Replacement(state, side)
				if side == singleplayer.Player {
					action = battle.SwitchAction{Target: state.Players[side].Reserve()[0]}
				}
				if _, err := session.SubmitReplacement(side, action); err != nil {
					t.Fatalf("replacementが拒否された: %v", err)
				}
				state, _ = session.State()
			}
			continue
		}

		player := state.Players[singleplayer.Player]
		active := player.Team[player.Active]
		if !active.HasUsableMove() {
			// ここまで来てはじめてStruggleが合法になる。
			assertEveryMoveSpent(t, active)
			assertStruggleResolves(t, session, opponent, state)
			return
		}

		if _, err := session.SubmitAction(singleplayer.Player, mostPP(state)); err != nil {
			t.Fatalf("playerのActionが拒否された: %v", err)
		}
		if _, err := session.SubmitAction(singleplayer.Opponent, opponent.Action(state, singleplayer.Opponent)); err != nil {
			t.Fatalf("相手のActionが拒否された: %v", err)
		}
	}
	t.Fatalf("seed %d で全技のPPが尽きなかった", struggleSeed)
}

// 11. 決着後に作る新しいsessionは、前の試合を引き継がない。
//
// UI側のseed導出は internal/game のtestが固定しているので、ここでは
// 別のseedからNewSessionを作り、状態が持ち越されないことだけを見る。
func TestNewSessionDoesNotCarryOver(t *testing.T) {
	t.Parallel()

	played := playMatch(t, scenarioSeed, firstUsable)
	if played.outcome == battle.Ongoing {
		t.Fatal("technical: 決着していない")
	}

	fresh := newSessionOrFatal(t, scenarioSeed+1)

	if got := fresh.Phase(); got != singleplayer.PhaseLeadSelection {
		t.Errorf("新しいsessionのphase = %s, want %s", got, singleplayer.PhaseLeadSelection)
	}
	if _, started := fresh.State(); started {
		t.Error("新しいsessionにBattleStateがある")
	}
	if got := len(fresh.History()); got != 0 {
		t.Errorf("新しいsessionに %d 件の履歴が残っている", got)
	}
	if _, decided := fresh.Outcome(); decided {
		t.Error("新しいsessionに決着がある")
	}

	before := newSessionOrFatal(t, scenarioSeed)
	for _, side := range sides {
		old, _ := before.Team(side)
		next, _ := fresh.Team(side)
		if speciesOf(old) == speciesOf(next) {
			t.Errorf("%s: 新しいsessionでも配布が同じ", side)
		}
	}
}

// 同じseedと同じ操作からは同じ試合になる。
func TestSameSeedAndScriptReproduceTheMatch(t *testing.T) {
	t.Parallel()

	for _, seed := range []uint64{scenarioSeed, scenarioSeed + 1, struggleSeed} {
		first := playMatch(t, seed, firstUsable)
		second := playMatch(t, seed, firstUsable)

		if !reflect.DeepEqual(first.history, second.history) {
			t.Errorf("seed %d: Event列が違う", seed)
		}
		if !reflect.DeepEqual(first.finalState, second.finalState) {
			t.Errorf("seed %d: 最終stateが違う", seed)
		}
		if !reflect.DeepEqual(first.eventKinds, second.eventKinds) {
			t.Errorf("seed %d: Eventの内訳が違う", seed)
		}
		if first.turns != second.turns || first.outcome != second.outcome {
			t.Errorf("seed %d: turn数か勝敗が違う", seed)
		}
	}
}

// 代表seedの結果を固定する。
//
// 巨大なEvent列は残さず、勝敗とturn数だけを置く。engineやBotを意図的に
// 変えたときはここが落ちるので、根拠を確かめてから更新する。
func TestRepresentativeOutcomesAreStable(t *testing.T) {
	t.Parallel()

	cases := []struct {
		seed    uint64
		turns   int
		outcome battle.Status
	}{
		{seed: 1, turns: 38, outcome: battle.Player1Won},
		{seed: 2, turns: 23, outcome: battle.Player2Won},
		{seed: 4, turns: 7, outcome: battle.Player2Won},
	}
	for _, tc := range cases {
		got := playMatch(t, tc.seed, firstUsable)
		if got.turns != tc.turns || got.outcome != tc.outcome {
			t.Errorf("seed %d: %d turn / %s（%d turn / %s のはず）",
				tc.seed, got.turns, got.outcome, tc.turns, tc.outcome)
		}
	}
}

// newSessionOrFatal はsessionを作る。
func newSessionOrFatal(t *testing.T, seed uint64) *singleplayer.Session {
	t.Helper()

	session, err := singleplayer.NewSession(singleplayer.Config{Seed: seed})
	if err != nil {
		t.Fatalf("NewSession(%d)に失敗: %v", seed, err)
	}
	return session
}

// startedMatch はleadまで決めた対戦とBotを返す。
func startedMatch(t *testing.T, seed uint64) (*singleplayer.Session, *bot.Bot) {
	t.Helper()

	session := newSessionOrFatal(t, seed)
	opponent := bot.New(roster.Data(), session.BotSeed())
	for _, side := range sides {
		own, _ := session.Team(side)
		foe, _ := session.Team(side.Opponent())
		lead := 0
		if side == singleplayer.Opponent {
			lead = opponent.Lead(own, foe)
		}
		if err := session.SubmitLead(side, lead); err != nil {
			t.Fatalf("seed %d: %s のleadが拒否された: %v", seed, side, err)
		}
	}
	return session, opponent
}

// assertEveryMoveSpent は4技すべてのPPが尽きていることを確かめる。
func assertEveryMoveSpent(t *testing.T, active battle.Pokemon) {
	t.Helper()

	for slot := range active.Moves {
		if move := active.Moves[slot]; move.Usable() {
			t.Fatalf("slot %d（%s）にPPが %d 残っている", slot, move.Move, move.PP)
		}
	}
}

// assertStruggleResolves はStruggleが受理されてEventになることを確かめる。
func assertStruggleResolves(t *testing.T, session *singleplayer.Session, opponent *bot.Bot, state battle.BattleState) {
	t.Helper()

	action := opponent.Action(state, singleplayer.Opponent)
	if _, err := session.SubmitAction(singleplayer.Player, battle.StruggleAction{}); err != nil {
		t.Fatalf("Struggleが拒否された: %v", err)
	}
	result, err := session.SubmitAction(singleplayer.Opponent, action)
	if err != nil {
		t.Fatalf("相手のActionが拒否された: %v", err)
	}

	for _, event := range result.Events {
		used, ok := event.(battle.MoveUsed)
		if !ok || used.Side != singleplayer.Player {
			continue
		}
		if used.Move != battle.MoveStruggle {
			t.Errorf("使った技が %s（Struggleのはず）", used.Move)
		}
		if used.Slot != battle.NoMoveSlot {
			t.Errorf("Slot = %d, want %d（技枠を使わない）", used.Slot, battle.NoMoveSlot)
		}
		return
	}
	t.Error("Struggleを使ったEventが出ていない")
}

// assertPhaseTransitions は取り得ない遷移が無いことを確かめる。
func assertPhaseTransitions(t *testing.T, phases []singleplayer.Phase) {
	t.Helper()

	allowed := map[singleplayer.Phase][]singleplayer.Phase{
		singleplayer.PhaseLeadSelection: {singleplayer.PhaseBattle},
		singleplayer.PhaseBattle:        {singleplayer.PhaseReplacement, singleplayer.PhaseFinished},
		singleplayer.PhaseReplacement:   {singleplayer.PhaseBattle, singleplayer.PhaseFinished},
	}
	for i := 1; i < len(phases); i++ {
		from, to := phases[i-1], phases[i]
		ok := false
		for _, next := range allowed[from] {
			if next == to {
				ok = true
			}
		}
		if !ok {
			t.Errorf("取り得ない遷移: %s -> %s", from, to)
		}
	}
}

// containsPhase は並びにphaseが含まれるかを返す。
func containsPhase(phases []singleplayer.Phase, want singleplayer.Phase) bool {
	for _, phase := range phases {
		if phase == want {
			return true
		}
	}
	return false
}

// speciesOf は3体のSpeciesIDを配布順で返す。
func speciesOf(team [battle.TeamSize]battle.Pokemon) [battle.TeamSize]battle.SpeciesID {
	var ids [battle.TeamSize]battle.SpeciesID
	for i, pokemon := range team {
		ids[i] = pokemon.Species
	}
	return ids
}
