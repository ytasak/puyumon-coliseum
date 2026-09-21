package bot_test

import (
	"reflect"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/bot"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// maxSmokeTurns は1試合を打ち切るturn数。決着しないことを検出するための安全装置。
const maxSmokeTurns = 500

// sides は両sideを順に処理するための一覧。
var sides = [...]battle.Side{singleplayer.Player, singleplayer.Opponent}

// step は1試合で実際にsubmitしたcommandの控え。
//
// Botを使わずに同じcommandをもう一度流すために取っておく。
type step struct {
	side        battle.Side
	action      battle.Action
	replacement battle.SwitchAction
	isReplace   bool
}

// Botはsessionへ注入でき、1000試合回しても合法でないActionを出さない。
func TestBotDrivesSessionsWithoutInvalidActions(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 1000; seed++ {
		session := newSession(t, seed)
		if _, steps := playWithBots(t, session, seed); len(steps) == 0 {
			t.Fatalf("seed %d: 1手も進まなかった", seed)
		}
		if session.Phase() != singleplayer.PhaseFinished {
			t.Fatalf("seed %d: %s のまま決着しなかった", seed, session.Phase())
		}
		if _, ok := session.Outcome(); !ok {
			t.Fatalf("seed %d: 決着後にOutcomeが取れない", seed)
		}
	}
}

// 同じseedのBotは同じ試合を再現する。
func TestBotIsDeterministic(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 20; seed++ {
		first := newSession(t, seed)
		_, firstSteps := playWithBots(t, first, seed)

		second := newSession(t, seed)
		_, secondSteps := playWithBots(t, second, seed)

		if !reflect.DeepEqual(firstSteps, secondSteps) {
			t.Fatalf("seed %d: 同じseedで違うcommand列になった", seed)
		}
		if !reflect.DeepEqual(first.History(), second.History()) {
			t.Fatalf("seed %d: 同じseedで違うEvent列になった", seed)
		}
	}
}

// Botは対戦の乱数列を消費しない。
//
// Botに選ばせた試合と、その選択をそのまま流し直した試合が完全に一致すれば、
// Botの判断がBattle Engineの乱数を進めていないと言える。進めていれば、
// Botの居ない再生側では命中や急所がずれて別の試合になる。
func TestBotDoesNotConsumeTheBattleRNG(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 20; seed++ {
		played := newSession(t, seed)
		leads, steps := playWithBots(t, played, seed)

		replayed := newSession(t, seed)
		for _, side := range sides {
			if err := replayed.SubmitLead(side, leads[side]); err != nil {
				t.Fatalf("seed %d: %s のleadが拒否された: %v", seed, side, err)
			}
		}
		for _, s := range steps {
			var err error
			if s.isReplace {
				_, err = replayed.SubmitReplacement(s.side, s.replacement)
			} else {
				_, err = replayed.SubmitAction(s.side, s.action)
			}
			if err != nil {
				t.Fatalf("seed %d: 再生中に拒否された: %v", seed, err)
			}
		}

		playedState, _ := played.State()
		replayedState, _ := replayed.State()
		if !reflect.DeepEqual(playedState, replayedState) {
			t.Errorf("seed %d: Botの有無で最終stateが変わった", seed)
		}
		if !reflect.DeepEqual(played.History(), replayed.History()) {
			t.Errorf("seed %d: Botの有無でEvent列が変わった", seed)
		}
	}
}

// newSession は1試合ぶんのsessionを作る。
func newSession(t *testing.T, seed uint64) *singleplayer.Session {
	t.Helper()

	session, err := singleplayer.NewSession(singleplayer.Config{Seed: seed})
	if err != nil {
		t.Fatalf("NewSession(seed %d)に失敗: %v", seed, err)
	}
	return session
}

// playWithBots は両sideをBotに任せて決着まで進め、選んだleadとcommand列を返す。
//
// 両sideに別のseedを渡すのはtestの都合で、同じ乱数列だと同点の割り方まで
// 揃ってしまうため。production側の割り当てではない。
func playWithBots(t *testing.T, session *singleplayer.Session, seed uint64) ([2]int, []step) {
	t.Helper()

	bots := map[battle.Side]*bot.Bot{
		singleplayer.Player:   bot.New(roster.Data(), session.BotSeed()+1),
		singleplayer.Opponent: bot.New(roster.Data(), session.BotSeed()),
	}

	var leads [2]int
	for _, side := range sides {
		own, ok := session.Team(side)
		if !ok {
			t.Fatalf("seed %d: %s のteamを取得できない", seed, side)
		}
		opponent, ok := session.Team(side.Opponent())
		if !ok {
			t.Fatalf("seed %d: %s のteamを取得できない", seed, side.Opponent())
		}

		leads[side] = bots[side].Lead(own, opponent)
		if err := session.SubmitLead(side, leads[side]); err != nil {
			t.Fatalf("seed %d: %s のleadが拒否された: %v", seed, side, err)
		}
	}

	var steps []step
	for turns := 0; session.Phase() != singleplayer.PhaseFinished; {
		switch session.Phase() {
		case singleplayer.PhaseReplacement:
			state, _ := session.State()
			for _, side := range sides {
				if !state.NeedsReplacement(side) {
					continue
				}
				replacement := bots[side].Replacement(state, side)
				if _, err := session.SubmitReplacement(side, replacement); err != nil {
					t.Fatalf("seed %d: %s のreplacementが拒否された: %v", seed, side, err)
				}
				steps = append(steps, step{side: side, replacement: replacement, isReplace: true})
				state, _ = session.State()
			}

		case singleplayer.PhaseBattle:
			state, _ := session.State()
			for _, side := range sides {
				action := bots[side].Action(state, side)
				if _, err := session.SubmitAction(side, action); err != nil {
					t.Fatalf("seed %d: turn %d の %s のActionが拒否された: %v", seed, state.Turn, side, err)
				}
				steps = append(steps, step{side: side, action: action})
			}
			turns++
			if turns >= maxSmokeTurns {
				t.Fatalf("seed %d: %d turnで決着しなかった", seed, maxSmokeTurns)
			}

		default:
			t.Fatalf("seed %d: 進められないphase: %s", seed, session.Phase())
		}
	}
	return leads, steps
}
