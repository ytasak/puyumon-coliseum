package singleplayer

import (
	"errors"
	"reflect"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
)

// maxTestTurns はtestで試合を進めるときの上限。決着しないことを検出するための安全装置。
const maxTestTurns = 200

// leadを選ぶまでBattleStateは存在しない。
func TestStateIsAbsentBeforeLeads(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 1})

	if got := session.Phase(); got != PhaseLeadSelection {
		t.Errorf("開始phaseが %s（%s のはず）", got, PhaseLeadSelection)
	}
	if _, ok := session.State(); ok {
		t.Error("lead選択前にBattleStateが取れてしまう")
	}
	if _, ok := session.Outcome(); ok {
		t.Error("lead選択前にOutcomeが取れてしまう")
	}
}

// lead選択前にActionは送れない。
func TestActionBeforeLeadsIsRejected(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 1})

	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 0}); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("lead選択前のActionが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
	if _, err := session.SubmitReplacement(Player, battle.SwitchAction{Target: 1}); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("lead選択前のReplacementが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
}

// 取れないlead indexは拒否する。
func TestInvalidLeadIsRejected(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 1})

	for _, index := range []int{-1, battle.TeamSize, battle.TeamSize + 1} {
		if err := session.SubmitLead(Player, index); !errors.Is(err, ErrInvalidCommand) {
			t.Errorf("lead index %d が %v で拒否された（ErrInvalidCommandのはず）", index, err)
		}
	}
	if err := session.SubmitLead(battle.Side(9), 0); !errors.Is(err, ErrInvalidCommand) {
		t.Error("不正なsideのleadが通ってしまう")
	}

	if err := session.SubmitLead(Player, 0); err != nil {
		t.Fatalf("正しいleadが拒否された: %v", err)
	}
	if err := session.SubmitLead(Player, 1); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("2度目のleadが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
}

// 両者がleadを選んだ時点で対戦が始まる。
func TestBattleStartsAfterBothLeads(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 3})

	if err := session.SubmitLead(Player, 2); err != nil {
		t.Fatalf("playerのleadが拒否された: %v", err)
	}
	if got := session.Phase(); got != PhaseLeadSelection {
		t.Errorf("片側だけでphaseが %s になった（%s のはず）", got, PhaseLeadSelection)
	}
	if _, ok := session.State(); ok {
		t.Error("片側だけでBattleStateが作られている")
	}

	if err := session.SubmitLead(Opponent, 1); err != nil {
		t.Fatalf("opponentのleadが拒否された: %v", err)
	}
	if got := session.Phase(); got != PhaseBattle {
		t.Errorf("両lead確定後のphaseが %s（%s のはず）", got, PhaseBattle)
	}

	state, ok := session.State()
	if !ok {
		t.Fatal("両lead確定後にBattleStateが取れない")
	}
	if got := state.Players[Player].Active; got != 2 {
		t.Errorf("playerのactiveが %d（2のはず）", got)
	}
	if got := state.Players[Opponent].Active; got != 1 {
		t.Errorf("opponentのactiveが %d（1のはず）", got)
	}
	if state.Turn != 1 {
		t.Errorf("開始turnが %d（1のはず）", state.Turn)
	}

	// 選んだのはactiveだけで、配布順は変えない。
	dealt := teamOrFatal(t, session, Player)
	if speciesOf(dealt) != speciesOf(state.Players[Player].Team) {
		t.Errorf("lead選択でteamの並びが変わった: %v -> %v", speciesOf(dealt), speciesOf(state.Players[Player].Team))
	}
}

// 両者のActionが揃った時点でturnが解決する。
func TestSubmitActionResolvesATurn(t *testing.T) {
	t.Parallel()

	session := startedSession(t, Config{Seed: 5})

	result, err := session.SubmitAction(Player, battle.MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("playerのActionが拒否された: %v", err)
	}
	if result.Resolved {
		t.Error("片側だけでturnが解決してしまった")
	}
	if result.Events != nil {
		t.Errorf("相手待ちなのにEventが返った: %v", result.Events)
	}

	result, err = session.SubmitAction(Opponent, battle.MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("opponentのActionが拒否された: %v", err)
	}
	if !result.Resolved {
		t.Fatal("両者揃ってもturnが解決しない")
	}
	if len(result.Events) == 0 {
		t.Error("解決したturnのEventが空")
	}

	state, _ := session.State()
	if state.Turn != 2 {
		t.Errorf("解決後のturnが %d（2のはず）", state.Turn)
	}
	if got := len(session.History()); got != len(result.Events) {
		t.Errorf("historyが %d 件（%d 件のはず）", got, len(result.Events))
	}
}

