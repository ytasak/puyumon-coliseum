package battleui

import (
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// lead選択中は3体すべてを選択肢にする。
func TestLeadSelectionOffersEveryDealtPokemon(t *testing.T) {
	t.Parallel()

	commands := snapshotOrFatal(t, leadInput(t), battle.Player1).Commands

	if commands.Kind != CommandChooseLead {
		t.Fatalf("Kind = %s, want %s", commands.Kind, CommandChooseLead)
	}
	if len(commands.Leads) != battle.TeamSize {
		t.Fatalf("lead候補が %d 件（%d 件のはず）", len(commands.Leads), battle.TeamSize)
	}
	for i, option := range commands.Leads {
		if option.Index != i || option.Species == "" {
			t.Errorf("lead候補 %d = %+v", i, option)
		}
	}
	if len(commands.Moves) != 0 || len(commands.Switches) != 0 {
		t.Error("lead選択中に技や交代の選択肢が出ている")
	}
}

// PPの尽きた技は選択肢から消さず、無効として見せる。
func TestZeroPPMoveIsDisabled(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.State.Players[battle.Player1].Team[0].Moves[2].PP = 0

	commands := snapshotOrFatal(t, in, battle.Player1).Commands

	if commands.Kind != CommandChooseAction {
		t.Fatalf("Kind = %s, want %s", commands.Kind, CommandChooseAction)
	}
	if len(commands.Moves) != battle.MoveSlots {
		t.Fatalf("技が %d 件（%d 件のはず）", len(commands.Moves), battle.MoveSlots)
	}
	if !commands.Moves[2].Disabled {
		t.Error("PP 0 の技がDisabledになっていない")
	}
	if commands.Moves[0].Disabled {
		t.Error("PPの残る技までDisabledになっている")
	}
	if commands.Struggle {
		t.Error("まだ使える技があるのにStruggleを出している")
	}
}

// 技が1つも使えなければStruggleを選べる状態にする。
func TestStruggleCommandWhenNoPPLeft(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	for i := range in.State.Players[battle.Player1].Team[0].Moves {
		in.State.Players[battle.Player1].Team[0].Moves[i].PP = 0
	}

	commands := snapshotOrFatal(t, in, battle.Player1).Commands

	if !commands.Struggle {
		t.Error("全PP 0 なのにStruggleを出していない")
	}
	for slot, move := range commands.Moves {
		if !move.Disabled {
			t.Errorf("slot %d がDisabledになっていない", slot)
		}
	}
}

// 交代先は生存している控えだけ。
func TestSwitchOptionsAreLivingReserves(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.State.Players[battle.Player1].Team[1].CurrentHP = 0

	commands := snapshotOrFatal(t, in, battle.Player1).Commands

	if len(commands.Switches) != 1 {
		t.Fatalf("交代候補が %d 件（1件のはず）", len(commands.Switches))
	}
	if commands.Switches[0].Index != 2 {
		t.Errorf("交代候補が %d（2のはず）", commands.Switches[0].Index)
	}
}

// 戦闘不能のあとは控えだけを出し、技は出さない。
func TestReplacementPhaseOffersOnlyReserves(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.Phase = singleplayer.PhaseReplacement
	in.State.Players[battle.Player1].Team[0].CurrentHP = 0

	commands := snapshotOrFatal(t, in, battle.Player1).Commands

	if commands.Kind != CommandChooseReplacement {
		t.Fatalf("Kind = %s, want %s", commands.Kind, CommandChooseReplacement)
	}
	if len(commands.Moves) != 0 {
		t.Error("replacement中に技の選択肢が出ている")
	}
	if commands.Struggle {
		t.Error("replacement中にStruggleを出している")
	}
	if len(commands.Switches) == 0 {
		t.Error("交代先が出ていない")
	}
}

// 倒れたのが相手だけなら、こちらは待つ。
func TestWaitingWhenOnlyTheOpponentReplaces(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.Phase = singleplayer.PhaseReplacement
	in.State.Players[battle.Player2].Team[0].CurrentHP = 0

	if got := snapshotOrFatal(t, in, battle.Player1).Commands.Kind; got != CommandWaiting {
		t.Errorf("Kind = %s, want %s", got, CommandWaiting)
	}
	// 倒れた側には交代の選択肢が出る。
	if got := snapshotOrFatal(t, in, battle.Player2).Commands.Kind; got != CommandChooseReplacement {
		t.Errorf("倒れた側のKind = %s, want %s", got, CommandChooseReplacement)
	}
}

// 自分が出したあとは相手待ちになり、二重に入力できない。
func TestPendingSuppressesInput(t *testing.T) {
	t.Parallel()

	lead := leadInput(t)
	lead.Pending.Lead = true
	if got := snapshotOrFatal(t, lead, battle.Player1).Commands.Kind; got != CommandWaiting {
		t.Errorf("lead提出後のKind = %s, want %s", got, CommandWaiting)
	}

	action := battleInput(t)
	action.Pending.Action = true
	if got := snapshotOrFatal(t, action, battle.Player1).Commands.Kind; got != CommandWaiting {
		t.Errorf("action提出後のKind = %s, want %s", got, CommandWaiting)
	}
}

// 対応しないphaseのpendingは参照しない。
//
// これはphase違いの取り違えを防ぐだけで、同じphaseに残った古い値までは救わない。
// 解決やphase遷移でのclearはcallerの責務。
func TestStalePendingDoesNotLeakAcrossPhases(t *testing.T) {
	t.Parallel()

	// lead提出のpendingが残ったままBattleへ移っても、行動は選べる。
	inBattle := battleInput(t)
	inBattle.Pending.Lead = true
	if got := snapshotOrFatal(t, inBattle, battle.Player1).Commands.Kind; got != CommandChooseAction {
		t.Errorf("Kind = %s, want %s", got, CommandChooseAction)
	}

	// 行動提出のpendingが残ったままreplacementへ移っても、控えを選べる。
	inReplacement := battleInput(t)
	inReplacement.Phase = singleplayer.PhaseReplacement
	inReplacement.State.Players[battle.Player1].Team[0].CurrentHP = 0
	inReplacement.Pending.Action = true
	if got := snapshotOrFatal(t, inReplacement, battle.Player1).Commands.Kind; got != CommandChooseReplacement {
		t.Errorf("Kind = %s, want %s", got, CommandChooseReplacement)
	}
}

// 決着後は新しい対戦を始められることを伝える。
func TestFinishedOffersANewMatch(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.Phase = singleplayer.PhaseFinished
	in.State.Status = battle.Player1Won

	commands := snapshotOrFatal(t, in, battle.Player1).Commands

	if commands.Kind != CommandFinished {
		t.Fatalf("Kind = %s, want %s", commands.Kind, CommandFinished)
	}
	if !commands.NewMatch {
		t.Error("NewMatchがfalse")
	}
	if len(commands.Moves) != 0 || len(commands.Switches) != 0 || len(commands.Leads) != 0 {
		t.Error("決着後に対戦中の選択肢が出ている")
	}
}
