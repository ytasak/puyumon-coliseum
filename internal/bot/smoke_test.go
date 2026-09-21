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

// matchStats は1試合の観測値。健全性の確認に使う。
//
// balanceの評価には使わない。ここで見るのは「session / Botが壊れていないか」だけ。
type matchStats struct {
	turns      int
	outcome    battle.Status
	unresolved bool

	// switches は自発的な交代の回数。戦闘不能後の交代は数えない。
	switches int

	// struggles はStruggleを選んだ回数。
	struggles int
}

// tally は複数試合ぶんの集計。
type tally struct {
	matches    int
	unresolved int
	p1Wins     int
	p2Wins     int
	draws      int
	turns      int
	maxTurns   int
	switches   int
	struggles  int
}

// add は1試合ぶんを足す。
func (t *tally) add(stats matchStats) {
	t.matches++
	t.turns += stats.turns
	t.switches += stats.switches
	t.struggles += stats.struggles

	if stats.turns > t.maxTurns {
		t.maxTurns = stats.turns
	}
	if stats.unresolved {
		t.unresolved++
		return
	}
	switch stats.outcome {
	case battle.Player1Won:
		t.p1Wins++
	case battle.Player2Won:
		t.p2Wins++
	case battle.Draw:
		t.draws++
	}
}

// meanTurns は1試合あたりの平均turn数を返す。
func (t tally) meanTurns() float64 {
	if t.matches == 0 {
		return 0
	}
	return float64(t.turns) / float64(t.matches)
}

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
//
// あわせて、YTA-32が求める健全性の集計もここで取る。**同じ目的の1000試合を
// 別に走らせない**ため、1回の走行から両方を得る。
//
// 集計のうち次の2つは「0と出る」のではなく「0でなければtestが落ちる」形で担保している。
//
//   - 合法でないAction: submitが拒否された時点でその場で失敗する
//   - panic: 起きればtestが失敗する
//
// つまりこのtestが通ったこと自体が、どちらも0だった根拠になる。
func TestBotDrivesSessionsWithoutInvalidActions(t *testing.T) {
	t.Parallel()

	var totals tally
	for seed := uint64(0); seed < 1000; seed++ {
		session := newSession(t, seed)
		_, steps, stats := playWithBots(t, session, seed)
		if len(steps) == 0 {
			t.Fatalf("seed %d: 1手も進まなかった", seed)
		}
		totals.add(stats)

		if stats.unresolved {
			continue
		}
		if session.Phase() != singleplayer.PhaseFinished {
			t.Fatalf("seed %d: %s のまま決着しなかった", seed, session.Phase())
		}
		if _, ok := session.Outcome(); !ok {
			t.Fatalf("seed %d: 決着後にOutcomeが取れない", seed)
		}
	}

	t.Logf("試合数=%d 合法でないAction=0 panic=0 未決着=%d 勝敗=%d/%d 引き分け=%d "+
		"平均turn=%.1f 最長turn=%d 自発交代=%d Struggle=%d",
		totals.matches, totals.unresolved, totals.p1Wins, totals.p2Wins, totals.draws,
		totals.meanTurns(), totals.maxTurns, totals.switches, totals.struggles)

	if totals.matches < 1000 {
		t.Errorf("試合数が %d 件（1000件以上のはず）", totals.matches)
	}
	if totals.unresolved != 0 {
		t.Errorf("%d 件が %d turnでも決着しなかった", totals.unresolved, maxSmokeTurns)
	}
	if got := totals.p1Wins + totals.p2Wins + totals.draws + totals.unresolved; got != totals.matches {
		t.Errorf("内訳の合計が %d 件で試合数 %d と合わない", got, totals.matches)
	}
}

// 同じseedのBotは同じ試合を再現する。
func TestBotIsDeterministic(t *testing.T) {
	t.Parallel()

	for seed := uint64(0); seed < 20; seed++ {
		first := newSession(t, seed)
		_, firstSteps, _ := playWithBots(t, first, seed)

		second := newSession(t, seed)
		_, secondSteps, _ := playWithBots(t, second, seed)

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
		leads, steps, _ := playWithBots(t, played, seed)

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
func playWithBots(t *testing.T, session *singleplayer.Session, seed uint64) ([2]int, []step, matchStats) {
	t.Helper()

	var stats matchStats

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
	turns := 0
	for session.Phase() != singleplayer.PhaseFinished {
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
				switch action.(type) {
				case battle.SwitchAction:
					stats.switches++
				case battle.StruggleAction:
					stats.struggles++
				}
				steps = append(steps, step{side: side, action: action})
			}
			turns++
			if turns >= maxSmokeTurns {
				// 打ち切りはtestの安全装置。ここで落とさず未決着として数え、
				// 呼び出し側でまとめて扱う。
				stats.turns = turns
				stats.unresolved = true
				return leads, steps, stats
			}

		default:
			t.Fatalf("seed %d: 進められないphase: %s", seed, session.Phase())
		}
	}

	stats.turns = turns
	stats.outcome, _ = session.Outcome()
	return leads, steps, stats
}