// 同じsideは同じturnに2度submitできない。
func TestSecondSubmitFromTheSameSideIsRejected(t *testing.T) {
	t.Parallel()

	session := startedSession(t, Config{Seed: 5})

	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 0}); err != nil {
		t.Fatalf("1度目のActionが拒否された: %v", err)
	}
	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 1}); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("2度目のActionが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
}

// 拒否されたcommandは、溜めてある相手のActionを壊さない。
func TestRejectedCommandKeepsTheBufferedAction(t *testing.T) {
	t.Parallel()

	session := startedSession(t, Config{Seed: 5})

	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 0}); err != nil {
		t.Fatalf("playerのActionが拒否された: %v", err)
	}
	// 拒否されるcommandをいくつか挟む。
	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 1}); err == nil {
		t.Fatal("2度目のActionが通ってしまった")
	}
	if _, err := session.SubmitAction(Opponent, nil); err == nil {
		t.Fatal("空のActionが通ってしまった")
	}
	if _, err := session.SubmitReplacement(Opponent, battle.SwitchAction{Target: 1}); err == nil {
		t.Fatal("phase違いのReplacementが通ってしまった")
	}

	// playerのActionが残っているので、opponentが出せばそのまま解決する。
	result, err := session.SubmitAction(Opponent, battle.MoveAction{Slot: 0})
	if err != nil {
		t.Fatalf("opponentのActionが拒否された: %v", err)
	}
	if !result.Resolved {
		t.Error("拒否されたcommandがplayerのActionを捨てている")
	}
}

// Teamは複製を返す。外から書き換えてもsessionの中は変わらない。
func TestTeamIsACopy(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 11})

	team := teamOrFatal(t, session, Player)
	before := team[0].CurrentHP
	team[0].CurrentHP = 0
	team[0].Species = "tampered"

	after := teamOrFatal(t, session, Player)
	if after[0].CurrentHP != before || after[0].Species == "tampered" {
		t.Error("Teamの戻り値を書き換えるとsessionの状態が変わってしまう")
	}
}

// 戦闘不能になるとreplacement phaseへ移り、控えを出すとbattleへ戻る。
func TestFaintMovesToReplacementAndBack(t *testing.T) {
	t.Parallel()

	session := startedSession(t, Config{Seed: 5})
	phases := play(t, session, maxTestTurns)

	replaced := false
	for i, phase := range phases {
		if phase != PhaseReplacement {
			continue
		}
		replaced = true
		if i+1 < len(phases) && phases[i+1] == PhaseReplacement {
			// 両者が同時に倒れた場合は続けて交代する。
			continue
		}
		if i+1 < len(phases) && phases[i+1] != PhaseBattle {
			t.Errorf("replacementの次が %s（%s のはず）", phases[i+1], PhaseBattle)
		}
	}
	if !replaced {
		t.Fatal("1試合を通してreplacement phaseへ一度も入らなかった")
	}
}

