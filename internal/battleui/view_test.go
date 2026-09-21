package battleui

import (
	"reflect"
	"testing"

	"github.com/ytasak/puyumon-coliseum/internal/battle"
	"github.com/ytasak/puyumon-coliseum/internal/roster"
	"github.com/ytasak/puyumon-coliseum/internal/singleplayer"
)

// BattleStateからHP・状態・PPが表示用の値になる。
func TestSnapshotShowsHPStatusAndPP(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.State.Players[battle.Player1].Team[0].CurrentHP = 40
	in.State.Players[battle.Player1].Team[0].Status = battle.Poison
	in.State.Players[battle.Player1].Team[0].Moves[1].PP = 3

	view := snapshotOrFatal(t, in, battle.Player1)

	active := view.You.Team[view.You.Active]
	if active.HP != 40 {
		t.Errorf("HP = %d, want 40", active.HP)
	}
	if active.MaxHP != in.State.Players[battle.Player1].Team[0].Stats.HP {
		t.Errorf("MaxHP = %d, want %d", active.MaxHP, in.State.Players[battle.Player1].Team[0].Stats.HP)
	}
	if active.Status != StatusPoison {
		t.Errorf("Status = %s, want %s", active.Status, StatusPoison)
	}
	if !active.Active {
		t.Error("場に出ている1体のActiveがfalse")
	}
	if active.Emoji == "" || active.Species == "" {
		t.Errorf("display resourceが空: species=%q emoji=%q", active.Species, active.Emoji)
	}
	if got := active.Moves[1]; got.PP != 3 || got.MaxPP == 0 || got.Move == "" {
		t.Errorf("move view = %+v", got)
	}

	// 相手側も同じ範囲の情報は出す。
	foe := view.Foe.Team[view.Foe.Active]
	if foe.MaxHP == 0 || foe.Level == 0 || foe.Species == "" {
		t.Errorf("相手のview = %+v", foe)
	}
}

// ひんしは状態異常より優先して表示する。
func TestFaintedBeatsStatusInTheView(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	target := &in.State.Players[battle.Player1].Team[0]
	target.Status = battle.Sleep
	target.CurrentHP = 0

	view := snapshotOrFatal(t, in, battle.Player1)

	got := view.You.Team[0]
	if got.Status != StatusFainted {
		t.Errorf("Status = %s, want %s（domainのstatusはSleepのまま）", got.Status, StatusFainted)
	}
	if !got.Fainted {
		t.Error("Faintedがfalse")
	}
}

// lead選択前はどちらのactiveも決まっていない。
func TestActiveIsUnsetBeforeLeadSelection(t *testing.T) {
	t.Parallel()

	in := leadInput(t)
	view := snapshotOrFatal(t, in, battle.Player1)

	if view.You.Active != NoActive || view.Foe.Active != NoActive {
		t.Errorf("active = %d / %d, want %d", view.You.Active, view.Foe.Active, NoActive)
	}
	for _, pokemon := range view.You.Team {
		if pokemon.Active {
			t.Error("lead選択前に場に出ている扱いの個体がある")
		}
	}
	// 配布3体は選択前から見えている。
	if view.You.Team[0].Species == "" || view.Foe.Team[2].Species == "" {
		t.Error("配布3体が見えていない")
	}
}

// 相手の技構成と残りPPは表に出さない。
func TestOpponentMovesAndPPAreNotExposed(t *testing.T) {
	t.Parallel()

	for _, viewer := range []battle.Side{battle.Player1, battle.Player2} {
		base := battleInput(t)
		changed := battleInput(t)

		// 相手側の技IDと残りPPだけを差し替える。
		foe := viewer.Opponent()
		changed.State.Players[foe].Team[0].Moves[0].Move = roster.MoveNeedles
		changed.State.Players[foe].Team[0].Moves[1].PP = 0
		changed.State.Players[foe].Team[1].Moves[2].PP = 1

		if reflect.DeepEqual(base.State.Players[foe].Team, changed.State.Players[foe].Team) {
			t.Fatal("technical: 相手の技を差し替えられていない")
		}

		before := snapshotOrFatal(t, base, viewer)
		after := snapshotOrFatal(t, changed, viewer)
		if !reflect.DeepEqual(before, after) {
			t.Errorf("viewer %s: 相手の技構成・PPでviewが変わった", viewer)
		}

		// 自分側は4技とPPが見える。
		own := before.You.Team[before.You.Active]
		for slot, move := range own.Moves {
			if move.Move == "" || move.MaxPP == 0 {
				t.Errorf("viewer %s: 自分のslot %d が見えていない: %+v", viewer, slot, move)
			}
		}
	}
}

