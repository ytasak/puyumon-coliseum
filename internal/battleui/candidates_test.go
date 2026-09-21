package battleui_test

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/battleui"
	"github.com/ytasak/puyumon-coliseum/internal/bot"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// maxTestTurns は1試合を打ち切るturn数。決着しないことを検出するための安全装置。
const maxTestTurns = 300

// sides は両sideを順に処理するための一覧。
var sides = [...]battle.Side{singleplayer.Player, singleplayer.Opponent}

// battleuiが出した候補は、対戦のルール上そのまま通る。
//
// 通常のturnの候補はResolver.ValidateActionで1件ずつ確かめる。
// 交代はValidateActionの対象外（phaseとreplacement要否を見ない）なので、
// stateを複製してResolveReplacementが受け付けるかで確かめる。
//
// UIがmechanicsを推測していないことを実際の対戦で見るためのtestで、
// これだけで「ルールを複製していない」ことの証明にはしない。
func TestOfferedCommandsAreAcceptedByTheEngine(t *testing.T) {
	t.Parallel()

	// sessionの乱数へ触れないよう、検証には別のresolverを使う。
	// ValidateActionは乱数を引かず、ResolveReplacementには複製したstateを渡す。
	probe := &battle.Resolver{Data: roster.Data(), RNG: battle.NewRand(0)}

	for seed := uint64(0); seed < 40; seed++ {
		session, err := singleplayer.NewSession(singleplayer.Config{Seed: seed})
		if err != nil {
			t.Fatalf("seed %d: NewSession()に失敗: %v", seed, err)
		}

		bots := map[battle.Side]*bot.Bot{
			singleplayer.Player:   bot.New(roster.Data(), session.BotSeed()+1),
			singleplayer.Opponent: bot.New(roster.Data(), session.BotSeed()),
		}
		for _, side := range sides {
			own, _ := session.Team(side)
			opponent, _ := session.Team(side.Opponent())
			if err := session.SubmitLead(side, bots[side].Lead(own, opponent)); err != nil {
				t.Fatalf("seed %d: %s のleadが拒否された: %v", seed, side, err)
			}
		}

		for turns := 0; session.Phase() != singleplayer.PhaseFinished; {
			state, ok := session.State()
			if !ok {
				t.Fatalf("seed %d: stateが取れない", seed)
			}
			checkCandidates(t, probe, session, state, seed)

			switch session.Phase() {
			case singleplayer.PhaseReplacement:
				for _, side := range sides {
					if !state.NeedsReplacement(side) {
						continue
					}
					if _, err := session.SubmitReplacement(side, bots[side].Replacement(state, side)); err != nil {
						t.Fatalf("seed %d: %s のreplacementが拒否された: %v", seed, side, err)
					}
					state, _ = session.State()
				}

			case singleplayer.PhaseBattle:
				for _, side := range sides {
					if _, err := session.SubmitAction(side, bots[side].Action(state, side)); err != nil {
						t.Fatalf("seed %d: %s のActionが拒否された: %v", seed, side, err)
					}
				}
				turns++
				if turns >= maxTestTurns {
					t.Fatalf("seed %d: %d turnで決着しなかった", seed, maxTestTurns)
				}

			default:
				t.Fatalf("seed %d: 進められないphase: %s", seed, session.Phase())
			}
		}
	}
}

// checkCandidates はviewerへ出している候補がすべて通ることを確かめる。
func checkCandidates(t *testing.T, probe *battle.Resolver, session *singleplayer.Session, state battle.BattleState, seed uint64) {
	t.Helper()

	viewer := singleplayer.Player
	teams := [2][battle.TeamSize]battle.Pokemon{}
	for _, side := range sides {
		teams[side], _ = session.Team(side)
	}

	view, ok := battleui.Snapshot(battleui.Input{
		Phase:   session.Phase(),
		Teams:   teams,
		State:   state,
		Started: true,
	}, viewer)
	if !ok {
		t.Fatalf("seed %d: Snapshotが失敗した", seed)
	}

	switch view.Commands.Kind {
	case battleui.CommandChooseAction:
		for slot, move := range view.Commands.Moves {
			if move.Disabled {
				continue
			}
			if err := probe.ValidateAction(state, viewer, battle.MoveAction{Slot: slot}); err != nil {
				t.Fatalf("seed %d turn %d: 出した技 slot %d が拒否された: %v", seed, state.Turn, slot, err)
			}
		}
		if view.Commands.Struggle {
			if err := probe.ValidateAction(state, viewer, battle.StruggleAction{}); err != nil {
				t.Fatalf("seed %d turn %d: Struggleが拒否された: %v", seed, state.Turn, err)
			}
		}
		for _, option := range view.Commands.Switches {
			if err := probe.ValidateAction(state, viewer, battle.SwitchAction{Target: option.Index}); err != nil {
				t.Fatalf("seed %d turn %d: 交代先 %d が拒否された: %v", seed, state.Turn, option.Index, err)
			}
		}

	case battleui.CommandChooseReplacement:
		for _, option := range view.Commands.Switches {
			// stateは値なので複製して渡す。sessionの状態は進めない。
			if _, _, err := probe.ResolveReplacement(state, viewer, battle.SwitchAction{Target: option.Index}); err != nil {
				t.Fatalf("seed %d turn %d: replacement先 %d が拒否された: %v", seed, state.Turn, option.Index, err)
			}
		}

	case battleui.CommandWaiting, battleui.CommandFinished:
		// viewerの入力を受け付けない状態。確かめる候補が無い。

	default:
		t.Fatalf("seed %d: 対戦中に出ないはずのcommand: %s", seed, view.Commands.Kind)
	}
}