// 決着まで進めるとFinishedになり、それ以上commandを受け付けない。
func TestBattleRunsToFinish(t *testing.T) {
	t.Parallel()

	session := startedSession(t, Config{Seed: 5})
	play(t, session, maxTestTurns)

	if got := session.Phase(); got != PhaseFinished {
		t.Fatalf("決着後のphaseが %s（%s のはず）", got, PhaseFinished)
	}

	outcome, ok := session.Outcome()
	if !ok {
		t.Fatal("決着後にOutcomeが取れない")
	}
	if outcome == battle.Ongoing {
		t.Error("決着したのにOutcomeがOngoing")
	}
	if len(session.History()) == 0 {
		t.Error("historyが空のまま決着した")
	}

	if _, err := session.SubmitAction(Player, battle.MoveAction{Slot: 0}); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("決着後のActionが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
	if err := session.SubmitLead(Player, 0); !errors.Is(err, ErrInvalidCommand) {
		t.Errorf("決着後のleadが %v で拒否された（ErrInvalidCommandのはず）", err)
	}
}

// 同じconfigと同じcommand列からは同じ試合になる。
func TestSameConfigAndActionsAreDeterministic(t *testing.T) {
	t.Parallel()

	first := startedSession(t, Config{Seed: 5})
	play(t, first, maxTestTurns)

	second := startedSession(t, Config{Seed: 5})
	play(t, second, maxTestTurns)

	firstState, _ := first.State()
	secondState, _ := second.State()
	if !reflect.DeepEqual(firstState, secondState) {
		t.Error("同じconfigと同じcommand列で最終stateが違う")
	}
	if !reflect.DeepEqual(first.History(), second.History()) {
		t.Error("同じconfigと同じcommand列でEvent列が違う")
	}
}

// Botのseedは対戦の乱数列とは別に取れる。
func TestBotSeedIsSeparateFromTheBattleStream(t *testing.T) {
	t.Parallel()

	session := newSessionOrFatal(t, Config{Seed: 5})
	derived := deriveSeeds(5)

	if got := session.BotSeed(); got != derived.bot {
		t.Errorf("BotSeedが %d（%d のはず）", got, derived.bot)
	}
	if session.BotSeed() == derived.battle || session.BotSeed() == derived.setup {
		t.Error("Bot用のseedが他の用途と同じ値になっている")
	}
}

// startedSession はleadまで決めてPhaseBattleにしたsessionを返す。
func startedSession(t *testing.T, cfg Config) *Session {
	t.Helper()

	session := newSessionOrFatal(t, cfg)
	for _, side := range sides {
		if err := session.SubmitLead(side, 0); err != nil {
			t.Fatalf("%s のleadが拒否された: %v", side, err)
		}
	}
	if session.Phase() != PhaseBattle {
		t.Fatalf("lead確定後のphaseが %s", session.Phase())
	}
	return session
}

// play は決着まで試合を進め、各回で見たphaseを順に返す。
//
// 行動の選び方はこのIssueの対象ではないので、使える技の先頭を選ぶだけにする。
func play(t *testing.T, s *Session, maxTurns int) []Phase {
	t.Helper()

	var phases []Phase
	for turns := 0; s.Phase() != PhaseFinished; {
		phases = append(phases, s.Phase())

		switch s.Phase() {
		case PhaseReplacement:
			state, _ := s.State()
			for _, side := range sides {
				if !state.NeedsReplacement(side) {
					continue
				}
				if _, err := s.SubmitReplacement(side, chooseReplacement(t, state, side)); err != nil {
					t.Fatalf("%s のreplacementが拒否された: %v", side, err)
				}
				state, _ = s.State()
			}
		case PhaseBattle:
			state, _ := s.State()
			for _, side := range sides {
				if _, err := s.SubmitAction(side, chooseAction(state, side)); err != nil {
					t.Fatalf("turn %d の %s のActionが拒否された: %v", state.Turn, side, err)
				}
			}
			turns++
			if turns >= maxTurns {
				t.Fatalf("%d turnで決着しなかった", maxTurns)
			}
		default:
			t.Fatalf("進められないphase: %s", s.Phase())
		}
	}
	return phases
}

// chooseAction は使える技の先頭を選ぶ。1つも無ければStruggleにする。
func chooseAction(state battle.BattleState, side battle.Side) battle.Action {
	active := state.Players[side].ActivePokemon()
	for slot, move := range active.Moves {
		if move.Usable() {
			return battle.MoveAction{Slot: slot}
		}
	}
	return battle.StruggleAction{}
}

// chooseReplacement は控えの先頭を出す。
func chooseReplacement(t *testing.T, state battle.BattleState, side battle.Side) battle.SwitchAction {
	t.Helper()

	player := state.Players[side]
	reserve := player.Reserve()
	if len(reserve) == 0 {
		t.Fatalf("%s に出せる控えが無いのにreplacementを求められた", side)
	}
	return battle.SwitchAction{Target: reserve[0]}
}