// 決着の結果はSnapshotのResultで表す。
func TestResultReflectsTheOutcome(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name   string
		status battle.Status
		want   Result
	}{
		{name: "未決着", status: battle.Ongoing, want: Result{}},
		{name: "player1の勝ち", status: battle.Player1Won, want: Result{Decided: true, Winner: battle.Player1}},
		{name: "player2の勝ち", status: battle.Player2Won, want: Result{Decided: true, Winner: battle.Player2}},
		{name: "引き分け", status: battle.Draw, want: Result{Decided: true, Draw: true}},
	}
	for _, tc := range tests {
		in := battleInput(t)
		in.State.Status = tc.status
		if tc.status != battle.Ongoing {
			in.Phase = singleplayer.PhaseFinished
		}

		if got := snapshotOrFatal(t, in, battle.Player1).Result; got != tc.want {
			t.Errorf("%s: Result = %+v, want %+v", tc.name, got, tc.want)
		}
	}
}

// 決着後にSnapshotを何度読んでも同じ値で、cueは生まれない。
func TestSnapshotIsIdempotentAfterTheBattleEnds(t *testing.T) {
	t.Parallel()

	in := battleInput(t)
	in.Phase = singleplayer.PhaseFinished
	in.State.Status = battle.Player1Won

	first := snapshotOrFatal(t, in, battle.Player1)
	for i := 0; i < 5; i++ {
		if got := snapshotOrFatal(t, in, battle.Player1); !reflect.DeepEqual(got, first) {
			t.Fatalf("%d回目のSnapshotが違う", i+2)
		}
	}
	if !first.Result.Decided || first.Commands.Kind != CommandFinished {
		t.Errorf("決着後のview = %+v / %+v", first.Result, first.Commands.Kind)
	}
}

// 両者のどちらでもないsideは受け付けない。
func TestSnapshotRejectsAnUnknownViewer(t *testing.T) {
	t.Parallel()

	if _, ok := Snapshot(battleInput(t), battle.Side(9)); ok {
		t.Error("不正なsideが通ってしまう")
	}
}

// leadInput はlead選択中の入力を返す。
func leadInput(t *testing.T) Input {
	t.Helper()

	return Input{
		Phase: singleplayer.PhaseLeadSelection,
		Teams: [2][battle.TeamSize]battle.Pokemon{
			teamOf3(t, roster.SpeciesBull, roster.SpeciesStar, roster.SpeciesJolt),
			teamOf3(t, roster.SpeciesPalm, roster.SpeciesWhale, roster.SpeciesCharm),
		},
	}
}

// battleInput は対戦中の入力を返す。
func battleInput(t *testing.T) Input {
	t.Helper()

	in := leadInput(t)
	state, err := battle.NewBattleState(in.Teams[battle.Player1], in.Teams[battle.Player2])
	if err != nil {
		t.Fatalf("NewBattleState()に失敗: %v", err)
	}
	in.Phase = singleplayer.PhaseBattle
	in.State = state
	in.Started = true
	return in
}

// teamOf3 は3体のteamを155の配分で作る。
func teamOf3(t *testing.T, ids ...battle.SpeciesID) [battle.TeamSize]battle.Pokemon {
	t.Helper()

	var trio [battle.TeamSize]battle.SpeciesID
	copy(trio[:], ids)

	team, err := roster.NewTeam(trio)
	if err != nil {
		t.Fatalf("NewTeam(%v)に失敗: %v", ids, err)
	}
	return team
}

// snapshotOrFatal はviewを取り出す。
func snapshotOrFatal(t *testing.T, in Input, viewer battle.Side) View {
	t.Helper()

	view, ok := Snapshot(in, viewer)
	if !ok {
		t.Fatalf("Snapshot(viewer %s)が失敗した", viewer)
	}
	return view
}
